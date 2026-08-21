package service

import "github.com/irgiys/tim1gow/backend/internal/domain"

// ParameterRepository adalah satu-satunya jalan service membaca tabel parameter.
// Service memanggilnya SETIAP KALI menghitung — tidak meng-cache ke variabel
// paket — supaya perubahan oleh ADM langsung berlaku tanpa restart (AC-15).
type ParameterRepository interface {
	// KomponenSkor mengembalikan baris aktif tabel parameter_skoring.
	KomponenSkor() ([]domain.ParameterKomponenSkor, error)

	// SkorRiwayatSlik mengembalikan skor komponen SLIK untuk suatu kolektibilitas.
	// ditemukan=false ketika barisnya belum diatur; service menolak melanjutkan,
	// bukan memakai nilai default (AGENTS.md Larangan 3 & 15).
	SkorRiwayatSlik(kolektibilitas int) (skor float64, ditemukan bool, err error)

	// Umum membaca parameter perhitungan bernama, mis. "hari_kerja_per_bulan"
	// dan "margin_usaha" untuk rumus kapasitas bayar (brief §4.4).
	Umum(kunci string) (nilai float64, ditemukan bool, err error)

	// RentangMarginPerGrade mengembalikan seluruh baris rentang_margin,
	// dipakai untuk menurunkan grade dari skor akhir.
	RentangMarginPerGrade() ([]domain.RentangMargin, error)

	// RentangMargin mengembalikan baris rentang_margin untuk satu grade.
	RentangMargin(grade int) (r domain.RentangMargin, ditemukan bool, err error)

	// AmbangApproval mengembalikan aturan ambang approval untuk nominal total plafon tertentu (Tabel 4.1).
	AmbangApproval(totalPlafon int64) (ambang domain.AmbangApproval, ditemukan bool, err error)

	// SemuaAmbangApproval mengembalikan seluruh baris ambang_approval.
	SemuaAmbangApproval() ([]domain.AmbangApproval, error)
}

// Kunci parameter umum. Ini KUNCI baris, bukan nilainya.
const (
	KunciHariKerjaPerBulan = "hari_kerja_per_bulan"
	KunciMarginUsaha       = "margin_usaha"

	// Batas plafon BR-01. Nilainya hidup di tabel parameter_umum supaya dapat
	// diubah ADM tanpa deploy ulang (FR-13, AGENTS.md Larangan 3).
	KunciPlafonMinimum  = "plafon_minimum"
	KunciPlafonMaksimum = "plafon_maksimum"
)
