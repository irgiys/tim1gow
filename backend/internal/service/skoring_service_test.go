package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/irgiys/tim1gow/backend/internal/domain"
)

// dataNasabahBaik adalah masukan skoring yang menghasilkan skor tinggi:
// angsuran ringan, usaha lama, survei baik. Kolektibilitas diisi pemanggil.
func dataNasabahBaik() domain.DataSkoring {
	return domain.DataSkoring{
		PengajuanID:     1,
		AngsuranBulanan: 1_000_000,
		OmzetHarian:     500_000, // kapasitas = 500rb x 25 x 0,30 = 3,75jt -> rasio 0,267
		LamaUsahaBulan:  48,
		NilaiSurvei:     5,
	}
}

// AC-07: skoring menampilkan rincian keempat komponen beserta bobot dan skornya.
// BR-08 mensyaratkan rincian ini DISIMPAN, bukan hanya angka akhirnya.
func TestHitung_AC07_RincianKeempatKomponenTersedia(t *testing.T) {
	repo := newFakeParameterRepo()
	svc := NewSkoringService(repo)

	d := dataNasabahBaik()
	d.Kolektibilitas = 1

	hasil, err := svc.Hitung(d)
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}

	if len(hasil.Rincian) != 4 {
		t.Fatalf("rincian komponen = %d, ingin 4", len(hasil.Rincian))
	}

	terlihat := map[string]bool{}
	for _, r := range hasil.Rincian {
		terlihat[r.Kode] = true
		if r.Nama == "" {
			t.Errorf("komponen %s: nama kosong, ANL tidak bisa membacanya", r.Kode)
		}
		if r.Bobot <= 0 {
			t.Errorf("komponen %s: bobot %v tidak masuk akal", r.Kode, r.Bobot)
		}
		if r.SkorMentah < 0 || r.SkorMentah > 100 {
			t.Errorf("komponen %s: skor mentah %v di luar 0-100", r.Kode, r.SkorMentah)
		}
		// Kontribusi wajib konsisten dengan skor x bobot, karena inilah angka
		// yang dibaca ANL untuk memeriksa perhitungan sistem.
		if ingin := r.SkorMentah * r.Bobot; !hampirSama(r.Kontribusi, ingin) {
			t.Errorf("komponen %s: kontribusi %v, ingin %v", r.Kode, r.Kontribusi, ingin)
		}
	}
	for _, kode := range []string{
		domain.KomponenKapasitasBayar, domain.KomponenRiwayatSlik,
		domain.KomponenLamaUsaha, domain.KomponenSurveiLapangan,
	} {
		if !terlihat[kode] {
			t.Errorf("komponen %s tidak ada di rincian", kode)
		}
	}
}

// AC-06: nasabah dengan kolektibilitas 2 dapat lanjut, tetapi grade risikonya
// TIDAK PERNAH lebih baik dari 3 (Tabel 4.2).
func TestHitung_AC06_Kolektibilitas2GradeMinimal3(t *testing.T) {
	repo := newFakeParameterRepo()
	svc := NewSkoringService(repo)

	d := dataNasabahBaik() // data ini sendiri menghasilkan grade 1-2
	d.Kolektibilitas = 2

	hasil, err := svc.Hitung(d)
	if err != nil {
		t.Fatalf("kolektibilitas 2 harus dapat lanjut, tetapi error: %v", err)
	}
	if hasil.Grade < 3 {
		t.Fatalf("grade = %d, ingin >= 3 untuk kolektibilitas 2", hasil.Grade)
	}
	if !hasil.GradeMinimalDipaksa {
		t.Error("GradeMinimalDipaksa = false; alasan grade dinaikkan tidak terlacak (BR-10)")
	}

	// Pembanding: data yang sama dengan kolektibilitas 1 harus lebih baik,
	// membuktikan grade 3 di atas benar-benar akibat aturan kolektibilitas.
	d.Kolektibilitas = 1
	baik, err := svc.Hitung(d)
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if baik.Grade >= 3 {
		t.Fatalf("data pembanding menghasilkan grade %d; kasus uji tidak membuktikan apa pun", baik.Grade)
	}
}

// AC-15: ADM mengubah bobot komponen "Lama usaha" dari 20 ke 25; skoring
// berikutnya memakai bobot baru TANPA restart aplikasi.
//
// Test ini yang membuktikan bobot dibaca dari data, bukan dari konstanta:
// baris parameter diubah di tengah test, pada instance service yang SAMA.
func TestHitung_AC15_UbahBobotLangsungBerlaku(t *testing.T) {
	repo := newFakeParameterRepo()
	svc := NewSkoringService(repo)

	// Data dengan lama usaha rendah supaya komponen itu bernilai kecil;
	// menaikkan bobotnya harus MENURUNKAN skor akhir.
	d := dataNasabahBaik()
	d.Kolektibilitas = 1
	d.LamaUsahaBulan = 6 // skor komponen lama usaha = 0

	sebelum, err := svc.Hitung(d)
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}

	repo.ubahBobot(domain.KomponenLamaUsaha, 25) // ADM lewat FR-13

	sesudah, err := svc.Hitung(d)
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}

	if sesudah.SkorAkhir == sebelum.SkorAkhir {
		t.Fatalf("skor tidak berubah (%d) setelah bobot diubah 20->25; "+
			"bobot kemungkinan di-hardcode atau di-cache", sebelum.SkorAkhir)
	}
	if sesudah.SkorAkhir >= sebelum.SkorAkhir {
		t.Errorf("skor %d -> %d; menaikkan bobot komponen bernilai 0 seharusnya menurunkan skor",
			sebelum.SkorAkhir, sesudah.SkorAkhir)
	}
	if sesudah.TotalBobot != 105 {
		t.Errorf("total bobot = %v, ingin 105 (35+25+25+20)", sesudah.TotalBobot)
	}
	if repo.jumlahBacaKomponen < 2 {
		t.Errorf("tabel parameter dibaca %d kali untuk 2 perhitungan; service meng-cache parameter",
			repo.jumlahBacaKomponen)
	}
}

// BR-07: skor akhir = SUM(skor x bobot) / SUM(bobot), dibulatkan ke bilangan
// bulat terdekat. Dihitung ulang di sini dari rincian yang dikembalikan service.
func TestHitung_BR07_RumusSkorAkhir(t *testing.T) {
	repo := newFakeParameterRepo()
	svc := NewSkoringService(repo)

	d := dataNasabahBaik()
	d.Kolektibilitas = 1

	hasil, err := svc.Hitung(d)
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}

	var totalKontribusi, totalBobot float64
	for _, r := range hasil.Rincian {
		totalKontribusi += r.Kontribusi
		totalBobot += r.Bobot
	}
	ingin := int(totalKontribusi/totalBobot + 0.5)
	if hasil.SkorAkhir != ingin {
		t.Errorf("skor akhir = %d, ingin %d (dari rincian yang ditampilkan ke ANL)",
			hasil.SkorAkhir, ingin)
	}
}

// BR-03: skoring tidak boleh jalan sebelum dokumen VERIFIED, ada survei VALID,
// dan SLIK check sudah dijalankan. AC-04 mensyaratkan pesannya menyebut BR-03.
func TestPastikanBolehSkoring_BR03(t *testing.T) {
	svc := NewSkoringService(newFakeParameterRepo())

	kasus := []struct {
		nama    string
		pra     PrasyaratSkoring
		inginOK bool
	}{
		{"semua prasyarat terpenuhi", PrasyaratSkoring{true, true, true}, true},
		{"dokumen belum verified", PrasyaratSkoring{false, true, true}, false},
		{"belum ada survei valid", PrasyaratSkoring{true, false, true}, false},
		{"slik belum dijalankan", PrasyaratSkoring{true, true, false}, false},
		{"tidak ada prasyarat sama sekali", PrasyaratSkoring{}, false},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			err := svc.PastikanBolehSkoring(k.pra)
			if k.inginOK {
				if err != nil {
					t.Fatalf("ingin lolos, dapat error: %v", err)
				}
				return
			}
			var brErr *domain.BusinessRuleError
			if !errors.As(err, &brErr) {
				t.Fatalf("ingin BusinessRuleError, dapat %v", err)
			}
			if brErr.Rule != "BR-03" {
				t.Errorf("rule = %q, ingin BR-03", brErr.Rule)
			}
		})
	}
}

// AC-04 berbunyi: pengajuan tanpa survei valid ditolak saat mencoba masuk
// skoring, "dengan pesan yang menyebut BR-03".
//
// Yang dinilai adalah pesan yang DILIHAT PENGGUNA, bukan field internal. Test
// BR-03 di atas hanya memeriksa `brErr.Rule`, sehingga pesan yang tidak menyebut
// BR-03 tetap lolos — padahal itu persis yang diminta AC-04.
func TestPastikanBolehSkoring_AC04_PesanMenyebutBR03(t *testing.T) {
	svc := NewSkoringService(newFakeParameterRepo())

	// Kasus AC-04: dokumen & SLIK beres, yang kurang hanya survei valid.
	err := svc.PastikanBolehSkoring(PrasyaratSkoring{
		SemuaDokumenVerified: true,
		AdaSurveiValid:       false,
		SlikSudahDijalankan:  true,
	})
	if err == nil {
		t.Fatal("tanpa survei valid: ingin ditolak, dapat nil")
	}

	// Pesan yang dirakit untuk pengguna wajib memuat kode BR-nya. Diperiksa
	// lewat Error() karena itulah yang berakhir di respons API dan di layar ANL.
	if !strings.Contains(err.Error(), "BR-03") {
		t.Errorf("pesan %q tidak menyebut BR-03 (AC-04)", err.Error())
	}

	// Sekaligus pastikan sebabnya ikut disebut, supaya ANL tahu apa yang kurang.
	if !strings.Contains(err.Error(), "survei") {
		t.Errorf("pesan %q tidak menjelaskan bahwa survei yang kurang", err.Error())
	}

	// Pembanding: prasyarat lengkap tidak boleh menghasilkan pesan apa pun.
	if err := svc.PastikanBolehSkoring(PrasyaratSkoring{true, true, true}); err != nil {
		t.Errorf("prasyarat lengkap seharusnya lolos: %v", err)
	}
}

// BR-05: grade 5 tidak dapat diajukan ke approval.
func TestPastikanBolehKeApproval_BR05(t *testing.T) {
	svc := NewSkoringService(newFakeParameterRepo())

	if err := svc.PastikanBolehKeApproval(4); err != nil {
		t.Errorf("grade 4 seharusnya boleh, dapat error: %v", err)
	}

	err := svc.PastikanBolehKeApproval(5)
	var brErr *domain.BusinessRuleError
	if !errors.As(err, &brErr) {
		t.Fatalf("grade 5: ingin BusinessRuleError, dapat %v", err)
	}
	if brErr.Rule != "BR-05" {
		t.Errorf("rule = %q, ingin BR-05", brErr.Rule)
	}
}

// Batas rentang skor->grade diuji tepat di tepinya (Tabel 4.3), karena inilah
// tempat kesalahan off-by-one paling mungkin lolos tanpa terlihat.
func TestGradeDariSkor_BatasRentang(t *testing.T) {
	svc := NewSkoringService(newFakeParameterRepo())

	kasus := []struct{ skor, grade int }{
		{100, 1}, {85, 1},
		{84, 2}, {70, 2},
		{69, 3}, {55, 3},
		{54, 4}, {40, 4},
		{39, 5}, {0, 5},
	}
	for _, k := range kasus {
		got, err := svc.GradeDariSkor(k.skor)
		if err != nil {
			t.Fatalf("skor %d: error %v", k.skor, err)
		}
		if got != k.grade {
			t.Errorf("skor %d -> grade %d, ingin %d", k.skor, got, k.grade)
		}
	}
}

// Parameter yang belum diatur harus MENGHENTIKAN perhitungan, bukan memakai
// nilai default diam-diam (AGENTS.md Larangan 3 & 15).
func TestHitung_ParameterKosongTidakMemakaiDefault(t *testing.T) {
	repo := newFakeParameterRepo()
	repo.komponen = nil
	svc := NewSkoringService(repo)

	d := dataNasabahBaik()
	d.Kolektibilitas = 1

	if _, err := svc.Hitung(d); err == nil {
		t.Fatal("parameter_skoring kosong: ingin error, dapat nil")
	}

	// Kolektibilitas yang skornya belum diatur juga harus berhenti — kegagalan
	// membaca riwayat SLIK tidak boleh dianggap SLIK bersih.
	repo2 := newFakeParameterRepo()
	svc2 := NewSkoringService(repo2)
	d2 := dataNasabahBaik()
	d2.Kolektibilitas = 3 // tidak ada di tabel; kol 3-5 seharusnya sudah ditolak lebih awal
	if _, err := svc2.Hitung(d2); err == nil {
		t.Fatal("skor kolektibilitas 3 belum diatur: ingin error, dapat nil")
	}
}

func hampirSama(a, b float64) bool {
	const eps = 1e-9
	d := a - b
	return d < eps && d > -eps
}
