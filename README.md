# texUtil

A CLI utility for working with texture files.

## Installation

### Download a release

Download the latest binary for your platform from the [releases page](../../releases/latest) and place it somewhere on your `PATH`.

### Build from source

Requires Go 1.26+.

```sh
git clone https://github.com/figglewatts/texUtil.git
cd texUtil
make build
```

The `texutil` binary will be built in the current directory.

## Usage

```
texUtil <command> [flags]
```

### Commands

#### `convert`

Convert texture files matching a glob pattern to a different image format.

```
texUtil convert <pattern...> --to <format> [--dir <directory>]
```

**Flags**

| Flag | Description | Default |
|------|-------------|---------|
| `--to` | Target format: `png`, `jpg`/`jpeg`, `tif`/`tiff`, `bmp` | *(required)* |
| `--dir` | Directory to search in | `.` (current directory) |

**Examples**

```sh
# Convert all TIFFs in the current directory to PNG
texUtil convert '*.tif' --to png

# Convert all PNGs in a specific directory to JPEG
texUtil convert '*.png' --to jpg --dir /path/to/textures

# Shell glob expansion also works
texUtil convert --to png -- textures/*.tif
```

Output files are written alongside the source files with the new extension. Files that cannot be decoded are skipped with an error printed to stderr.

## Supported Formats

| Format | Extensions |
|--------|------------|
| PNG    | `.png` |
| JPEG   | `.jpg`, `.jpeg` |
| TIFF   | `.tif`, `.tiff` |
| BMP    | `.bmp` |

## Building for all platforms

```sh
make build-all
```

Binaries are written to `dist/` with the naming convention `texutil-{os}-{arch}[.exe]`.

Individual platforms can also be targeted:

```sh
make linux/amd64
make darwin/arm64
make windows/amd64
```
