package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/irgiys/tim1gow/backend/internal/config"
	"github.com/irgiys/tim1gow/backend/internal/domain"
)

// sumberPalsu adalah SumberPengguna in-memory.
type sumberPalsu struct {
	pengguna map[string]*PenggunaLogin
	galat    error
}

func (s sumberPalsu) CariByEmailUntukLogin(_ context.Context, email string) (*PenggunaLogin, error) {
	if s.galat != nil {
		return nil, s.galat
	}
	return s.pengguna[email], nil
}

// cocokPalsu menggantikan bcrypt di test: hash dianggap cocok bila berbentuk
// "hash:" + password. bcrypt asli diuji lewat jalur seed dan integrasi.
func cocokPalsu(hash, password string) bool { return hash == "hash:"+password }

func handlerAuthUji(aktif bool) *AuthHandler {
	sumber := sumberPalsu{pengguna: map[string]*PenggunaLogin{
		"anl@imitra.test": {
			ID: 2, Nama: "Andi Analis Mikro", Email: "anl@imitra.test",
			PasswordHash: "hash:Demo1234!", Peran: domain.PeranANL, Aktif: aktif,
		},
	}}
	return NewAuthHandler(sumber, secretMw, time.Hour, cocokPalsu)
}

func postLogin(h http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// TestAC01_LoginBerhasilMengembalikanToken adalah langkah pertama AC-01.
func TestAC01_LoginBerhasilMengembalikanToken(t *testing.T) {
	h := handlerAuthUji(true)

	rec := postLogin(http.HandlerFunc(h.Login), `{"email":"anl@imitra.test","password":"Demo1234!"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200. body: %s", rec.Code, rec.Body.String())
	}

	var got jawabanLogin
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("membaca respons: %v", err)
	}
	if got.Token == "" {
		t.Fatal("token kosong")
	}
	if got.Peran != domain.PeranANL {
		t.Errorf("peran = %q, mau ANL", got.Peran)
	}

	// Token hasil login harus benar-benar diterima middleware.
	prot := MiddlewareAuth(secretMw, pemeriksaPalsu{aktif: true})(handlerEcho())
	if rec := panggil(prot, "Bearer "+got.Token); rec.Code != http.StatusOK {
		t.Errorf("token hasil login ditolak middleware: status = %d", rec.Code)
	}
}

// TestLogin_PasswordSalahDitolak401 dan email tak dikenal harus memberi pesan
// yang SAMA: membedakannya membocorkan email mana yang terdaftar.
func TestLogin_KredensialSalahDitolak401(t *testing.T) {
	h := handlerAuthUji(true)

	kasus := map[string]string{
		"password salah":        `{"email":"anl@imitra.test","password":"salah"}`,
		"email tidak terdaftar": `{"email":"hantu@imitra.test","password":"Demo1234!"}`,
	}

	var pesan []string
	for nama, body := range kasus {
		rec := postLogin(http.HandlerFunc(h.Login), body)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, mau 401", nama, rec.Code)
			continue
		}
		var e errorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
			t.Fatalf("%s: %v", nama, err)
		}
		pesan = append(pesan, e.Message)
	}
	if len(pesan) == 2 && pesan[0] != pesan[1] {
		t.Errorf("pesan berbeda antara password salah dan email tidak terdaftar (%q vs %q) — membocorkan email terdaftar", pesan[0], pesan[1])
	}
}

func TestLogin_AkunNonaktifDitolak(t *testing.T) {
	h := handlerAuthUji(false)

	rec := postLogin(http.HandlerFunc(h.Login), `{"email":"anl@imitra.test","password":"Demo1234!"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, mau 401", rec.Code)
	}
}

func TestLogin_InputTidakValidDitolak400(t *testing.T) {
	h := handlerAuthUji(true)

	for nama, body := range map[string]string{
		"bukan json":              `{`,
		"email kosong":            `{"email":"","password":"x"}`,
		"password kosong":         `{"email":"anl@imitra.test","password":""}`,
		"tanpa field sama sekali": `{}`,
	} {
		if rec := postLogin(http.HandlerFunc(h.Login), body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, mau 400", nama, rec.Code)
		}
	}
}

// TestLogin_RespronsTidakMemuatHash: password_hash tidak boleh pernah keluar
// dari server, walaupun bukan plaintext.
func TestLogin_ResponsTidakMemuatHash(t *testing.T) {
	h := handlerAuthUji(true)

	rec := postLogin(http.HandlerFunc(h.Login), `{"email":"anl@imitra.test","password":"Demo1234!"}`)
	body := strings.ToLower(rec.Body.String())
	for _, dilarang := range []string{"password", "hash"} {
		if strings.Contains(body, dilarang) {
			t.Errorf("respons login memuat %q: %s", dilarang, rec.Body.String())
		}
	}
}

// TestRouter_EndpointTerlindungiTanpaTokenDitolak401 memastikan wiring router
// produksi benar-benar memasang autentikasi pada seluruh route /api.
func TestRouter_EndpointTerlindungiTanpaTokenDitolak401(t *testing.T) {
	cfg := config.Config{AppEnv: "test", JWTSecret: string(secretMw), JWTExpiresIn: time.Hour}
	h := NewRouterLengkap(cfg, nil, nil, nil, nil, handlerAuthUji(true), nil, nil, pemeriksaPalsu{aktif: true})

	// Login tetap publik.
	if rec := postLogin(h, `{"email":"anl@imitra.test","password":"Demo1234!"}`); rec.Code != http.StatusOK {
		t.Fatalf("login seharusnya publik: status = %d", rec.Code)
	}

	// /auth/me tanpa token -> 401.
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /api/auth/me tanpa token: status = %d, mau 401", rec.Code)
	}
}
