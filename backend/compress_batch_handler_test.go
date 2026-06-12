package main

import (
	"archive/zip"
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"strings"
	"testing"
)

func TestBuildCompressedZip(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("images", "sample.png")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(testPNG(t)); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(1024 * 1024)
	if err != nil {
		t.Fatalf("ReadForm: %v", err)
	}

	archive, manifest, err := buildCompressedZip(form.File["images"], compressImageOptions{
		Output:  "png",
		Quality: 82,
	})
	if err != nil {
		t.Fatalf("buildCompressedZip returned error: %v", err)
	}
	if manifest.SuccessCount != 1 || manifest.FailedCount != 0 {
		t.Fatalf("manifest counts = success %d failed %d, want 1/0", manifest.SuccessCount, manifest.FailedCount)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatalf("zip.NewReader: %v", err)
	}

	var hasImage bool
	var hasManifest bool
	for _, file := range zipReader.File {
		if strings.HasSuffix(file.Name, ".png") {
			hasImage = true
		}
		if file.Name == "manifest.json" {
			hasManifest = true
		}
	}
	if !hasImage {
		t.Fatal("zip archive has no compressed image")
	}
	if !hasManifest {
		t.Fatal("zip archive has no manifest.json")
	}
}

func testPNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y += 1 {
		for x := 0; x < 4; x += 1 {
			img.SetRGBA(x, y, color.RGBA{R: 40, G: 140, B: 220, A: 255})
		}
	}

	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	return buffer.Bytes()
}
