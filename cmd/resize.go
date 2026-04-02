package cmd

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/image/bmp"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/tiff"
)

var resizeSize string
var resizeDir string
var resizeSuffix string
var resizeFilter string

var resizeCmd = &cobra.Command{
	Use:   "resize <pattern...>",
	Short: "Resize textures matching a glob pattern to a given size",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runResize,
}

func init() {
	resizeCmd.Flags().StringVar(&resizeSize, "size", "", "Target size in WxH format, e.g. 1024x1024 (required)")
	resizeCmd.Flags().StringVar(&resizeDir, "dir", ".", "Directory to search in (default: current directory)")
	resizeCmd.Flags().StringVar(&resizeSuffix, "suffix", "", "Suffix to append to output filenames; if omitted, files are resized in place")
	resizeCmd.Flags().StringVar(&resizeFilter, "filter", "bilinear", "Resampling filter: nearest, bilinear, catmull-rom")
	if err := resizeCmd.MarkFlagRequired("size"); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(resizeCmd)
}

func parseSize(s string) (int, int, error) {
	parts := strings.SplitN(strings.ToLower(s), "x", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid size %q: expected WxH format (e.g. 1024x1024)", s)
	}
	w, err := strconv.Atoi(parts[0])
	if err != nil || w <= 0 {
		return 0, 0, fmt.Errorf("invalid width in size %q", s)
	}
	h, err := strconv.Atoi(parts[1])
	if err != nil || h <= 0 {
		return 0, 0, fmt.Errorf("invalid height in size %q", s)
	}
	return w, h, nil
}

func parseFilter(s string) (xdraw.Interpolator, error) {
	switch strings.ToLower(s) {
	case "nearest":
		return xdraw.NearestNeighbor, nil
	case "bilinear":
		return xdraw.BiLinear, nil
	case "catmull-rom", "catmullrom", "cubic":
		return xdraw.CatmullRom, nil
	default:
		return nil, fmt.Errorf("unsupported filter %q: must be nearest, bilinear, or catmull-rom", s)
	}
}

func runResize(_ *cobra.Command, args []string) error {
	w, h, err := parseSize(resizeSize)
	if err != nil {
		return err
	}

	interp, err := parseFilter(resizeFilter)
	if err != nil {
		return err
	}

	var matches []string
	for _, pattern := range args {
		m, err := filepath.Glob(filepath.Join(resizeDir, pattern))
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
		if err := resizeFile(src, w, h, resizeSuffix, interp); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "skipping %s: %v\n", src, err)
		}
	}
	return nil
}

func resizeFile(src string, w, h int, suffix string, interp xdraw.Interpolator) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	img, format, err := image.Decode(f)
	_ = f.Close()
	if err != nil {
		return fmt.Errorf("decode failed: %w", err)
	}

	resized := image.NewRGBA(image.Rect(0, 0, w, h))
	interp.Scale(resized, resized.Bounds(), img, img.Bounds(), xdraw.Src, nil)

	ext := filepath.Ext(src)
	dst := strings.TrimSuffix(src, ext) + suffix + ext

	tmp, err := os.CreateTemp(filepath.Dir(src), ".texutil-resize-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	encErr := encodeFormat(tmp, resized, format)
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

	if suffix == "" {
		fmt.Printf("%s (resized in place)\n", dst)
	} else {
		fmt.Printf("%s -> %s\n", src, dst)
	}
	return nil
}

func encodeFormat(w io.Writer, img image.Image, format string) error {
	switch format {
	case "png":
		return png.Encode(w, img)
	case "jpeg":
		return jpeg.Encode(w, img, nil)
	case "tiff":
		return tiff.Encode(w, img, nil)
	case "bmp":
		return bmp.Encode(w, img)
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}
