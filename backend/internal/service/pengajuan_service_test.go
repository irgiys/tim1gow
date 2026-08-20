package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/irgiys/tim1gow/backend/internal/domain"
)

// Test-test di berkas ini diturunkan dari BR-12 dan BR-01 pada AGENTS.md
// bagian 5, bukan dari kode yang sudah ada — pada saat ditulis,
// pengajuan_service.go belum ada sama sekali.
//
// Keduanya sebelumnya berstatus "Done" di docs/TRACEABILITY.md tanpa satu pun
// test yang benar-benar mengujinya (lihat commit b9adbe8).

// BR-12: nomor referensi berformat IMT-YYYYMMDD-NNNN, unik, dan tidak pernah
// dipakai ulang — termasuk untuk pengajuan yang ditolak.
//
// AC-01 memeriksa format ini secara langsung. Nomor dibangkitkan di server
// (AGENTS.md Larangan 4: tidak boleh dibangkitkan di frontend).
func TestBuatNomorReferensi_BR12_Format(t *testing.T) {
	repo := newFakeNomorRepo()
	svc := NewPengajuanService(repo, newFakeParameterRepo())

	tanggal := time.Date(2026, 8, 20, 10, 30, 0, 0, time.UTC)

	nomor, err := svc.BuatNomorReferensi(tanggal)
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}

	if ingin := "IMT-20260820-0001"; nomor != ingin {
		t.Errorf("nomor = %q, ingin %q", nomor, ingin)
	}
}

// Urutan berlanjut dalam hari yang sama, dan NNNNN di-pad 4 digit.
func TestBuatNomorReferensi_BR12_UrutanDalamSatuHari(t *testing.T) {
	repo := newFakeNomorRepo()
	svc := NewPengajuanService(repo, newFakeParameterRepo())

	tanggal := time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC)

	var terakhir string
	for i := 1; i <= 3; i++ {
		n, err := svc.BuatNomorReferensi(tanggal)
		if err != nil {
			t.Fatalf("pengajuan ke-%d: %v", i, err)
		}
		if n == terakhir {
			t.Fatalf("nomor ke-%d sama dengan sebelumnya (%q); nomor dipakai ulang", i, n)
		}
		terakhir = n
	}

	if ingin := "IMT-20260820-0003"; terakhir != ingin {
		t.Errorf("nomor ketiga = %q, ingin %q", terakhir, ingin)
	}
}

// Urutan direset per tanggal: hari baru mulai dari 0001 lagi.
func TestBuatNomorReferensi_BR12_ResetPerTanggal(t *testing.T) {
	repo := newFakeNomorRepo()
	svc := NewPengajuanService(repo, newFakeParameterRepo())

	hariIni := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	besok := time.Date(2026, 8, 21, 9, 0, 0, 0, time.UTC)

	if _, err := svc.BuatNomorReferensi(hariIni); err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}

	nomorBesok, err := svc.BuatNomorReferensi(besok)
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if ingin := "IMT-20260821-0001"; nomorBesok != ingin {
		t.Errorf("nomor hari berikutnya = %q, ingin %q", nomorBesok, ingin)
	}
}

// BR-12 bagian yang paling mudah terlewat: nomor tidak boleh dipakai ulang
// walaupun pengajuan pemiliknya DITOLAK. Urutan maju terus, tidak mengisi
// kembali lubang yang ditinggalkan pengajuan yang gagal.
func TestBuatNomorReferensi_BR12_NomorPengajuanDitolakTidakDipakaiUlang(t *testing.T) {
	repo := newFakeNomorRepo()
	svc := NewPengajuanService(repo, newFakeParameterRepo())

	tanggal := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)

	pertama, err := svc.BuatNomorReferensi(tanggal)
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}

	// Pengajuan pertama ditolak (REJECTED_SLIK / REJECTED_SCORING / REJECTED).
	// Nomornya tetap terpakai selamanya.
	repo.tandaiDitolak(pertama)

	kedua, err := svc.BuatNomorReferensi(tanggal)
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}

	if kedua == pertama {
		t.Fatalf("nomor %q dipakai ulang setelah pengajuan sebelumnya ditolak (BR-12)", kedua)
	}
	if ingin := "IMT-20260820-0002"; kedua != ingin {
		t.Errorf("nomor kedua = %q, ingin %q", kedua, ingin)
	}
}

// BR-01: plafon < Rp 5.000.000 atau > Rp 500.000.000 ditolak saat submit,
// dengan pesan yang menjelaskan batasnya.
//
// Batas atas & bawah diuji TEPAT di tepinya beserta kasus pembanding yang
// harus diterima (AGENTS.md Larangan 18) — fungsi yang argumen ambangnya
// tertukar akan meloloskan seluruh kasus penolakan kalau hanya satu arah
// yang diuji.
func TestPastikanPlafonValid_BR01_BatasBawahDanAtas(t *testing.T) {
	svc := NewPengajuanService(newFakeNomorRepo(), newFakeParameterRepo())

	kasus := []struct {
		nama    string
		plafon  int64
		inginOK bool
	}{
		// Ditolak — di bawah batas bawah.
		{"nol", 0, false},
		{"4 juta", 4_000_000, false},
		{"tepat satu rupiah di bawah batas bawah", 4_999_999, false},

		// Diterima — tepat di batas dan di dalam rentang.
		{"tepat batas bawah 5 juta", 5_000_000, true},
		{"30 juta (AC-01)", 30_000_000, true},
		{"120 juta (AC-10)", 120_000_000, true},
		{"tepat batas atas 500 juta", 500_000_000, true},

		// Ditolak — di atas batas atas.
		{"tepat satu rupiah di atas batas atas", 500_000_001, false},
		{"600 juta", 600_000_000, false},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			err := svc.PastikanPlafonValid(k.plafon)

			if k.inginOK {
				if err != nil {
					t.Fatalf("plafon %d seharusnya diterima, dapat error: %v", k.plafon, err)
				}
				return
			}

			if err == nil {
				t.Fatalf("plafon %d seharusnya ditolak, dapat nil", k.plafon)
			}

			var brErr *domain.BusinessRuleError
			if !errors.As(err, &brErr) {
				t.Fatalf("ingin BusinessRuleError, dapat %T: %v", err, err)
			}
			if brErr.Rule != "BR-01" {
				t.Errorf("rule = %q, ingin BR-01", brErr.Rule)
			}
		})
	}
}

// BR-01 mewajibkan pesannya MENJELASKAN BATAS, bukan sekadar "plafon tidak
// valid" — AO harus tahu angka mana yang berlaku tanpa membuka dokumen.
func TestPastikanPlafonValid_BR01_PesanMenyebutBatas(t *testing.T) {
	svc := NewPengajuanService(newFakeNomorRepo(), newFakeParameterRepo())

	err := svc.PastikanPlafonValid(4_000_000)

	var brErr *domain.BusinessRuleError
	if !errors.As(err, &brErr) {
		t.Fatalf("ingin BusinessRuleError, dapat %v", err)
	}

	// Kedua batas wajib disebut supaya AO tahu rentang yang berlaku.
	for _, potongan := range []string{"5.000.000", "500.000.000"} {
		if !strings.Contains(brErr.Message, potongan) {
			t.Errorf("pesan %q tidak menyebut batas %s", brErr.Message, potongan)
		}
	}
}

// Batas plafon BR-01 adalah PARAMETER, bukan konstanta di kode (AGENTS.md
// Larangan 3). Test ini yang membuktikannya: barisnya diubah di tengah test
// pada instance service yang sama, lalu nilai yang tadinya ditolak harus
// diterima.
func TestPastikanPlafonValid_BR01_BatasDibacaDariParameter(t *testing.T) {
	repo := newFakeParameterRepo()
	svc := NewPengajuanService(newFakeNomorRepo(), repo)

	// Sebelum diubah: 4 juta di bawah batas bawah 5 juta.
	if err := svc.PastikanPlafonValid(4_000_000); err == nil {
		t.Fatal("4 juta seharusnya ditolak sebelum batas diubah")
	}

	repo.ubahNilaiUmum(KunciPlafonMinimum, 3_000_000) // ADM lewat FR-13

	if err := svc.PastikanPlafonValid(4_000_000); err != nil {
		t.Fatalf("4 juta seharusnya diterima setelah batas bawah diturunkan ke 3 juta: %v", err)
	}
}

// Batas yang belum diatur harus MENGHENTIKAN submit, bukan diam-diam memakai
// nilai default — kegagalan konfigurasi bukan alasan melewati BR-01
// (AGENTS.md Larangan 3 & 15).
func TestPastikanPlafonValid_BR01_ParameterKosongTidakMemakaiDefault(t *testing.T) {
	repo := newFakeParameterRepo()
	repo.umum = map[string]float64{}
	svc := NewPengajuanService(newFakeNomorRepo(), repo)

	if err := svc.PastikanPlafonValid(30_000_000); err == nil {
		t.Fatal("batas plafon belum diatur: ingin error, dapat nil")
	}
}
