package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/irgiys/tim1gow/backend/internal/config"
	"github.com/irgiys/tim1gow/backend/internal/domain"
	"github.com/irgiys/tim1gow/backend/internal/service"
)

// Test di berkas ini menjaga satu hal yang sebelumnya hilang: handler FR-02,
// FR-03, dan FR-04 sudah ada, tetapi TIDAK SATU PUN terdaftar sebagai route.
// Dari luar — lewat API, yang justru dipakai penilai — ketiga fitur itu tidak
// ada. Test service yang hijau tidak menangkap keadaan itu, karena test service
// memanggil service secara langsung tanpa melewati router.
//
// Karena itu test di sini menembak lewat HTTP dengan token asli, bukan lewat
// router test tanpa autentikasi: yang diuji adalah route + penegakan peran
// di server (AGENTS.md Larangan 6), bukan lagi perhitungannya.

// routerPengajuan membangun router produksi lengkap dengan autentikasi aktif.
// Batas plafon dan daftar dokumen wajib dikembalikan sebagai nilai yang bisa
// diubah test, bukan konstanta di kode (Larangan 3).
func routerPengajuan(t *testing.T) (http.Handler, *fakePengajuanRepoHTTP, *fakeDokumenRepoHTTP, *fakeBatasPlafonHTTP) {
	t.Helper()

	pjnRepo := newFakePengajuanRepoHTTP()
	dokRepo := newFakeDokumenRepoHTTP()
	svRepo := newFakeSurveiRepoHTTP()
	batas := &fakeBatasPlafonHTTP{minimum: 5_000_000, maksimum: 500_000_000, ditemukan: true}
	wajib := &fakeDokumenWajibHTTP{jenis: []string{service.JenisDokumenKTP}}

	h := NewPengajuanHandler(
		service.NewPengajuanService(pjnRepo, batas),
		service.NewDokumenService(dokRepo, wajib),
		service.NewSurveiService(svRepo),
	)

	r := NewRouterLengkap(
		config.Config{AppEnv: "test", JWTSecret: string(secretMw)},
		nil, nil, nil, nil, nil, h, pemeriksaPalsu{aktif: true},
	)
	return r, pjnRepo, dokRepo, batas
}

// kirim menembak endpoint dengan token peran tertentu.
func kirim(t *testing.T, h http.Handler, metode, path string, peran domain.Peran, id int64, body any) *httptest.ResponseRecorder {
	t.Helper()

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(metode, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenUntuk(t, id, peran))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func dataPengajuan() map[string]any {
	return map[string]any{
		"tipe":           "INDIVIDU",
		"namaNasabah":    "Nasabah Uji",
		"nik":            "3404000000000001",
		"alamatUsaha":    "Jalan Uji 1",
		"jenisUsaha":     "Warung",
		"jenisAkad":      string(domain.AkadMurabahah),
		"plafonDiajukan": 20_000_000,
		"tenorBulan":     12,
	}
}

func dataSurvei() map[string]any {
	return map[string]any{
		"latitude":       -7.05,
		"longitude":      110.44,
		"fotoUrl":        "s3://bukti/1.jpg",
		"omzetHarian":    500_000,
		"lamaUsahaBulan": 24,
		"catatanKondisi": "usaha aktif, stok terisi",
	}
}

// TestRoute_FR02_PengajuanTerdaftarDanNomorReferensiDariServer membuktikan route
// POST /api/pengajuan benar-benar terpasang, bukan hanya handler yang menganggur.
// Sebelum wiring ini, permintaan yang sama dijawab 404 oleh NotFound router.
func TestRoute_FR02_PengajuanTerdaftarDanNomorReferensiDariServer(t *testing.T) {
	h, _, _, _ := routerPengajuan(t)

	rec := kirim(t, h, http.MethodPost, "/api/pengajuan", domain.PeranAO, 11, dataPengajuan())
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, mau 201 — route POST /api/pengajuan belum terpasang? (body=%s)",
			rec.Code, rec.Body.String())
	}

	var resp pengajuanResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode respons: %v", err)
	}
	// Nomor referensi dibangkitkan server, bukan dikirim klien (Larangan 4).
	if len(resp.NomorReferensi) != len("IMT-20260821-0001") || resp.NomorReferensi[:4] != "IMT-" {
		t.Errorf("nomorReferensi = %q; mau format IMT-YYYYMMDD-NNNN", resp.NomorReferensi)
	}
	// NIK adalah data pribadi dan tidak boleh ikut ke respons (BR-11).
	if bytes.Contains(rec.Body.Bytes(), []byte("3404000000000001")) {
		t.Error("NIK muncul di respons; melanggar BR-11")
	}
}

// TestRoute_FR02_PlafonDiLuarBatasDitolak422 memastikan BR-01 benar-benar sampai
// ke klien lewat route baru, dengan kode BR di field `rule`.
//
// Kasus pembanding ada di test sebelumnya (plafon 20 juta -> 201), sesuai
// Larangan 18: test penolakan tidak berdiri sendiri.
func TestRoute_FR02_PlafonDiLuarBatasDitolak422(t *testing.T) {
	h, _, _, batas := routerPengajuan(t)

	body := dataPengajuan()
	body["plafonDiajukan"] = 1_000_000 // di bawah batas minimum tersimpan

	rec := kirim(t, h, http.MethodPost, "/api/pengajuan", domain.PeranAO, 11, body)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, mau 422 (body=%s)", rec.Code, rec.Body.String())
	}
	if r := decodeError(t, rec).Rule; r != "BR-01" {
		t.Errorf("rule = %q, mau BR-01", r)
	}

	// Batasnya data, bukan konstanta: turunkan minimum di "tabel parameter"
	// dan plafon yang sama harus jadi diterima (AC-15, Larangan 3).
	batas.minimum = 1_000_000
	if rec2 := kirim(t, h, http.MethodPost, "/api/pengajuan", domain.PeranAO, 11, body); rec2.Code != http.StatusCreated {
		t.Fatalf("setelah batas diturunkan: status = %d, mau 201 — batas plafon tampaknya "+
			"tidak dibaca dari data (body=%s)", rec2.Code, rec2.Body.String())
	}
}

// TestRoute_FR02_AOTidakDapatMembukaPengajuanAOLain menjaga lingkup akses tetap
// ditegakkan di server pada route yang baru dipasang.
func TestRoute_FR02_AOTidakDapatMembukaPengajuanAOLain(t *testing.T) {
	h, repo, _, _ := routerPengajuan(t)
	repo.baris[9] = service.Pengajuan{ID: 9, AOID: 11, NomorReferensi: "IMT-20260821-0009"}

	if rec := kirim(t, h, http.MethodGet, "/api/pengajuan/9", domain.PeranAO, 11, nil); rec.Code != http.StatusOK {
		t.Fatalf("pemilik: status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}
	// AO lain tidak boleh tahu id itu ada.
	if rec := kirim(t, h, http.MethodGet, "/api/pengajuan/9", domain.PeranAO, 22, nil); rec.Code != http.StatusNotFound {
		t.Errorf("AO lain: status = %d, mau 404", rec.Code)
	}
	// ANL memang bertugas memeriksa pengajuan orang lain.
	if rec := kirim(t, h, http.MethodGet, "/api/pengajuan/9", domain.PeranANL, 33, nil); rec.Code != http.StatusOK {
		t.Errorf("ANL: status = %d, mau 200", rec.Code)
	}
}

// TestRoute_FR03_UploadDanVerifikasiTerpisahPeran menjaga pemisahan maker dan
// checker pada tahap dokumen: AO mengunggah, ANL memverifikasi, dan keduanya
// tidak boleh bertukar peran (AC-02, Larangan 6).
func TestRoute_FR03_UploadDanVerifikasiTerpisahPeran(t *testing.T) {
	h, _, _, _ := routerPengajuan(t)
	upload := map[string]any{"jenisDokumen": service.JenisDokumenKTP, "urlBerkas": "s3://dok/ktp.jpg"}

	rec := kirim(t, h, http.MethodPost, "/api/pengajuan/9/dokumen", domain.PeranAO, 11, upload)
	if rec.Code != http.StatusCreated {
		t.Fatalf("AO upload: status = %d, mau 201 (body=%s)", rec.Code, rec.Body.String())
	}
	// ANL tidak mengunggah berkas nasabah.
	if rec := kirim(t, h, http.MethodPost, "/api/pengajuan/9/dokumen", domain.PeranANL, 33, upload); rec.Code != http.StatusForbidden {
		t.Errorf("ANL upload: status = %d, mau 403", rec.Code)
	}
	// AO tidak memverifikasi berkas yang ia unggah sendiri.
	setujui := map[string]any{"setujui": true}
	if rec := kirim(t, h, http.MethodPatch, "/api/pengajuan/9/dokumen/1/verifikasi", domain.PeranAO, 11, setujui); rec.Code != http.StatusForbidden {
		t.Errorf("AO verifikasi: status = %d, mau 403", rec.Code)
	}
	if rec := kirim(t, h, http.MethodPatch, "/api/pengajuan/9/dokumen/1/verifikasi", domain.PeranANL, 33, setujui); rec.Code != http.StatusOK {
		t.Errorf("ANL verifikasi: status = %d, mau 200", rec.Code)
	}
}

// TestRoute_FR03_PenolakanTanpaKodeAlasanDitolak400 memastikan aturan AC-03
// terlihat lewat API, dan bukan jatuh ke 500 karena pemetaan error yang kurang.
func TestRoute_FR03_PenolakanTanpaKodeAlasanDitolak400(t *testing.T) {
	h, _, dok, _ := routerPengajuan(t)
	dok.baris[1] = service.Dokumen{ID: 1, PengajuanID: 9, JenisDokumen: service.JenisDokumenKTP}

	tolak := map[string]any{"setujui": false}
	rec := kirim(t, h, http.MethodPatch, "/api/pengajuan/9/dokumen/1/verifikasi", domain.PeranANL, 33, tolak)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, mau 400 (body=%s)", rec.Code, rec.Body.String())
	}

	// Kasus pembanding (Larangan 18): dengan kode alasan, penolakan diterima.
	tolak["kodeAlasan"] = "DOK_TIDAK_JELAS"
	if rec2 := kirim(t, h, http.MethodPatch, "/api/pengajuan/9/dokumen/1/verifikasi", domain.PeranANL, 33, tolak); rec2.Code != http.StatusOK {
		t.Fatalf("dengan kode alasan: status = %d, mau 200 (body=%s)", rec2.Code, rec2.Body.String())
	}
}

// TestRoute_FR04_SurveiHanyaAODanWajibLengkap menjaga route survei terpasang,
// terbatas untuk AO, dan survei tidak lengkap tidak tersimpan.
func TestRoute_FR04_SurveiHanyaAODanWajibLengkap(t *testing.T) {
	h, _, _, _ := routerPengajuan(t)

	rec := kirim(t, h, http.MethodPost, "/api/pengajuan/9/survei", domain.PeranAO, 11, dataSurvei())
	if rec.Code != http.StatusCreated {
		t.Fatalf("AO: status = %d, mau 201 — route survei belum terpasang? (body=%s)",
			rec.Code, rec.Body.String())
	}
	if rec := kirim(t, h, http.MethodPost, "/api/pengajuan/9/survei", domain.PeranANL, 33, dataSurvei()); rec.Code != http.StatusForbidden {
		t.Errorf("ANL: status = %d, mau 403", rec.Code)
	}

	kurang := dataSurvei()
	delete(kurang, "fotoUrl")
	if rec := kirim(t, h, http.MethodPost, "/api/pengajuan/9/survei", domain.PeranAO, 11, kurang); rec.Code != http.StatusBadRequest {
		t.Errorf("tanpa foto: status = %d, mau 400", rec.Code)
	}
}

// TestRoute_FR02_TanpaTokenDitolak401 memastikan route baru berada DI DALAM
// grup terautentikasi. Route yang lupa dimasukkan ke grup itu akan menjawab
// 201 di sini, bukan 401.
func TestRoute_FR02_TanpaTokenDitolak401(t *testing.T) {
	h, _, _, _ := routerPengajuan(t)

	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(dataPengajuan())
	req := httptest.NewRequest(http.MethodPost, "/api/pengajuan", &buf)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401 (body=%s)", rec.Code, rec.Body.String())
	}
}
