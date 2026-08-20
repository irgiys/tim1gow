package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/irgiys/tim1gow/backend/internal/domain"
	"github.com/irgiys/tim1gow/backend/internal/service"
)

// ApprovalHandler menangani endpoint HTTP untuk alur approval berjenjang (FR-08).
type ApprovalHandler struct {
	approvalService service.ApprovalService
}

// NewApprovalHandler membuat instance ApprovalHandler baru.
func NewApprovalHandler(approvalService service.ApprovalService) *ApprovalHandler {
	return &ApprovalHandler{approvalService: approvalService}
}

// getActor membaca identitas aktor dari header (atau context saat ada middleware auth).
func getActor(r *http.Request) (int64, domain.Peran) {
	actorIDStr := r.Header.Get("X-Actor-ID")
	actorRoleStr := r.Header.Get("X-Actor-Role")

	actorID, _ := strconv.ParseInt(actorIDStr, 10, 64)
	if actorID <= 0 {
		actorID = 1 // default actor fallback bila belum lewat token
	}

	actorRole := domain.Peran(actorRoleStr)
	if actorRole == "" {
		actorRole = domain.PeranANL
	}

	return actorID, actorRole
}

// AjukanApproval menangani POST /api/pengajuan/{id}/ajukan-approval.
func (h *ApprovalHandler) AjukanApproval(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "id pengajuan tidak valid", "")
		return
	}

	actorID, actorRole := getActor(r)
	if err := h.approvalService.AjukanKeApproval(r.Context(), id, actorID, actorRole); err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"message": "pengajuan berhasil diajukan ke approval",
	})
}

// PutuskanApproval menangani POST /api/pengajuan/{id}/approval.
func (h *ApprovalHandler) PutuskanApproval(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "id pengajuan tidak valid", "")
		return
	}

	var req domain.ApprovalDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "format JSON tidak valid", "")
		return
	}

	actorID, actorRole := getActor(r)
	dec, err := h.approvalService.PutuskanApproval(r.Context(), id, req, actorID, actorRole)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"keputusan": dec,
	})
}

// DetailApproval menangani GET /api/pengajuan/{id}/approval.
func (h *ApprovalHandler) DetailApproval(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "id pengajuan tidak valid", "")
		return
	}

	detail, err := h.approvalService.AmbilDetailApproval(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, detail)
}

// handleServiceError memetakan error domain/service ke status HTTP yang sesuai (AGENTS.md 4.3).
func handleServiceError(w http.ResponseWriter, err error) {
	var brErr *domain.BusinessRuleError
	if errors.As(err, &brErr) {
		switch brErr.Rule {
		case "FORBIDDEN":
			respondError(w, http.StatusForbidden, "FORBIDDEN", brErr.Message, "")
		case "NOT_FOUND":
			respondError(w, http.StatusNotFound, "NOT_FOUND", brErr.Message, "")
		case "VALIDATION_ERROR":
			respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", brErr.Message, "")
		default:
			// BR-xx pelanggaran aturan bisnis -> 422 Unprocessable Entity
			respondError(w, http.StatusUnprocessableEntity, "BUSINESS_RULE_VIOLATION", brErr.Message, brErr.Rule)
		}
		return
	}

	var cfgErr *domain.ConfigError
	if errors.As(err, &cfgErr) {
		respondError(w, http.StatusInternalServerError, "CONFIG_ERROR", cfgErr.Message, "")
		return
	}

	respondError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "terjadi kesalahan internal", "")
}
