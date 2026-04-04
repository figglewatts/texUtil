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

var applyAOIntensity float64
var applyAOSuffix string
var applyAODir string
var applyAOAOSuffix string

var applyAOCmd = &cobra.Command{
	Use:   "applyao <pattern...>",
	Short: "Multiply colors from an AO texture into textures matching a glob pattern",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runApplyAO,
}

func init() {
	applyAOCmd.Flags().Float64Var(&applyAOIntensity, "intensity", 0.75, "Intensity of the AO effect (0.0 to 1.0, 0 is invalid)")
	applyAOCmd.Flags().StringVar(&applyAOAOSuffix, "aosuffix", "_ao", "Suffix to find the AO file (e.g. '_ao' for 'base.png' -> 'base_ao.png')")
	applyAOCmd.Flags().StringVar(&applyAOSuffix, "suffix", "", "Suffix to append to output filenames; if omitted, files are modified in place")
	applyAOCmd.Flags().StringVar(&applyAODir, "dir", ".", "Directory to search in (default: current directory)")

	rootCmd.AddCommand(applyAOCmd)
}

func runApplyAO(_ *cobra.Command, args []string) error {
	if applyAOIntensity <= 0 || applyAOIntensity > 1 {
		return fmt.Errorf("intensity must be between 0 (exclusive) and 1 (inclusive)")
	}

	var matches []string
	for _, pattern := range args {
		m, err := filepath.Glob(filepath.Join(applyAODir, pattern))
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
		// Skip files that are themselves AO files (if the suffix matches)
		if strings.HasSuffix(strings.TrimSuffix(src, filepath.Ext(src)), applyAOAOSuffix) {
			continue
		}

		if err := applyAOToFile(src); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "skipping %s: %v\n", src, err)
		}
	}
	return nil
}

func applyAOToFile(src string) (retErr error) {
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

	// 2. Find and decode AO image
	ext := filepath.Ext(src)
	base := strings.TrimSuffix(src, ext)

	// Try common texture suffixes to replace: _albedo, _diffuse, _basecolor, _col, _base
	commonSuffixes := []string{"_albedo", "_diffuse", "_basecolor", "_col", "_base", "_color", "_Color", "_Albedo", "_Diffuse", "_BaseColor"}
	aoPath := ""
	for _, suffix := range commonSuffixes {
		if strings.HasSuffix(base, suffix) {
			aoPath = strings.TrimSuffix(base, suffix) + applyAOAOSuffix + ext
			if _, err := os.Stat(aoPath); err == nil {
				break
			}
			aoPath = ""
		}
	}

	// Fallback to appending if no common suffix was found or replacement didn't exist
	if aoPath == "" {
		aoPath = base + applyAOAOSuffix + ext
	}

	af, err := os.Open(aoPath)
	if err != nil {
		return fmt.Errorf("could not open AO file %s: %w", aoPath, err)
	}
	aoImg, _, err := image.Decode(af)
	_ = af.Close()
	if err != nil {
		return fmt.Errorf("decode AO image failed: %w", err)
	}

	// 3. Process image
	bounds := img.Bounds()
	aoBounds := aoImg.Bounds()
	if bounds.Dx() != aoBounds.Dx() || bounds.Dy() != aoBounds.Dy() {
		return fmt.Errorf("image sizes do not match: main %dx%d, AO %dx%d", bounds.Dx(), bounds.Dy(), aoBounds.Dx(), aoBounds.Dy())
	}

	outImg := image.NewRGBA(bounds)
	// Draw the original image onto outImg to preserve alpha if any (though we'll calculate it below)
	draw.Draw(outImg, bounds, img, bounds.Min, draw.Src)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)
			ac := aoImg.At(x, y)

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
			// Typically AO is grayscale, but let's average or just use one channel
			// Most AO maps are grayscale, so R should be same as G and B.
			// Let's use average to be safe.
			fao := (far + fag + fab) / 3.0

			// Multiply logic with intensity:
			// result = color * (1 - intensity + intensity * ao)
			// This means if intensity is 1, result = color * ao.
			// If intensity is 0, result = color (but 0 is invalid).
			factor := 1.0 - applyAOIntensity + (applyAOIntensity * fao)

			nr := uint16(fr * factor * 65535.0)
			ng := uint16(fg * factor * 65535.0)
			nb := uint16(fb * factor * 65535.0)
			na := uint16(fa * 65535.0)

			outImg.SetRGBA64(x, y, color.RGBA64{R: nr, G: ng, B: nb, A: na})
		}
	}

	// 4. Save result
	dst := base + applyAOSuffix + ext
	tmp, err := os.CreateTemp(filepath.Dir(src), ".texutil-applyao-*")
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

	if applyAOSuffix == "" {
		fmt.Printf("%s (AO applied in place)\n", dst)
	} else {
		fmt.Printf("%s -> %s\n", src, dst)
	}

	return nil
}
