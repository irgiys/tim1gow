package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/irgiys/tim1gow/backend/internal/config"
	"github.com/irgiys/tim1gow/backend/internal/domain"
	"github.com/irgiys/tim1gow/backend/internal/service"
)

// Test di berkas ini mengunci satu celah otorisasi yang sempat ada di jalur
// produksi: handler approval dan override skoring membaca identitas aktor dari
// header X-Actor-ID / X-Actor-Role, dengan fallback ke (id=1, ANL).
//
// Dua akibatnya, keduanya terbukti lewat panggilan API nyata sebelum diperbaiki:
//
//   - Identitas dapat DIPALSUKAN. KCP yang sah mengirim "X-Actor-Role: KOM"
//     membuat server menilai BR-02 memakai peran palsu, sehingga KCP justru
//     ditolak "level 2 tidak dapat memutuskan sebelum level 1".
//   - Tanpa header, SETIAP aktor dianggap id=1. Audit trail mencatat
//     AJUKAN_APPROVAL sebagai actor_id=1 padahal ANL adalah id=2 — BR-10
//     dilanggar karena jejaknya menunjuk orang yang salah.
//
// Bug ini tidak tertangkap test manapun karena test handler SENDIRI yang
// mengirim header itu. Test di bawah memakai router PRODUKSI (pemeriksa
// terpasang) supaya yang diuji adalah perilaku yang benar-benar dipakai.

// routerProduksiApproval membangun router dengan autentikasi aktif, sehingga
// identitas hanya boleh berasal dari token.
//
// appH dan skoH ikut dipasang supaya route-nya benar-benar terdaftar: tanpa
// itu semua permintaan dijawab 404 oleh NotFound router dan test ini akan
// hijau/merah untuk alasan yang salah.
func routerProduksiApproval(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Config{
		AppEnv:       "test",
		JWTSecret:    string(secretMw),
		JWTExpiresIn: time.Hour,
	}

	appRepo := newFakeApprovalRepoForHTTP()
	paramRepo := &fakeParameterRepoForHTTP{
		ambang: []domain.AmbangApproval{
			{PlafonMin: 5_000_000, PlafonMaks: 50_000_000, Level: []domain.Peran{domain.PeranKCP}},
		},
	}
	auditSvc := service.NewAuditService(&fakeAuditRepoForHTTP{})
	appH := NewApprovalHandler(service.NewApprovalService(appRepo, paramRepo, auditSvc))

	skoH := NewSkoringHandler(
		service.NewSkoringServiceWithAudit(newFakeParamRepoSkoring(), auditSvc),
		service.NewMarginService(newFakeParamRepoSkoring()),
	)

	return NewRouterLengkap(cfg, nil, appH, nil, skoH, handlerAuthUji(true), nil, nil,
		pemeriksaPalsu{aktif: true})
}

// TestProduksi_HeaderXActorTidakDapatMemalsukanIdentitas adalah regression test
// utama: header X-Actor-* dari klien WAJIB diabaikan di jalur produksi.
//
// Endpoint approval hanya untuk KCP/KC/KOM. Token AO yang menyuntik
// "X-Actor-Role: KOM" harus tetap 403 — kalau header dipercaya, ia akan lolos
// pemeriksaan peran atau justru mengubah penilaian BR-02/BR-09.
func TestProduksi_HeaderXActorTidakDapatMemalsukanIdentitas(t *testing.T) {
	h := routerProduksiApproval(t)

	req := httptest.NewRequest(http.MethodPost, "/api/pengajuan/7/approval",
		strings.NewReader(`{"keputusan":"APPROVE"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenUntuk(t, 11, domain.PeranAO))
	// Klien berusaha naik pangkat lewat header.
	req.Header.Set("X-Actor-ID", "99")
	req.Header.Set("X-Actor-Role", string(domain.PeranKOM))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, mau 403 — header X-Actor-* seharusnya diabaikan, "+
			"peran diambil dari token (body=%s)", rec.Code, rec.Body.String())
	}
}

// TestProduksi_TanpaTokenTidakDianggapAktorDefault memastikan tidak ada lagi
// fallback (id=1, ANL). Ketiadaan identitas dijawab 401, bukan ditebak.
//
// Ini yang paling berbahaya sebelumnya: perubahan keadaan berjalan dengan aktor
// yang ditebak, sehingga audit trail-nya menunjuk orang yang salah (BR-10).
func TestProduksi_TanpaTokenTidakDianggapAktorDefault(t *testing.T) {
	h := routerProduksiApproval(t)

	for _, jalur := range []string{
		"/api/pengajuan/7/ajukan-approval",
		"/api/pengajuan/7/approval",
	} {
		req := httptest.NewRequest(http.MethodPost, jalur,
			strings.NewReader(`{"keputusan":"APPROVE"}`))
		req.Header.Set("Content-Type", "application/json")
		// Hanya header aktor, TANPA token.
		req.Header.Set("X-Actor-ID", "1")
		req.Header.Set("X-Actor-Role", string(domain.PeranANL))

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, mau 401 — aktor tidak boleh ditebak dari header",
				jalur, rec.Code)
		}
	}
}

// TestProduksi_OverrideSkoringJugaMemakaiTokenBukanHeader menutup jalur kedua
// yang memakai getActor. AC-08 menuntut override tercatat dengan identitas ANL
// yang sebenarnya; kalau identitasnya dari header, catatan audit itu tidak dapat
// dipercaya.
func TestProduksi_OverrideSkoringJugaMemakaiTokenBukanHeader(t *testing.T) {
	h := routerProduksiApproval(t)

	req := httptest.NewRequest(http.MethodPatch, "/api/pengajuan/7/skoring/override",
		strings.NewReader(`{"gradeSemula":2,"gradeBaru":3,"alasan":"uji"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenUntuk(t, 11, domain.PeranAO))
	req.Header.Set("X-Actor-ID", "2")
	req.Header.Set("X-Actor-Role", string(domain.PeranANL))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	// Override hanya untuk ANL. Token AO tetap 403 walau header mengaku ANL.
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, mau 403 — token AO tidak boleh naik jadi ANL lewat header (body=%s)",
			rec.Code, rec.Body.String())
	}
}

// TestProduksi_HeaderTidakDapatMengubahAktorYangDinilaiBR09 adalah test yang
// benar-benar menyentuh getActor.
//
// Test lain di berkas ini dijaga lebih dulu oleh WajibPeran, jadi ia hijau
// bahkan ketika getActor salah — terbukti lewat mutation check. Test ini
// memakai token dengan peran yang SAH (KCP, jadi lolos WajibPeran) tetapi
// menyuntik X-Actor-ID milik maker pengajuan.
//
// Kalau getActor mempercayai header, aktor yang dinilai menjadi maker dan
// server menjawab 422 BR-09. Kalau ia memakai token, KCP bukan maker sehingga
// keputusannya diproses (200). Perbedaan inilah yang membuktikan sumber
// identitasnya.
func TestProduksi_HeaderTidakDapatMengubahAktorYangDinilaiBR09(t *testing.T) {
	cfg := config.Config{AppEnv: "test", JWTSecret: string(secretMw), JWTExpiresIn: time.Hour}

	const makerID = int64(10)
	const kcpID = int64(33)

	appRepo := newFakeApprovalRepoForHTTP()
	_ = appRepo.SimpanPengajuan(context.Background(), &domain.Pengajuan{
		ID:             1,
		NomorReferensi: "IMT-20260821-0001",
		PlafonDiajukan: 30_000_000,
		Grade:          1,
		Status:         domain.StatusWaitingApprovalL1,
		AOID:           makerID, // maker
		DibuatPada:     time.Now(),
	})
	paramRepo := &fakeParameterRepoForHTTP{
		ambang: []domain.AmbangApproval{
			{PlafonMin: 5_000_000, PlafonMaks: 50_000_000, Level: []domain.Peran{domain.PeranKCP}},
		},
	}
	appH := NewApprovalHandler(service.NewApprovalService(
		appRepo, paramRepo, service.NewAuditService(&fakeAuditRepoForHTTP{})))

	h := NewRouterLengkap(cfg, nil, appH, nil, nil, handlerAuthUji(true), nil, nil,
		pemeriksaPalsu{aktif: true})

	req := httptest.NewRequest(http.MethodPost, "/api/pengajuan/1/approval",
		strings.NewReader(`{"keputusan":"APPROVE"}`))
	req.Header.Set("Content-Type", "application/json")
	// Token: KCP yang sah dan BUKAN maker.
	req.Header.Set("Authorization", "Bearer "+tokenUntuk(t, kcpID, domain.PeranKCP))
	// Header berusaha mengganti aktor menjadi maker.
	req.Header.Set("X-Actor-ID", "10")
	req.Header.Set("X-Actor-Role", string(domain.PeranKCP))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code == http.StatusUnprocessableEntity {
		t.Fatalf("status = 422 (%s); aktor yang dinilai berasal dari header, "+
			"bukan dari token — getActor tidak boleh mempercayai X-Actor-*",
			strings.TrimSpace(rec.Body.String()))
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}
}

// CATATAN HASIL MUTATION CHECK — dibaca sebelum menambah test di sini.
//
// Mutasi "pasang injeksiIdentitasUji juga di jalur produksi" TIDAK menjatuhkan
// test manapun, dan setelah diperiksa itu memang benar: MiddlewareAuth berjalan
// setelah middleware global dan menimpa identitas apa pun yang disuntikkan
// lebih awal, sedangkan route yang memakai getActor semuanya berada di dalam
// grup terautentikasi. Jadi pemisahan itu bersifat pertahanan berlapis, bukan
// satu-satunya penghalang — penghalang sebenarnya adalah MiddlewareAuth.
//
// Yang benar-benar diperbaiki commit ini, dan yang dijaga test di atas, adalah
// getActor: dulu ia membaca header dari klien dengan fallback (id=1, ANL),
// sehingga identitas dapat dipalsukan dan audit trail mencatat orang yang
// salah. Itu bug nyata yang terbukti lewat panggilan API, bukan hipotesis.
//
// Jangan menambah test yang "membuktikan" injeksi berbahaya di route publik:
// tidak ada route publik yang memakai getActor, jadi test seperti itu akan
// hijau untuk alasan yang salah.

// TestProduksi_PeranSahDariTokenTetapDilayani adalah kasus pembanding wajib
// (Larangan 18). Tanpa ini, middleware yang menolak SEMUA request akan
// meloloskan ketiga test di atas.
//
// KCP yang sah harus melewati lapisan peran dan sampai ke service — jawabannya
// boleh 404/422 (data uji tidak ada di router tanpa DB), yang penting BUKAN
// 401/403.
func TestProduksi_PeranSahDariTokenTetapDilayani(t *testing.T) {
	h := routerProduksiApproval(t)

	req := httptest.NewRequest(http.MethodPost, "/api/pengajuan/7/approval",
		strings.NewReader(`{"keputusan":"APPROVE"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenUntuk(t, 33, domain.PeranKCP))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden {
		t.Fatalf("status = %d; KCP yang sah seharusnya lolos lapisan peran (body=%s)",
			rec.Code, rec.Body.String())
	}
}
