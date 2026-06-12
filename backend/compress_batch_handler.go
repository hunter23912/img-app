package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

const maxBatchCompressFiles = 30

type compressBatchManifestEntry struct {
	SourceName      string `json:"sourceName"`
	OutputName      string `json:"outputName,omitempty"`
	Width           int    `json:"width,omitempty"`
	Height          int    `json:"height,omitempty"`
	SourceFormat    string `json:"sourceFormat,omitempty"`
	Output          string `json:"output,omitempty"`
	OriginalBytes   int    `json:"originalBytes,omitempty"`
	CompressedBytes int    `json:"compressedBytes,omitempty"`
	SavedBytes      int    `json:"savedBytes,omitempty"`
	Error           string `json:"error,omitempty"`
}

type compressBatchManifest struct {
	Files           []compressBatchManifestEntry `json:"files"`
	TotalFiles      int                          `json:"totalFiles"`
	SuccessCount    int                          `json:"successCount"`
	FailedCount     int                          `json:"failedCount"`
	OriginalBytes   int                          `json:"originalBytes"`
	CompressedBytes int                          `json:"compressedBytes"`
	SavedBytes      int                          `json:"savedBytes"`
}

func compressBatchHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
			return
		}

		if err := r.ParseMultipartForm(256 * 1024 * 1024); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid multipart form"})
			return
		}

		files := uploadedBatchFiles(r.MultipartForm)
		if len(files) == 0 {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "image files are required"})
			return
		}
		if len(files) > maxBatchCompressFiles {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: fmt.Sprintf("too many images; maximum is %d", maxBatchCompressFiles)})
			return
		}

		quality, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("quality")))
		options := compressImageOptions{
			Output:  strings.TrimSpace(r.FormValue("output")),
			Quality: quality,
		}

		archive, manifest, err := buildCompressedZip(files, options)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}

		log.Printf(
			"batch compress: files=%d success=%d failed=%d original_bytes=%d compressed_bytes=%d",
			manifest.TotalFiles,
			manifest.SuccessCount,
			manifest.FailedCount,
			manifest.OriginalBytes,
			manifest.CompressedBytes,
		)

		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="compressed-images.zip"`)
		w.Header().Set("X-Batch-File-Count", strconv.Itoa(manifest.TotalFiles))
		w.Header().Set("X-Batch-Success-Count", strconv.Itoa(manifest.SuccessCount))
		w.Header().Set("X-Batch-Failed-Count", strconv.Itoa(manifest.FailedCount))
		w.Header().Set("X-Batch-Original-Bytes", strconv.Itoa(manifest.OriginalBytes))
		w.Header().Set("X-Batch-Compressed-Bytes", strconv.Itoa(manifest.CompressedBytes))
		w.Header().Set("X-Batch-Saved-Bytes", strconv.Itoa(manifest.SavedBytes))
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(archive); err != nil {
			log.Printf("write compressed zip: %v", err)
		}
	}
}

func uploadedBatchFiles(form *multipart.Form) []*multipart.FileHeader {
	if form == nil || form.File == nil {
		return nil
	}

	files := append([]*multipart.FileHeader{}, form.File["images"]...)
	if len(files) == 0 {
		files = append(files, form.File["image"]...)
	}
	return files
}

func buildCompressedZip(files []*multipart.FileHeader, options compressImageOptions) ([]byte, compressBatchManifest, error) {
	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)
	manifest := compressBatchManifest{
		Files:      make([]compressBatchManifestEntry, 0, len(files)),
		TotalFiles: len(files),
	}
	usedNames := make(map[string]int)

	for _, header := range files {
		entry := compressBatchManifestEntry{SourceName: header.Filename}
		file, err := header.Open()
		if err != nil {
			entry.Error = "open image file failed"
			manifest.FailedCount += 1
			manifest.Files = append(manifest.Files, entry)
			continue
		}

		result, err := compressUploadedImage(file, options)
		closeErr := file.Close()
		if err != nil {
			entry.Error = err.Error()
			manifest.FailedCount += 1
			manifest.Files = append(manifest.Files, entry)
			continue
		}
		if closeErr != nil {
			entry.Error = "close image file failed"
			manifest.FailedCount += 1
			manifest.Files = append(manifest.Files, entry)
			continue
		}

		outputName := uniqueZipName(compressedOutputName(header.Filename, result.Output), usedNames)
		zipFile, err := zipWriter.Create(outputName)
		if err != nil {
			entry.Error = "create zip entry failed"
			manifest.FailedCount += 1
			manifest.Files = append(manifest.Files, entry)
			continue
		}
		if _, err := zipFile.Write(result.Data); err != nil {
			entry.Error = "write zip entry failed"
			manifest.FailedCount += 1
			manifest.Files = append(manifest.Files, entry)
			continue
		}

		entry.OutputName = outputName
		entry.Width = result.Width
		entry.Height = result.Height
		entry.SourceFormat = result.SourceFormat
		entry.Output = result.Output
		entry.OriginalBytes = result.OriginalBytes
		entry.CompressedBytes = result.CompressedBytes
		entry.SavedBytes = result.OriginalBytes - result.CompressedBytes
		manifest.SuccessCount += 1
		manifest.OriginalBytes += result.OriginalBytes
		manifest.CompressedBytes += result.CompressedBytes
		manifest.Files = append(manifest.Files, entry)
	}

	manifest.SavedBytes = manifest.OriginalBytes - manifest.CompressedBytes
	if manifest.SuccessCount == 0 {
		_ = zipWriter.Close()
		return nil, manifest, fmt.Errorf("all images failed to compress")
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err == nil {
		manifestFile, createErr := zipWriter.Create("manifest.json")
		if createErr == nil {
			_, _ = manifestFile.Write(manifestBytes)
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, manifest, fmt.Errorf("close zip archive: %w", err)
	}

	return buffer.Bytes(), manifest, nil
}

func compressedOutputName(sourceName string, output string) string {
	extension := ".jpg"
	if output == "png" {
		extension = ".png"
	}

	base := strings.TrimSuffix(filepath.Base(sourceName), filepath.Ext(sourceName))
	base = sanitizeZipName(base)
	if base == "" || base == "." {
		base = "image"
	}

	return base + "-compressed" + extension
}

func sanitizeZipName(name string) string {
	var builder strings.Builder
	lastDash := false
	for _, char := range strings.ToLower(name) {
		valid := (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')
		if valid {
			builder.WriteRune(char)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}

	return strings.Trim(builder.String(), "-")
}

func uniqueZipName(name string, used map[string]int) string {
	count := used[name]
	used[name] = count + 1
	if count == 0 {
		return name
	}

	extension := filepath.Ext(name)
	base := strings.TrimSuffix(name, extension)
	return fmt.Sprintf("%s-%d%s", base, count+1, extension)
}
