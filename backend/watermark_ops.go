package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
)

type localWatermarkResult struct {
	ContentType  string
	Filename     string
	Data         []byte
	Width        int
	Height       int
	MaskedPixels int
	Iterations   int
}

type fillPixel struct {
	index int
	r     uint8
	g     uint8
	b     uint8
	a     uint8
}

func removeWatermarkLocal(imageReader io.Reader, maskReader io.Reader) (localWatermarkResult, error) {
	imageData, err := io.ReadAll(io.LimitReader(imageReader, 32*1024*1024))
	if err != nil {
		return localWatermarkResult{}, fmt.Errorf("read image: %w", err)
	}
	if len(imageData) == 0 {
		return localWatermarkResult{}, fmt.Errorf("image file is empty")
	}

	maskData, err := io.ReadAll(io.LimitReader(maskReader, 8*1024*1024))
	if err != nil {
		return localWatermarkResult{}, fmt.Errorf("read mask: %w", err)
	}
	if len(maskData) == 0 {
		return localWatermarkResult{}, fmt.Errorf("mask file is empty")
	}

	source, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return localWatermarkResult{}, fmt.Errorf("unsupported or invalid image")
	}

	maskImage, _, err := image.Decode(bytes.NewReader(maskData))
	if err != nil {
		return localWatermarkResult{}, fmt.Errorf("unsupported or invalid mask")
	}

	bounds := source.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return localWatermarkResult{}, fmt.Errorf("image has invalid dimensions")
	}

	rgba := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(rgba, rgba.Bounds(), source, bounds.Min, draw.Src)

	mask, maskedPixels, minX, minY, maxX, maxY := buildScaledMask(maskImage, width, height)
	if maskedPixels == 0 {
		return localWatermarkResult{}, fmt.Errorf("mask file has no marked pixels")
	}

	mask, maskedPixels, minX, minY, maxX, maxY = expandMask(mask, width, height, minX, minY, maxX, maxY)
	iterations := inpaintMaskedArea(rgba, mask, width, height, minX, minY, maxX, maxY)

	var output bytes.Buffer
	if err := png.Encode(&output, rgba); err != nil {
		return localWatermarkResult{}, fmt.Errorf("encode watermark result: %w", err)
	}

	return localWatermarkResult{
		ContentType:  "image/png",
		Filename:     "watermark-result.png",
		Data:         output.Bytes(),
		Width:        width,
		Height:       height,
		MaskedPixels: maskedPixels,
		Iterations:   iterations,
	}, nil
}

func buildScaledMask(maskImage image.Image, width int, height int) ([]bool, int, int, int, int, int) {
	mask := make([]bool, width*height)
	maskBounds := maskImage.Bounds()
	maskWidth := maskBounds.Dx()
	maskHeight := maskBounds.Dy()

	minX := width
	minY := height
	maxX := -1
	maxY := -1
	count := 0

	for y := 0; y < height; y += 1 {
		sourceY := maskBounds.Min.Y
		if maskHeight > 0 {
			sourceY += y * maskHeight / height
		}

		for x := 0; x < width; x += 1 {
			sourceX := maskBounds.Min.X
			if maskWidth > 0 {
				sourceX += x * maskWidth / width
			}

			if !isMarkedMaskPixel(maskImage.At(sourceX, sourceY)) {
				continue
			}

			index := y*width + x
			mask[index] = true
			count += 1
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x > maxX {
				maxX = x
			}
			if y > maxY {
				maxY = y
			}
		}
	}

	return mask, count, minX, minY, maxX, maxY
}

func isMarkedMaskPixel(pixel color.Color) bool {
	r, g, b, a := pixel.RGBA()
	if a < 0x1000 {
		return false
	}

	luma := (299*r + 587*g + 114*b) / 1000
	return luma > 0x1000
}

func expandMask(mask []bool, width int, height int, minX int, minY int, maxX int, maxY int) ([]bool, int, int, int, int, int) {
	if maxX < minX || maxY < minY {
		return mask, 0, minX, minY, maxX, maxY
	}

	expanded := make([]bool, len(mask))
	copy(expanded, mask)

	for y := minY; y <= maxY; y += 1 {
		for x := minX; x <= maxX; x += 1 {
			if !mask[y*width+x] {
				continue
			}

			for dy := -1; dy <= 1; dy += 1 {
				ny := y + dy
				if ny < 0 || ny >= height {
					continue
				}

				for dx := -1; dx <= 1; dx += 1 {
					nx := x + dx
					if nx < 0 || nx >= width {
						continue
					}
					expanded[ny*width+nx] = true
				}
			}
		}
	}

	nextMinX := width
	nextMinY := height
	nextMaxX := -1
	nextMaxY := -1
	count := 0
	for y := max(0, minY-1); y <= min(height-1, maxY+1); y += 1 {
		for x := max(0, minX-1); x <= min(width-1, maxX+1); x += 1 {
			if !expanded[y*width+x] {
				continue
			}
			count += 1
			if x < nextMinX {
				nextMinX = x
			}
			if y < nextMinY {
				nextMinY = y
			}
			if x > nextMaxX {
				nextMaxX = x
			}
			if y > nextMaxY {
				nextMaxY = y
			}
		}
	}

	return expanded, count, nextMinX, nextMinY, nextMaxX, nextMaxY
}

func inpaintMaskedArea(rgba *image.RGBA, mask []bool, width int, height int, minX int, minY int, maxX int, maxY int) int {
	known := make([]bool, len(mask))
	remaining := 0
	for index, masked := range mask {
		known[index] = !masked
		if masked {
			remaining += 1
		}
	}

	iterations := 0
	maxIterations := min(max(width, height), 512)
	for remaining > 0 && iterations < maxIterations {
		fills := make([]fillPixel, 0, min(remaining, 4096))

		for y := minY; y <= maxY; y += 1 {
			for x := minX; x <= maxX; x += 1 {
				index := y*width + x
				if known[index] {
					continue
				}

				if fill, ok := averageKnownNeighbors(rgba, known, width, height, x, y, index); ok {
					fills = append(fills, fill)
				}
			}
		}

		if len(fills) == 0 {
			break
		}

		for _, fill := range fills {
			offset := fill.index * 4
			rgba.Pix[offset] = fill.r
			rgba.Pix[offset+1] = fill.g
			rgba.Pix[offset+2] = fill.b
			rgba.Pix[offset+3] = fill.a
			known[fill.index] = true
			remaining -= 1
		}

		iterations += 1
	}

	if remaining > 0 {
		fillRemainingWithAverage(rgba, known)
	}

	return iterations
}

func averageKnownNeighbors(rgba *image.RGBA, known []bool, width int, height int, x int, y int, index int) (fillPixel, bool) {
	var rTotal int
	var gTotal int
	var bTotal int
	var aTotal int
	count := 0

	for dy := -1; dy <= 1; dy += 1 {
		ny := y + dy
		if ny < 0 || ny >= height {
			continue
		}

		for dx := -1; dx <= 1; dx += 1 {
			if dx == 0 && dy == 0 {
				continue
			}

			nx := x + dx
			if nx < 0 || nx >= width {
				continue
			}

			neighborIndex := ny*width + nx
			if !known[neighborIndex] {
				continue
			}

			offset := neighborIndex * 4
			rTotal += int(rgba.Pix[offset])
			gTotal += int(rgba.Pix[offset+1])
			bTotal += int(rgba.Pix[offset+2])
			aTotal += int(rgba.Pix[offset+3])
			count += 1
		}
	}

	if count == 0 {
		return fillPixel{}, false
	}

	return fillPixel{
		index: index,
		r:     uint8(rTotal / count),
		g:     uint8(gTotal / count),
		b:     uint8(bTotal / count),
		a:     uint8(aTotal / count),
	}, true
}

func fillRemainingWithAverage(rgba *image.RGBA, known []bool) {
	var rTotal int
	var gTotal int
	var bTotal int
	var aTotal int
	count := 0

	for index, isKnown := range known {
		if !isKnown {
			continue
		}

		offset := index * 4
		rTotal += int(rgba.Pix[offset])
		gTotal += int(rgba.Pix[offset+1])
		bTotal += int(rgba.Pix[offset+2])
		aTotal += int(rgba.Pix[offset+3])
		count += 1
	}

	if count == 0 {
		return
	}

	r := uint8(rTotal / count)
	g := uint8(gTotal / count)
	b := uint8(bTotal / count)
	a := uint8(aTotal / count)

	for index, isKnown := range known {
		if isKnown {
			continue
		}

		offset := index * 4
		rgba.Pix[offset] = r
		rgba.Pix[offset+1] = g
		rgba.Pix[offset+2] = b
		rgba.Pix[offset+3] = a
	}
}
