package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/irgiys/tim1gow/backend/internal/config"
	"github.com/irgiys/tim1gow/backend/internal/domain"
	"github.com/irgiys/tim1gow/backend/internal/service"
)

// Test di berkas ini diturunkan dari AC-08 (docs/SRS-iMitra.md):
//
//	"ANL override grade dari 2 ke 3; sistem menolak jika alasan kosong;
//	 setelah diisi, override tercatat di audit trail dengan identitas ANL."
//
// Verifikasinya: alasan kosong -> 400; alasan terisi -> baris audit_trail baru
// dengan actor = ANL terkait.

// fakeAuditRecorder merekam panggilan Catat supaya test bisa memeriksa isi
// baris audit yang benar-benar ditulis, bukan hanya status HTTP-nya.
type fakeAuditRecorder struct {
	entries []domain.CatatAuditInput
	gagal   error
}

func (f *fakeAuditRecorder) Catat(_ context.Context, in domain.CatatAuditInput) error {
	if f.gagal != nil {
		return f.gagal
	}
	f.entries = append(f.entries, in)
	return nil
}

func (f *fakeAuditRecorder) AmbilRiwayatPengajuan(_ context.Context, pengajuanID int64) ([]domain.AuditTrailEntry, error) {
	out := make([]domain.AuditTrailEntry, 0, len(f.entries))
	for _, e := range f.entries {
		if e.PengajuanID != nil && *e.PengajuanID == pengajuanID {
			out = append(out, domain.AuditTrailEntry{
				PengajuanID: e.PengajuanID,
				Aksi:        e.Aksi,
				Catatan:     e.Catatan,
				ActorID:     e.ActorID,
				ActorRole:   e.ActorRole,
			})
		}
	}
	return out, nil
}

func (f *fakeAuditRecorder) AmbilSemua(_ context.Context, _, _ int) ([]domain.AuditTrailEntry, error) {
	return nil, nil
}

// routerOverride membangun router dengan skoring service yang memakai audit.
func routerOverride(param service.ParameterRepository, audit service.AuditService) http.Handler {
	h := NewSkoringHandler(
		service.NewSkoringServiceWithAudit(param, audit),
		service.NewMarginService(param),
	)
	return NewRouterWithAllHandlers(config.Config{AppEnv: "test"}, nil, nil, nil, h)
}

// patchJSON mengirim PATCH dengan identitas aktor di header.
func patchJSON(t *testing.T, h http.Handler, path string, body any, actorID, actorRole string) *httptest.ResponseRecorder {
	t.Helper()
	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPatch, path, bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	if actorID != "" {
		req.Header.Set("X-Actor-ID", actorID)
	}
	if actorRole != "" {
		req.Header.Set("X-Actor-Role", actorRole)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

var _ = errors.New // dipakai oleh test kegagalan audit di bawah

// TestHTTP_AC08_OverrideGrade2Ke3TercatatDiAuditTrail adalah AC-08 utuh:
// alasan kosong ditolak, alasan terisi menghasilkan baris audit dengan
// identitas ANL. Kedua arah ada dalam satu test (Larangan 18).
func TestHTTP_AC08_OverrideGrade2Ke3TercatatDiAuditTrail(t *testing.T) {
	audit := &fakeAuditRecorder{}
	h := routerOverride(newFakeParamRepoSkoring(), audit)

	t.Run("alasan kosong ditolak", func(t *testing.T) {
		rec := patchJSON(t, h, "/api/pengajuan/7/skoring/override", map[string]any{
			"gradeSemula": 2, "gradeBaru": 3, "alasan": "",
		}, "42", "ANL")

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, mau 400 (body=%s)", rec.Code, rec.Body.String())
		}
		if len(audit.entries) != 0 {
			t.Errorf("override ditolak tetapi %d baris audit tertulis; tidak boleh ada jejak perubahan yang tidak terjadi",
				len(audit.entries))
		}
	})

	t.Run("alasan hanya spasi juga ditolak", func(t *testing.T) {
		rec := patchJSON(t, h, "/api/pengajuan/7/skoring/override", map[string]any{
			"gradeSemula": 2, "gradeBaru": 3, "alasan": "   ",
		}, "42", "ANL")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, mau 400", rec.Code)
		}
	})

	t.Run("alasan terisi tercatat dengan identitas ANL", func(t *testing.T) {
		rec := patchJSON(t, h, "/api/pengajuan/7/skoring/override", map[string]any{
			"gradeSemula": 2, "gradeBaru": 3,
			"alasan": "kondisi usaha membaik setelah kunjungan ulang",
		}, "42", "ANL")

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
		}
		if len(audit.entries) != 1 {
			t.Fatalf("jumlah baris audit = %d, mau 1", len(audit.entries))
		}

		e := audit.entries[0]
		if e.Aksi != domain.AksiOverrideSkor {
			t.Errorf("aksi = %q, mau %q", e.Aksi, domain.AksiOverrideSkor)
		}
		if e.ActorID != 42 {
			t.Errorf("actorId = %d, mau 42 — identitas ANL wajib tercatat (BR-10)", e.ActorID)
		}
		if e.ActorRole != domain.PeranANL {
			t.Errorf("actorRole = %q, mau ANL", e.ActorRole)
		}
		if e.PengajuanID == nil || *e.PengajuanID != 7 {
			t.Errorf("pengajuanId audit tidak sesuai path")
		}
		if e.Catatan == "" {
			t.Error("catatan audit kosong; alasan override wajib ikut tersimpan")
		}
	})
}

// TestHTTP_Override_HanyaANL memastikan otorisasi ditegakkan di server, bukan
// dengan menyembunyikan tombol di UI (Larangan 6).
func TestHTTP_Override_HanyaANL(t *testing.T) {
	audit := &fakeAuditRecorder{}
	h := routerOverride(newFakeParamRepoSkoring(), audit)

	body := map[string]any{"gradeSemula": 2, "gradeBaru": 3, "alasan": "alasan sah"}

	rec := patchJSON(t, h, "/api/pengajuan/7/skoring/override", body, "9", "AO")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("AO: status = %d, mau 403 (body=%s)", rec.Code, rec.Body.String())
	}
	if len(audit.entries) != 0 {
		t.Errorf("AO ditolak tetapi audit tertulis %d baris", len(audit.entries))
	}

	// Kasus pembanding: ANL diterima.
	if rec2 := patchJSON(t, h, "/api/pengajuan/7/skoring/override", body, "42", "ANL"); rec2.Code != http.StatusOK {
		t.Fatalf("ANL: status = %d, mau 200 (body=%s)", rec2.Code, rec2.Body.String())
	}
}

// TestHTTP_Override_AuditGagalMakaOverrideGagal menegakkan BR-10: tidak ada
// perubahan tanpa jejak. Kalau pencatatan audit gagal, override TIDAK boleh
// dilaporkan sukses.
func TestHTTP_Override_AuditGagalMakaOverrideGagal(t *testing.T) {
	audit := &fakeAuditRecorder{gagal: errors.New("audit trail tidak dapat ditulis")}
	h := routerOverride(newFakeParamRepoSkoring(), audit)

	rec := patchJSON(t, h, "/api/pengajuan/7/skoring/override", map[string]any{
		"gradeSemula": 2, "gradeBaru": 3, "alasan": "alasan sah",
	}, "42", "ANL")

	if rec.Code == http.StatusOK {
		t.Fatalf("status = 200 padahal audit gagal; override tanpa jejak melanggar BR-10")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, mau 500", rec.Code)
	}
}

// TestHTTP_Override_GradeTidakDikenalDitolak memastikan grade tujuan
// divalidasi terhadap tabel rentang_margin, bukan rentang di kode.
func TestHTTP_Override_GradeTidakDikenalDitolak(t *testing.T) {
	h := routerOverride(newFakeParamRepoSkoring(), &fakeAuditRecorder{})

	rec := patchJSON(t, h, "/api/pengajuan/7/skoring/override", map[string]any{
		"gradeSemula": 2, "gradeBaru": 9, "alasan": "alasan sah",
	}, "42", "ANL")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("grade 9: status = %d, mau 400 (body=%s)", rec.Code, rec.Body.String())
	}

	// Grade sama dengan semula bukan override.
	rec2 := patchJSON(t, h, "/api/pengajuan/7/skoring/override", map[string]any{
		"gradeSemula": 3, "gradeBaru": 3, "alasan": "alasan sah",
	}, "42", "ANL")
	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("grade sama: status = %d, mau 400", rec2.Code)
	}
}

// TestHTTP_Override_CatatanAuditTanpaDataPribadi menegakkan BR-11: NIK yang
// diselipkan ANL ke kolom alasan tidak boleh ikut ke catatan audit apa adanya.
func TestHTTP_Override_CatatanAuditTanpaDataPribadi(t *testing.T) {
	audit := &fakeAuditRecorder{}
	h := routerOverride(newFakeParamRepoSkoring(), audit)

	rec := patchJSON(t, h, "/api/pengajuan/7/skoring/override", map[string]any{
		"gradeSemula": 2, "gradeBaru": 3,
		"alasan": "hasil kunjungan ulang",
		"nik":    "3404123456789012",
	}, "42", "ANL")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200", rec.Code)
	}
	if len(audit.entries) != 1 {
		t.Fatalf("jumlah baris audit = %d, mau 1", len(audit.entries))
	}
	if bytes.Contains([]byte(audit.entries[0].Catatan), []byte("3404123456789012")) {
		t.Errorf("catatan audit memuat NIK — melanggar BR-11: %s", audit.entries[0].Catatan)
	}
}
