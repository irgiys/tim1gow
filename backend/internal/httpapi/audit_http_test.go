package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/irgiys/tim1gow/backend/internal/config"
	"github.com/irgiys/tim1gow/backend/internal/domain"
	"github.com/irgiys/tim1gow/backend/internal/service"
)

type fakeAuditRepoForHTTP struct {
	entries []domain.AuditTrailEntry
}

func (f *fakeAuditRepoForHTTP) Catat(_ context.Context, entry *domain.AuditTrailEntry) error {
	entry.ID = int64(len(f.entries) + 1)
	f.entries = append(f.entries, *entry)
	return nil
}

func (f *fakeAuditRepoForHTTP) AmbilRiwayatByPengajuan(_ context.Context, pengajuanID int64) ([]domain.AuditTrailEntry, error) {
	var out []domain.AuditTrailEntry
	for _, e := range f.entries {
		if e.PengajuanID != nil && *e.PengajuanID == pengajuanID {
			out = append(out, e)
		}
	}
	return out, nil
}

func (f *fakeAuditRepoForHTTP) AmbilSemua(_ context.Context, _, _ int) ([]domain.AuditTrailEntry, error) {
	out := make([]domain.AuditTrailEntry, len(f.entries))
	copy(out, f.entries)
	return out, nil
}

// Test AC-13: Tidak ada endpoint yang bisa mengubah atau menghapus baris audit trail.
// Memastikan percobaan PUT/PATCH/DELETE pada resource audit menghasilkan 404 / 405.
func TestAuditHTTP_AC13_AppendOnlyTidakAdaMutasi(t *testing.T) {
	auditRepo := &fakeAuditRepoForHTTP{}
	auditSvc := service.NewAuditService(auditRepo)
	audH := NewAuditHandler(auditSvc)

	cfg := config.Config{AppEnv: "test"}
	handler := NewRouterWithHandlers(cfg, nil, nil, audH)

	forbiddenMethods := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

	// AC-13 meminta bukti dari DAFTAR ROUTE, jadi SEMUA route audit yang
	// terdaftar di router harus diperiksa — bukan hanya satu. Sebelumnya
	// /api/audit tidak diuji sama sekali padahal ia route audit juga.
	auditRoutes := []string{
		"/api/pengajuan/1/audit",
		"/api/audit",
	}

	for _, route := range auditRoutes {
		for _, method := range forbiddenMethods {
			t.Run("Metode_"+method+"_Ditolak_pada_"+route, func(t *testing.T) {
				req := httptest.NewRequest(method, route, nil)
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, req)

				if w.Code != http.StatusNotFound && w.Code != http.StatusMethodNotAllowed {
					t.Fatalf("metode %s pada %s harus ditolak (404/405), dapat status %d",
						method, route, w.Code)
				}
			})
		}
	}

	// Pembanding: GET pada kedua route harus TETAP bisa diakses. Tanpa ini,
	// router yang memblokir seluruh metode akan meloloskan test di atas.
	for _, route := range auditRoutes {
		t.Run("GET_tetap_diizinkan_pada_"+route, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, route, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code == http.StatusNotFound || w.Code == http.StatusMethodNotAllowed {
				t.Fatalf("GET %s seharusnya tersedia (audit trail dapat dibaca), dapat status %d",
					route, w.Code)
			}
		})
	}
}

// Test GET /api/pengajuan/{id}/audit mengembalikan riwayat audit dengan benar.
func TestAuditHTTP_GetRiwayatPengajuan(t *testing.T) {
	auditRepo := &fakeAuditRepoForHTTP{}
	pID := int64(10)
	auditRepo.entries = append(auditRepo.entries, domain.AuditTrailEntry{
		ID:            1,
		PengajuanID:   &pID,
		Aksi:          domain.AksiBuatPengajuan,
		StatusSebelum: "",
		StatusSesudah: domain.StatusDraft,
		Catatan:       "dibuat oleh AO",
		ActorID:       1,
		ActorRole:     domain.PeranAO,
		CreatedAt:     time.Now(),
	})

	auditSvc := service.NewAuditService(auditRepo)
	audH := NewAuditHandler(auditSvc)
	cfg := config.Config{AppEnv: "test"}
	handler := NewRouterWithHandlers(cfg, nil, nil, audH)

	req := httptest.NewRequest(http.MethodGet, "/api/pengajuan/10/audit", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/pengajuan/10/audit gagal dengan status %d", w.Code)
	}
}
