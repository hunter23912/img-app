package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
)

func compressHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
			return
		}

		// 压缩接口保持原图尺寸，只做 JPG/PNG 重编码和元数据剥离。
		if err := r.ParseMultipartForm(32 * 1024 * 1024); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid multipart form"})
			return
		}

		imageFile, _, err := r.FormFile("image")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "image file is required"})
			return
		}
		defer imageFile.Close()

		quality, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("quality")))
		result, err := compressUploadedImage(imageFile, compressImageOptions{
			Output:  strings.TrimSpace(r.FormValue("output")),
			Quality: quality,
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}

		w.Header().Set("Content-Type", result.ContentType)
		w.Header().Set("Content-Disposition", `attachment; filename="`+result.Filename+`"`)
		w.Header().Set("X-Image-Width", strconv.Itoa(result.Width))
		w.Header().Set("X-Image-Height", strconv.Itoa(result.Height))
		w.Header().Set("X-Image-Source-Format", result.SourceFormat)
		w.Header().Set("X-Image-Output", result.Output)
		w.Header().Set("X-Original-Bytes", strconv.Itoa(result.OriginalBytes))
		w.Header().Set("X-Compressed-Bytes", strconv.Itoa(result.CompressedBytes))
		w.Header().Set("X-Saved-Bytes", strconv.Itoa(result.OriginalBytes-result.CompressedBytes))
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(result.Data); err != nil {
			log.Printf("write compressed image: %v", err)
		}
	}
}
