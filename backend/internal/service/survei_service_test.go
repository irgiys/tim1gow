package service

import (
	"context"
	"testing"
	"time"
)

func surveiServiceUji(t *testing.T) (*SurveiService, *fakeSurveiRepo) {
	t.Helper()
	repo := newFakeSurveiRepo()
	svc := NewSurveiService(repo).DenganWaktu(func() time.Time { return waktuUji })
	return svc, repo
}

func inputSurveiValid() InputSurvei {
	return InputSurvei{
		PengajuanID:    1,
		AOID:           7,
		Latitude:       -7.801389,
		Longitude:      110.364444,
		FotoURL:        "berkas/survei-1.jpg",
		OmzetHarian:    500_000,
		LamaUsahaBulan: 24,
		CatatanKondisi: "Warung ramai, stok terisi, lokasi di tepi jalan pasar.",
	}
}

// FR-04: survei lengkap tersimpan sebagai VALID dan terbaca oleh guard BR-03.
func TestSurvei_RekamLengkapTersimpanValid(t *testing.T) {
	svc, _ := surveiServiceUji(t)
	ctx := context.Background()

	sv, err := svc.Rekam(ctx, inputSurveiValid())
	if err != nil {
		t.Fatalf("survei lengkap seharusnya diterima, dapat: %v", err)
	}
	if sv.Status != StatusSurveiValid {
		t.Errorf("status = %q, mau VALID", sv.Status)
	}
	if sv.ID == 0 {
		t.Error("survei tersimpan seharusnya punya ID")
	}

	ada, err := svc.AdaSurveiValid(ctx, 1)
	if err != nil {
		t.Fatalf("cek survei valid gagal: %v", err)
	}
	if !ada {
		t.Error("setelah survei VALID direkam, AdaSurveiValid seharusnya true")
	}
}

// Masukan BR-03: tanpa survei VALID, guard skoring harus melihat false.
// Survei TIDAK_VALID tidak boleh dihitung sebagai memenuhi syarat.
func TestSurvei_BR03_HanyaSurveiValidYangDihitung(t *testing.T) {
	svc, _ := surveiServiceUji(t)
	ctx := context.Background()

	ada, err := svc.AdaSurveiValid(ctx, 1)
	if err != nil {
		t.Fatalf("cek gagal: %v", err)
	}
	if ada {
		t.Error("tanpa survei sama sekali, AdaSurveiValid seharusnya false")
	}

	in := inputSurveiValid()
	in.Status = StatusSurveiTidakValid
	if _, err := svc.Rekam(ctx, in); err != nil {
		t.Fatalf("perekaman survei TIDAK_VALID gagal: %v", err)
	}

	ada, err = svc.AdaSurveiValid(ctx, 1)
	if err != nil {
		t.Fatalf("cek gagal: %v", err)
	}
	if ada {
		t.Error("survei TIDAK_VALID tidak boleh memenuhi syarat BR-03")
	}

	// Kasus pembanding: setelah ada survei VALID, syarat terpenuhi.
	if _, err := svc.Rekam(ctx, inputSurveiValid()); err != nil {
		t.Fatalf("perekaman survei VALID gagal: %v", err)
	}
	ada, err = svc.AdaSurveiValid(ctx, 1)
	if err != nil {
		t.Fatalf("cek gagal: %v", err)
	}
	if !ada {
		t.Error("dengan satu survei VALID, syarat BR-03 seharusnya terpenuhi")
	}
}

// FR-04 mewajibkan koordinat, foto, omzet, lama usaha, dan catatan kondisi.
func TestSurvei_ValidasiKelengkapan(t *testing.T) {
	kasus := []struct {
		nama  string
		ubah  func(*InputSurvei)
		tolak bool
	}{
		{"survei lengkap diterima", func(*InputSurvei) {}, false},
		{"tanpa koordinat ditolak", func(in *InputSurvei) { in.Latitude, in.Longitude = 0, 0 }, true},
		{"tanpa foto ditolak", func(in *InputSurvei) { in.FotoURL = "" }, true},
		{"omzet nol ditolak", func(in *InputSurvei) { in.OmzetHarian = 0 }, true},
		{"omzet negatif ditolak", func(in *InputSurvei) { in.OmzetHarian = -1 }, true},
		{"lama usaha nol ditolak", func(in *InputSurvei) { in.LamaUsahaBulan = 0 }, true},
		{"tanpa catatan kondisi ditolak", func(in *InputSurvei) { in.CatatanKondisi = "  " }, true},
		{"latitude di luar rentang ditolak", func(in *InputSurvei) { in.Latitude = 95 }, true},
		{"longitude di luar rentang ditolak", func(in *InputSurvei) { in.Longitude = 181 }, true},
		{"usaha baru satu bulan diterima", func(in *InputSurvei) { in.LamaUsahaBulan = 1 }, false},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			svc, _ := surveiServiceUji(t)
			in := inputSurveiValid()
			k.ubah(&in)

			_, err := svc.Rekam(context.Background(), in)

			if k.tolak && err == nil {
				t.Fatal("seharusnya ditolak, tetapi diterima")
			}
			if !k.tolak && err != nil {
				t.Fatalf("seharusnya diterima, dapat error: %v", err)
			}
		})
	}
}

// BR-11: path foto tidak boleh bocor lewat pesan error.
func TestSurvei_BR11_PesanErrorTidakMemuatPathFoto(t *testing.T) {
	svc, _ := surveiServiceUji(t)
	in := inputSurveiValid()
	in.OmzetHarian = 0

	_, err := svc.Rekam(context.Background(), in)
	if err == nil {
		t.Fatal("omzet nol seharusnya ditolak")
	}
	if pesanMemuat(err.Error(), in.FotoURL) {
		t.Errorf("pesan error membocorkan path foto: %q", err.Error())
	}
}
