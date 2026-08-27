package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"img-app/backend/internal/store"
)

func modelsHandler(config appConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if config.Database == nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "database is not initialized"})
			return
		}

		scopeKey, err := config.imageModelScope()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to resolve image model scope"})
			return
		}

		path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/models"), "/")
		switch r.Method {
		case http.MethodGet:
			if path != "" {
				writeJSON(w, http.StatusNotFound, errorResponse{Error: "image model not found"})
				return
			}
			handleImageModelList(w, config.Database, scopeKey)
		case http.MethodPost:
			if path != "" {
				writeJSON(w, http.StatusNotFound, errorResponse{Error: "image model not found"})
				return
			}
			handleImageModelCreate(w, r, config.Database, scopeKey)
		case http.MethodDelete:
			if path == "" {
				writeJSON(w, http.StatusBadRequest, errorResponse{Error: "model id is required"})
				return
			}
			handleImageModelDelete(w, config.Database, scopeKey, path)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		}
	}
}

func handleImageModelList(w http.ResponseWriter, database *appDatabase, scopeKey string) {
	customModels, err := database.ListImageModels(scopeKey)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to load image models"})
		return
	}
	options := make([]modelOptionResponse, 0)
	for _, option := range store.FixedModelOptions() {
		options = append(options, modelOptionResponse{ID: option.ID, Value: option.Value, Label: option.Label, BuiltIn: option.BuiltIn})
	}
	for _, model := range customModels {
		options = append(options, modelOptionResponse{
			ID:      model.ID,
			Value:   model.Model,
			Label:   model.Model,
			BuiltIn: false,
		})
	}
	writeJSON(w, http.StatusOK, imageModelsResponse{Models: options})
}

func handleImageModelCreate(w http.ResponseWriter, r *http.Request, database *appDatabase, scopeKey string) {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	var input imageModelRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
		return
	}
	if err := ensureJSONBodyEnded(decoder); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
		return
	}
	model, err := database.CreateImageModel(scopeKey, input.Model)
	if err != nil {
		writeImageModelError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, modelOptionResponse{
		ID:      model.ID,
		Value:   model.Model,
		Label:   model.Model,
		BuiltIn: false,
	})
}

func handleImageModelDelete(w http.ResponseWriter, database *appDatabase, scopeKey, id string) {
	if strings.HasPrefix(id, "builtin:") {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "固定模型不能删除"})
		return
	}
	if err := database.DeleteImageModel(scopeKey, id); err != nil {
		if err == errNotFound {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "image model not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "删除自定义模型失败"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeImageModelError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	if strings.Contains(err.Error(), "请") || strings.Contains(err.Error(), "固定模型") || strings.Contains(err.Error(), "同名") || strings.Contains(err.Error(), "最多") || strings.Contains(err.Error(), "控制字符") {
		status = http.StatusBadRequest
	}
	writeJSON(w, status, errorResponse{Error: err.Error()})
}
