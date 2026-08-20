package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/irgiys/tim1gow/backend/internal/domain"
)

func dokumenServiceUji(t *testing.T) (*DokumenService, *fakeDokumenRepo, *fakeDokumenWajibRepo) {
	t.Helper()
	dok := newFakeDokumenRepo()
	wajib := newFakeDokumenWajibRepo()
	svc := NewDokumenService(dok, wajib).DenganWaktu(func() time.Time { return waktuUji })
	return svc, dok, wajib
}

// unggahTiganya mengunggah ketiga dokumen wajib dan mengembalikan ID-nya.
func unggahTiganya(t *testing.T, svc *DokumenService, pengajuanID int64) map[string]int64 {
	t.Helper()
	id := map[string]int64{}
	for _, jenis := range []string{
		JenisDokumenKTP,
		JenisDokumenKK,
		JenisDokumenSuratKeteranganUsaha,
	} {
		d, err := svc.Upload(context.Background(), pengajuanID, jenis, "berkas/"+jenis+".jpg")
		if err != nil {
			t.Fatalf("upload %s gagal: %v", jenis, err)
		}
		id[jenis] = d.ID
	}
	return id
}

// AC-03: ANL menolak KTP; AO mengunggah ulang HANYA KTP. Dokumen lain yang sudah
// terverifikasi tetap VERIFIED dan tidak perlu diunggah ulang.
func TestDokumen_AC03_ReuploadHanyaBerkasYangDitolak(t *testing.T) {
	svc, _, _ := dokumenServiceUji(t)
	ctx := context.Background()
	const pengajuanID = int64(1)

	id := unggahTiganya(t, svc, pengajuanID)

	// ANL memverifikasi KK dan SKU, menolak KTP.
	if _, err := svc.Verifikasi(ctx, id[JenisDokumenKK], 9, true, ""); err != nil {
		t.Fatalf("verifikasi KK gagal: %v", err)
	}
	if _, err := svc.Verifikasi(ctx, id[JenisDokumenSuratKeteranganUsaha], 9, true, ""); err != nil {
		t.Fatalf("verifikasi SKU gagal: %v", err)
	}
	ktp, err := svc.Verifikasi(ctx, id[JenisDokumenKTP], 9, false, "FOTO_BURAM")
	if err != nil {
		t.Fatalf("penolakan KTP gagal: %v", err)
	}
	if ktp.Status != StatusDokumenRejected {
		t.Fatalf("status KTP = %q, mau REJECTED", ktp.Status)
	}
	if ktp.AlasanPenolakan == nil || *ktp.AlasanPenolakan != "FOTO_BURAM" {
		t.Error("kode alasan penolakan tidak tersimpan")
	}

	// Belum lengkap: KTP masih ditolak.
	lengkap, err := svc.SemuaDokumenWajibVerified(ctx, pengajuanID)
	if err != nil {
		t.Fatalf("cek kelengkapan gagal: %v", err)
	}
	if lengkap {
		t.Error("dengan KTP REJECTED, dokumen wajib belum boleh dianggap lengkap")
	}

	// AO mengunggah ulang KTP saja.
	ktpBaru, err := svc.Upload(ctx, pengajuanID, JenisDokumenKTP, "berkas/KTP-v2.jpg")
	if err != nil {
		t.Fatalf("re-upload KTP seharusnya diizinkan, dapat: %v", err)
	}
	if ktpBaru.Status != StatusDokumenUploaded {
		t.Errorf("status KTP baru = %q, mau UPLOADED", ktpBaru.Status)
	}

	// KK dan SKU tidak tersentuh oleh re-upload KTP.
	daftar, err := svc.dok.DaftarPerPengajuan(ctx, pengajuanID)
	if err != nil {
		t.Fatalf("baca dokumen gagal: %v", err)
	}
	for _, d := range daftar {
		if d.ID == id[JenisDokumenKK] || d.ID == id[JenisDokumenSuratKeteranganUsaha] {
			if d.Status != StatusDokumenVerified {
				t.Errorf("dokumen %s ikut berubah menjadi %q; seharusnya tetap VERIFIED", d.JenisDokumen, d.Status)
			}
		}
	}

	// Setelah KTP baru diverifikasi, barulah lengkap.
	if _, err := svc.Verifikasi(ctx, ktpBaru.ID, 9, true, ""); err != nil {
		t.Fatalf("verifikasi KTP baru gagal: %v", err)
	}
	lengkap, err = svc.SemuaDokumenWajibVerified(ctx, pengajuanID)
	if err != nil {
		t.Fatalf("cek kelengkapan gagal: %v", err)
	}
	if !lengkap {
		t.Error("ketiga dokumen sudah VERIFIED, seharusnya dianggap lengkap")
	}
}

// Penolakan wajib berkode alasan; persetujuan tidak memerlukannya.
func TestDokumen_PenolakanWajibKodeAlasan(t *testing.T) {
	svc, _, _ := dokumenServiceUji(t)
	ctx := context.Background()
	id := unggahTiganya(t, svc, 1)

	if _, err := svc.Verifikasi(ctx, id[JenisDokumenKTP], 9, false, "   "); err == nil {
		t.Error("penolakan tanpa kode alasan seharusnya ditolak")
	}
	if _, err := svc.Verifikasi(ctx, id[JenisDokumenKTP], 9, true, ""); err != nil {
		t.Errorf("persetujuan tanpa kode alasan seharusnya diterima, dapat: %v", err)
	}
}

// Dokumen yang sudah VERIFIED tidak boleh ditimpa diam-diam.
func TestDokumen_YangSudahVerifiedTidakBisaDiunggahUlang(t *testing.T) {
	svc, _, _ := dokumenServiceUji(t)
	ctx := context.Background()
	id := unggahTiganya(t, svc, 1)

	if _, err := svc.Verifikasi(ctx, id[JenisDokumenKTP], 9, true, ""); err != nil {
		t.Fatalf("verifikasi gagal: %v", err)
	}

	_, err := svc.Upload(ctx, 1, JenisDokumenKTP, "berkas/KTP-v2.jpg")
	var br *domain.BusinessRuleError
	if !errors.As(err, &br) {
		t.Fatalf("re-upload dokumen VERIFIED seharusnya ditolak, dapat: %v", err)
	}
}

// AC-15: daftar dokumen wajib berupa data. Menambah satu jenis membuat pengajuan
// yang tadinya lengkap menjadi belum lengkap, tanpa mengubah kode.
func TestDokumen_DaftarWajibDibacaDariParameter(t *testing.T) {
	svc, _, wajib := dokumenServiceUji(t)
	ctx := context.Background()
	id := unggahTiganya(t, svc, 1)
	for _, dokID := range id {
		if _, err := svc.Verifikasi(ctx, dokID, 9, true, ""); err != nil {
			t.Fatalf("verifikasi gagal: %v", err)
		}
	}

	lengkap, err := svc.SemuaDokumenWajibVerified(ctx, 1)
	if err != nil || !lengkap {
		t.Fatalf("seharusnya lengkap; lengkap=%v err=%v", lengkap, err)
	}

	// ADM menambah satu jenis dokumen wajib lewat tabel parameter.
	wajib.jenis = append(wajib.jenis, "NPWP")

	lengkap, err = svc.SemuaDokumenWajibVerified(ctx, 1)
	if err != nil {
		t.Fatalf("cek kelengkapan gagal: %v", err)
	}
	if lengkap {
		t.Error("setelah NPWP menjadi wajib, pengajuan seharusnya belum lengkap")
	}
}

func TestDokumen_DaftarWajibKosongAdalahSalahKonfigurasi(t *testing.T) {
	svc, _, wajib := dokumenServiceUji(t)
	wajib.jenis = nil

	_, err := svc.SemuaDokumenWajibVerified(context.Background(), 1)

	var cfg *domain.ConfigError
	if !errors.As(err, &cfg) {
		t.Fatalf("daftar wajib kosong seharusnya ConfigError, dapat: %v", err)
	}
}
