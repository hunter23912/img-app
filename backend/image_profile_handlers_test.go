package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestImageProfilesHandlerSupportsCRUDAndActivation(t *testing.T) {
	database, err := openDatabase(filepath.Join(t.TempDir(), "img-app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	config := appConfig{Database: database}

	create := func(name, endpoint, key string) imageProfileResponse {
		body, _ := json.Marshal(imageProfileRequest{Name: name, Endpoint: endpoint, APIKey: key})
		recorder := httptest.NewRecorder()
		imageProfilesHandler(config).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/image-profiles", bytes.NewReader(body)))
		if recorder.Code != http.StatusCreated {
			t.Fatalf("create status = %d: %s", recorder.Code, recorder.Body.String())
		}
		var profile imageProfileResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &profile); err != nil {
			t.Fatal(err)
		}
		return profile
	}

	first := create("A", "https://a.example", "key-a")
	second := create("B", "https://b.example", "key-b")
	if !first.IsActive || second.IsActive {
		t.Fatalf("created profiles active states = %v, %v", first.IsActive, second.IsActive)
	}

	activate := httptest.NewRecorder()
	imageProfilesHandler(config).ServeHTTP(activate, httptest.NewRequest(http.MethodPost, "/api/image-profiles/"+second.ID+"/activate", nil))
	if activate.Code != http.StatusOK {
		t.Fatalf("activate status = %d: %s", activate.Code, activate.Body.String())
	}

	list := httptest.NewRecorder()
	imageProfilesHandler(config).ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/image-profiles", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d", list.Code)
	}
	var response imageProfilesResponse
	if err := json.Unmarshal(list.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Profiles) != 2 || response.Profiles[0].IsActive || response.Profiles[0].APIKey != "key-a" || !response.Profiles[1].IsActive || response.Profiles[1].APIKey != "key-b" {
		t.Fatalf("listed profiles = %#v", response.Profiles)
	}

	remove := httptest.NewRecorder()
	imageProfilesHandler(config).ServeHTTP(remove, httptest.NewRequest(http.MethodDelete, "/api/image-profiles/"+first.ID, nil))
	if remove.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d: %s", remove.Code, remove.Body.String())
	}
}

func TestImageProfilesHandlerNormalizesEndpointsAndGeneratesNames(t *testing.T) {
	database, err := openDatabase(filepath.Join(t.TempDir(), "img-app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	config := appConfig{Database: database}

	create := func(name, endpoint string) imageProfileResponse {
		body, err := json.Marshal(imageProfileRequest{Name: name, Endpoint: endpoint})
		if err != nil {
			t.Fatal(err)
		}
		recorder := httptest.NewRecorder()
		imageProfilesHandler(config).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/image-profiles", bytes.NewReader(body)))
		if recorder.Code != http.StatusCreated {
			t.Fatalf("create status = %d: %s", recorder.Code, recorder.Body.String())
		}
		var profile imageProfileResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &profile); err != nil {
			t.Fatal(err)
		}
		return profile
	}

	first := create("", "example.com")
	if first.Name != "中转站 1" || first.Endpoint != "https://example.com" {
		t.Fatalf("first profile = %#v", first)
	}

	second := create("", "http://http.example")
	if second.Name != "中转站 2" || second.Endpoint != "http://http.example" {
		t.Fatalf("second profile = %#v", second)
	}

	third := create("", "https://https.example")
	if third.Name != "中转站 3" || third.Endpoint != "https://https.example" {
		t.Fatalf("third profile = %#v", third)
	}

	body, err := json.Marshal(imageProfileRequest{Name: "  ", Endpoint: "updated.example", APIKey: "updated-key"})
	if err != nil {
		t.Fatal(err)
	}
	updated := httptest.NewRecorder()
	imageProfilesHandler(config).ServeHTTP(updated, httptest.NewRequest(http.MethodPut, "/api/image-profiles/"+first.ID, bytes.NewReader(body)))
	if updated.Code != http.StatusOK {
		t.Fatalf("update status = %d: %s", updated.Code, updated.Body.String())
	}
	var updatedProfile imageProfileResponse
	if err := json.Unmarshal(updated.Body.Bytes(), &updatedProfile); err != nil {
		t.Fatal(err)
	}
	if updatedProfile.Name != "中转站 1" || updatedProfile.Endpoint != "https://updated.example" || updatedProfile.APIKey != "updated-key" {
		t.Fatalf("updated profile = %#v", updatedProfile)
	}
}
