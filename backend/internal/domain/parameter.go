package domain

// Tipe di berkas ini mewakili BARIS TABEL PARAMETER di database
// (parameter_skoring, ambang_approval, rentang_margin). Nilainya TIDAK PERNAH
// ditulis sebagai konstanta di kode — ADM dapat mengubahnya tanpa deploy ulang
// (FR-13, AC-15, AGENTS.md Larangan 3).

// ParameterKomponenSkor adalah satu baris tabel parameter_skoring.
//
// Ambang batas perhitungan ikut disimpan sebagai data, bukan hardcode: untuk
// KAPASITAS_BAYAR, Batas1/Batas2 adalah rasio angsuran (mis. 0.30 dan 0.60);
// untuk LAMA_USAHA, keduanya adalah jumlah bulan (mis. 6 dan 36).
type ParameterKomponenSkor struct {
	Kode   string
	Nama   string
	Bobot  float64
	Batas1 float64 // batas skor penuh
	Batas2 float64 // batas skor nol
	Aktif  bool
}

// ParameterRiwayatSlik adalah pemetaan kolektibilitas ke skor komponen SLIK.
// Kol-1 dan Kol-2 disimpan sebagai data karena bisa diubah ADM.
type ParameterRiwayatSlik struct {
	Kolektibilitas int
	Skor           float64
}

// RentangMargin adalah satu baris tabel rentang_margin: rentang skor per grade
// beserta rentang margin murabahah dan nisbah bank musyarakah yang disetujui.
type RentangMargin struct {
	Grade         int
	SkorMin       int
	SkorMaks      int
	MarginMin     float64
	MarginMaks    float64
	NisbahMin     float64
	NisbahMaks    float64
	DapatDibiayai bool
}

// AmbangApproval adalah satu baris tabel ambang_approval: level approval yang
// diperlukan untuk suatu rentang total plafon.
type AmbangApproval struct {
	PlafonMin  int64
	PlafonMaks int64
	Level      []Peran
}

// HasilSlik adalah hasil SLIK check yang tersimpan, dipakai sebagai masukan
// skoring. Sengaja TIDAK memuat NIK supaya data pribadi tidak ikut mengalir ke
// lapisan perhitungan dan log (BR-11).
type HasilSlik struct {
	PengajuanID    int64
	Kolektibilitas int
	DiperiksaPada  string
}

// DataSkoring adalah masukan perhitungan skor kelayakan. Semua nilai berasal
// dari data pengajuan, survei, dan hasil SLIK yang sudah tersimpan.
type DataSkoring struct {
	PengajuanID int64

	// Kapasitas bayar
	AngsuranBulanan float64
	OmzetHarian     float64

	// Lama usaha berjalan, dalam bulan
	LamaUsahaBulan int

	// Riwayat SLIK
	Kolektibilitas int

	// Penilaian ANL atas kondisi usaha, skala 1-5
	NilaiSurvei int
}

// RincianKomponenSkor adalah kontribusi satu komponen terhadap skor akhir.
// WAJIB ditampilkan ke ANL dan DISIMPAN bersama hasil skoring (BR-08) —
// bukan hanya angka akhirnya.
type RincianKomponenSkor struct {
	Kode       string
	Nama       string
	SkorMentah float64 // 0-100, sebelum dibobot
	Bobot      float64
	Kontribusi float64 // SkorMentah * Bobot
}

// HasilSkoring adalah keluaran perhitungan skor kelayakan.
type HasilSkoring struct {
	PengajuanID int64
	SkorAkhir   int
	Grade       int
	Rincian     []RincianKomponenSkor
	TotalBobot  float64

	// GradeMinimalDipaksa terisi ketika kolektibilitas 2 memaksa grade risiko
	// minimal 3 (Tabel 4.2). Disimpan supaya alasannya terlacak (BR-10).
	GradeMinimalDipaksa bool
}
