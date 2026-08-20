package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/irgiys/tim1gow/backend/internal/domain"
)

// AC-09: margin 10,0 % untuk grade 1 (di bawah batas 11,0 %) DIBLOKIR sistem.
func TestValidasi_AC09_MarginDiBawahBatasGrade1Diblokir(t *testing.T) {
	repo := newFakeParameterRepo()
	svc := NewMarginService(repo)

	_, err := svc.Validasi(domain.AkadMurabahah, 1, 10.0)
	if err == nil {
		t.Fatal("margin 10,0 persen grade 1: ingin diblokir, dapat nil")
	}

	var brErr *domain.BusinessRuleError
	if !errors.As(err, &brErr) {
		t.Fatalf("ingin BusinessRuleError, dapat %T: %v", err, err)
	}
	if brErr.Rule != "BR-06" {
		t.Errorf("rule = %q, ingin BR-06", brErr.Rule)
	}
	// AC-04 menuntut pesan pelanggaran menyebut kode BR-nya.
	if !strings.Contains(brErr.Message, "grade 1") {
		t.Errorf("pesan %q tidak menyebut grade-nya; ANL tidak tahu batas mana yang dilanggar", brErr.Message)
	}

	// Yang di dalam rentang harus lolos, supaya test di atas tidak lolos hanya
	// karena fungsinya selalu menolak.
	if _, err := svc.Validasi(domain.AkadMurabahah, 1, 11.0); err != nil {
		t.Errorf("margin 11,0 persen grade 1 (tepat batas bawah) seharusnya lolos: %v", err)
	}
	if _, err := svc.Validasi(domain.AkadMurabahah, 1, 13.0); err != nil {
		t.Errorf("margin 13,0 persen grade 1 (tepat batas atas) seharusnya lolos: %v", err)
	}
	if _, err := svc.Validasi(domain.AkadMurabahah, 1, 13.01); err == nil {
		t.Error("margin 13,01 persen grade 1 (di atas batas) seharusnya diblokir")
	}
}

// BR-06 berlaku juga untuk nisbah musyarakah, dengan rentang yang berbeda.
// Angka yang sah untuk murabahah belum tentu sah untuk musyarakah.
func TestValidasi_BR06_NisbahMusyarakahMemakaiRentangSendiri(t *testing.T) {
	svc := NewMarginService(newFakeParameterRepo())

	// 22 persen sah sebagai nisbah grade 1 (20-25), tetapi di luar rentang margin
	// murabahah grade 1 (11-13).
	if _, err := svc.Validasi(domain.AkadMusyarakah, 1, 22.0); err != nil {
		t.Errorf("nisbah 22 persen grade 1 seharusnya lolos: %v", err)
	}
	if _, err := svc.Validasi(domain.AkadMurabahah, 1, 22.0); err == nil {
		t.Error("margin 22 persen grade 1 seharusnya diblokir; rentang akad tertukar")
	}
}

// BR-05: grade yang tidak dibiayai ditolak sebelum angkanya divalidasi.
func TestValidasi_BR05_Grade5TidakDibiayai(t *testing.T) {
	svc := NewMarginService(newFakeParameterRepo())

	_, err := svc.Validasi(domain.AkadMurabahah, 5, 25.0)
	var brErr *domain.BusinessRuleError
	if !errors.As(err, &brErr) {
		t.Fatalf("ingin BusinessRuleError, dapat %v", err)
	}
	if brErr.Rule != "BR-05" {
		t.Errorf("rule = %q, ingin BR-05", brErr.Rule)
	}
}

// AC-15 untuk rentang margin: ADM mengubah batas grade, validasi berikutnya
// memakai batas baru tanpa restart. Ini yang membuktikan rentang dibaca dari
// data, bukan dari konstanta di kode.
func TestValidasi_AC15_UbahRentangLangsungBerlaku(t *testing.T) {
	repo := newFakeParameterRepo()
	svc := NewMarginService(repo)

	// Sebelum diubah: 10,5 % di luar rentang grade 1 (11,0-13,0).
	if _, err := svc.Validasi(domain.AkadMurabahah, 1, 10.5); err == nil {
		t.Fatal("10,5 persen seharusnya diblokir sebelum rentang diubah")
	}

	repo.ubahBatasMargin(1, 10.0, 13.0) // ADM menurunkan batas bawah lewat FR-13

	if _, err := svc.Validasi(domain.AkadMurabahah, 1, 10.5); err != nil {
		t.Fatalf("10,5 persen seharusnya lolos setelah batas bawah diturunkan ke 10,0: %v", err)
	}
}

// Rentang yang belum diatur harus menghentikan perhitungan, bukan meloloskan
// nilai apa pun (kegagalan konfigurasi bukan alasan melewati BR-06).
func TestValidasi_GradeTanpaRentangDitolak(t *testing.T) {
	repo := newFakeParameterRepo()
	repo.rentang = nil
	svc := NewMarginService(repo)

	if _, err := svc.Validasi(domain.AkadMurabahah, 1, 12.0); err == nil {
		t.Fatal("rentang_margin kosong: ingin error, dapat nil")
	}
}

// Hasil validasi wajib memuat rentang yang dipakai, supaya ANL melihat dasar
// keputusan sistem, bukan hanya lolos/tidak.
func TestValidasi_HasilMemuatRentangYangDipakai(t *testing.T) {
	svc := NewMarginService(newFakeParameterRepo())

	h, err := svc.Validasi(domain.AkadMurabahah, 2, 14.0)
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if h.RentangMin != 13.0 || h.RentangMaks != 15.5 {
		t.Errorf("rentang = %v-%v, ingin 13-15.5", h.RentangMin, h.RentangMaks)
	}
	if h.Grade != 2 || h.Nilai != 14.0 {
		t.Errorf("hasil = grade %d nilai %v, ingin grade 2 nilai 14", h.Grade, h.Nilai)
	}
}
