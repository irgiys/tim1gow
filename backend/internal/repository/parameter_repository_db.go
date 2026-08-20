package repository

import (
	"errors"

	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/irgiys/tim1gow/backend/internal/domain"
	"github.com/irgiys/tim1gow/backend/internal/service"
)

type dbParameterKomponenSkor struct {
	Kode   string  `gorm:"column:kode;primaryKey"`
	Nama   string  `gorm:"column:nama"`
	Bobot  float64 `gorm:"column:bobot"`
	Batas1 float64 `gorm:"column:batas_1"`
	Batas2 float64 `gorm:"column:batas_2"`
	Aktif  bool    `gorm:"column:aktif"`
}

func (dbParameterKomponenSkor) TableName() string {
	return "parameter_skoring"
}

type dbParameterRiwayatSlik struct {
	Kolektibilitas int     `gorm:"column:kolektibilitas;primaryKey"`
	Skor           float64 `gorm:"column:skor"`
}

func (dbParameterRiwayatSlik) TableName() string {
	return "parameter_riwayat_slik"
}

type dbParameterUmum struct {
	Kunci string  `gorm:"column:kunci;primaryKey"`
	Nilai float64 `gorm:"column:nilai"`
}

func (dbParameterUmum) TableName() string {
	return "parameter_umum"
}

type dbRentangMargin struct {
	Grade         int      `gorm:"column:grade;primaryKey"`
	SkorMin       int      `gorm:"column:skor_min"`
	SkorMaks      int      `gorm:"column:skor_maks"`
	MarginMin     *float64 `gorm:"column:margin_min"`
	MarginMaks    *float64 `gorm:"column:margin_maks"`
	NisbahMin     *float64 `gorm:"column:nisbah_min"`
	NisbahMaks    *float64 `gorm:"column:nisbah_maks"`
	DapatDibiayai bool     `gorm:"column:dapat_dibiayai"`
}

func (dbRentangMargin) TableName() string {
	return "rentang_margin"
}

type dbAmbangApproval struct {
	ID         int64          `gorm:"column:id;primaryKey"`
	PlafonMin  int64          `gorm:"column:plafon_min"`
	PlafonMaks int64          `gorm:"column:plafon_maks"`
	Level      pq.StringArray `gorm:"column:level;type:varchar(8)[]"`
}

func (dbAmbangApproval) TableName() string {
	return "ambang_approval"
}

type gormParameterRepo struct {
	db *gorm.DB
}

// NewParameterRepository membuat repository parameter berbasis database GORM.
func NewParameterRepository(db *gorm.DB) service.ParameterRepository {
	return &gormParameterRepo{db: db}
}

func (r *gormParameterRepo) KomponenSkor() ([]domain.ParameterKomponenSkor, error) {
	var rows []dbParameterKomponenSkor
	if err := r.db.Where("aktif = true").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ParameterKomponenSkor, len(rows))
	for i, row := range rows {
		out[i] = domain.ParameterKomponenSkor{
			Kode:   row.Kode,
			Nama:   row.Nama,
			Bobot:  row.Bobot,
			Batas1: row.Batas1,
			Batas2: row.Batas2,
			Aktif:  row.Aktif,
		}
	}
	return out, nil
}

func (r *gormParameterRepo) SkorRiwayatSlik(kolektibilitas int) (float64, bool, error) {
	var row dbParameterRiwayatSlik
	if err := r.db.Where("kolektibilitas = ?", kolektibilitas).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return row.Skor, true, nil
}

func (r *gormParameterRepo) Umum(kunci string) (float64, bool, error) {
	var row dbParameterUmum
	if err := r.db.Where("kunci = ?", kunci).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return row.Nilai, true, nil
}

func (r *gormParameterRepo) RentangMarginPerGrade() ([]domain.RentangMargin, error) {
	var rows []dbRentangMargin
	if err := r.db.Order("grade ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.RentangMargin, len(rows))
	for i, row := range rows {
		rm := domain.RentangMargin{
			Grade:         row.Grade,
			SkorMin:       row.SkorMin,
			SkorMaks:      row.SkorMaks,
			DapatDibiayai: row.DapatDibiayai,
		}
		if row.MarginMin != nil {
			rm.MarginMin = *row.MarginMin
		}
		if row.MarginMaks != nil {
			rm.MarginMaks = *row.MarginMaks
		}
		if row.NisbahMin != nil {
			rm.NisbahMin = *row.NisbahMin
		}
		if row.NisbahMaks != nil {
			rm.NisbahMaks = *row.NisbahMaks
		}
		out[i] = rm
	}
	return out, nil
}

func (r *gormParameterRepo) RentangMargin(grade int) (domain.RentangMargin, bool, error) {
	var row dbRentangMargin
	if err := r.db.Where("grade = ?", grade).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.RentangMargin{}, false, nil
		}
		return domain.RentangMargin{}, false, err
	}
	rm := domain.RentangMargin{
		Grade:         row.Grade,
		SkorMin:       row.SkorMin,
		SkorMaks:      row.SkorMaks,
		DapatDibiayai: row.DapatDibiayai,
	}
	if row.MarginMin != nil {
		rm.MarginMin = *row.MarginMin
	}
	if row.MarginMaks != nil {
		rm.MarginMaks = *row.MarginMaks
	}
	if row.NisbahMin != nil {
		rm.NisbahMin = *row.NisbahMin
	}
	if row.NisbahMaks != nil {
		rm.NisbahMaks = *row.NisbahMaks
	}
	return rm, true, nil
}

func (r *gormParameterRepo) AmbangApproval(totalPlafon int64) (domain.AmbangApproval, bool, error) {
	var row dbAmbangApproval
	if err := r.db.Where("plafon_min <= ? AND plafon_maks >= ?", totalPlafon, totalPlafon).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.AmbangApproval{}, false, nil
		}
		return domain.AmbangApproval{}, false, err
	}
	levels := make([]domain.Peran, len(row.Level))
	for i, l := range row.Level {
		levels[i] = domain.Peran(l)
	}
	return domain.AmbangApproval{
		PlafonMin:  row.PlafonMin,
		PlafonMaks: row.PlafonMaks,
		Level:      levels,
	}, true, nil
}

func (r *gormParameterRepo) SemuaAmbangApproval() ([]domain.AmbangApproval, error) {
	var rows []dbAmbangApproval
	if err := r.db.Order("plafon_min ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.AmbangApproval, len(rows))
	for i, row := range rows {
		levels := make([]domain.Peran, len(row.Level))
		for j, l := range row.Level {
			levels[j] = domain.Peran(l)
		}
		out[i] = domain.AmbangApproval{
			PlafonMin:  row.PlafonMin,
			PlafonMaks: row.PlafonMaks,
			Level:      levels,
		}
	}
	return out, nil
}
