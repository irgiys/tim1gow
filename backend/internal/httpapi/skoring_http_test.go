package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/irgiys/tim1gow/backend/internal/config"
	"github.com/irgiys/tim1gow/backend/internal/domain"
	"github.com/irgiys/tim1gow/backend/internal/service"
)

// Test di berkas ini diturunkan dari AC di docs/SRS-iMitra.md, BUKAN dari kode
// handler yang sedang diuji:
//
//	AC-07  Skoring menampilkan rincian keempat komponen beserta bobot dan skornya
//	AC-09  Margin 10,0 % untuk grade 1 (di bawah batas 11,0 %) diblokir sistem
//	AC-06  Kolektibilitas 2 dapat lanjut, tetapi grade tidak pernah lebih baik dari 3
//	AC-04  Skoring tanpa survei valid ditolak dengan pesan yang menyebut BR-03
//	AC-15  Mengubah baris tabel parameter langsung berlaku tanpa restart
//
// Nilai bobot dan rentang di fake repo di bawah SENGAJA bukan angka brief:
// kalau handler diam-diam memakai konstanta di kode alih-alih membaca
// parameter, test AC-15 akan gagal — itu justru yang ingin ditangkap.

// fakeParamRepoSkoring adalah tabel parameter yang bisa diubah di tengah test,
// meniru ADM mengubah baris lewat FR-13.
type fakeParamRepoSkoring struct {
	komponen []domain.ParameterKomponenSkor
	slik     map[int]float64
	umum     map[string]float64
	rentang  map[int]domain.RentangMargin
}

func newFakeParamRepoSkoring() *fakeParamRepoSkoring {
	return &fakeParamRepoSkoring{
		komponen: []domain.ParameterKomponenSkor{
			{Kode: domain.KomponenKapasitasBayar, Nama: "Kapasitas bayar", Bobot: 35, Batas1: 0.30, Batas2: 0.60, Aktif: true},
			{Kode: domain.KomponenRiwayatSlik, Nama: "Riwayat SLIK", Bobot: 25, Aktif: true},
			{Kode: domain.KomponenLamaUsaha, Nama: "Lama usaha", Bobot: 20, Batas1: 36, Batas2: 6, Aktif: true},
			{Kode: domain.KomponenSurveiLapangan, Nama: "Hasil survei lapangan", Bobot: 20, Aktif: true},
		},
		slik: map[int]float64{1: 100, 2: 40},
		umum: map[string]float64{
			service.KunciHariKerjaPerBulan: 25,
			service.KunciMarginUsaha:       0.30,
		},
		rentang: map[int]domain.RentangMargin{
			1: {Grade: 1, SkorMin: 85, SkorMaks: 100, MarginMin: 11.0, MarginMaks: 13.0, NisbahMin: 20, NisbahMaks: 25, DapatDibiayai: true},
			2: {Grade: 2, SkorMin: 70, SkorMaks: 84, MarginMin: 13.0, MarginMaks: 15.5, NisbahMin: 25, NisbahMaks: 30, DapatDibiayai: true},
			3: {Grade: 3, SkorMin: 55, SkorMaks: 69, MarginMin: 15.5, MarginMaks: 18.0, NisbahMin: 30, NisbahMaks: 35, DapatDibiayai: true},
			4: {Grade: 4, SkorMin: 40, SkorMaks: 54, MarginMin: 18.0, MarginMaks: 21.0, NisbahMin: 35, NisbahMaks: 40, DapatDibiayai: true},
			5: {Grade: 5, SkorMin: 0, SkorMaks: 39, DapatDibiayai: false},
		},
	}
}

func (f *fakeParamRepoSkoring) KomponenSkor() ([]domain.ParameterKomponenSkor, error) {
	out := make([]domain.ParameterKomponenSkor, len(f.komponen))
	copy(out, f.komponen)
	return out, nil
}

func (f *fakeParamRepoSkoring) SkorRiwayatSlik(kol int) (float64, bool, error) {
	v, ok := f.slik[kol]
	return v, ok, nil
}

func (f *fakeParamRepoSkoring) Umum(kunci string) (float64, bool, error) {
	v, ok := f.umum[kunci]
	return v, ok, nil
}

func (f *fakeParamRepoSkoring) RentangMarginPerGrade() ([]domain.RentangMargin, error) {
	out := make([]domain.RentangMargin, 0, len(f.rentang))
	for g := 1; g <= 5; g++ {
		if r, ok := f.rentang[g]; ok {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeParamRepoSkoring) RentangMargin(grade int) (domain.RentangMargin, bool, error) {
	r, ok := f.rentang[grade]
	return r, ok, nil
}

func (f *fakeParamRepoSkoring) AmbangApproval(int64) (domain.AmbangApproval, bool, error) {
	return domain.AmbangApproval{}, false, nil
}

func (f *fakeParamRepoSkoring) SemuaAmbangApproval() ([]domain.AmbangApproval, error) {
	return nil, nil
}

// fakePrasyaratRepo meniru keadaan nyata pengajuan di database (dokumen,
// survei, hasil SLIK). Test mengubah keadaan DI SINI, bukan lewat badan
// request — persis seperti produksi, di mana klien tidak punya suara soal
// terpenuhi atau tidaknya BR-03.
type fakePrasyaratRepo struct {
	keadaan service.KeadaanPrasyarat
	err     error
}

func newFakePrasyaratRepo() *fakePrasyaratRepo {
	return &fakePrasyaratRepo{keadaan: service.KeadaanPrasyarat{
		SemuaDokumenVerified: true,
		AdaSurveiValid:       true,
		SlikSudahDijalankan:  true,
		Kolektibilitas:       1,
	}}
}

func (f *fakePrasyaratRepo) KeadaanPrasyaratSkoring(_ context.Context, _ int64) (service.KeadaanPrasyarat, error) {
	if f.err != nil {
		return service.KeadaanPrasyarat{}, f.err
	}
	return f.keadaan, nil
}

// routerSkoring membangun router dengan HANYA handler skoring/margin terpasang,
// dengan seluruh prasyarat BR-03 terpenuhi.
func routerSkoring(param service.ParameterRepository) http.Handler {
	h, _ := routerSkoringDenganPrasyarat(param)
	return h
}

// routerSkoringDenganPrasyarat mengembalikan router beserta fake keadaannya,
// supaya test dapat mengubah keadaan pengajuan di tengah jalan.
func routerSkoringDenganPrasyarat(param service.ParameterRepository) (http.Handler, *fakePrasyaratRepo) {
	pra := newFakePrasyaratRepo()
	h := NewSkoringHandler(
		service.NewSkoringService(param).DenganPrasyarat(pra),
		service.NewMarginService(param),
	)
	return NewRouterWithAllHandlers(config.Config{AppEnv: "test"}, nil, nil, nil, h), pra
}

func postJSON(t *testing.T, h http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// decodeError membaca bentuk error API yang seragam: {error, message, rule}.
func decodeError(t *testing.T, rec *httptest.ResponseRecorder) errorResponse {
	t.Helper()
	var out errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("respons error bukan JSON yang dikenal: %v (body=%s)", err, rec.Body.String())
	}
	return out
}

// dataSkoring adalah badan request skoring. Prasyarat BR-03 dan
// kolektibilitas TIDAK ada di sini: keduanya dibaca dari keadaan pengajuan.
func dataSkoring() map[string]any {
	return map[string]any{
		"angsuranBulanan": 1000000.0,
		"omzetHarian":     500000.0,
		"lamaUsahaBulan":  36,
		"nilaiSurvei":     5,
	}
}

// TestHTTP_AC07_RincianKeempatKomponenAdaDiRespons memverifikasi AC-07: respons
// endpoint skoring memuat 4 komponen beserta bobot dan skor masing-masing.
// AC-07 secara eksplisit meminta test INTEGRASI endpoint, bukan unit test.
func TestHTTP_AC07_RincianKeempatKomponenAdaDiRespons(t *testing.T) {
	h := routerSkoring(newFakeParamRepoSkoring())

	rec := postJSON(t, h, "/api/pengajuan/7/skoring", dataSkoring())
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}

	var resp skoringResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode respons: %v", err)
	}

	if len(resp.Rincian) != 4 {
		t.Fatalf("jumlah rincian komponen = %d, mau 4", len(resp.Rincian))
	}
	if resp.PengajuanID != 7 {
		t.Errorf("pengajuanId = %d, mau 7 (diambil dari path)", resp.PengajuanID)
	}

	// Setiap komponen wajib membawa bobot dan skornya, bukan hanya nama.
	wajib := map[string]bool{
		domain.KomponenKapasitasBayar: false,
		domain.KomponenRiwayatSlik:    false,
		domain.KomponenLamaUsaha:      false,
		domain.KomponenSurveiLapangan: false,
	}
	for _, r := range resp.Rincian {
		if _, ada := wajib[r.Kode]; !ada {
			t.Errorf("komponen tak dikenal di respons: %q", r.Kode)
			continue
		}
		wajib[r.Kode] = true
		if r.Bobot <= 0 {
			t.Errorf("komponen %s: bobot = %v, mau > 0", r.Kode, r.Bobot)
		}
		if r.Nama == "" {
			t.Errorf("komponen %s: nama kosong", r.Kode)
		}
	}
	for kode, ada := range wajib {
		if !ada {
			t.Errorf("komponen %s tidak ada di respons", kode)
		}
	}
}

// TestHTTP_AC09_MarginDiBawahBatasGrade1Diblokir memverifikasi AC-09: margin
// 10,0 % untuk grade 1 (batas bawah 11,0 %) diblokir dengan 422 + rule BR-06.
//
// Kasus pembanding 11,5 % (di dalam rentang) WAJIB ada di test yang sama
// (AGENTS.md Larangan 18): tanpa itu, handler yang menolak SEMUA nilai akan
// tetap meloloskan test ini.
func TestHTTP_AC09_MarginDiBawahBatasGrade1Diblokir(t *testing.T) {
	h := routerSkoring(newFakeParamRepoSkoring())

	t.Run("10,0% grade 1 diblokir", func(t *testing.T) {
		rec := postJSON(t, h, "/api/pengajuan/7/margin", map[string]any{
			"akad": "MURABAHAH", "grade": 1, "nilai": 10.0,
		})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, mau 422 (body=%s)", rec.Code, rec.Body.String())
		}
		errResp := decodeError(t, rec)
		if errResp.Rule != "BR-06" {
			t.Errorf(`rule = %q, mau "BR-06"`, errResp.Rule)
		}
		if errResp.Error != "BUSINESS_RULE_VIOLATION" {
			t.Errorf(`error = %q, mau "BUSINESS_RULE_VIOLATION"`, errResp.Error)
		}
	})

	// Kasus pembanding: nilai di dalam rentang harus DITERIMA.
	t.Run("11,5% grade 1 diterima", func(t *testing.T) {
		rec := postJSON(t, h, "/api/pengajuan/7/margin", map[string]any{
			"akad": "MURABAHAH", "grade": 1, "nilai": 11.5,
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
		}
		var resp marginResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode respons: %v", err)
		}
		// Rentang yang dipakai ikut dikembalikan supaya ANL melihat dasarnya.
		if resp.RentangMin != 11.0 || resp.RentangMaks != 13.0 {
			t.Errorf("rentang = %v-%v, mau 11-13", resp.RentangMin, resp.RentangMaks)
		}
	})

	// Batas atas juga diblokir — memastikan pemeriksaannya dua arah.
	t.Run("13,5% grade 1 diblokir (di atas batas)", func(t *testing.T) {
		rec := postJSON(t, h, "/api/pengajuan/7/margin", map[string]any{
			"akad": "MURABAHAH", "grade": 1, "nilai": 13.5,
		})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, mau 422", rec.Code)
		}
		if r := decodeError(t, rec).Rule; r != "BR-06" {
			t.Errorf(`rule = %q, mau "BR-06"`, r)
		}
	})
}

// TestHTTP_AC04_SkoringTanpaSurveiValidDitolak memverifikasi AC-04: skoring
// tanpa survei VALID ditolak 422 dan pesannya menyebut BR-03.
func TestHTTP_AC04_SkoringTanpaSurveiValidDitolak(t *testing.T) {
	h, pra := routerSkoringDenganPrasyarat(newFakeParamRepoSkoring())

	// Keadaan pengajuan yang menentukan, bukan badan request.
	pra.keadaan.AdaSurveiValid = false

	rec := postJSON(t, h, "/api/pengajuan/7/skoring", dataSkoring())
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, mau 422 (body=%s)", rec.Code, rec.Body.String())
	}
	errResp := decodeError(t, rec)
	if errResp.Rule != "BR-03" {
		t.Errorf(`rule = %q, mau "BR-03"`, errResp.Rule)
	}
	if errResp.Message == "" {
		t.Error("message kosong; AC-04 meminta pesan yang menjelaskan sebabnya")
	}

	// Kasus pembanding (Larangan 18): prasyarat lengkap harus DITERIMA.
	pra.keadaan.AdaSurveiValid = true
	if rec2 := postJSON(t, h, "/api/pengajuan/7/skoring", dataSkoring()); rec2.Code != http.StatusOK {
		t.Fatalf("prasyarat lengkap: status = %d, mau 200 (body=%s)", rec2.Code, rec2.Body.String())
	}
}

// TestHTTP_AC06_Kolektibilitas2GradeTidakLebihBaikDari3 memverifikasi AC-06 pada
// lapisan HTTP: pengajuan tetap lanjut (200), tetapi grade minimal 3.
func TestHTTP_AC06_Kolektibilitas2GradeTidakLebihBaikDari3(t *testing.T) {
	h, pra := routerSkoringDenganPrasyarat(newFakeParamRepoSkoring())

	// Komponen lain sempurna supaya perhitungan mentah menghasilkan grade 1;
	// pemaksaan grade harus tetap terjadi. Kolektibilitas berasal dari hasil
	// SLIK tersimpan, bukan dari klien.
	pra.keadaan.Kolektibilitas = 2

	rec := postJSON(t, h, "/api/pengajuan/7/skoring", dataSkoring())
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 — kol-2 tetap lanjut (body=%s)", rec.Code, rec.Body.String())
	}

	var resp skoringResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode respons: %v", err)
	}
	if resp.Grade < 3 {
		t.Errorf("grade = %d, mau >= 3 untuk kolektibilitas 2", resp.Grade)
	}
	if !resp.GradeMinimalDipaksa {
		t.Error("gradeMinimalDipaksa = false, mau true supaya alasannya terlacak (BR-10)")
	}

	// Kasus pembanding: kol-1 dengan masukan sama boleh lebih baik dari 3.
	pra.keadaan.Kolektibilitas = 1
	rec2 := postJSON(t, h, "/api/pengajuan/7/skoring", dataSkoring())
	var resp2 skoringResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("decode respons kol-1: %v", err)
	}
	if resp2.Grade >= resp.Grade && resp2.Grade > 2 {
		t.Errorf("kol-1 grade = %d, kol-2 grade = %d — kol-1 seharusnya lebih baik",
			resp2.Grade, resp.Grade)
	}
	if resp2.GradeMinimalDipaksa {
		t.Error("kol-1: gradeMinimalDipaksa = true, mau false")
	}
}

// TestHTTP_AC15_UbahBobotLangsungBerlakuTanpaRestart memverifikasi AC-15 lewat
// endpoint: baris tabel parameter diubah di tengah jalan, router TIDAK dibangun
// ulang, dan hasilnya wajib berubah. Ini yang membuktikan handler membaca
// parameter dari repository, bukan dari konstanta di kode (Larangan 3).
func TestHTTP_AC15_UbahBobotLangsungBerlakuTanpaRestart(t *testing.T) {
	param := newFakeParamRepoSkoring()
	h := routerSkoring(param) // dibangun SEKALI

	body := dataSkoring()
	body["lamaUsahaBulan"] = 6 // komponen lama usaha bernilai rendah

	rec1 := postJSON(t, h, "/api/pengajuan/7/skoring", body)
	if rec1.Code != http.StatusOK {
		t.Fatalf("status awal = %d, mau 200 (body=%s)", rec1.Code, rec1.Body.String())
	}
	var awal skoringResponse
	if err := json.Unmarshal(rec1.Body.Bytes(), &awal); err != nil {
		t.Fatalf("decode awal: %v", err)
	}

	// ADM menaikkan bobot komponen yang skornya rendah -> skor akhir turun.
	for i := range param.komponen {
		if param.komponen[i].Kode == domain.KomponenLamaUsaha {
			param.komponen[i].Bobot = 80
		}
	}

	rec2 := postJSON(t, h, "/api/pengajuan/7/skoring", body)
	if rec2.Code != http.StatusOK {
		t.Fatalf("status sesudah = %d, mau 200", rec2.Code)
	}
	var sesudah skoringResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &sesudah); err != nil {
		t.Fatalf("decode sesudah: %v", err)
	}

	if sesudah.SkorAkhir == awal.SkorAkhir {
		t.Fatalf("skor tidak berubah (%d) setelah bobot diubah — parameter tidak dibaca dari tabel",
			awal.SkorAkhir)
	}
	if sesudah.SkorAkhir >= awal.SkorAkhir {
		t.Errorf("skor %d -> %d; menaikkan bobot komponen berskor rendah seharusnya menurunkan skor",
			awal.SkorAkhir, sesudah.SkorAkhir)
	}

	// Bobot yang DIPAKAI harus ikut terlihat di rincian, bukan hanya
	// memengaruhi skor akhir. Tanpa pemeriksaan ini, handler yang mengabaikan
	// kolom Bobot dan memakai konstanta tetap meloloskan test (Larangan 3).
	var bobotLamaUsaha float64
	for _, r := range sesudah.Rincian {
		if r.Kode == domain.KomponenLamaUsaha {
			bobotLamaUsaha = r.Bobot
		}
	}
	if bobotLamaUsaha != 80 {
		t.Errorf("bobot LAMA_USAHA di rincian = %v, mau 80 — nilai yang dipakai bukan dari tabel parameter",
			bobotLamaUsaha)
	}

	// Kontribusi wajib konsisten dengan bobot yang dilaporkan (BR-07/BR-08):
	// kontribusi = skorMentah x bobot. Ini mengunci ketiganya sekaligus.
	for _, r := range sesudah.Rincian {
		if mau := r.SkorMentah * r.Bobot; mau != r.Kontribusi {
			t.Errorf("komponen %s: kontribusi = %v, mau %v (skorMentah %v x bobot %v)",
				r.Kode, r.Kontribusi, mau, r.SkorMentah, r.Bobot)
		}
	}

	// Total bobot yang dilaporkan harus sama dengan jumlah bobot komponen.
	var jumlah float64
	for _, r := range sesudah.Rincian {
		jumlah += r.Bobot
	}
	if jumlah != sesudah.TotalBobot {
		t.Errorf("totalBobot = %v, mau %v (jumlah bobot rincian)", sesudah.TotalBobot, jumlah)
	}
}

// TestHTTP_AC15_UbahRentangMarginLangsungBerlaku adalah pasangan AC-15 untuk
// FR-07: nilai yang tadinya diblokir harus lolos setelah ADM melebarkan rentang.
func TestHTTP_AC15_UbahRentangMarginLangsungBerlaku(t *testing.T) {
	param := newFakeParamRepoSkoring()
	h := routerSkoring(param)

	req := map[string]any{"akad": "MURABAHAH", "grade": 1, "nilai": 10.0}

	if rec := postJSON(t, h, "/api/pengajuan/7/margin", req); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("sebelum: status = %d, mau 422", rec.Code)
	}

	// ADM melebarkan batas bawah grade 1 menjadi 9,0 %.
	r := param.rentang[1]
	r.MarginMin = 9.0
	param.rentang[1] = r

	if rec := postJSON(t, h, "/api/pengajuan/7/margin", req); rec.Code != http.StatusOK {
		t.Fatalf("sesudah: status = %d, mau 200 — rentang baru tidak terbaca (body=%s)",
			rec.Code, rec.Body.String())
	}
}

// TestHTTP_BR05_Grade5DitolakDiEndpointMargin memverifikasi BR-05 di lapisan
// HTTP: grade 5 tidak dibiayai, jadi permintaan margin ditolak 422 rule BR-05.
func TestHTTP_BR05_Grade5DitolakDiEndpointMargin(t *testing.T) {
	h := routerSkoring(newFakeParamRepoSkoring())

	rec := postJSON(t, h, "/api/pengajuan/7/margin", map[string]any{
		"akad": "MURABAHAH", "grade": 5, "nilai": 25.0,
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, mau 422 (body=%s)", rec.Code, rec.Body.String())
	}
	if r := decodeError(t, rec).Rule; r != "BR-05" {
		t.Errorf(`rule = %q, mau "BR-05"`, r)
	}

	// Kasus pembanding: grade 4 masih dibiayai.
	rec2 := postJSON(t, h, "/api/pengajuan/7/margin", map[string]any{
		"akad": "MURABAHAH", "grade": 4, "nilai": 19.0,
	})
	if rec2.Code != http.StatusOK {
		t.Fatalf("grade 4: status = %d, mau 200 (body=%s)", rec2.Code, rec2.Body.String())
	}
}

// TestHTTP_NisbahMusyarakahMemakaiRentangSendiri memastikan akad musyarakah
// divalidasi terhadap kolom nisbah, bukan kolom margin murabahah.
func TestHTTP_NisbahMusyarakahMemakaiRentangSendiri(t *testing.T) {
	h := routerSkoring(newFakeParamRepoSkoring())

	// 22 % ada di dalam rentang nisbah grade 1 (20-25) tetapi di LUAR rentang
	// margin murabahah (11-13). Kalau handler tertukar kolom, ini gagal.
	rec := postJSON(t, h, "/api/pengajuan/7/margin", map[string]any{
		"akad": "MUSYARAKAH", "grade": 1, "nilai": 22.0,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}

	// Sebaliknya, 12 % sah untuk murabahah tetapi di luar rentang nisbah.
	rec2 := postJSON(t, h, "/api/pengajuan/7/margin", map[string]any{
		"akad": "MUSYARAKAH", "grade": 1, "nilai": 12.0,
	})
	if rec2.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, mau 422 untuk nisbah 12%% grade 1", rec2.Code)
	}
}

// TestHTTP_SkoringIDPengajuanTidakValid memastikan path yang tidak masuk akal
// dibalas 400, bukan 500 atau 200.
func TestHTTP_SkoringIDPengajuanTidakValid(t *testing.T) {
	h := routerSkoring(newFakeParamRepoSkoring())

	rec := postJSON(t, h, "/api/pengajuan/abc/skoring", dataSkoring())
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, mau 400 (body=%s)", rec.Code, rec.Body.String())
	}
	if e := decodeError(t, rec).Error; e != "VALIDATION_ERROR" {
		t.Errorf(`error = %q, mau "VALIDATION_ERROR"`, e)
	}
}

// TestHTTP_MarginPesanErrorTanpaDataPribadi menegakkan BR-11 pada jalur error:
// pesan yang dikirim ke klien tidak boleh memuat NIK atau nomor dokumen.
func TestHTTP_MarginPesanErrorTanpaDataPribadi(t *testing.T) {
	h := routerSkoring(newFakeParamRepoSkoring())

	rec := postJSON(t, h, "/api/pengajuan/7/margin", map[string]any{
		"akad": "MURABAHAH", "grade": 1, "nilai": 10.0,
		// Field asing seperti ini diabaikan decoder; disertakan untuk memastikan
		// nilainya tidak pernah muncul kembali di pesan error.
		"nik": "3404123456789012",
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, mau 422", rec.Code)
	}
	if body := rec.Body.String(); bytes.Contains([]byte(body), []byte("3404123456789012")) {
		t.Errorf("respons memuat NIK — melanggar BR-11: %s", body)
	}
}
