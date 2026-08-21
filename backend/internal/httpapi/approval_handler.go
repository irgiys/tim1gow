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

// getActor membaca identitas aktor dari token yang sudah diverifikasi.
//
// SEBELUMNYA berkas ini membaca header X-Actor-ID / X-Actor-Role dengan fallback
// ke (id=1, ANL). Itu dua lubang sekaligus:
//
//   - Identitas dapat dipalsukan klien. KCP yang sah mengirim
//     "X-Actor-Role: KOM" membuat server menilai BR-02 memakai peran palsu.
//   - Tanpa header, SETIAP aktor dianggap id=1. Akibatnya audit trail mencatat
//     AJUKAN_APPROVAL sebagai actor_id=1 padahal ANL adalah id=2 (BR-10
//     dilanggar: jejaknya menunjuk orang yang salah), dan BR-09 menolak KCP
//     yang sah karena id=1 kebetulan adalah AO pembuat pengajuan.
//
// Identitas sekarang SELALU dari context yang diisi MiddlewareAuth. Ketika
// identitas tidak ada, pemanggil dijawab 401 dan tidak ada aktor yang ditebak —
// perubahan keadaan tanpa aktor yang pasti melanggar BR-10.
func getActor(w http.ResponseWriter, r *http.Request) (int64, domain.Peran, bool) {
	ident, ok := IdentitasDari(r.Context())
	if !ok || ident.PenggunaID <= 0 {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "autentikasi diperlukan", "")
		return 0, "", false
	}
	return ident.PenggunaID, ident.Peran, true
}

// AjukanApproval menangani POST /api/pengajuan/{id}/ajukan-approval.
func (h *ApprovalHandler) AjukanApproval(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "id pengajuan tidak valid", "")
		return
	}

	actorID, actorRole, ok := getActor(w, r)
	if !ok {
		return
	}
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

	actorID, actorRole, ok := getActor(w, r)
	if !ok {
		return
	}
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

	// Kegagalan validasi masukan -> 400, BUKAN 500 (AGENTS.md bagian 4.3).
	// Tanpa cabang ini, "nama nasabah wajib diisi" dari PengajuanService
	// terlihat seperti kerusakan server dan pesannya hilang dari respons.
	var valErr *service.ValidationError
	if errors.As(err, &valErr) {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", valErr.Message, "")
		return
	}

	// Baris yang diminta tidak ada -> 404, bukan 500. Tanpa pemetaan ini,
	// id pengajuan yang salah ketik terlihat seperti kerusakan server.
	if errors.Is(err, service.ErrTidakDitemukan) {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "sumber daya tidak ditemukan", "")
		return
	}

	if errors.Is(err, service.ErrNIKTidakDitemukanSlik) {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "NIK tidak ditemukan di SLIK", "")
		return
	}

	// Layanan SLIK tidak dapat dihubungi (503/timeout) -> 502 Bad Gateway (AGENTS.md 4.3).
	// Backend gagal memakai dependensi hulu; pengajuan TIDAK boleh dianggap bersih.
	if errors.Is(err, service.ErrSlikTidakTersedia) {
		respondError(w, http.StatusBadGateway, "SLIK_UNAVAILABLE", "layanan SLIK tidak tersedia", "")
		return
	}

	respondError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "terjadi kesalahan internal", "")
}
