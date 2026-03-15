package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"apiservices/address-validation/internal/address/geo"
)

type Handler struct {
	service *geo.Service
}

func NewHandler(service *geo.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/v1/address/") {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/address/"), "/")
	switch path {
	case "validate":
		h.handleValidate(w, r)
	case "geocode":
		h.handleGeocode(w, r)
	case "reverse":
		h.handleReverse(w, r)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (h *Handler) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req geo.ValidateInput
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.service.ValidateAddress(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}

func (h *Handler) handleGeocode(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req geo.GeocodeInput
		if err := decodeJSONBody(w, r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		result, err := h.service.Geocode(r.Context(), req)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": result})
	case http.MethodGet:
		limit := 1
		if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				writeError(w, http.StatusBadRequest, "limit must be an integer")
				return
			}
			limit = parsed
		}
		result, err := h.service.Geocode(r.Context(), geo.GeocodeInput{
			Address: r.URL.Query().Get("address"),
			Region:  r.URL.Query().Get("region"),
			Limit:   limit,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": result})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleReverse(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req geo.ReverseInput
		if err := decodeJSONBody(w, r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		result, err := h.service.Reverse(r.Context(), req)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": result})
	case http.MethodGet:
		lat, err := strconv.ParseFloat(strings.TrimSpace(r.URL.Query().Get("lat")), 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "lat must be a float")
			return
		}
		lon, err := strconv.ParseFloat(strings.TrimSpace(r.URL.Query().Get("lon")), 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "lon must be a float")
			return
		}

		result, err := h.service.Reverse(r.Context(), geo.ReverseInput{
			Latitude:  lat,
			Longitude: lon,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": result})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"failed to marshal response"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, out any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return errors.New("invalid json body")
	}

	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("json body must contain a single object")
	}
	return nil
}
