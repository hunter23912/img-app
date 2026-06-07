package main

import (
	"log"
	"net/http"
	"strings"
	"time"
)

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 当前只放行本地 Vite 开发服务器；正式部署时应改成真实前端域名或由网关处理。
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Expose-Headers", "X-Image-Width, X-Image-Height, X-Image-Source-Format, X-Image-Output, X-Original-Bytes, X-Compressed-Bytes, X-Saved-Bytes, X-Watermark-Mode")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	bytes      int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(data)
	r.bytes += n
	return n, err
}

func withRequestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// 包一层 ResponseWriter，用于记录 handler 最终写出的状态码和响应大小。
		recorder := &responseRecorder{ResponseWriter: w}

		next.ServeHTTP(recorder, r)

		statusCode := recorder.statusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}

		log.Printf(
			"http request: method=%s path=%s status=%d bytes=%d duration_ms=%d client_ip=%s user_agent=%q",
			r.Method,
			r.URL.RequestURI(),
			statusCode,
			recorder.bytes,
			time.Since(start).Milliseconds(),
			clientIP(r),
			r.UserAgent(),
		)
	})
}

func clientIP(r *http.Request) string {
	// 优先读取反向代理传入的客户端 IP，方便后续部署到 Nginx/CDN 后仍能定位请求来源。
	if forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwardedFor != "" {
		ip, _, _ := strings.Cut(forwardedFor, ",")
		if ip = strings.TrimSpace(ip); ip != "" {
			return ip
		}
	}

	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}

	host, _, err := strings.Cut(r.RemoteAddr, ":")
	if err {
		return host
	}

	return r.RemoteAddr
}
