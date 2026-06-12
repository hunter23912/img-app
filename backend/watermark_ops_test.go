package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestRemoveWatermarkLocalFillsMarkedPixels(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 5, 5))
	for y := 0; y < 5; y += 1 {
		for x := 0; x < 5; x += 1 {
			source.SetRGBA(x, y, color.RGBA{R: 20, G: 80, B: 220, A: 255})
		}
	}
	source.SetRGBA(2, 2, color.RGBA{R: 240, G: 20, B: 20, A: 255})

	mask := image.NewRGBA(image.Rect(0, 0, 5, 5))
	mask.SetRGBA(2, 2, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	result, err := removeWatermarkLocal(bytes.NewReader(mustPNG(t, source)), bytes.NewReader(mustPNG(t, mask)))
	if err != nil {
		t.Fatalf("removeWatermarkLocal returned error: %v", err)
	}
	if result.ContentType != "image/png" {
		t.Fatalf("ContentType = %q, want image/png", result.ContentType)
	}
	if result.MaskedPixels == 0 {
		t.Fatal("MaskedPixels = 0, want marked pixels")
	}
	if result.Iterations == 0 {
		t.Fatal("Iterations = 0, want at least one fill pass")
	}

	decoded, err := png.Decode(bytes.NewReader(result.Data))
	if err != nil {
		t.Fatalf("result is not a valid png: %v", err)
	}

	r, g, b, _ := decoded.At(2, 2).RGBA()
	if r>>8 > 80 || g>>8 < 40 || b>>8 < 160 {
		t.Fatalf("center pixel was not filled from surrounding color: r=%d g=%d b=%d", r>>8, g>>8, b>>8)
	}
}

func TestRemoveWatermarkLocalRejectsEmptyMask(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 3, 3))
	mask := image.NewRGBA(image.Rect(0, 0, 3, 3))

	_, err := removeWatermarkLocal(bytes.NewReader(mustPNG(t, source)), bytes.NewReader(mustPNG(t, mask)))
	if err == nil {
		t.Fatal("removeWatermarkLocal returned nil error for empty mask")
	}
	if !strings.Contains(err.Error(), "no marked pixels") {
		t.Fatalf("error = %q, want no marked pixels", err)
	}
}

func mustPNG(t *testing.T, img image.Image) []byte {
	t.Helper()

	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buffer.Bytes()
}
