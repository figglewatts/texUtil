# texutil

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
texutil <command> [flags]
```

### Commands

#### `convert`

Convert texture files matching a glob pattern to a different image format.

```sh
texutil convert <pattern...> --to <format> [--dir <directory>] [--remove]
```

**Flags**

| Flag | Description | Default |
|------|-------------|---------|
| `--to` | Target format: `png`, `jpg`/`jpeg`, `tif`/`tiff`, `bmp` | *(required)* |
| `--dir` | Directory to search in | `.` (current directory) |
| `--remove` | Remove source files after successful conversion | `false` |

**Examples**

```sh
# Convert all TIFFs in current directory to PNG
texutil convert '*.tif' --to png

# Convert all PNGs in a specific directory to JPEG
texutil convert '*.png' --to jpg --dir /path/to/textures

# Convert and delete the originals
texutil convert '*.tif' --to png --remove

# Shell glob expansion also works
texutil convert --to png -- textures/*.tif
```

Output files are written alongside the source files with the new extension. Files that cannot be decoded are skipped with an error printed to stderr.

#### `resize`

Resize texture files matching a glob pattern to a specific size.

```sh
texutil resize <pattern...> --size <width>x<height> [--dir <directory>] [--suffix <suffix>] [--filter <filter>]
```

**Flags**

| Flag | Description | Default |
|------|-------------|---------|
| `--size` | Target size in `WxH` format (e.g. `1024x1024`) | *(required)* |
| `--dir` | Directory to search in | `.` (current directory) |
| `--suffix` | Suffix to append to output filenames; if omitted, files are modified in place | `""` |
| `--filter` | Resampling filter: `nearest`, `bilinear`, `catmull-rom` | `bilinear` |

**Examples**

```sh
# Resize all PNGs to 1024x1024 in place
texutil resize '*.png' --size 1024x1024

# Resize and save with a suffix
texutil resize '*.png' --size 512x512 --suffix _512

# Resize using a specific filter
texutil resize '*.jpg' --size 2048x2048 --filter catmull-rom
```

#### `applyao`

Multiply colors from an Ambient Occlusion (AO) texture into textures matching a glob pattern.

```sh
texutil applyao <pattern...> [--intensity <0.0-1.0>] [--aosuffix <suffix>] [--suffix <suffix>] [--dir <directory>]
```

**Flags**

| Flag | Description | Default |
|------|-------------|---------|
| `--intensity` | Intensity of the AO effect (0.0 to 1.0, 0 is invalid) | `0.75` |
| `--aosuffix` | Suffix to find the AO file (e.g. `_ao` for `base.png` -> `base_ao.png`) | `_ao` |
| `--suffix` | Suffix to append to output filenames; if omitted, files are modified in place | `""` |
| `--dir` | Directory to search in | `.` (current directory) |

**Examples**

```sh
# Apply AO to all albedo maps with default intensity
texutil applyao '*_albedo.png'

# Apply AO with a specific intensity and suffix
texutil applyao '*_diffuse.jpg' --intensity 0.5 --suffix _ao_applied

# Custom AO suffix
texutil applyao '*.png' --aosuffix _AmbientOcclusion
```

The tool will automatically attempt to match AO files by replacing common texture suffixes (like `_albedo`, `_diffuse`, etc.) with the AO suffix.

#### `applyalpha`

Bake an alpha/opacity map into a diffuse texture. The alpha map is expected to be grayscale.

```sh
texutil applyalpha <pattern...> [--alphasuffix <suffix>] [--suffix <suffix>] [--dir <directory>]
```

**Flags**

| Flag | Description | Default |
|------|-------------|---------|
| `--alphasuffix` | Suffix to find the alpha/opacity file (e.g. `_alpha` for `base.png` -> `base_alpha.png`) | `_alpha` |
| `--suffix` | Suffix to append to output filenames; if omitted, files are modified in place | `""` |
| `--dir` | Directory to search in | `.` (current directory) |

**Examples**

```sh
# Apply alpha to all albedo maps
texutil applyalpha '*_albedo.png'

# Apply alpha with a specific suffix
texutil applyalpha '*_diffuse.jpg' --suffix _alpha_applied

# Custom alpha suffix
texutil applyalpha '*.png' --alphasuffix _opacity
```

The tool will automatically attempt to match alpha files by replacing common texture suffixes (like `_albedo`, `_diffuse`, etc.) with the alpha suffix.

#### `invert`

Invert the RGB channels of images matching a glob pattern. The alpha channel is preserved.

```sh
texutil invert <pattern...> [--suffix <suffix>] [--dir <directory>]
```

**Flags**

| Flag | Description | Default |
|------|-------------|---------|
| `--suffix` | Suffix to append to output filenames; if omitted, files are modified in place | `""` |
| `--dir` | Directory to search in | `.` (current directory) |

**Examples**

```sh
# Invert all masks in the current directory
texutil invert '*_mask.png'

# Invert images and save with a suffix
texutil invert '*.jpg' --suffix _inverted
```

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
