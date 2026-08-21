// Berkas ini menutup satu lubang otorisasi aturan bisnis: prasyarat BR-03
// dulu dikirim klien lewat badan request (semuaDokumenVerified, adaSurveiValid,
// slikSudahDijalankan). Siapa pun yang memegang token ANL bisa mengirim `true`
// dan skoring tetap jalan walau dokumennya belum diverifikasi.
//
// Sekarang keadaan itu DIBACA DARI DATABASE. Klien tidak lagi punya suara soal
// apakah BR-03 terpenuhi.
package service

import (
	"context"

	"github.com/irgiys/tim1gow/backend/internal/domain"
)

// KeadaanPrasyarat adalah keadaan nyata sebuah pengajuan yang menentukan
// boleh/tidaknya skoring dijalankan (BR-03), apa adanya dari database.
//
// Kolektibilitas ikut di sini, bukan dari klien: nilai itu menentukan skor
// komponen Riwayat SLIK dan memaksa grade minimal 3 (tabel 4.2 brief). Kalau
// klien boleh mengirimnya, ia bisa menaikkan grade-nya sendiri.
type KeadaanPrasyarat struct {
	SemuaDokumenVerified bool
	AdaSurveiValid       bool
	SlikSudahDijalankan  bool

	// Kolektibilitas hasil SLIK terakhir yang SUKSES. Bernilai 0 ketika SLIK
	// belum pernah berhasil dijalankan — 0 bukan "bersih", melainkan "tidak
	// ada data", dan PastikanBolehSkoringPengajuan menolaknya lewat
	// SlikSudahDijalankan (AGENTS.md Larangan 15).
	Kolektibilitas int
}

// PrasyaratSkoringRepository membaca keadaan prasyarat BR-03 dari penyimpanan.
//
// Kontraknya hidup di paket service mengikuti pola parameter_repository.go dan
// pengajuan_repository.go, sehingga paket repository boleh meng-import service
// tanpa import cycle. Tidak ada aturan bisnis di sini — implementasinya hanya
// melaporkan fakta; keputusan menolak tetap diambil SkoringService.
type PrasyaratSkoringRepository interface {
	// KeadaanPrasyaratSkoring melaporkan keadaan satu pengajuan.
	// ErrTidakDitemukan bila pengajuannya sendiri tidak ada.
	KeadaanPrasyaratSkoring(ctx context.Context, pengajuanID int64) (KeadaanPrasyarat, error)
}

// DenganPrasyarat memasang sumber keadaan BR-03. Mengikuti pola DenganWaktu
// pada service lain supaya konstruktor yang sudah dipakai tidak berubah.
func (s *SkoringService) DenganPrasyarat(repo PrasyaratSkoringRepository) *SkoringService {
	s.prasyarat = repo
	return s
}

// PastikanBolehSkoringPengajuan menegakkan BR-03 memakai keadaan yang dibaca
// dari database, bukan yang diklaim pemanggil.
//
// Sifatnya FAIL-CLOSED dan itu disengaja: bila sumber keadaan belum dipasang,
// permintaan ditolak sebagai kesalahan konfigurasi, BUKAN dianggap lolos.
// Guard yang diam-diam menyerah ketika dependensinya hilang sama saja dengan
// tidak ada guard — dan pada jalur SLIK, menganggap ketiadaan data sebagai
// keadaan bersih persis yang dilarang AGENTS.md Larangan 15.
func (s *SkoringService) PastikanBolehSkoringPengajuan(ctx context.Context, pengajuanID int64) (KeadaanPrasyarat, error) {
	var keadaan KeadaanPrasyarat

	if s.prasyarat == nil {
		return keadaan, domain.NewConfigError(
			"sumber prasyarat skoring belum dipasang; BR-03 tidak dapat diperiksa")
	}

	keadaan, err := s.prasyarat.KeadaanPrasyaratSkoring(ctx, pengajuanID)
	if err != nil {
		return keadaan, err
	}

	if err := s.PastikanBolehSkoring(PrasyaratSkoring{
		SemuaDokumenVerified: keadaan.SemuaDokumenVerified,
		AdaSurveiValid:       keadaan.AdaSurveiValid,
		SlikSudahDijalankan:  keadaan.SlikSudahDijalankan,
	}); err != nil {
		return keadaan, err
	}

	return keadaan, nil
}
