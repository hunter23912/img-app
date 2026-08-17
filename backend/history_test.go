package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

const testHistoryPNGDataURL = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="

func TestImageHistoryAddKeepsRecentSupportedImages(t *testing.T) {
	history := newImageHistory()
	for _, image := range []string{
		"https://images.example.com/one.png",
		"https://images.example.com/two.png",
		"https://images.example.com/three.png",
		"https://images.example.com/four.png",
		"https://images.example.com/five.png",
		"https://images.example.com/six.png",
		"data:image/png;base64,aGVsbG8=",
	} {
		history.Add(image)
	}

	want := []string{
		"data:image/png;base64,aGVsbG8=",
		"https://images.example.com/six.png",
		"https://images.example.com/five.png",
		"https://images.example.com/four.png",
		"https://images.example.com/three.png",
	}
	if got := history.List(); !reflect.DeepEqual(got, want) {
		t.Fatalf("List() = %#v, want %#v", got, want)
	}

	for _, image := range []string{
		"data:image/png;base64,abc",
		"data:image/png,not-base64",
		"http://images.example.com/image.png",
		"not a URL",
		"https://",
	} {
		history.Add(image)
	}
	if got := history.List(); !reflect.DeepEqual(got, want) {
		t.Fatalf("List() after invalid Add() = %#v, want %#v", got, want)
	}
}

func TestImageHistoryDuplicateMovesToFront(t *testing.T) {
	history := newImageHistory()
	history.Add("https://images.example.com/one.png")
	history.Add("https://images.example.com/two.png")
	history.Add("https://images.example.com/one.png")

	want := []string{
		"https://images.example.com/one.png",
		"https://images.example.com/two.png",
	}
	if got := history.List(); !reflect.DeepEqual(got, want) {
		t.Fatalf("List() = %#v, want %#v", got, want)
	}
}

func TestImageHistoryRemove(t *testing.T) {
	history := newImageHistory()
	history.Add("https://images.example.com/one.png")
	history.Add("https://images.example.com/two.png")
	history.Remove("https://images.example.com/one.png")

	want := []string{"https://images.example.com/two.png"}
	if got := history.List(); !reflect.DeepEqual(got, want) {
		t.Fatalf("List() = %#v, want %#v", got, want)
	}

	history.Remove("http://images.example.com/two.png")
	if got := history.List(); !reflect.DeepEqual(got, want) {
		t.Fatalf("invalid Remove() changed List() to %#v", got)
	}
}

func TestImageHistoryConcurrentAccess(t *testing.T) {
	history := newImageHistory()
	var waitGroup sync.WaitGroup

	for worker := 0; worker < 8; worker++ {
		worker := worker
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for index := 0; index < 100; index++ {
				image := fmt.Sprintf("https://images.example.com/%d/%d.png", worker, index)
				history.Add(image)
				_ = history.List()
				history.Remove(image)
			}
		}()
	}

	waitGroup.Wait()
	if got := len(history.List()); got > maxImageHistory {
		t.Fatalf("List() returned %d items, want at most %d", got, maxImageHistory)
	}
}

func TestHistoryHandlerGet(t *testing.T) {
	history := newImageHistory()
	history.Add("https://images.example.com/one.png")
	config := appConfig{ImageHistory: history}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/history", nil)
	historyHandler(config).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response imageHistoryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if want := []string{"https://images.example.com/one.png"}; !reflect.DeepEqual(response.Images, want) {
		t.Fatalf("images = %#v, want %#v", response.Images, want)
	}
}

func TestHistoryHandlerEmptyHistoryReturnsEmptyArray(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/history", nil)
	historyHandler(appConfig{ImageHistory: newImageHistory()}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Body.String(); got != "{\"images\":[]}\n" {
		t.Fatalf("body = %q, want empty images array", got)
	}
}

func TestDatabaseHistoryServesBase64ImageByReference(t *testing.T) {
	database, err := openDatabase(filepath.Join(t.TempDir(), "img-app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	taskID, err := database.createTask("edit", generateRequest{Model: seedVRModel, Prompt: "Upscale", Size: "2048x2048"})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.completeTask(taskID, testHistoryPNGDataURL); err != nil {
		t.Fatal(err)
	}

	pageRecorder := httptest.NewRecorder()
	pageRequest := httptest.NewRequest(http.MethodGet, "/api/history", nil)
	historyHandler(appConfig{Database: database}).ServeHTTP(pageRecorder, pageRequest)
	if pageRecorder.Code != http.StatusOK {
		t.Fatalf("history status = %d, want %d", pageRecorder.Code, http.StatusOK)
	}
	var page historyPage
	if err := json.Unmarshal(pageRecorder.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Tasks) != 1 || page.Tasks[0].Image != historyImagePath(taskID) {
		t.Fatalf("history task = %#v, want reference %q", page.Tasks, historyImagePath(taskID))
	}

	imageRecorder := httptest.NewRecorder()
	imageRequest := httptest.NewRequest(http.MethodGet, historyImagePath(taskID), nil)
	historyHandler(appConfig{Database: database}).ServeHTTP(imageRecorder, imageRequest)
	if imageRecorder.Code != http.StatusOK || imageRecorder.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("image response = %d %q, want 200 image/png", imageRecorder.Code, imageRecorder.Header().Get("Content-Type"))
	}
}

func TestHistoryHandlerDelete(t *testing.T) {
	history := newImageHistory()
	image := "https://images.example.com/one.png"
	history.Add(image)
	body, err := json.Marshal(historyDeleteRequest{Image: image})
	if err != nil {
		t.Fatalf("marshal delete request: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/api/history", bytes.NewReader(body))
	historyHandler(appConfig{ImageHistory: history}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := history.List(); len(got) != 0 {
		t.Fatalf("history after delete = %#v, want empty", got)
	}
}

func TestHistoryHandlerRejectsInvalidDeleteRequest(t *testing.T) {
	for _, body := range []string{
		"",
		"not json",
		`{"image":"http://images.example.com/one.png"}`,
		`{"image":"data:image/png;base64,abc"}`,
		`{"image":"not a URL"}`,
		`{"image":"https://images.example.com/one.png"} {}`,
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodDelete, "/api/history", bytes.NewBufferString(body))
		historyHandler(appConfig{ImageHistory: newImageHistory()}).ServeHTTP(recorder, request)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("body %q status = %d, want %d", body, recorder.Code, http.StatusBadRequest)
		}
	}
}
