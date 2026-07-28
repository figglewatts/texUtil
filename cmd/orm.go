package cmd

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
)

var (
	ormOcclusionSuffix string
	ormRoughnessSuffix string
	ormMetallicSuffix  string
	ormOutputSuffix    string
	ormDir             string
	ormDefaultO        float64
	ormDefaultR        float64
	ormDefaultM        float64
)

var ormCmd = &cobra.Command{
	Use:   "orm <pattern...>",
	Short: "Create ORM (Occlusion, Roughness, Metallic) maps from separate textures",
	Long:  "Packs Occlusion (R), Roughness (G), and Metallic (B) maps into a single texture. Matches files based on patterns and searches for corresponding maps using suffixes.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runORM,
}

func init() {
	ormCmd.Flags().StringVar(&ormOcclusionSuffix, "osuffix", "_ao", "Suffix for the occlusion map")
	ormCmd.Flags().StringVar(&ormRoughnessSuffix, "rsuffix", "_roughness", "Suffix for the roughness map")
	ormCmd.Flags().StringVar(&ormMetallicSuffix, "msuffix", "_metallic", "Suffix for the metallic map")
	ormCmd.Flags().StringVar(&ormOutputSuffix, "suffix", "_orm", "Suffix for the output ORM map")
	ormCmd.Flags().StringVar(&ormDir, "dir", ".", "Directory to search in")
	ormCmd.Flags().Float64Var(&ormDefaultO, "default-o", 1.0, "Default value for occlusion if map is missing (0.0 to 1.0)")
	ormCmd.Flags().Float64Var(&ormDefaultR, "default-r", 1.0, "Default value for roughness if map is missing (0.0 to 1.0)")
	ormCmd.Flags().Float64Var(&ormDefaultM, "default-m", 0.0, "Default value for metallic if map is missing (0.0 to 1.0)")

	rootCmd.AddCommand(ormCmd)
}

func runORM(_ *cobra.Command, args []string) error {
	var matches []string
	for _, pattern := range args {
		m, err := filepath.Glob(filepath.Join(ormDir, pattern))
		if err != nil {
			return fmt.Errorf("invalid pattern %q: %w", pattern, err)
		}
		matches = append(matches, m...)
	}

	if len(matches) == 0 {
		fmt.Println("No files matched the pattern.")
		return nil
	}

	// To avoid processing the same set multiple times, we'll keep track of base names
	processed := make(map[string]bool)

	for _, src := range matches {
		ext := filepath.Ext(src)
		base := strings.TrimSuffix(src, ext)

		// Determine the root base name by stripping any of the suffixes
		var root string
		commonSuffixes := []string{"_albedo", "_diffuse", "_basecolor", "_col", "_base", "_color", "_Color", "_Albedo", "_Diffuse", "_BaseColor"}
		foundSuffix := false

		for _, suffix := range append(commonSuffixes, ormOcclusionSuffix, ormRoughnessSuffix, ormMetallicSuffix) {
			if suffix != "" && strings.HasSuffix(base, suffix) {
				root = strings.TrimSuffix(base, suffix)
				foundSuffix = true
				break
			}
		}

		if !foundSuffix {
			root = base
		}

		if processed[root] {
			continue
		}
		processed[root] = true

		if err := ormToFile(root, ext); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "skipping %s: %v\n", root, err)
		}
	}
	return nil
}

func decodeImage(path string) (image.Image, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	return image.Decode(f)
}

func getGrayscale(img image.Image, x, y int) float64 {
	c := img.At(x, y)
	r, g, b, _ := c.RGBA()
	// Average for grayscale
	return (float64(r) + float64(g) + float64(b)) / (3.0 * 65535.0)
}

func ormToFile(root, ext string) error {
	oPath := root + ormOcclusionSuffix + ext
	rPath := root + ormRoughnessSuffix + ext
	mPath := root + ormMetallicSuffix + ext

	var oImg, rImg, mImg image.Image
	var format string
	var bounds image.Rectangle

	oImg, format, _ = decodeImage(oPath)
	rImg, _, _ = decodeImage(rPath)
	mImg, _, _ = decodeImage(mPath)

	if oImg != nil {
		bounds = oImg.Bounds()
	} else if rImg != nil {
		bounds = rImg.Bounds()
	} else if mImg != nil {
		bounds = mImg.Bounds()
	} else {
		// If no maps found, we still might want to create it if we have an albedo or something
		// But we need a size. Let's try to find any image with that root.
		matches, _ := filepath.Glob(root + "*" + ext)
		for _, m := range matches {
			if img, f, err := decodeImage(m); err == nil {
				bounds = img.Bounds()
				format = f
				break
			}
		}

		if bounds.Empty() {
			return fmt.Errorf("no maps found for %s and could not determine bounds", root)
		}
	}

	if format == "" {
		format = "png" // Default to png if no files were loaded (unlikely given above check)
	}

	if (oImg != nil && oImg.Bounds() != bounds) ||
		(rImg != nil && rImg.Bounds() != bounds) ||
		(mImg != nil && mImg.Bounds() != bounds) {
		return fmt.Errorf("image dimensions do not match for %s", root)
	}

	outImg := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			var o, r, m float64
			if oImg != nil {
				o = getGrayscale(oImg, x, y)
			} else {
				o = ormDefaultO
			}

			if rImg != nil {
				r = getGrayscale(rImg, x, y)
			} else {
				r = ormDefaultR
			}

			if mImg != nil {
				m = getGrayscale(mImg, x, y)
			} else {
				m = ormDefaultM
			}

			outImg.Set(x, y, color.RGBA{
				R: uint8(o * 255),
				G: uint8(r * 255),
				B: uint8(m * 255),
				A: 255,
			})
		}
	}

	dst := root + ormOutputSuffix + ext
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".texutil-orm-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	encErr := encodeFormat(tmp, outImg, format)
	closeErr := tmp.Close()
	if encErr != nil || closeErr != nil {
		_ = os.Remove(tmpPath)
		if encErr != nil {
			return encErr
		}
		return closeErr
	}

	if err := os.Rename(tmpPath, dst); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	fmt.Printf("Created ORM map: %s\n", dst)
	return nil
}
