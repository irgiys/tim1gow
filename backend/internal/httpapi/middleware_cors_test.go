package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/irgiys/tim1gow/backend/internal/config"
)

// Test di berkas ini menjaga satu hal yang tidak pernah tertangkap go test
// sebelumnya: config.CorsAllowedOrigins dibaca dari environment lalu DIBUANG,
// karena tidak ada middleware yang memakainya. Preflight dijawab 405 oleh
// MethodNotAllowed router, sehingga halaman login (Client Component yang fetch
// dari BROWSER, lintas origin :3000 -> :8080) tidak pernah dapat memanggil API.
//
// Bug ini hanya muncul saat aplikasi benar-benar dijalankan, bukan dari test
// yang memanggil handler langsung — curl dan httptest tidak menegakkan CORS.
// Karena itu yang diuji di sini adalah HEADER RESPONS, bukan kode status saja.

const originDemo = "http://localhost:3000"

// routerCORS membangun router dengan autentikasi aktif, supaya test membuktikan
// preflight tidak tersangkut MiddlewareAuth.
func routerCORS(t *testing.T, origins ...string) http.Handler {
	t.Helper()
	if len(origins) == 0 {
		origins = []string{originDemo}
	}
	cfg := config.Config{
		AppEnv:             "test",
		JWTSecret:          string(secretMw),
		CorsAllowedOrigins: origins,
	}
	return NewRouterLengkap(cfg, nil, nil, nil, nil, handlerAuthUji(true), nil, nil,
		pemeriksaPalsu{aktif: true})
}

func preflight(h http.Handler, path, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodOptions, path, nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "content-type")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// TestCORS_PreflightLoginDiizinkan adalah regression test utama: inilah request
// yang dikirim browser sebelum tombol login bekerja. Sebelum middleware ini
// dipasang, jawabannya 405 tanpa satu pun header Access-Control-*.
func TestCORS_PreflightLoginDiizinkan(t *testing.T) {
	rec := preflight(routerCORS(t), "/api/auth/login", originDemo)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, mau 204 — preflight seharusnya dijawab middleware CORS, "+
			"bukan MethodNotAllowed router", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != originDemo {
		t.Errorf("Allow-Origin = %q, mau %q", got, originDemo)
	}
	// Tanpa content-type di Allow-Headers, browser memblokir request JSON
	// walau preflight-nya sendiri 204.
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Error("Allow-Headers kosong; fetch dengan Content-Type: application/json akan diblokir")
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("Allow-Methods kosong")
	}
}

// TestCORS_PreflightTidakTersangkutAuth menjaga urutan middleware. Browser
// tidak mengirim Authorization pada preflight; kalau CORS dipasang SETELAH
// MiddlewareAuth, preflight dijawab 401 dan request aslinya tidak pernah
// dikirim. Endpoint di bawah sengaja yang terlindungi, bukan /auth/login.
func TestCORS_PreflightTidakTersangkutAuth(t *testing.T) {
	rec := preflight(routerCORS(t), "/api/pengajuan", originDemo)

	if rec.Code == http.StatusUnauthorized {
		t.Fatal("preflight dijawab 401; CORS harus dipasang SEBELUM MiddlewareAuth")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, mau 204", rec.Code)
	}
}

// TestCORS_OriginAsingTidakDiberiHeader memastikan daftar izin benar-benar
// menyaring. Ini kasus pembanding wajib untuk test di atas (Larangan 18):
// middleware yang mengizinkan SEMUA origin akan meloloskan test 204 tadi.
func TestCORS_OriginAsingTidakDiberiHeader(t *testing.T) {
	rec := preflight(routerCORS(t), "/api/auth/login", "http://penyerang.test")

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin = %q untuk origin asing; harus kosong", got)
	}
	if rec.Code == http.StatusNoContent {
		t.Error("preflight origin asing dijawab 204; seharusnya ditolak")
	}
}

// TestCORS_PrefixMiripTidakLolos menutup celah pencocokan prefix/suffix.
// "http://localhost:3000.penyerang.com" berawalan sama dengan origin yang
// diizinkan; pencocokan harus PERSIS.
func TestCORS_PrefixMiripTidakLolos(t *testing.T) {
	for _, jahat := range []string{
		"http://localhost:3000.penyerang.com",
		"http://localhost:30001",
		"https://localhost:3000", // skema berbeda
	} {
		rec := preflight(routerCORS(t), "/api/auth/login", jahat)
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("origin %q diberi Allow-Origin %q; pencocokan harus persis", jahat, got)
		}
	}
}

// TestCORS_ResponsAsliMembawaHeader memastikan bukan hanya preflight yang
// diurus. Tanpa header di respons POST yang sebenarnya, browser tetap
// memblokir pembacaan hasilnya walau preflight sudah lolos.
func TestCORS_ResponsAsliMembawaHeader(t *testing.T) {
	h := routerCORS(t)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login",
		strings.NewReader(`{"email":"anl@imitra.test","password":"Demo1234!"}`))
	req.Header.Set("Origin", originDemo)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != originDemo {
		t.Errorf("Allow-Origin pada respons asli = %q, mau %q", got, originDemo)
	}
	// Vary: Origin mencegah cache menyajikan respons origin A ke origin B.
	if got := rec.Header().Get("Vary"); got == "" {
		t.Error("Vary kosong; cache dapat membocorkan respons antar origin")
	}
}

// TestCORS_TanpaOriginTidakBerubah memastikan panggilan non-browser (curl,
// server-to-server, healthcheck container) tidak terpengaruh sama sekali.
func TestCORS_TanpaOriginTidakBerubah(t *testing.T) {
	h := routerCORS(t)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin = %q padahal request tidak punya Origin", got)
	}
}
