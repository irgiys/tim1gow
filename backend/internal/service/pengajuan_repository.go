// Berkas ini memuat entitas persistensi dan kontrak akses data untuk
// pengajuan, dokumen, dan survei (FR-02, FR-03, FR-04).
//
// Kontrak repository hidup di paket service mengikuti pola
// service/parameter_repository.go, sehingga paket repository yang
// mengimplementasikannya boleh meng-import service tanpa import cycle.
// Bentuk kolom mengikuti docs/SDD-iMitra.md BAB 4.1. Tidak ada aturan
// bisnis di sini (AGENTS.md Larangan 17) — hanya bentuk data.
package service

import (
	"context"
	"errors"
	"time"
)

// StatusDokumen adalah status verifikasi satu berkas dokumen (FR-03).
//
// Nilai UPLOADED mengikuti docs/SDD-iMitra.md BAB 4.1 kolom `dokumen.status`.
// AGENTS.md bagian 4.1 menyebut PENDING untuk keadaan yang sama; selisih ini
// sudah dilaporkan ke Tech Lead. SDD dipakai lebih dulu karena berkas inilah
// yang mengimplementasikan tabelnya.
type StatusDokumen string

const (
	StatusDokumenUploaded StatusDokumen = "UPLOADED"
	StatusDokumenVerified StatusDokumen = "VERIFIED"
	StatusDokumenRejected StatusDokumen = "REJECTED"
)

// StatusSurvei adalah status hasil survei lapangan (FR-04).
type StatusSurvei string

const (
	StatusSurveiValid      StatusSurvei = "VALID"
	StatusSurveiTidakValid StatusSurvei = "TIDAK_VALID"
)

// TipePengajuan membedakan pengajuan perorangan dan kelompok/majelis.
type TipePengajuan string

const (
	TipeIndividu TipePengajuan = "INDIVIDU"
	TipeKelompok TipePengajuan = "KELOMPOK"
)

// Jenis dokumen wajib untuk pengajuan mikro (FR-03). Konstanta ini adalah KODE
// jenis berkas, bukan ambang atau bobot aturan bisnis.
const (
	JenisDokumenKTP                  = "KTP"
	JenisDokumenKK                   = "KK"
	JenisDokumenSuratKeteranganUsaha = "SURAT_KETERANGAN_USAHA"
)

// Pengajuan adalah satu baris tabel `pengajuan`.
//
// NIK disimpan karena dibutuhkan SLIK check, tetapi TIDAK BOLEH ikut ke log,
// pesan error, atau URL (BR-11). Pemakai memakai ID atau NomorReferensi.
type Pengajuan struct {
	ID               int64
	NomorReferensi   string
	Tipe             TipePengajuan
	AOID             int64
	NamaNasabah      string
	NIK              string
	AlamatUsaha      string
	JenisUsaha       string
	JenisAkad        string
	PlafonDiajukan   int64
	PlafonDisetujui  *int64
	TenorBulan       int
	MarginAtauNisbah *float64
	Status           string
	DibuatPada       time.Time
	DiperbaruiPada   time.Time
}

// PengajuanAnggota adalah satu baris tabel `pengajuan_anggota` (pembiayaan
// kelompok). Total plafon kelompok dihitung ulang dari baris-baris ini.
type PengajuanAnggota struct {
	ID            int64
	PengajuanID   int64
	NamaAnggota   string
	NIKAnggota    string
	PlafonAnggota int64
	StatusAnggota string
}

// Dokumen adalah satu baris tabel `dokumen` (FR-03).
type Dokumen struct {
	ID               int64
	PengajuanID      int64
	JenisDokumen     string
	URLBerkas        string
	Status           StatusDokumen
	AlasanPenolakan  *string
	DiverifikasiOleh *int64
	DiverifikasiPada *time.Time
	DibuatPada       time.Time
}

// Survei adalah satu baris tabel `survei` (FR-04).
type Survei struct {
	ID             int64
	PengajuanID    int64
	AOID           int64
	Latitude       float64
	Longitude      float64
	FotoURL        string
	OmzetHarian    int64
	LamaUsahaBulan int
	CatatanKondisi string
	Status         StatusSurvei
	DibuatPada     time.Time
}

// ErrTidakDitemukan dikembalikan repository ketika baris yang diminta tidak ada.
// Service memetakannya ke error domain yang sesuai; repository tidak menentukan
// kode HTTP maupun kode BR.
var ErrTidakDitemukan = errors.New("baris tidak ditemukan")

// PengajuanRepository adalah satu-satunya jalan service menyentuh tabel
// `pengajuan` dan `pengajuan_anggota` (FR-02).
type PengajuanRepository interface {
	// Simpan menyisipkan pengajuan baru dan mengisi ID beserta stempel waktunya.
	Simpan(ctx context.Context, p *Pengajuan) error

	// Perbarui menyimpan perubahan pada pengajuan yang sudah ada.
	Perbarui(ctx context.Context, p *Pengajuan) error

	// CariID mengembalikan satu pengajuan. ErrTidakDitemukan bila tidak ada.
	CariID(ctx context.Context, id int64) (Pengajuan, error)

	// DaftarMilikAO mengembalikan pengajuan milik seorang AO, terbaru lebih dulu.
	// Dipakai FR-02 "daftar pengajuan milik AO".
	DaftarMilikAO(ctx context.Context, aoID int64) ([]Pengajuan, error)

	// DaftarSemua mengembalikan seluruh pengajuan, terbaru lebih dulu.
	// Dipakai oleh peran non-AO (ANL, KCP, KC, KOM, ADM).
	DaftarSemua(ctx context.Context) ([]Pengajuan, error)

	// AmbilNomorUrutHarian menaikkan dan mengembalikan urutan berikutnya untuk
	// suatu tanggal, di dalam transaksi pemanggil, memakai penguncian baris
	// (SELECT ... FOR UPDATE) sehingga dua permintaan bersamaan tidak pernah
	// memperoleh angka yang sama (BR-12, SDD BAB 4.1).
	AmbilNomorUrutHarian(ctx context.Context, tanggal time.Time) (int, error)

	// SimpanAnggota menyimpan daftar anggota untuk pengajuan kelompok.
	SimpanAnggota(ctx context.Context, pengajuanID int64, anggota []PengajuanAnggota) error

	// DaftarAnggota mengembalikan anggota sebuah pengajuan kelompok.
	DaftarAnggota(ctx context.Context, pengajuanID int64) ([]PengajuanAnggota, error)

	// DalamTransaksi menjalankan fn di dalam satu transaksi database. Repository
	// yang diterima fn terikat pada transaksi tersebut. Pembuatan nomor referensi
	// wajib memakai ini supaya penguncian counter dan penyimpanan pengajuan
	// berada dalam satu transaksi (BR-12).
	DalamTransaksi(ctx context.Context, fn func(tx PengajuanRepository) error) error
}

// DokumenRepository adalah akses data tabel `dokumen` (FR-03).
type DokumenRepository interface {
	// Simpan menyisipkan satu dokumen baru.
	Simpan(ctx context.Context, d *Dokumen) error

	// Perbarui menyimpan perubahan status/verifikasi satu dokumen.
	Perbarui(ctx context.Context, d *Dokumen) error

	// CariID mengembalikan satu dokumen. ErrTidakDitemukan bila tidak ada.
	CariID(ctx context.Context, id int64) (Dokumen, error)

	// DaftarPerPengajuan mengembalikan seluruh dokumen sebuah pengajuan.
	DaftarPerPengajuan(ctx context.Context, pengajuanID int64) ([]Dokumen, error)

	// CariAktif mengembalikan dokumen berjenis tertentu yang masih berlaku pada
	// sebuah pengajuan. Dipakai saat re-upload agar hanya berkas yang ditolak
	// yang tergantikan, sementara dokumen lain tidak tersentuh (AC-03).
	CariAktif(ctx context.Context, pengajuanID int64, jenis string) (Dokumen, error)
}

// SurveiRepository adalah akses data tabel `survei` (FR-04).
type SurveiRepository interface {
	// Simpan menyisipkan satu hasil survei lapangan.
	Simpan(ctx context.Context, s *Survei) error

	// DaftarPerPengajuan mengembalikan seluruh survei sebuah pengajuan.
	DaftarPerPengajuan(ctx context.Context, pengajuanID int64) ([]Survei, error)

	// AdaSurveiValid melaporkan apakah pengajuan sudah punya minimal satu survei
	// berstatus VALID. Dipakai service sebagai masukan guard BR-03; keputusan
	// menolak/melanjutkan tetap diambil di service, bukan di sini.
	AdaSurveiValid(ctx context.Context, pengajuanID int64) (bool, error)
}
