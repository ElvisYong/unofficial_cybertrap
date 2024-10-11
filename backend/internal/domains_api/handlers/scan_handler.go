package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/gorilla/schema"

	"github.com/shannevie/unofficial_cybertrap/backend/internal/domains_api/dto"
	s "github.com/shannevie/unofficial_cybertrap/backend/internal/domains_api/service"
)

type ScansHandler struct {
	ScansService s.ScansService
}

// NewUserHandler creates a new instance of userHandler
func NewScansHandler(r *chi.Mux, service s.ScansService) {
	handler := &ScansHandler{
		ScansService: service,
	}

	r.Route("/v1/scans", func(r chi.Router) {
		r.Get("/", handler.GetAllScans)
		r.Post("/", handler.SingleScanDomain)
		r.Post("/multi", handler.MultiScanDomain)
		r.Get("/schedule", handler.GetAllScheduledScans)
		r.Post("/schedule", handler.ScheduleSingleScan)
		r.Delete("/schedule", handler.DeleteScheduledScanRequest)
	})
}

func (h *ScansHandler) GetAllScans(w http.ResponseWriter, r *http.Request) {
	scans, err := h.ScansService.GetAllScans()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to json body and return
	w.Header().Set("Content-Type", "application/json")
	// Encode domains and write to response
	json.NewEncoder(w).Encode(scans)
	w.WriteHeader(http.StatusOK)
}

func (h *ScansHandler) GetAllScheduledScans(w http.ResponseWriter, r *http.Request) {
	scans, err := h.ScansService.GetAllScheduledScans()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to json body and return
	w.Header().Set("Content-Type", "application/json")
	// Encode domains and write to response
	json.NewEncoder(w).Encode(scans)
	w.WriteHeader(http.StatusOK)
}

func (h *ScansHandler) SingleScanDomain(w http.ResponseWriter, r *http.Request) {
	req := &dto.ScanDomainRequest{}

	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	err := h.ScansService.ScanDomain(req.DomainID, req.TemplateIDs)
	if err != nil {
		http.Error(w, "Failed to scan domain", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ScansHandler) MultiScanDomain(w http.ResponseWriter, r *http.Request) {
	req := &[]dto.ScanDomainRequest{}

	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	err := h.ScansService.ScanMultiDomain(*req)
	if err != nil {
		http.Error(w, "Failed to scan domain", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ScansHandler) ScheduleSingleScan(w http.ResponseWriter, r *http.Request) {
	req := &dto.ScheduleSingleScanRequest{}

	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Call to ScansService to save the new scan domain and templates
	err := h.ScansService.CreateScheduleScanRecord(req.DomainID, req.ScheduledDate, req.TemplateIDs)
	if err != nil {
		http.Error(w, "Failed to create scan record", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Scan record created successfully"))
}

func (h *ScansHandler) DeleteScheduledScanRequest(w http.ResponseWriter, r *http.Request) {
	req := &dto.DeleteScheduledScanRequest{}

	if err := schema.NewDecoder().Decode(req, r.URL.Query()); err != nil {
		http.Error(w, "Invalid query parameters", http.StatusBadRequest)
		return
	}

	err := h.ScansService.DeleteScheduledScanRequest(req.ID)
	if err != nil {
		http.Error(w, "Failed to delete scheduled scan", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Scan record deleted successfully"))

}
