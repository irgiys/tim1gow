package slik

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Test di berkas ini memakai httptest, bukan mock-slik nyata, supaya jalur
// error (503, timeout, kontrak dilanggar) dapat dipaksa dengan pasti. Kontrak
// yang ditiru adalah AGENTS.md bagian 5.2.

const nikUji = "3404123456780001"

func serverPalsu(t *testing.T, tangani http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(tangani)
	t.Cleanup(srv.Close)
	return srv
}

func clientKe(srv *httptest.Server, timeout time.Duration) *Client {
	return NewClient(Opsi{BaseURL: srv.URL, InquiryPath: "/slik/inquiry", Timeout: timeout})
}

// AC-05: kolektibilitas 1 lolos dan nilainya terbaca utuh.
//
// Ini kasus PEMBANDING untuk seluruh test penolakan di bawah (Larangan 18).
// Tanpa kasus ini, client yang selalu mengembalikan "gagal" akan meloloskan
// setiap test jalur error.
func TestInquiry_Kol1_SuksesDanTerbacaUtuh(t *testing.T) {
	srv := serverPalsu(t, func(w http.ResponseWriter, r *http.Request) {
		// BR-11: NIK wajib di badan, bukan di URL.
		if strings.Contains(r.URL.String(), nikUji) {
			t.Errorf("NIK muncul di URL: %s", r.URL.String())
		}
		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("badan request tidak terbaca: %v", err)
		}
		if req["nik"] != nikUji {
			t.Errorf("NIK di badan = %q, mau %q", req["nik"], nikUji)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jawabanInquiry{
			NIK: nikUji, Nama: "Nasabah Uji", Kolektibilitas: 1,
			JumlahFasilitasAktif: 2, TotalBakiDebet: 15_000_000,
			TanggalData: "2026-08-20", ReferenceID: "SLIK-REF-001",
		})
	})

	h, err := clientKe(srv, 2*time.Second).Inquiry(context.Background(), nikUji)
	if err != nil {
		t.Fatalf("Inquiry() error tak terduga: %v", err)
	}
	if !h.Sukses() {
		t.Fatalf("Sukses() = false, mau true (status=%s)", h.Status)
	}
	if *h.Kolektibilitas != 1 {
		t.Errorf("kolektibilitas = %d, mau 1", *h.Kolektibilitas)
	}
	if h.TanggalData == nil {
		t.Fatal("tanggalData nil — berlaku_sampai (BR-04) tidak bisa dihitung")
	}
	if got := h.TanggalData.Format("2006-01-02"); got != "2026-08-20" {
		t.Errorf("tanggalData = %s, mau 2026-08-20", got)
	}
	if h.ReferenceID != "SLIK-REF-001" {
		t.Errorf("referenceId = %q, mau SLIK-REF-001", h.ReferenceID)
	}
}

// Kol-5 tetap SUKSES di lapisan client. Penolakan otomatis (REJECTED_SLIK)
// adalah keputusan service, bukan client — client hanya melaporkan fakta.
// Test ini menjaga batas lapisan itu agar tidak bergeser diam-diam.
func TestInquiry_Kol5_TetapSuksesKeputusanMilikService(t *testing.T) {
	srv := serverPalsu(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(jawabanInquiry{
			NIK: nikUji, Kolektibilitas: 5, TanggalData: "2026-08-20",
		})
	})

	h, err := clientKe(srv, 2*time.Second).Inquiry(context.Background(), nikUji)
	if err != nil {
		t.Fatalf("Inquiry() error tak terduga: %v", err)
	}
	if !h.Sukses() || *h.Kolektibilitas != 5 {
		t.Fatalf("kol-5 harus dilaporkan apa adanya, dapat status=%s", h.Status)
	}
}

// AC-06 / Larangan 15: 503 TIDAK boleh jadi SLIK bersih.
func TestInquiry_503_JadiStatusLayananTidakAdaBukanBersih(t *testing.T) {
	srv := serverPalsu(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(jawabanGalat{Error: "SERVICE_UNAVAILABLE"})
	})

	h, err := clientKe(srv, 2*time.Second).Inquiry(context.Background(), nikUji)
	if err != nil {
		t.Fatalf("503 harus jadi status, bukan error Go: %v", err)
	}
	if h.Status != StatusLayananTidakAda {
		t.Errorf("status = %s, mau %s", h.Status, StatusLayananTidakAda)
	}
	if h.Kolektibilitas != nil {
		t.Errorf("kolektibilitas = %d, mau nil — nilai default = Larangan 15", *h.Kolektibilitas)
	}
	if h.Sukses() {
		t.Error("Sukses() = true saat SLIK mati — ini kegagalan yang dilarang Larangan 15")
	}
}

// 404 NIK_NOT_FOUND juga bukan SLIK bersih, dan pesannya tidak boleh membawa NIK.
func TestInquiry_404_JadiStatusNIKTidakDitemukan(t *testing.T) {
	srv := serverPalsu(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(jawabanGalat{Error: "NIK_NOT_FOUND"})
	})

	h, err := clientKe(srv, 2*time.Second).Inquiry(context.Background(), nikUji)
	if err != nil {
		t.Fatalf("404 harus jadi status, bukan error Go: %v", err)
	}
	if h.Status != StatusNIKTidakDitemukan {
		t.Errorf("status = %s, mau %s", h.Status, StatusNIKTidakDitemukan)
	}
	if h.Sukses() {
		t.Error("Sukses() = true untuk NIK tidak ditemukan")
	}
}

// Timeout wajib tercatat sebagai TIMEOUT, bukan menggantung atau jadi bersih.
func TestInquiry_Timeout_JadiStatusTimeout(t *testing.T) {
	srv := serverPalsu(t, func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(300 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(jawabanInquiry{Kolektibilitas: 1})
	})

	h, err := clientKe(srv, 50*time.Millisecond).Inquiry(context.Background(), nikUji)
	if err != nil {
		t.Fatalf("timeout harus jadi status, bukan error Go: %v", err)
	}
	if h.Status != StatusTimeout {
		t.Errorf("status = %s, mau %s", h.Status, StatusTimeout)
	}
	if h.Sukses() {
		t.Error("Sukses() = true saat timeout — Larangan 15")
	}
}

// 200 dengan kolektibilitas di luar 1..5 adalah pelanggaran kontrak, dan harus
// menjadi error Go — bukan Hasil yang kelihatan sah. Nilai 0 di sini penting:
// itu bentuk paling berbahaya, karena terlihat seperti "belum ada data".
func TestInquiry_Kol0_DitolakSebagaiPelanggaranKontrak(t *testing.T) {
	for _, kol := range []int{0, 6, -1} {
		srv := serverPalsu(t, func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(jawabanInquiry{Kolektibilitas: kol})
		})

		h, err := clientKe(srv, 2*time.Second).Inquiry(context.Background(), nikUji)
		if err == nil {
			t.Errorf("kolektibilitas %d: error = nil, mau ErrKontrakSlik (status=%s)", kol, h.Status)
		}
		if h.Sukses() {
			t.Errorf("kolektibilitas %d: Sukses() = true", kol)
		}
	}
}

// 500 dari hulu tidak boleh ditelan menjadi status apa pun yang bisa lolos.
func TestInquiry_500_DitolakSebagaiPelanggaranKontrak(t *testing.T) {
	srv := serverPalsu(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	if _, err := clientKe(srv, 2*time.Second).Inquiry(context.Background(), nikUji); err == nil {
		t.Error("500 dari SLIK: error = nil, mau ErrKontrakSlik")
	}
}
