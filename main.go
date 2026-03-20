package main

import (
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"

	"texUtil/cmd"
)

func main() {
	cmd.Execute()
}