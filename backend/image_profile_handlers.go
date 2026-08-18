package main

import (
	"net/http"
	"strings"
)

func imageProfilesHandler(config appConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if config.Database == nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "database is not initialized"})
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/api/image-profiles")
		path = strings.Trim(path, "/")
		parts := []string{}
		if path != "" {
			parts = strings.Split(path, "/")
		}

		if len(parts) == 0 {
			handleImageProfileCollection(w, r, config)
			return
		}
		if len(parts) == 2 && parts[1] == "activate" && r.Method == http.MethodPost {
			handleImageProfileActivate(w, config, parts[0])
			return
		}
		if len(parts) != 1 {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "image profile not found"})
			return
		}

		switch r.Method {
		case http.MethodPut:
			handleImageProfileUpdate(w, r, config, parts[0])
		case http.MethodDelete:
			handleImageProfileDelete(w, config, parts[0])
		default:
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		}
	}
}

func handleImageProfileCollection(w http.ResponseWriter, r *http.Request, config appConfig) {
	switch r.Method {
	case http.MethodGet:
		profiles, err := config.Database.listImageProfiles()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to load image profiles"})
			return
		}
		result := make([]imageProfileResponse, 0, len(profiles))
		for _, profile := range profiles {
			result = append(result, imageProfileResponseFrom(profile))
		}
		writeJSON(w, http.StatusOK, imageProfilesResponse{Profiles: result})
	case http.MethodPost:
		var input imageProfileRequest
		if !decodeJSONBody(w, r, &input) {
			return
		}
		profile, err := config.Database.createImageProfile(input.Name, input.Endpoint, input.APIKey)
		if err != nil {
			writeImageProfileError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, imageProfileResponseFrom(profile))
	default:
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
	}
}

func handleImageProfileUpdate(w http.ResponseWriter, r *http.Request, config appConfig, id string) {
	var input imageProfileRequest
	if !decodeJSONBody(w, r, &input) {
		return
	}
	profile, err := config.Database.updateImageProfile(id, input.Name, input.Endpoint, input.APIKey)
	if err != nil {
		writeImageProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, imageProfileResponseFrom(profile))
}

func handleImageProfileActivate(w http.ResponseWriter, config appConfig, id string) {
	profile, err := config.Database.activateImageProfile(id)
	if err != nil {
		writeImageProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, imageProfileResponseFrom(profile))
}

func handleImageProfileDelete(w http.ResponseWriter, config appConfig, id string) {
	if err := config.Database.deleteImageProfile(id); err != nil {
		writeImageProfileError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func imageProfileResponseFrom(profile imageProfile) imageProfileResponse {
	return imageProfileResponse{
		ID:        profile.ID,
		Name:      profile.Name,
		Endpoint:  profile.Endpoint,
		APIKey:    profile.APIKey,
		IsActive:  profile.IsActive,
		CreatedAt: profile.CreatedAt,
		UpdatedAt: profile.UpdatedAt,
	}
}

func writeImageProfileError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	if err == errNotFound {
		status = http.StatusNotFound
	} else if strings.Contains(err.Error(), "请") || strings.Contains(err.Error(), "endpoint must") {
		status = http.StatusBadRequest
	}
	writeJSON(w, status, errorResponse{Error: err.Error()})
}
