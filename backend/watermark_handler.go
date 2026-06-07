package main

import (
	"io"
	"log"
	"net/http"
)

func removeWatermarkHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
			return
		}

		// 最小流程先收 image + mask，后续再把这里替换成真实图像修复算法或模型接口。
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

		// 读取 mask 是为了验证前端确实生成并上传了标记区域。
		maskBytes, err := io.ReadAll(io.LimitReader(maskFile, 8*1024*1024))
		if err != nil || len(maskBytes) == 0 {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "mask file is invalid"})
			return
		}

		contentType := imageContentType(imageHeader.Filename)
		if contentType == "" {
			contentType = "image/png"
		}

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", `attachment; filename="watermark-result.png"`)
		w.Header().Set("X-Watermark-Mode", "placeholder")
		w.WriteHeader(http.StatusOK)
		if _, err := io.Copy(w, imageFile); err != nil {
			log.Printf("write watermark placeholder image: %v", err)
		}
	}
}
