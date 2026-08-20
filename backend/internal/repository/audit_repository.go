package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/irgiys/tim1gow/backend/internal/domain"
)

// AuditRepository adalah interface akses database untuk audit trail (append-only).
type AuditRepository interface {
	// Catat merekam satu baris log audit baru.
	Catat(ctx context.Context, entry *domain.AuditTrailEntry) error

	// AmbilRiwayatByPengajuan mengambil jejak audit satu pengajuan urut waktu ASC (AC-12).
	AmbilRiwayatByPengajuan(ctx context.Context, pengajuanID int64) ([]domain.AuditTrailEntry, error)

	// AmbilSemua mengambil seluruh audit trail terbaru (untuk audit sistem).
	AmbilSemua(ctx context.Context, limit, offset int) ([]domain.AuditTrailEntry, error)
}

type gormAuditRepo struct {
	db *gorm.DB
}

// NewAuditRepository membuat implementasi GORM untuk AuditRepository.
func NewAuditRepository(db *gorm.DB) AuditRepository {
	return &gormAuditRepo{db: db}
}

func (r *gormAuditRepo) Catat(ctx context.Context, entry *domain.AuditTrailEntry) error {
	return r.db.WithContext(ctx).Create(entry).Error
}

func (r *gormAuditRepo) AmbilRiwayatByPengajuan(ctx context.Context, pengajuanID int64) ([]domain.AuditTrailEntry, error) {
	var list []domain.AuditTrailEntry
	err := r.db.WithContext(ctx).
		Where("pengajuan_id = ?", pengajuanID).
		Order("created_at ASC, id ASC").
		Find(&list).Error
	return list, err
}

func (r *gormAuditRepo) AmbilSemua(ctx context.Context, limit, offset int) ([]domain.AuditTrailEntry, error) {
	var list []domain.AuditTrailEntry
	if limit <= 0 {
		limit = 100
	}
	err := r.db.WithContext(ctx).
		Order("created_at ASC, id ASC").
		Limit(limit).
		Offset(offset).
		Find(&list).Error
	return list, err
}
