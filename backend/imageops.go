package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
)

type compressImageOptions struct {
	Output  string
	Quality int
}

type compressedImage struct {
	ContentType     string
	Filename        string
	Data            []byte
	Width           int
	Height          int
	SourceFormat    string
	Output          string
	OriginalBytes   int
	CompressedBytes int
}

func compressUploadedImage(file io.Reader, options compressImageOptions) (compressedImage, error) {
	imageData, err := io.ReadAll(io.LimitReader(file, 32*1024*1024))
	if err != nil {
		return compressedImage{}, fmt.Errorf("read image: %w", err)
	}
	if len(imageData) == 0 {
		return compressedImage{}, fmt.Errorf("image file is empty")
	}

	decoded, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return compressedImage{}, fmt.Errorf("unsupported or invalid image")
	}

	bounds := decoded.Bounds()
	output := normalizeCompressOutput(options.Output, format)
	quality := normalizeJPEGQuality(options.Quality)

	var buffer bytes.Buffer
	result := compressedImage{
		Width:         bounds.Dx(),
		Height:        bounds.Dy(),
		SourceFormat:  format,
		Output:        output,
		OriginalBytes: len(imageData),
	}

	switch output {
	case "jpg":
		if err := jpeg.Encode(&buffer, decoded, &jpeg.Options{Quality: quality}); err != nil {
			return compressedImage{}, fmt.Errorf("encode jpeg: %w", err)
		}
		result.ContentType = "image/jpeg"
		result.Filename = "compressed-image.jpg"
	case "png":
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		if err := encoder.Encode(&buffer, decoded); err != nil {
			return compressedImage{}, fmt.Errorf("encode png: %w", err)
		}
		result.ContentType = "image/png"
		result.Filename = "compressed-image.png"
	default:
		return compressedImage{}, fmt.Errorf("output must be jpg or png")
	}

	result.Data = buffer.Bytes()
	result.CompressedBytes = len(result.Data)
	return result, nil
}

func normalizeCompressOutput(output string, sourceFormat string) string {
	switch output {
	case "", "auto":
		if sourceFormat == "png" {
			return "png"
		}
		return "jpg"
	case "jpg", "jpeg":
		return "jpg"
	case "png":
		return "png"
	}

	return "jpg"
}

func normalizeJPEGQuality(quality int) int {
	if quality == 0 {
		return 82
	}
	if quality < 45 {
		return 45
	}
	if quality > 95 {
		return 95
	}
	return quality
}
