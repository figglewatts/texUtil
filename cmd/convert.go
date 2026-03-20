package cmd

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
)

var convertTo string
var convertDir string

var convertCmd = &cobra.Command{
	Use:   "convert <pattern...>",
	Short: "Convert textures matching a glob pattern to a given format",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runConvert,
}

func init() {
	convertCmd.Flags().StringVar(&convertTo, "to", "", "Target format: png, jpg, tiff, bmp (required)")
	convertCmd.Flags().StringVar(&convertDir, "dir", ".", "Directory to search in (default: current directory)")
	if err := convertCmd.MarkFlagRequired("to"); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(convertCmd)
}

func runConvert(_ *cobra.Command, args []string) error {
	ext := strings.ToLower(convertTo)
	switch ext {
	case "jpg":
		ext = "jpeg"
	case "tif":
		ext = "tiff"
	case "png", "jpeg", "tiff", "bmp":
	default:
		return fmt.Errorf("unsupported format %q: must be png, jpg/jpeg, tif/tiff, or bmp", convertTo)
	}

	var matches []string
	for _, pattern := range args {
		m, err := filepath.Glob(filepath.Join(convertDir, pattern))
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
		if err := convertFile(src, ext); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "skipping %s: %v\n", src, err)
		}
	}
	return nil
}

func convertFile(src, ext string) (retErr error) {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer closeWithErr(f, "source file", &retErr)

	img, _, err := image.Decode(f)
	if err != nil {
		return fmt.Errorf("decode failed: %w", err)
	}

	outExt := ext
	dst := strings.TrimSuffix(src, filepath.Ext(src)) + "." + outExt

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		closeWithErr(out, "output file", &retErr)
		if retErr != nil {
			if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
				_, _ = fmt.Fprintf(os.Stderr, "warning: failed to remove partial output %s: %v\n", dst, err)
			}
		}
	}()

	switch ext {
	case "png":
		err = png.Encode(out, img)
	case "jpeg":
		err = jpeg.Encode(out, img, nil)
	case "tiff":
		err = tiff.Encode(out, img, nil)
	case "bmp":
		err = bmp.Encode(out, img)
	}
	if err != nil {
		return fmt.Errorf("encode failed: %w", err)
	}

	fmt.Printf("%s -> %s\n", src, dst)
	return nil
}
