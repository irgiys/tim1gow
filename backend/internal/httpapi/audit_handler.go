package httpapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/irgiys/tim1gow/backend/internal/service"
)

// AuditHandler menangani endpoint HTTP untuk jejak audit (FR-09, AC-12, AC-13).
// Hanya menyediakan method GET (read-only). Tidak ada endpoint PUT/PATCH/DELETE (AC-13).
type AuditHandler struct {
	auditService service.AuditService
}

// NewAuditHandler membuat instance AuditHandler baru.
func NewAuditHandler(auditService service.AuditService) *AuditHandler {
	return &AuditHandler{auditService: auditService}
}

// RiwayatPengajuan menangani GET /api/pengajuan/{id}/audit (AC-12).
func (h *AuditHandler) RiwayatPengajuan(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "id pengajuan tidak valid", "")
		return
	}

	riwayat, err := h.auditService.AmbilRiwayatPengajuan(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"pengajuan_id": id,
		"total":        len(riwayat),
		"riwayat":      riwayat,
	})
}

// SemuaAudit menangani GET /api/audit.
func (h *AuditHandler) SemuaAudit(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	list, err := h.auditService.AmbilSemua(r.Context(), limit, offset)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"total": len(list),
		"data":  list,
	})
}
