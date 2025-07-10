package products

import (
	"amankumarsingh77/specmatic-coding-test/internal/logger"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

func WriteError(w http.ResponseWriter, r *http.Request, status int, msg string, lg logger.Logger) {
	if lg != nil {
		lg.Errorf("%s %s -> %d: %s", r.Method, r.URL.Path, status, msg)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := ErrorResponse{
		Timestamp: time.Now().UTC(),
		Status:    status,
		Error:     msg,
		Path:      r.URL.Path,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func HandlePostProduct(store Store, lg logger.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			WriteError(w, r, http.StatusBadRequest, "failed to read request body", lg)
			return
		}

		var raw map[string]*json.RawMessage
		if err := json.Unmarshal(bodyBytes, &raw); err != nil {
			WriteError(w, r, http.StatusBadRequest, "invalid JSON payload", lg)
			return
		}

		// here just to make sure the api is backward compatible this check is required. Valid inputs: {cost:200}, Invalid input: {cost:null}
		if rawCost, ok := raw["cost"]; ok && (rawCost == nil || string(*rawCost) == "null") {
			WriteError(w, r, http.StatusBadRequest, "cost cannot be null", lg)
			return
		}

		var payload ProductDetails
		if err := json.Unmarshal(bodyBytes, &payload); err != nil {
			WriteError(w, r, http.StatusBadRequest, "invalid JSON body format", lg)
			return
		}
		if err := validateProductDetails(&payload); err != nil {
			WriteError(w, r, http.StatusBadRequest, err.Error(), lg)
			return
		}

		created := store.Add(payload)
		if lg != nil {
			lg.Infof("Added product id=%d type=%s name=%s", created.ID, created.Type, created.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(struct {
			ID int `json:"id"`
		}{ID: created.ID})
	}
}

func HandleGetProducts(store Store, lg logger.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		qType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("type")))

		var products []Product
		if qType == "" {
			products = store.ListAll()
			if len(products) == 0 {
				WriteError(w, r, http.StatusBadRequest, "no products added yet", lg)
				return
			}
		} else {
			if _, ok := validProductTypes[qType]; !ok {
				WriteError(w, r, http.StatusBadRequest, "missing or invalid type query param", lg)
				return
			}
			products = store.ListByType(qType)
		}

		if lg != nil {
			lg.Infof("Fetched %d products", len(products))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(products)
	}
}
