package httpapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"img-app/backend/internal/history"
	"img-app/backend/internal/imageops"
	"img-app/backend/internal/store"
)

func TestLegacyURLDownloadCachesLocallyAcrossRestart(t *testing.T) {
	data := testPNG(t, color.NRGBA{R: 44, G: 66, B: 88, A: 255})
	calls := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(data)
	}))
	defer server.Close()
	originalTransport := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	defer func() { http.DefaultTransport = originalTransport }()

	path := filepath.Join(t.TempDir(), "history.db")
	db, err := store.OpenDatabase(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { db.Close() }()
	id, err := db.CreateTask("edit", store.ImageTaskInput{Prompt: "old image", Model: defaultModel})
	if err != nil {
		t.Fatal(err)
	}
	source := server.URL + "/old.png"
	if err := db.CompleteTask(id, source); err != nil {
		t.Fatal(err)
	}

	for _, format := range []string{"jpg", "png"} {
		res := serveDownloadRequest(t, downloadImageHandler(appConfig{Database: db}), downloadRequest{Source: source, Format: format})
		if res.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", format, res.Code, res.Body.String())
		}
		if format == "png" && !bytes.Equal(res.Body.Bytes(), data) {
			t.Fatal("PNG changed")
		}
		if calls != 1 {
			t.Fatalf("remote requests = %d, want 1", calls)
		}
		db.Close()
		db, err = store.OpenDatabase(path)
		if err != nil {
			t.Fatal(err)
		}
		server.Close()
	}
	page, err := db.ListTasks(5, "")
	if err != nil || len(page.Tasks) != 1 || page.Tasks[0].Image != history.ImagePath(id) {
		t.Fatalf("cached history: %#v, %v", page, err)
	}
	res := serveDownloadRequest(t, downloadImageHandler(appConfig{Database: db}), downloadRequest{Source: history.ImagePath(id), Format: "jpg"})
	if res.Code != http.StatusOK {
		t.Fatalf("local history: %d %s", res.Code, res.Body.String())
	}
}

func TestGeneratedURLIsStoredLocally(t *testing.T) {
	data := testPNG(t, color.NRGBA{R: 10, A: 255})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(data)
	}))
	defer server.Close()
	originalTransport := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	defer func() { http.DefaultTransport = originalTransport }()
	db, err := store.OpenDatabase(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	source := persistImageResult(appConfig{Database: db}, "generate", generateRequest{Prompt: "test"}, server.URL+"/image")
	for i := 0; i < 50; i++ {
		if cached, cacheErr := db.CachedImageSource(source); cacheErr == nil && history.IsBase64ImageDataURL(cached) {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if cached, cacheErr := db.CachedImageSource(source); cacheErr != nil || !history.IsBase64ImageDataURL(cached) {
		t.Fatalf("image was not cached: %v", cacheErr)
	}
	server.Close()
	res := serveDownloadRequest(t, downloadImageHandler(appConfig{Database: db}), downloadRequest{Source: source, Format: "png"})
	if res.Code != http.StatusOK || !bytes.Equal(res.Body.Bytes(), data) {
		t.Fatal("local download failed")
	}
}

func TestFailedLegacyDownloadPreservesURL(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusGone) }))
	defer server.Close()
	originalTransport := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	defer func() { http.DefaultTransport = originalTransport }()
	db, err := store.OpenDatabase(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	id, err := db.CreateTask("edit", store.ImageTaskInput{Prompt: "expired"})
	if err != nil {
		t.Fatal(err)
	}
	source := server.URL + "/expired"
	if err := db.CompleteTask(id, source); err != nil {
		t.Fatal(err)
	}
	if _, err := loadDownloadImageForConfig(source, appConfig{Database: db}); err == nil {
		t.Fatal("expected download error")
	}
	value, err := db.CachedImageSource(source)
	if err != nil || value != source {
		t.Fatal("failed download changed original URL")
	}
}

func TestPNGDownloadConvertsJPEGSource(t *testing.T) {
	var encoded bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	if err := jpeg.Encode(&encoded, img, nil); err != nil {
		t.Fatal(err)
	}
	source := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(encoded.Bytes())
	res := serveDownloadRequest(t, downloadImageHandler(appConfig{}), downloadRequest{Source: source, Format: "png"})
	if res.Code != http.StatusOK || res.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("response: %d %s", res.Code, res.Body.String())
	}
	if _, err := png.Decode(bytes.NewReader(res.Body.Bytes())); err != nil {
		t.Fatal(err)
	}
}

func TestDownloadImagePNGPreservesSource(t *testing.T) {
	source := testPNG(t, color.NRGBA{R: 12, G: 34, B: 56, A: 255})
	recorder := serveDownloadRequest(t, downloadImageHandler(appConfig{Endpoint: "https://images.example.com"}), downloadRequest{
		Source: dataURL(source),
		Format: "png",
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if !bytes.Equal(recorder.Body.Bytes(), source) {
		t.Fatal("PNG download did not preserve the source bytes")
	}
	if got := recorder.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("Content-Type = %q, want image/png", got)
	}
	if got := recorder.Header().Get("Content-Disposition"); !strings.Contains(got, `filename="gpt-image.png"`) {
		t.Fatalf("Content-Disposition = %q, want PNG filename", got)
	}
}

func TestDownloadImageJPGQualityAndDefault(t *testing.T) {
	source := testPNG(t, color.NRGBA{R: 100, G: 120, B: 140, A: 255})
	var bodies = map[string][]byte{}

	for _, quality := range []int{1, 95, 100} {
		recorder := serveDownloadRequest(t, downloadImageHandler(appConfig{Endpoint: "https://images.example.com"}), downloadRequest{
			Source:  dataURL(source),
			Format:  "jpg",
			Quality: &quality,
		})
		assertJPGResponse(t, recorder)
		bodies[fmt.Sprint(quality)] = recorder.Body.Bytes()
	}

	defaultRecorder := serveDownloadRequest(t, downloadImageHandler(appConfig{Endpoint: "https://images.example.com"}), downloadRequest{
		Source: dataURL(source),
		Format: "jpg",
	})
	assertJPGResponse(t, defaultRecorder)
	if !bytes.Equal(defaultRecorder.Body.Bytes(), bodies["95"]) {
		t.Fatal("missing quality did not default to quality 95")
	}
}

func TestDownloadImageRejectsInvalidQuality(t *testing.T) {
	source := testPNG(t, color.NRGBA{R: 1, G: 2, B: 3, A: 255})
	for _, quality := range []int{0, 101} {
		recorder := serveDownloadRequest(t, downloadImageHandler(appConfig{Endpoint: "https://images.example.com"}), downloadRequest{
			Source:  dataURL(source),
			Format:  "jpg",
			Quality: &quality,
		})
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("quality %d status = %d, want %d", quality, recorder.Code, http.StatusBadRequest)
		}
	}
}

func TestDownloadImageFillsTransparentPixelsWithWhite(t *testing.T) {
	transparent := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	transparent.SetNRGBA(0, 0, color.NRGBA{A: 0})
	var source bytes.Buffer
	if err := png.Encode(&source, transparent); err != nil {
		t.Fatalf("encode transparent PNG: %v", err)
	}

	quality := 100
	recorder := serveDownloadRequest(t, downloadImageHandler(appConfig{Endpoint: "https://images.example.com"}), downloadRequest{
		Source:  dataURL(source.Bytes()),
		Format:  "jpg",
		Quality: &quality,
	})
	assertJPGResponse(t, recorder)

	decoded, err := jpeg.Decode(bytes.NewReader(recorder.Body.Bytes()))
	if err != nil {
		t.Fatalf("decode downloaded JPG: %v", err)
	}
	red, green, blue, _ := decoded.At(0, 0).RGBA()
	if red < 65000 || green < 65000 || blue < 65000 {
		t.Fatalf("transparent pixel was not filled with white: %d %d %d", red, green, blue)
	}
}

func TestDownloadImageRejectsUntrustedSourceURL(t *testing.T) {
	for _, source := range []string{
		"http://images.example.com/image.png",
		"https://evil.example.com/image.png",
		"https://images.example.com/image.png#fragment",
	} {
		recorder := serveDownloadRequest(t, downloadImageHandler(appConfig{Endpoint: "https://images.example.com"}), downloadRequest{
			Source: source,
			Format: "jpg",
		})
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("source %q status = %d, want %d", source, recorder.Code, http.StatusBadRequest)
		}
	}
}

func TestFetchExternalImageUsesConfiguredHost(t *testing.T) {
	source := testPNG(t, color.NRGBA{R: 22, G: 44, B: 66, A: 255})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(source)
	}))
	defer server.Close()

	got, err := imageops.FetchExternalImageWithClient(server.URL+"/result", server.URL, server.Client())
	if err != nil {
		t.Fatalf("fetchExternalImageWithClient() error: %v", err)
	}
	if !bytes.Equal(got, source) {
		t.Fatal("external image fetch changed the response bytes")
	}
}

func TestDownloadImageAllowsURLTrustedByBackend(t *testing.T) {
	source := testPNG(t, color.NRGBA{R: 22, G: 44, B: 66, A: 255})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(source)
	}))
	defer server.Close()

	imageURL := server.URL + "/result"
	registry := imageops.NewSourceRegistry()
	registry.Trust(imageURL)

	got, err := imageops.LoadImageWithClient(imageURL, registry, server.Client())
	if err != nil {
		t.Fatalf("loadDownloadImageWithClient() error: %v", err)
	}
	if !bytes.Equal(got, source) {
		t.Fatal("trusted external image fetch changed the response bytes")
	}
}

func TestDownloadImageAllowsPersistedHistoryImageReference(t *testing.T) {
	source := testPNG(t, color.NRGBA{R: 44, G: 66, B: 88, A: 255})
	database, err := store.OpenDatabase(filepath.Join(t.TempDir(), "img-app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	taskID, err := database.CreateTask("edit", store.ImageTaskInput{Model: defaultModel, Prompt: "Upscale", Size: "2048x2048"})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.CompleteTask(taskID, dataURL(source)); err != nil {
		t.Fatal(err)
	}

	recorder := serveDownloadRequest(t, downloadImageHandler(appConfig{Database: database}), downloadRequest{
		Source: history.ImagePath(taskID),
		Format: "png",
	})
	if recorder.Code != http.StatusOK || !bytes.Equal(recorder.Body.Bytes(), source) {
		t.Fatalf("history image download = %d, %d bytes; want %d, %d bytes", recorder.Code, recorder.Body.Len(), http.StatusOK, len(source))
	}
}

type downloadRequest struct {
	Source  string
	Format  string
	Quality *int
}

func serveDownloadRequest(t *testing.T, handler http.Handler, input downloadRequest) *httptest.ResponseRecorder {
	t.Helper()
	body := map[string]any{
		"source": input.Source,
		"format": input.Format,
	}
	if input.Quality != nil {
		body["quality"] = *input.Quality
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/download/image", bytes.NewReader(encoded))
	handler.ServeHTTP(recorder, request)
	return recorder
}

func testPNG(t *testing.T, pixel color.NRGBA) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, pixel)
	img.SetNRGBA(1, 0, pixel)
	var output bytes.Buffer
	if err := png.Encode(&output, img); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
	return output.Bytes()
}

func dataURL(imageBytes []byte) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageBytes)
}

func assertJPGResponse(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "image/jpeg" {
		t.Fatalf("Content-Type = %q, want image/jpeg", got)
	}
	if got := recorder.Header().Get("Content-Disposition"); !strings.Contains(got, `filename="gpt-image.jpg"`) {
		t.Fatalf("Content-Disposition = %q, want JPG filename", got)
	}
	if _, err := jpeg.Decode(bytes.NewReader(recorder.Body.Bytes())); err != nil {
		t.Fatalf("response is not a valid JPEG: %v", err)
	}
	if _, err := io.Copy(io.Discard, bytes.NewReader(recorder.Body.Bytes())); err != nil {
		t.Fatalf("read response body: %v", err)
	}
}
