package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"go.mongodb.org/mongo-driver/bson/primitive"

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
		r.Get("/{id}", handler.GetScanById)
		r.Get("/multi", handler.GetAllMultiScans)
		r.Get("/multiscan/{multiScanId}", handler.GetScansByMultiScanId)
		r.Get("/schedule", handler.GetAllScheduledScans)
		r.Post("/", handler.ScanDomains)
		r.Post("/all", handler.ScanAllDomains)
		r.Post("/schedule", handler.ScheduleScan)
		r.Delete("/schedule/{id}", handler.DeleteScheduledScanRequest)
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

func (h *ScansHandler) ScanAllDomains(w http.ResponseWriter, r *http.Request) {
	err := h.ScansService.ScanAllDomains()
	if err != nil {
		http.Error(w, "Failed to scan all domains", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ScansHandler) ScanDomains(w http.ResponseWriter, r *http.Request) {
	req := &dto.ScanDomainRequest{}

	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// If allNucleiTemplates is false and templateIds is empty, return error
	if !req.ScanAllNuclei && len(req.TemplateIds) == 0 {
		http.Error(w, "Template IDs are required when scanAllNuclei is false", http.StatusBadRequest)
		return
	}

	// Convert domainIds to []primitive.ObjectID
	domainIds := make([]primitive.ObjectID, 0)
	for _, domainId := range req.DomainIds {
		domainObjectID, err := primitive.ObjectIDFromHex(domainId)
		if err != nil {
			http.Error(w, "Invalid domain ID", http.StatusBadRequest)
			return
		}
		domainIds = append(domainIds, domainObjectID)
	}

	// Convert templateIds to []primitive.ObjectID
	templateIds := make([]primitive.ObjectID, 0)
	for _, templateId := range req.TemplateIds {
		templateObjectID, err := primitive.ObjectIDFromHex(templateId)
		if err != nil {
			http.Error(w, "Invalid template ID", http.StatusBadRequest)
			return
		}
		templateIds = append(templateIds, templateObjectID)
	}

	err := h.ScansService.ScanDomains(domainIds, templateIds, req.ScanAllNuclei)
	if err != nil {
		http.Error(w, "Failed to scan domain", http.StatusInternalServerError)
		return
	}

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

func (h *ScansHandler) ScheduleScan(w http.ResponseWriter, r *http.Request) {
	req := &dto.ScheduleScanRequest{}

	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Call to ScansService to schedule scan for all domains
	err := h.ScansService.ScheduleScan(req)
	if err != nil {
		http.Error(w, "Failed to schedule scan for all domains", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Scan scheduled for all domains successfully"))
}

func (h *ScansHandler) DeleteScheduledScanRequest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing ID parameter", http.StatusBadRequest)
		return
	}

	err := h.ScansService.DeleteScheduledScanRequest(id)
	if err != nil {
		http.Error(w, "Failed to delete scheduled scan", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Scan record deleted successfully"))

}

func (h *ScansHandler) GetAllMultiScans(w http.ResponseWriter, r *http.Request) {
	multiScans, err := h.ScansService.GetAllMultiScans()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(multiScans)
	w.WriteHeader(http.StatusOK)
}

func (h *ScansHandler) GetScansByMultiScanId(w http.ResponseWriter, r *http.Request) {
	multiScanId := chi.URLParam(r, "multiScanId")
	if multiScanId == "" {
		http.Error(w, "Missing multiScanId parameter", http.StatusBadRequest)
		return
	}

	response, err := h.ScansService.GetScansByMultiScanId(multiScanId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *ScansHandler) GetScanById(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing ID parameter", http.StatusBadRequest)
		return
	}

	scan, err := h.ScansService.GetScanById(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scan)
}
