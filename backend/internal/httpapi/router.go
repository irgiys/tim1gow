// Package httpapi berisi handler, route, dan middleware HTTP. Ia TIDAK memuat
// aturan bisnis (AGENTS.md Larangan 17) — hanya menghubungkan HTTP ke service.
package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"

	"github.com/irgiys/tim1gow/backend/internal/config"
	"github.com/irgiys/tim1gow/backend/internal/repository"
	"github.com/irgiys/tim1gow/backend/internal/repository/db"
	"github.com/irgiys/tim1gow/backend/internal/service"
)

// errorResponse adalah satu-satunya bentuk respons error untuk seluruh API
// (AGENTS.md bagian 4.3). Handler tidak menyusun JSON error manual.
type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Rule    string `json:"rule,omitempty"`
}

// respondJSON menulis payload JSON dengan status yang diberikan.
func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// respondError adalah helper terpusat untuk seluruh error API. Pesan tidak
// boleh memuat data pribadi (BR-11).
func respondError(w http.ResponseWriter, status int, code, message, rule string) {
	respondJSON(w, status, errorResponse{Error: code, Message: message, Rule: rule})
}

// NewRouter membangun router Chi lengkap dengan middleware dasar dan route.
// gdb boleh nil (mode tanpa DB); saat itu /readyz melaporkan 503.
func NewRouter(cfg config.Config, gdb *gorm.DB) http.Handler {
	var appH *ApprovalHandler
	var audH *AuditHandler

	if gdb != nil {
		paramRepo := repository.NewParameterRepository(gdb)
		approvalRepo := repository.NewApprovalRepository(gdb)
		auditRepo := repository.NewAuditRepository(gdb)

		auditSvc := service.NewAuditService(auditRepo)
		approvalSvc := service.NewApprovalService(approvalRepo, paramRepo, auditSvc)

		appH = NewApprovalHandler(approvalSvc)
		audH = NewAuditHandler(auditSvc)
	}

	return NewRouterWithHandlers(cfg, gdb, appH, audH)
}

// NewRouterWithHandlers membangun router Chi dengan handler yang disediakan (mudah di-test/mock).
func NewRouterWithHandlers(cfg config.Config, gdb *gorm.DB, appH *ApprovalHandler, audH *AuditHandler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	// Liveness: proses hidup, tidak menyentuh dependensi.
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
			"env":    cfg.AppEnv,
		})
	})

	// Readiness: pastikan DB benar-benar bisa dihubungi.
	r.Get("/readyz", func(w http.ResponseWriter, req *http.Request) {
		if gdb == nil {
			respondError(w, http.StatusServiceUnavailable, "DB_NOT_CONFIGURED",
				"database belum dikonfigurasi", "")
			return
		}
		ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
		defer cancel()
		if err := db.Ping(ctx, gdb); err != nil {
			respondError(w, http.StatusServiceUnavailable, "DB_UNAVAILABLE",
				"database tidak dapat dihubungi", "")
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"status": "ready", "db": "ok"})
	})

	// Route API
	r.Route("/api", func(api chi.Router) {
		if appH != nil {
			api.Post("/pengajuan/{id}/ajukan-approval", appH.AjukanApproval)
			api.Post("/pengajuan/{id}/approval", appH.PutuskanApproval)
			api.Get("/pengajuan/{id}/approval", appH.DetailApproval)
		}

		if audH != nil {
			// FR-09 & AC-12 / AC-13: Audit Trail hanya ada method GET (append-only)
			api.Get("/pengajuan/{id}/audit", audH.RiwayatPengajuan)
			api.Get("/audit", audH.SemuaAudit)
		}
	})

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "sumber daya tidak ditemukan", "")
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "metode HTTP tidak diizinkan", "")
	})

	return r
}
