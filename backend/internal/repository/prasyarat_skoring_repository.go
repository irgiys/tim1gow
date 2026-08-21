// Implementasi pembacaan keadaan prasyarat BR-03 (FR-06).
//
// Repository ini HANYA melaporkan fakta dari tabel `dokumen`, `survei`, dan
// `hasil_slik`. Keputusan menolak/melanjutkan skoring diambil di
// service.SkoringService (AGENTS.md Larangan 17).
package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/irgiys/tim1gow/backend/internal/service"
)

type gormPrasyaratSkoringRepo struct {
	db *gorm.DB

	// sekarang dipisah supaya masa berlaku SLIK (BR-04) dapat diuji tanpa
	// menunggu 30 hari berlalu.
	sekarang func() time.Time
}

// NewPrasyaratSkoringRepository membuat implementasi GORM untuk
// service.PrasyaratSkoringRepository.
func NewPrasyaratSkoringRepository(db *gorm.DB) service.PrasyaratSkoringRepository {
	return &gormPrasyaratSkoringRepo{db: db, sekarang: time.Now}
}

// KeadaanPrasyaratSkoring membaca keadaan satu pengajuan apa adanya.
//
// Tiga hal yang sengaja ketat di sini:
//
//  1. Pengajuan yang tidak ada menghasilkan ErrTidakDitemukan, bukan keadaan
//     kosong. Keadaan kosong akan terbaca sebagai "prasyarat belum terpenuhi"
//     dan menghasilkan 422 BR-03 yang menyesatkan untuk id yang salah ketik.
//  2. Hanya baris hasil_slik berstatus SUKSES yang dihitung. Panggilan yang
//     gagal tercatat sebagai bukti percobaan, tetapi TIDAK membuat SLIK
//     dianggap sudah dijalankan (AGENTS.md Larangan 15).
//  3. Hasil SLIK yang kedaluwarsa (BR-04, kolom berlaku_sampai) diperlakukan
//     sama dengan belum pernah SLIK: skoring berhenti dan ANL harus SLIK ulang.
func (r *gormPrasyaratSkoringRepo) KeadaanPrasyaratSkoring(ctx context.Context, pengajuanID int64) (service.KeadaanPrasyarat, error) {
	var keadaan service.KeadaanPrasyarat

	db := r.db.WithContext(ctx)

	var jumlahPengajuan int64
	if err := db.Table("pengajuan").Where("id = ?", pengajuanID).
		Count(&jumlahPengajuan).Error; err != nil {
		return keadaan, err
	}
	if jumlahPengajuan == 0 {
		return keadaan, service.ErrTidakDitemukan
	}

	// Dokumen wajib: daftarnya berupa data (tabel dokumen_wajib), bukan
	// konstanta di kode (AGENTS.md Larangan 3). Prasyarat terpenuhi ketika
	// tidak ada satu pun jenis wajib yang belum punya dokumen VERIFIED.
	var jenisWajibBelumVerified int64
	err := db.Raw(`
		SELECT COUNT(*)
		FROM dokumen_wajib dw
		WHERE dw.aktif = TRUE
		  AND NOT EXISTS (
		      SELECT 1 FROM dokumen d
		      WHERE d.pengajuan_id = ?
		        AND d.jenis_dokumen = dw.jenis_dokumen
		        AND d.status = 'VERIFIED'
		  )`, pengajuanID).Scan(&jenisWajibBelumVerified).Error
	if err != nil {
		return keadaan, err
	}

	// Daftar dokumen wajib yang kosong TIDAK boleh terbaca sebagai "semua
	// sudah lengkap" — itu membuat BR-03 lolos hanya karena tabel parameter
	// belum di-seed.
	var jumlahJenisWajib int64
	if err := db.Table("dokumen_wajib").Where("aktif = TRUE").
		Count(&jumlahJenisWajib).Error; err != nil {
		return keadaan, err
	}
	keadaan.SemuaDokumenVerified = jumlahJenisWajib > 0 && jenisWajibBelumVerified == 0

	var jumlahSurveiValid int64
	if err := db.Table("survei").
		Where("pengajuan_id = ? AND status = ?", pengajuanID, "VALID").
		Count(&jumlahSurveiValid).Error; err != nil {
		return keadaan, err
	}
	keadaan.AdaSurveiValid = jumlahSurveiValid > 0

	// Hasil SLIK SUKSES terbaru yang masih berlaku (BR-04).
	var slik struct {
		Kolektibilitas *int16
		BerlakuSampai  *time.Time
	}
	err = db.Table("hasil_slik").
		Select("kolektibilitas, berlaku_sampai").
		Where("pengajuan_id = ? AND status_panggilan = ?", pengajuanID, "SUKSES").
		Order("dibuat_pada DESC, id DESC").
		Limit(1).
		Scan(&slik).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return keadaan, err
	}

	if slik.Kolektibilitas != nil {
		masihBerlaku := slik.BerlakuSampai == nil ||
			!r.sekarang().Truncate(24*time.Hour).After(*slik.BerlakuSampai)
		if masihBerlaku {
			keadaan.SlikSudahDijalankan = true
			keadaan.Kolektibilitas = int(*slik.Kolektibilitas)
		}
	}

	return keadaan, nil
}
