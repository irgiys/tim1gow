package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/irgiys/tim1gow/backend/internal/auth"
	"github.com/irgiys/tim1gow/backend/internal/domain"
)

var secretMw = []byte("secret-test-middleware-bukan-nilai-produksi")

// pemeriksaPalsu mengendalikan status aktif pengguna untuk test pencabutan.
type pemeriksaPalsu struct {
	aktif bool
	galat error
}

func (p pemeriksaPalsu) PenggunaAktif(context.Context, int64) (bool, error) {
	return p.aktif, p.galat
}

// handlerEcho membalas 200 dan menuliskan identitas, supaya test dapat
// memastikan identitas benar-benar sampai ke handler.
func handlerEcho() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ident, ok := IdentitasDari(r.Context())
		if !ok {
			http.Error(w, "identitas tidak ada", http.StatusInternalServerError)
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"pengguna_id": ident.PenggunaID,
			"peran":       string(ident.Peran),
		})
	})
}

func tokenUntuk(t *testing.T, id int64, peran domain.Peran) string {
	t.Helper()
	tok, err := auth.Terbitkan(secretMw, id, string(peran), time.Hour, time.Now())
	if err != nil {
		t.Fatalf("menerbitkan token: %v", err)
	}
	return tok
}

func panggil(h http.Handler, header string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/uji", nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// TestAuth_TokenValidDiterima adalah kasus pembanding untuk semua test
// penolakan (Larangan 18): middleware yang menolak SEMUA request akan
// meloloskan setiap test 401/403 tanpa test ini.
func TestAuth_TokenValidDiterima(t *testing.T) {
	h := MiddlewareAuth(secretMw, pemeriksaPalsu{aktif: true})(handlerEcho())

	rec := panggil(h, "Bearer "+tokenUntuk(t, 42, domain.PeranANL))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200. body: %s", rec.Code, rec.Body.String())
	}

	var got struct {
		PenggunaID int64  `json:"pengguna_id"`
		Peran      string `json:"peran"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("membaca respons: %v", err)
	}
	if got.PenggunaID != 42 || got.Peran != "ANL" {
		t.Errorf("identitas = %d/%s, mau 42/ANL", got.PenggunaID, got.Peran)
	}
}

func TestAuth_TanpaTokenDitolak401(t *testing.T) {
	h := MiddlewareAuth(secretMw, pemeriksaPalsu{aktif: true})(handlerEcho())

	for nama, header := range map[string]string{
		"tanpa header":    "",
		"tanpa skema":     tokenUntuk(t, 1, domain.PeranAO),
		"skema salah":     "Basic abcdef",
		"bearer kosong":   "Bearer ",
		"token sembarang": "Bearer bukan-token",
	} {
		rec := panggil(h, header)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, mau 401", nama, rec.Code)
		}
	}
}

// TestAuth_SkemaBearerCaseInsensitive: RFC 7235 menyebut skema auth tidak
// case-sensitive, dan sebagian klien mengirim "bearer".
func TestAuth_SkemaBearerCaseInsensitive(t *testing.T) {
	h := MiddlewareAuth(secretMw, pemeriksaPalsu{aktif: true})(handlerEcho())

	for _, skema := range []string{"Bearer ", "bearer ", "BEARER "} {
		if rec := panggil(h, skema+tokenUntuk(t, 1, domain.PeranAO)); rec.Code != http.StatusOK {
			t.Errorf("skema %q: status = %d, mau 200", skema, rec.Code)
		}
	}
}

// TestAuth_PenggunaNonaktifDitolak menjaga mekanisme pencabutan SDD BAB 7:
// token yang masih berlaku harus ditolak begitu akunnya dinonaktifkan.
func TestAuth_PenggunaNonaktifDitolak(t *testing.T) {
	token := tokenUntuk(t, 5, domain.PeranAO)

	aktif := MiddlewareAuth(secretMw, pemeriksaPalsu{aktif: true})(handlerEcho())
	if rec := panggil(aktif, "Bearer "+token); rec.Code != http.StatusOK {
		t.Fatalf("pengguna aktif: status = %d, mau 200", rec.Code)
	}

	nonaktif := MiddlewareAuth(secretMw, pemeriksaPalsu{aktif: false})(handlerEcho())
	if rec := panggil(nonaktif, "Bearer "+token); rec.Code != http.StatusUnauthorized {
		t.Errorf("pengguna nonaktif: status = %d, mau 401 — token lama masih diterima", rec.Code)
	}
}

func TestAuth_TokenKedaluwarsaDitolak(t *testing.T) {
	// Diterbitkan di masa lalu dengan masa berlaku singkat.
	token, err := auth.Terbitkan(secretMw, 1, "AO", time.Minute, time.Now().Add(-2*time.Hour))
	if err != nil {
		t.Fatalf("Terbitkan: %v", err)
	}

	h := MiddlewareAuth(secretMw, pemeriksaPalsu{aktif: true})(handlerEcho())
	if rec := panggil(h, "Bearer "+token); rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, mau 401", rec.Code)
	}
}

// TestAC02_PeranLainDitolak403 adalah AC-02 secara langsung: panggilan API
// lintas peran mengembalikan 403, bukan 200 dan bukan 404.
func TestAC02_PeranLainDitolak403(t *testing.T) {
	// Endpoint verifikasi dokumen hanya untuk ANL (SDD BAB 5).
	h := MiddlewareAuth(secretMw, pemeriksaPalsu{aktif: true})(
		WajibPeran(domain.PeranANL)(handlerEcho()),
	)

	// AO memanggil endpoint milik ANL -> 403.
	rec := panggil(h, "Bearer "+tokenUntuk(t, 1, domain.PeranAO))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("AO ke endpoint ANL: status = %d, mau 403. body: %s", rec.Code, rec.Body.String())
	}

	var body errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("membaca respons: %v", err)
	}
	if body.Error != "FORBIDDEN" {
		t.Errorf("error = %q, mau FORBIDDEN", body.Error)
	}

	// Pembanding wajib (Larangan 18): ANL sendiri tetap boleh masuk, sehingga
	// test membuktikan penolakannya berdasar peran, bukan route yang rusak.
	if rec := panggil(h, "Bearer "+tokenUntuk(t, 2, domain.PeranANL)); rec.Code != http.StatusOK {
		t.Errorf("ANL ke endpoint ANL: status = %d, mau 200", rec.Code)
	}
}

func TestWajibPeran_BeberapaPeranDiizinkan(t *testing.T) {
	// Route approval terbuka untuk tiga level approver.
	h := MiddlewareAuth(secretMw, pemeriksaPalsu{aktif: true})(
		WajibPeran(domain.PeranKCP, domain.PeranKC, domain.PeranKOM)(handlerEcho()),
	)

	for _, p := range []domain.Peran{domain.PeranKCP, domain.PeranKC, domain.PeranKOM} {
		if rec := panggil(h, "Bearer "+tokenUntuk(t, 1, p)); rec.Code != http.StatusOK {
			t.Errorf("peran %s: status = %d, mau 200", p, rec.Code)
		}
	}
	for _, p := range []domain.Peran{domain.PeranAO, domain.PeranANL, domain.PeranADM} {
		if rec := panggil(h, "Bearer "+tokenUntuk(t, 1, p)); rec.Code != http.StatusForbidden {
			t.Errorf("peran %s: status = %d, mau 403", p, rec.Code)
		}
	}
}

// TestWajibPeran_TanpaMiddlewareAuthTidakLolos: route yang salah pasang tidak
// boleh terbuka. Ini menjaga kesalahan wiring menjadi 401, bukan 200.
func TestWajibPeran_TanpaMiddlewareAuthTidakLolos(t *testing.T) {
	h := WajibPeran(domain.PeranADM)(handlerEcho())

	if rec := panggil(h, "Bearer "+tokenUntuk(t, 1, domain.PeranADM)); rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, mau 401", rec.Code)
	}
}
