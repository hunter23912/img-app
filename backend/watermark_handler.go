package main

import (
	"log"
	"net/http"
	"strconv"
)

func removeWatermarkHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
			return
		}

		// 本地去水印先使用遮罩邻域扩散算法，避免把用户图片发给外部服务。
		if err := r.ParseMultipartForm(32 * 1024 * 1024); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid multipart form"})
			return
		}

		imageFile, imageHeader, err := r.FormFile("image")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "image file is required"})
			return
		}
		defer imageFile.Close()

		maskFile, _, err := r.FormFile("mask")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "mask file is required"})
			return
		}
		defer maskFile.Close()

		result, err := removeWatermarkLocal(imageFile, maskFile)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}

		log.Printf("local watermark remove: image=%q width=%d height=%d masked_pixels=%d iterations=%d", imageHeader.Filename, result.Width, result.Height, result.MaskedPixels, result.Iterations)
		w.Header().Set("Content-Type", result.ContentType)
		w.Header().Set("Content-Disposition", `attachment; filename="`+result.Filename+`"`)
		w.Header().Set("X-Watermark-Mode", "local")
		w.Header().Set("X-Image-Width", strconv.Itoa(result.Width))
		w.Header().Set("X-Image-Height", strconv.Itoa(result.Height))
		w.Header().Set("X-Masked-Pixels", strconv.Itoa(result.MaskedPixels))
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(result.Data); err != nil {
			log.Printf("write watermark result image: %v", err)
		}
	}
}
