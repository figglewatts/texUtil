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

var invertSuffix string
var invertDir string

var invertCmd = &cobra.Command{
	Use:   "invert <pattern...>",
	Short: "Invert colors of images matching a glob pattern",
	Long:  "Inverts the RGB channels of the image. Alpha channel is preserved.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runInvert,
}

func init() {
	invertCmd.Flags().StringVar(&invertSuffix, "suffix", "", "Suffix to append to output filenames; if omitted, files are modified in place")
	invertCmd.Flags().StringVar(&invertDir, "dir", ".", "Directory to search in (default: current directory)")

	rootCmd.AddCommand(invertCmd)
}

func runInvert(_ *cobra.Command, args []string) error {
	var matches []string
	for _, pattern := range args {
		m, err := filepath.Glob(filepath.Join(invertDir, pattern))
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
		if err := invertFile(src); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "skipping %s: %v\n", src, err)
		}
	}
	return nil
}

func invertFile(src string) error {
	// 1. Decode main image
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	img, format, err := image.Decode(f)
	_ = f.Close()
	if err != nil {
		return fmt.Errorf("decode image failed: %w", err)
	}

	// 2. Process image
	bounds := img.Bounds()
	outImg := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)
			r, g, b, a := c.RGBA()

			nr := 65535 - r
			ng := 65535 - g
			nb := 65535 - b

			outImg.SetRGBA64(x, y, color.RGBA64{
				R: uint16(nr),
				G: uint16(ng),
				B: uint16(nb),
				A: uint16(a),
			})
		}
	}

	// 3. Save result
	ext := filepath.Ext(src)
	base := strings.TrimSuffix(src, ext)
	dst := base + invertSuffix + ext

	tmp, err := os.CreateTemp(filepath.Dir(src), ".texutil-invert-*")
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

	if invertSuffix == "" {
		fmt.Printf("%s (inverted in place)\n", dst)
	} else {
		fmt.Printf("%s -> %s\n", src, dst)
	}

	return nil
}
