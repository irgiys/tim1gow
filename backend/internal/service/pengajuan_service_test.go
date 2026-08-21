package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/irgiys/tim1gow/backend/internal/domain"
)

// waktuUji dipatok supaya tanggal pada nomor referensi dapat diperiksa persis.
var waktuUji = time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)

func pengajuanServiceUji(t *testing.T) (*PengajuanService, *fakePengajuanRepo, *fakeBatasPlafonRepo) {
	t.Helper()
	repo := newFakePengajuanRepo()
	batas := newFakeBatasPlafonRepo()
	svc := NewPengajuanService(repo, batas).DenganWaktu(func() time.Time { return waktuUji })
	return svc, repo, batas
}

func inputValid() InputPengajuan {
	return InputPengajuan{
		AOID:           7,
		Tipe:           TipeIndividu,
		NamaNasabah:    "Siti Aminah",
		NIK:            "3404010101900001",
		AlamatUsaha:    "Pasar Beringharjo Blok C",
		JenisUsaha:     "Warung kelontong",
		JenisAkad:      domain.AkadMurabahah,
		PlafonDiajukan: 30_000_000,
		TenorBulan:     12,
	}
}

// AC-01: AO membuat pengajuan Rp 30 juta akad murabahah; sistem menyimpannya
// sebagai DRAFT dan memberi nomor referensi berformat IMT-YYYYMMDD-NNNN.
func TestBuatPengajuan_AC01_NomorReferensiDanStatusDraft(t *testing.T) {
	svc, repo, _ := pengajuanServiceUji(t)

	p, err := svc.Buat(context.Background(), inputValid())
	if err != nil {
		t.Fatalf("pengajuan sah seharusnya diterima, dapat error: %v", err)
	}

	if want := "IMT-20260820-0001"; p.NomorReferensi != want {
		t.Errorf("nomor referensi = %q, mau %q", p.NomorReferensi, want)
	}
	if p.Status != string(domain.StatusDraft) {
		t.Errorf("status awal = %q, mau DRAFT", p.Status)
	}
	if p.ID == 0 {
		t.Error("pengajuan tersimpan seharusnya punya ID")
	}
	if repo.jumlahTransaksi != 1 {
		t.Errorf("pembuatan nomor referensi harus dalam 1 transaksi, dapat %d", repo.jumlahTransaksi)
	}
}

// BR-12: nomor referensi tidak pernah dipakai ulang, termasuk pada hari yang sama.
func TestBuatPengajuan_BR12_NomorReferensiTidakPernahDiulang(t *testing.T) {
	svc, _, _ := pengajuanServiceUji(t)
	ctx := context.Background()

	terlihat := map[string]bool{}
	for i := 0; i < 5; i++ {
		p, err := svc.Buat(ctx, inputValid())
		if err != nil {
			t.Fatalf("pengajuan ke-%d gagal: %v", i+1, err)
		}
		if terlihat[p.NomorReferensi] {
			t.Fatalf("nomor referensi %q dipakai ulang", p.NomorReferensi)
		}
		terlihat[p.NomorReferensi] = true
	}

	if want := "IMT-20260820-0005"; !terlihat[want] {
		t.Errorf("urutan harian tidak naik; %q tidak pernah muncul", want)
	}
}

// BR-01: plafon di luar batas ditolak — DAN plafon di dalam batas diterima.
// Kasus pembanding wajib ada (AGENTS.md Larangan 18): tanpa itu, fungsi yang
// menolak segalanya akan lolos seluruh test penolakan.
func TestBuatPengajuan_BR01_BatasPlafon(t *testing.T) {
	kasus := []struct {
		nama     string
		plafon   int64
		mauTolak bool
	}{
		{"tepat di batas bawah diterima", 5_000_000, false},
		{"satu rupiah di bawah batas bawah ditolak", 4_999_999, true},
		{"di tengah rentang diterima", 30_000_000, false},
		{"tepat di batas atas diterima", 500_000_000, false},
		{"satu rupiah di atas batas atas ditolak", 500_000_001, true},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			svc, _, _ := pengajuanServiceUji(t)
			in := inputValid()
			in.PlafonDiajukan = k.plafon

			_, err := svc.Buat(context.Background(), in)

			if !k.mauTolak {
				if err != nil {
					t.Fatalf("plafon %d seharusnya diterima, dapat error: %v", k.plafon, err)
				}
				return
			}

			var br *domain.BusinessRuleError
			if !errors.As(err, &br) {
				t.Fatalf("plafon %d seharusnya ditolak dengan BusinessRuleError, dapat: %v", k.plafon, err)
			}
			if br.Rule != "BR-01" {
				t.Errorf("kode aturan = %q, mau BR-01", br.Rule)
			}
		})
	}
}

// AC-15: batas plafon berupa data. Mengubah baris parameter harus mengubah
// hasil, tanpa menyentuh kode service.
func TestBuatPengajuan_BR01_BatasDibacaDariParameter(t *testing.T) {
	svc, _, batas := pengajuanServiceUji(t)
	ctx := context.Background()

	in := inputValid()
	in.PlafonDiajukan = 4_000_000

	if _, err := svc.Buat(ctx, in); err == nil {
		t.Fatal("dengan batas bawah 5 juta, plafon 4 juta seharusnya ditolak")
	}

	// ADM menurunkan batas bawah lewat tabel parameter.
	batas.minimum = 1_000_000

	if _, err := svc.Buat(ctx, in); err != nil {
		t.Fatalf("setelah batas bawah diturunkan, plafon 4 juta seharusnya diterima, dapat: %v", err)
	}
}

// Batas plafon belum di-seed adalah salah konfigurasi, bukan izin melanjutkan
// diam-diam dengan nilai default (AGENTS.md Larangan 3 & 15).
func TestBuatPengajuan_BatasBelumDiaturDitolak(t *testing.T) {
	svc, _, batas := pengajuanServiceUji(t)
	batas.ditemukan = false

	_, err := svc.Buat(context.Background(), inputValid())

	var cfg *domain.ConfigError
	if !errors.As(err, &cfg) {
		t.Fatalf("batas belum diatur seharusnya ConfigError, dapat: %v", err)
	}
}

// BR-11: pesan error tidak boleh memuat NIK.
func TestBuatPengajuan_BR11_PesanErrorTidakMemuatNIK(t *testing.T) {
	svc, _, _ := pengajuanServiceUji(t)
	in := inputValid()
	in.PlafonDiajukan = 1_000

	_, err := svc.Buat(context.Background(), in)
	if err == nil {
		t.Fatal("plafon di bawah batas seharusnya ditolak")
	}
	if pesanMemuat(err.Error(), in.NIK) {
		t.Errorf("pesan error membocorkan NIK: %q", err.Error())
	}
}

// Pengajuan kelompok dinilai dari TOTAL plafon anggota, bukan per anggota.
func TestBuatPengajuan_KelompokDinilaiDariTotalPlafon(t *testing.T) {
	svc, repo, _ := pengajuanServiceUji(t)

	in := inputValid()
	in.Tipe = TipeKelompok
	in.PlafonDiajukan = 0
	in.Anggota = []PengajuanAnggota{
		{NamaAnggota: "Anggota A", NIKAnggota: "3404010101900002", PlafonAnggota: 10_000_000},
		{NamaAnggota: "Anggota B", NIKAnggota: "3404010101900003", PlafonAnggota: 15_000_000},
	}

	p, err := svc.Buat(context.Background(), in)
	if err != nil {
		t.Fatalf("kelompok dengan total 25 juta seharusnya diterima, dapat: %v", err)
	}
	if p.PlafonDiajukan != 25_000_000 {
		t.Errorf("total plafon kelompok = %d, mau 25000000", p.PlafonDiajukan)
	}

	anggota, err := repo.DaftarAnggota(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("gagal membaca anggota: %v", err)
	}
	if len(anggota) != 2 {
		t.Errorf("jumlah anggota tersimpan = %d, mau 2", len(anggota))
	}
}

func TestBuatPengajuan_ValidasiInputWajib(t *testing.T) {
	kasus := []struct {
		nama  string
		ubah  func(*InputPengajuan)
		tolak bool
	}{
		{"input lengkap diterima", func(*InputPengajuan) {}, false},
		{"nama nasabah kosong ditolak", func(in *InputPengajuan) { in.NamaNasabah = "  " }, true},
		{"NIK kosong ditolak", func(in *InputPengajuan) { in.NIK = "" }, true},
		{"jenis usaha kosong ditolak", func(in *InputPengajuan) { in.JenisUsaha = "" }, true},
		{"akad tidak dikenal ditolak", func(in *InputPengajuan) { in.JenisAkad = "IJARAH" }, true},
		{"tenor nol ditolak", func(in *InputPengajuan) { in.TenorBulan = 0 }, true},
		{"akad musyarakah diterima", func(in *InputPengajuan) { in.JenisAkad = domain.AkadMusyarakah }, false},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			svc, _, _ := pengajuanServiceUji(t)
			in := inputValid()
			k.ubah(&in)

			_, err := svc.Buat(context.Background(), in)

			if k.tolak && err == nil {
				t.Fatal("seharusnya ditolak, tetapi diterima")
			}
			if !k.tolak && err != nil {
				t.Fatalf("seharusnya diterima, dapat error: %v", err)
			}
		})
	}
}

func TestFormatNomorReferensi(t *testing.T) {
	got := FormatNomorReferensi(time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC), 7)
	if want := "IMT-20260820-0007"; got != want {
		t.Errorf("FormatNomorReferensi = %q, mau %q", got, want)
	}
}

// pesanMemuat memeriksa keberadaan potongan teks tanpa membocorkannya ke output.
func pesanMemuat(pesan, potongan string) bool {
	if potongan == "" {
		return false
	}
	return len(pesan) >= len(potongan) && contains(pesan, potongan)
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
