package main

import (
	"log"
	"net/http"
)

func main() {
	config := loadConfig()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", healthHandler(config))
	mux.HandleFunc("/api/generate", generateHandler(config))
	mux.HandleFunc("/api/edit", editHandler(config))
	mux.HandleFunc("/api/compress", compressHandler())
	mux.HandleFunc("/api/watermark/remove", removeWatermarkHandler())

	log.Printf("backend starting")
	log.Printf("listen addr: %s", config.Addr)
	log.Printf("image endpoint: %s", config.Endpoint)
	log.Printf("api key configured: %t", config.APIKey != "")
	if config.APIKey == "" {
		log.Printf("warning: IMG_API_KEY is empty; image requests will fail until it is set")
	}

	if err := http.ListenAndServe(config.Addr, withRequestLog(withCORS(mux))); err != nil {
		log.Fatal(err)
	}
}
