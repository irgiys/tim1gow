package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fixtureNyata memuat fixtures/nasabah-uji.csv yang sebenarnya, bukan salinan
// di dalam test. Kalau QA mengubah berkas itu dan mock berhenti cocok, test
// ini yang memberi tahu — salinan inline justru menyembunyikannya.
func fixtureNyata(t *testing.T) map[string]Nasabah {
	t.Helper()
	n, err := MuatFixturesDariBerkas("../fixtures/nasabah-uji.csv")
	if err != nil {
		t.Fatalf("memuat fixtures nyata: %v", err)
	}
	return n
}

func serverUji(t *testing.T) http.Handler {
	t.Helper()
	s := NewServer(fixtureNyata(t))
	s.sekarang = func() time.Time { return time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC) }
	return s.Handler()
}

func inquiry(t *testing.T, h http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/slik/inquiry", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// TestInquiry_NIKAdaMengembalikan200 adalah kasus pembanding untuk seluruh
// test penolakan di bawah (AGENTS.md Larangan 18): tanpa ini, mock yang
// menolak SEMUA NIK akan meloloskan test 404 dan 503.
func TestInquiry_NIKAdaMengembalikan200(t *testing.T) {
	h := serverUji(t)

	rec := inquiry(t, h, `{"nik":"3404110985000001"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200. body: %s", rec.Code, rec.Body.String())
	}

	var got jawabanInquiry
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("membaca respons: %v", err)
	}

	// Nilai dibandingkan dengan baris fixtures, bukan dengan angka yang
	// diketik ulang di sini.
	mau := fixtureNyata(t)["3404110985000001"]
	if got.NIK != mau.NIK || got.Nama != mau.Nama {
		t.Errorf("identitas = %q/%q, mau %q/%q", got.NIK, got.Nama, mau.NIK, mau.Nama)
	}
	if got.Kolektibilitas != mau.Kolektibilitas {
		t.Errorf("kolektibilitas = %d, mau %d", got.Kolektibilitas, mau.Kolektibilitas)
	}
	if got.JumlahFasilitasAktif != mau.JumlahFasilitasAktif {
		t.Errorf("jumlahFasilitasAktif = %d, mau %d", got.JumlahFasilitasAktif, mau.JumlahFasilitasAktif)
	}
	if got.TotalBakiDebet != mau.TotalBakiDebet {
		t.Errorf("totalBakiDebet = %d, mau %d", got.TotalBakiDebet, mau.TotalBakiDebet)
	}
	if got.TanggalData != "2026-08-20" {
		t.Errorf("tanggalData = %q, mau 2026-08-20", got.TanggalData)
	}
	if !strings.HasPrefix(got.ReferenceID, "SLIK-") {
		t.Errorf("referenceId = %q, mau berawalan SLIK-", got.ReferenceID)
	}
}

// TestInquiry_ReferenceIDTidakMemuatNIK menjaga BR-11: referenceId ikut
// tercatat di log backend, jadi ia tidak boleh membawa data pribadi.
func TestInquiry_ReferenceIDTidakMemuatNIK(t *testing.T) {
	h := serverUji(t)
	const nik = "3404110985000001"

	rec := inquiry(t, h, `{"nik":"`+nik+`"}`)
	var got jawabanInquiry
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("membaca respons: %v", err)
	}
	if strings.Contains(got.ReferenceID, nik) {
		t.Errorf("referenceId %q memuat NIK — melanggar BR-11", got.ReferenceID)
	}
}

// TestInquiry_NIKTidakDikenalMengembalikan404 memakai NIK pemicu dari
// fixtures (skenario "NIK tidak ditemukan - uji respons 404").
func TestInquiry_NIKTidakDikenalMengembalikan404(t *testing.T) {
	h := serverUji(t)

	rec := inquiry(t, h, `{"nik":"3404999999999999"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, mau 404. body: %s", rec.Code, rec.Body.String())
	}

	var got jawabanGalat
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("membaca respons: %v", err)
	}
	if got.Error != "NIK_NOT_FOUND" {
		t.Errorf("error = %q, mau NIK_NOT_FOUND", got.Error)
	}
}

// TestInquiry_NIKPemicuMengembalikan503 membuktikan jalur kegagalan dapat
// didemokan tanpa mematikan container (brief §6.1).
func TestInquiry_NIKPemicuMengembalikan503(t *testing.T) {
	h := serverUji(t)

	rec := inquiry(t, h, `{"nik":"`+NIKPemicu503+`"}`)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, mau 503. body: %s", rec.Code, rec.Body.String())
	}

	var got jawabanGalat
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("membaca respons: %v", err)
	}
	if got.Error != "SERVICE_UNAVAILABLE" {
		t.Errorf("error = %q, mau SERVICE_UNAVAILABLE", got.Error)
	}
}

// TestInquiry_QueryPaksa503 memaksa 503 pada NIK yang sebenarnya valid.
// Pasangan asersinya penting: NIK yang sama TANPA ?paksa harus 200, sehingga
// test membuktikan query-nya yang bekerja, bukan NIK-nya yang rusak.
func TestInquiry_QueryPaksa503(t *testing.T) {
	h := serverUji(t)
	const nik = "3404110985000001"

	req := httptest.NewRequest(http.MethodPost, "/slik/inquiry?paksa=503", strings.NewReader(`{"nik":"`+nik+`"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("dengan ?paksa=503 status = %d, mau 503", rec.Code)
	}

	if rec := inquiry(t, h, `{"nik":"`+nik+`"}`); rec.Code != http.StatusOK {
		t.Fatalf("tanpa ?paksa status = %d, mau 200 — NIK-nya seharusnya valid", rec.Code)
	}
}

// TestInquiry_Kolektibilitas345AdaDiFixtures memastikan data untuk AC-05
// (penolakan otomatis) benar-benar tersedia. Backend tidak dapat menguji
// REJECTED_SLIK kalau mock tidak pernah bisa mengembalikan kol 3-5.
func TestInquiry_Kolektibilitas345AdaDiFixtures(t *testing.T) {
	h := serverUji(t)

	// NIK -> kolektibilitas yang diharapkan, diambil dari kolom skenario CSV.
	kasus := map[string]int{
		"3404270995000006": 3, // "SLIK kol-3 - penolakan otomatis"
		"3404031292000004": 4, // "SLIK kol-4 - penolakan otomatis"
		"3404121189000008": 5, // "SLIK kol-5 - penolakan otomatis"
		"3404150688000003": 2, // "SLIK kol-2 - grade dipaksa minimal 3" (AC-06)
	}

	for nik, mauKol := range kasus {
		rec := inquiry(t, h, `{"nik":"`+nik+`"}`)
		if rec.Code != http.StatusOK {
			t.Errorf("NIK %s: status = %d, mau 200", nik, rec.Code)
			continue
		}
		var got jawabanInquiry
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Errorf("NIK %s: %v", nik, err)
			continue
		}
		if got.Kolektibilitas != mauKol {
			t.Errorf("NIK %s: kolektibilitas = %d, mau %d", nik, got.Kolektibilitas, mauKol)
		}
	}
}

func TestInquiry_MethodSelainPostDitolak(t *testing.T) {
	h := serverUji(t)

	req := httptest.NewRequest(http.MethodGet, "/slik/inquiry", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /slik/inquiry status = %d, mau 405", rec.Code)
	}
}

func TestInquiry_BadanRusakDitolak(t *testing.T) {
	h := serverUji(t)

	for nama, body := range map[string]string{
		"bukan json":  `{nik:}`,
		"nik kosong":  `{"nik":""}`,
		"tanpa field": `{}`,
	} {
		if rec := inquiry(t, h, body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, mau 400", nama, rec.Code)
		}
	}
}

func TestHealth_MelaporkanJumlahNasabah(t *testing.T) {
	h := serverUji(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200", rec.Code)
	}

	var got struct {
		Status string `json:"status"`
		Jumlah int    `json:"jumlah_nasabah"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("membaca respons: %v", err)
	}
	if got.Status != "ok" {
		t.Errorf("status = %q, mau ok", got.Status)
	}
	// Healthcheck yang melaporkan 0 nasabah berarti fixtures gagal dimuat.
	if got.Jumlah == 0 {
		t.Error("jumlah_nasabah = 0 — fixtures tidak termuat")
	}
}

// TestMuatFixtures_KolomHilangGagalKeras: fixtures rusak harus terlihat saat
// start, bukan menjadi 404 misterius saat demo.
func TestMuatFixtures_KolomHilangGagalKeras(t *testing.T) {
	csv := "nik,nama\n3404110985000001,Siti\n"
	if _, err := MuatFixtures(strings.NewReader(csv)); err == nil {
		t.Error("header tanpa kolom kolektibilitas seharusnya menghasilkan error")
	}
}

// TestMuatFixtures_BarisPenandaSkenarioDilewati: baris pemicu 404/503 punya
// kolektibilitas "-" dan BUKAN data nasabah. Kalau ikut dimuat, NIK pemicu
// 404 justru ditemukan dan dijawab 200 dengan kolektibilitas 0 — nilai
// default diam-diam yang dilarang AGENTS.md bagian 4.3.
func TestMuatFixtures_BarisPenandaSkenarioDilewati(t *testing.T) {
	csv := "nik,nama,kolektibilitas,jumlah_fasilitas_aktif,total_baki_debet\n" +
		"3404999999999999,TIDAK ADA,-,-,-\n" +
		"3404110985000001,Siti,1,1,8000000\n"

	got, err := MuatFixtures(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if _, ada := got["3404999999999999"]; ada {
		t.Error("baris penanda skenario ikut dimuat — NIK pemicu 404 akan dijawab 200")
	}
	// Pembanding: baris data biasa tetap dimuat.
	if _, ada := got["3404110985000001"]; !ada {
		t.Error("baris data nasabah justru tidak dimuat")
	}
}

// TestMuatFixtures_NilaiTidakValidGagal adalah pembanding test di atas:
// "-" boleh, tetapi teks sembarang tidak boleh diam-diam menjadi 0.
func TestMuatFixtures_NilaiTidakValidGagal(t *testing.T) {
	csv := "nik,nama,kolektibilitas,jumlah_fasilitas_aktif,total_baki_debet\n" +
		"3404110985000001,Siti,satu,1,8000000\n"

	if _, err := MuatFixtures(strings.NewReader(csv)); err == nil {
		t.Error("kolektibilitas \"satu\" seharusnya menghasilkan error, bukan 0")
	}
}
