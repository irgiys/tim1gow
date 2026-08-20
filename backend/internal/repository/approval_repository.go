package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/irgiys/tim1gow/backend/internal/domain"
)

// ApprovalRepository adalah interface akses database untuk alur approval.
type ApprovalRepository interface {
	AmbilPengajuan(ctx context.Context, id int64) (*domain.Pengajuan, error)
	UpdateStatusPengajuan(ctx context.Context, id int64, statusBaru domain.StatusPengajuan) error
	SimpanPengajuan(ctx context.Context, p *domain.Pengajuan) error
	SimpanKeputusan(ctx context.Context, k *domain.KeputusanApprovalRecord) error
	AmbilKeputusanByPengajuan(ctx context.Context, pengajuanID int64) ([]domain.KeputusanApprovalRecord, error)
}

type gormApprovalRepo struct {
	db *gorm.DB
}

// NewApprovalRepository membuat implementasi GORM untuk ApprovalRepository.
func NewApprovalRepository(db *gorm.DB) ApprovalRepository {
	return &gormApprovalRepo{db: db}
}

func (r *gormApprovalRepo) AmbilPengajuan(ctx context.Context, id int64) (*domain.Pengajuan, error) {
	var p domain.Pengajuan
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *gormApprovalRepo) UpdateStatusPengajuan(ctx context.Context, id int64, statusBaru domain.StatusPengajuan) error {
	return r.db.WithContext(ctx).Model(&domain.Pengajuan{}).
		Where("id = ?", id).
		Update("status", statusBaru).Error
}

func (r *gormApprovalRepo) SimpanPengajuan(ctx context.Context, p *domain.Pengajuan) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *gormApprovalRepo) SimpanKeputusan(ctx context.Context, k *domain.KeputusanApprovalRecord) error {
	return r.db.WithContext(ctx).Create(k).Error
}

func (r *gormApprovalRepo) AmbilKeputusanByPengajuan(ctx context.Context, pengajuanID int64) ([]domain.KeputusanApprovalRecord, error) {
	var list []domain.KeputusanApprovalRecord
	err := r.db.WithContext(ctx).
		Where("pengajuan_id = ?", pengajuanID).
		Order("created_at ASC, id ASC").
		Find(&list).Error
	return list, err
}
