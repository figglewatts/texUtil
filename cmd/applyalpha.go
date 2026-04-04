package cmd

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
)

var applyAlphaSuffix string
var applyAlphaDir string
var applyAlphaMapSuffix string

var applyAlphaCmd = &cobra.Command{
	Use:   "applyalpha <pattern...>",
	Short: "Apply an alpha/opacity map to textures matching a glob pattern",
	Long:  "Bakes an alpha/opacity map into the diffuse texture. The alpha map is expected to be grayscale.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runApplyAlpha,
}

func init() {
	applyAlphaCmd.Flags().StringVar(&applyAlphaMapSuffix, "alphasuffix", "_alpha", "Suffix to find the alpha/opacity file (e.g. '_alpha' for 'base.png' -> 'base_alpha.png')")
	applyAlphaCmd.Flags().StringVar(&applyAlphaSuffix, "suffix", "", "Suffix to append to output filenames; if omitted, files are modified in place")
	applyAlphaCmd.Flags().StringVar(&applyAlphaDir, "dir", ".", "Directory to search in (default: current directory)")

	rootCmd.AddCommand(applyAlphaCmd)
}

func runApplyAlpha(_ *cobra.Command, args []string) error {
	var matches []string
	for _, pattern := range args {
		m, err := filepath.Glob(filepath.Join(applyAlphaDir, pattern))
		if err != nil {
			return fmt.Errorf("invalid pattern %q: %w", pattern, err)
		}
		matches = append(matches, m...)
	}

	if len(matches) == 0 {
		fmt.Println("No files matched the pattern.")
		return nil
	}

	for _, src := range matches {
		// Skip files that are themselves alpha files (if the suffix matches)
		if strings.HasSuffix(strings.TrimSuffix(src, filepath.Ext(src)), applyAlphaMapSuffix) {
			continue
		}

		if err := applyAlphaToFile(src); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "skipping %s: %v\n", src, err)
		}
	}
	return nil
}

func applyAlphaToFile(src string) (retErr error) {
	// 1. Decode main image
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	img, format, err := image.Decode(f)
	_ = f.Close()
	if err != nil {
		return fmt.Errorf("decode main image failed: %w", err)
	}

	// 2. Find and decode Alpha image
	ext := filepath.Ext(src)
	base := strings.TrimSuffix(src, ext)

	// Try common texture suffixes to replace
	commonSuffixes := []string{"_albedo", "_diffuse", "_basecolor", "_col", "_base", "_color", "_Color", "_Albedo", "_Diffuse", "_BaseColor"}
	alphaPath := ""
	for _, suffix := range commonSuffixes {
		if strings.HasSuffix(base, suffix) {
			alphaPath = strings.TrimSuffix(base, suffix) + applyAlphaMapSuffix + ext
			if _, err := os.Stat(alphaPath); err == nil {
				break
			}
			alphaPath = ""
		}
	}

	// Fallback to appending if no common suffix was found or replacement didn't exist
	if alphaPath == "" {
		alphaPath = base + applyAlphaMapSuffix + ext
	}

	af, err := os.Open(alphaPath)
	if err != nil {
		return fmt.Errorf("could not open alpha file %s: %w", alphaPath, err)
	}
	alphaImg, _, err := image.Decode(af)
	_ = af.Close()
	if err != nil {
		return fmt.Errorf("decode alpha image failed: %w", err)
	}

	// 3. Process image
	bounds := img.Bounds()
	alphaBounds := alphaImg.Bounds()
	if bounds.Dx() != alphaBounds.Dx() || bounds.Dy() != alphaBounds.Dy() {
		return fmt.Errorf("image sizes do not match: main %dx%d, alpha %dx%d", bounds.Dx(), bounds.Dy(), alphaBounds.Dx(), alphaBounds.Dy())
	}

	outImg := image.NewRGBA(bounds)
	// Draw the original image onto outImg
	draw.Draw(outImg, bounds, img, bounds.Min, draw.Src)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)
			ac := alphaImg.At(x, y)

			r, g, b, a := c.RGBA()
			ar, ag, ab, _ := ac.RGBA()

			// Scale down to 0-1 range
			fr := float64(r) / 65535.0
			fg := float64(g) / 65535.0
			fb := float64(b) / 65535.0
			fa := float64(a) / 65535.0

			far := float64(ar) / 65535.0
			fag := float64(ag) / 65535.0
			fab := float64(ab) / 65535.0
			// Alpha map is typically grayscale
			falpha := (far + fag + fab) / 3.0

			// Bake alpha: result alpha = original alpha * alpha map value
			newAlpha := fa * falpha

			nr := uint16(fr * falpha * 65535.0)
			ng := uint16(fg * falpha * 65535.0)
			nb := uint16(fb * falpha * 65535.0)
			na := uint16(newAlpha * 65535.0)

			outImg.SetRGBA64(x, y, color.RGBA64{R: nr, G: ng, B: nb, A: na})
		}
	}

	// 4. Save result
	dst := base + applyAlphaSuffix + ext
	tmp, err := os.CreateTemp(filepath.Dir(src), ".texutil-applyalpha-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	encErr := encodeFormat(tmp, outImg, format)
	closeErr := tmp.Close()
	if encErr != nil || closeErr != nil {
		_ = os.Remove(tmpPath)
		if encErr != nil {
			return fmt.Errorf("encode failed: %w", encErr)
		}
		return fmt.Errorf("closing temp file: %w", closeErr)
	}

	if err := os.Rename(tmpPath, dst); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("finalizing output: %w", err)
	}

	if applyAlphaSuffix == "" {
		fmt.Printf("%s (alpha applied in place)\n", dst)
	} else {
		fmt.Printf("%s -> %s\n", src, dst)
	}

	return nil
}
