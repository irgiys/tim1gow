package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/irgiys/tim1gow/backend/internal/service"
)

// hasilSlikModel adalah pemetaan GORM ke tabel `hasil_slik` (migrasi 000004).
type hasilSlikModel struct {
	ID                   int64      `gorm:"primaryKey;autoIncrement"`
	PengajuanID          int64      `gorm:"column:pengajuan_id;not null"`
	Kolektibilitas       *int       `gorm:"column:kolektibilitas"`
	JumlahFasilitasAktif *int       `gorm:"column:jumlah_fasilitas_aktif"`
	TotalBakiDebet       *int64     `gorm:"column:total_baki_debet"`
	TanggalData          *time.Time `gorm:"column:tanggal_data"`
	ReferenceID          string     `gorm:"column:reference_id"`
	StatusPanggilan      string     `gorm:"column:status_panggilan;not null"`
	BerlakuSampai        *time.Time `gorm:"column:berlaku_sampai"`
	DicekOleh            *int64     `gorm:"column:dicek_oleh"`
	DibuatPada           time.Time  `gorm:"column:dibuat_pada;not null"`
}

func (hasilSlikModel) TableName() string {
	return "hasil_slik"
}

type gormSlikRepo struct {
	db *gorm.DB
}

// NewSlikRepository membuat implementasi GORM untuk service.SlikRepository.
func NewSlikRepository(db *gorm.DB) service.SlikRepository {
	return &gormSlikRepo{db: db}
}

// Simpan menyisipkan satu baris hasil SLIK ke database dan mengisi h.ID.
func (r *gormSlikRepo) Simpan(ctx context.Context, h *service.HasilSlik) error {
	m := hasilSlikModel{
		PengajuanID:          h.PengajuanID,
		Kolektibilitas:       h.Kolektibilitas,
		JumlahFasilitasAktif: h.JumlahFasilitasAktif,
		TotalBakiDebet:       h.TotalBakiDebet,
		TanggalData:          h.TanggalData,
		ReferenceID:          h.ReferenceID,
		StatusPanggilan:      h.StatusPanggilan,
		BerlakuSampai:        h.BerlakuSampai,
		DicekOleh:            h.DicekOleh,
		DibuatPada:           h.DibuatPada,
	}
	if m.DibuatPada.IsZero() {
		m.DibuatPada = time.Now()
	}

	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return err
	}
	h.ID = m.ID
	return nil
}

// TerakhirSukses mengembalikan baris hasil_slik berstatus SUKSES terbaru untuk
// pengajuan yang diberikan. Mengembalikan service.ErrTidakDitemukan bila belum ada.
func (r *gormSlikRepo) TerakhirSukses(ctx context.Context, pengajuanID int64) (service.HasilSlik, error) {
	var m hasilSlikModel
	err := r.db.WithContext(ctx).
		Where("pengajuan_id = ? AND status_panggilan = ?", pengajuanID, "SUKSES").
		Order("dibuat_pada DESC").
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return service.HasilSlik{}, service.ErrTidakDitemukan
		}
		return service.HasilSlik{}, err
	}

	return service.HasilSlik{
		ID:                   m.ID,
		PengajuanID:          m.PengajuanID,
		Kolektibilitas:       m.Kolektibilitas,
		JumlahFasilitasAktif: m.JumlahFasilitasAktif,
		TotalBakiDebet:       m.TotalBakiDebet,
		TanggalData:          m.TanggalData,
		ReferenceID:          m.ReferenceID,
		StatusPanggilan:      m.StatusPanggilan,
		BerlakuSampai:        m.BerlakuSampai,
		DicekOleh:            m.DicekOleh,
		DibuatPada:           m.DibuatPada,
	}, nil
}
