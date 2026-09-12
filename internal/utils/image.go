package utils

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
)

// Compression thresholds and targets. Tuned so a typical phone photo
// (3–8 MB JPEG) drops below 1 MB with no visible quality loss, while
// small/already-optimized files are left alone.
const (
	// Only compress if the file is larger than this.
	CompressSizeThreshold = 1 * 1024 * 1024 // 1 MB

	// Maximum dimension on the longest side. Larger images are downscaled,
	// preserving aspect ratio. 1200px covers any reasonable display size.
	MaxDimension = 1200

	// JPEG quality. 85 is the sweet spot: 40–50% smaller than 100 with no
	// perceptible difference on screen.
	JPEGQuality = 85

	// PNG compression level (0–9). 9 is max lossless compression.
	PNGCompressionLevel = png.BestCompression
)

// ShouldCompress reports whether the given file should be processed.
// We only touch images larger than CompressSizeThreshold — smaller files
// are already fine and re-encoding them would waste CPU for no gain.
func ShouldCompress(path string, size int64) bool {
	if size <= CompressSizeThreshold {
		return false
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg", ".png":
		return true
	default:
		// Skip WebP (already compact) and GIF (animation would break).
		return false
	}
}

// CompressImage compresses the image at inPath in place. Returns the new
// file size after compression. If compression fails or doesn't help, the
// original file is left untouched and the original size is returned.
//
// Behavior:
//   - JPEG: decode → optionally downscale → re-encode at JPEGQuality
//   - PNG:  decode → optionally downscale → re-encode losslessly at max level
//   - Other extensions: no-op
func CompressImage(inPath string) (int64, error) {
	info, err := os.Stat(inPath)
	if err != nil {
		return 0, err
	}
	if !ShouldCompress(inPath, info.Size()) {
		return info.Size(), nil
	}

	ext := strings.ToLower(filepath.Ext(inPath))

	// Decode once, then reuse for both downscale and re-encode.
	src, err := imaging.Open(inPath, imaging.AutoOrientation(true))
	if err != nil {
		return info.Size(), fmt.Errorf("decode failed: %w", err)
	}

	// Downscale if the longest side exceeds MaxDimension.
	src = downscaleIfNeeded(src)

	// Encode into a buffer first so we can compare sizes before overwriting.
	var buf bytes.Buffer
	switch ext {
	case ".jpg", ".jpeg":
		if err := jpeg.Encode(&buf, src, &jpeg.Options{Quality: JPEGQuality}); err != nil {
			return info.Size(), fmt.Errorf("jpeg encode failed: %w", err)
		}
	case ".png":
		enc := &png.Encoder{CompressionLevel: png.BestCompression}
		if err := enc.Encode(&buf, src); err != nil {
			return info.Size(), fmt.Errorf("png encode failed: %w", err)
		}
	default:
		return info.Size(), nil
	}

	// If the "compressed" version is larger (rare for tiny sources, possible
	// for already-optimized JPEGs), keep the original.
	if int64(buf.Len()) >= info.Size() {
		return info.Size(), nil
	}

	// Atomic-ish replace: write to a temp file in the same directory, then
	// rename over the original. Rename is atomic on POSIX, so readers never
	// see a half-written file.
	tmpPath := inPath + ".tmp"
	if err := os.WriteFile(tmpPath, buf.Bytes(), 0o644); err != nil {
		return info.Size(), fmt.Errorf("write temp failed: %w", err)
	}
	if err := os.Rename(tmpPath, inPath); err != nil {
		_ = os.Remove(tmpPath)
		return info.Size(), fmt.Errorf("rename failed: %w", err)
	}

	// Best-effort: trim to the actual bytes written.
	return int64(buf.Len()), nil
}

// downscaleIfNeeded returns a possibly-smaller copy of img whose longest
// side is at most MaxDimension. Aspect ratio is preserved.
func downscaleIfNeeded(img image.Image) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= MaxDimension && h <= MaxDimension {
		return img
	}

	if w >= h {
		return imaging.Resize(img, MaxDimension, 0, imaging.Lanczos)
	}
	return imaging.Resize(img, 0, MaxDimension, imaging.Lanczos)
}

// CompressReader is a convenience for tests: compress an in-memory image
// and return the encoded bytes. Not used by the HTTP path.
func CompressReader(r io.Reader, ext string) ([]byte, error) {
	src, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	src = downscaleIfNeeded(src)

	var buf bytes.Buffer
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		if err := jpeg.Encode(&buf, src, &jpeg.Options{Quality: JPEGQuality}); err != nil {
			return nil, err
		}
	case ".png":
		enc := &png.Encoder{CompressionLevel: png.BestCompression}
		if err := enc.Encode(&buf, src); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported extension: %s", ext)
	}
	return buf.Bytes(), nil
}

// ensure image and gif imports are used even if the build tag changes.
var _ = gif.GIF{}