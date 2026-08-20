package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/irgiys/tim1gow/backend/internal/domain"
)

// AuditRepository adalah antarmuka akses database untuk audit trail yang dibutuhkan service.
type AuditRepository interface {
	Catat(ctx context.Context, entry *domain.AuditTrailEntry) error
	AmbilRiwayatByPengajuan(ctx context.Context, pengajuanID int64) ([]domain.AuditTrailEntry, error)
	AmbilSemua(ctx context.Context, limit, offset int) ([]domain.AuditTrailEntry, error)
}

// AuditService adalah layanan untuk mengelola jejak audit sistem (FR-09).
// Sifatnya append-only: tidak ada fungsi untuk mengubah atau menghapus rekaman (AC-13).
type AuditService interface {
	// Catat merekam satu jejak audit baru (BR-10).
	Catat(ctx context.Context, input domain.CatatAuditInput) error

	// AmbilRiwayatPengajuan mengembalikan riwayat lengkap satu pengajuan urut waktu ASC (AC-12).
	AmbilRiwayatPengajuan(ctx context.Context, pengajuanID int64) ([]domain.AuditTrailEntry, error)

	// AmbilSemua mengembalikan daftar audit log (untuk audit keseluruhan).
	AmbilSemua(ctx context.Context, limit, offset int) ([]domain.AuditTrailEntry, error)
}

type auditService struct {
	repo AuditRepository
}

// NewAuditService membuat instance AuditService baru.
func NewAuditService(repo AuditRepository) AuditService {
	return &auditService{repo: repo}
}

func (s *auditService) Catat(ctx context.Context, input domain.CatatAuditInput) error {
	if input.ActorID <= 0 {
		return errors.New("actor_id wajib diisi untuk pencatatan audit trail (BR-10)")
	}
	if input.ActorRole == "" {
		return errors.New("actor_role wajib diisi untuk pencatatan audit trail (BR-10)")
	}
	if input.Aksi == "" {
		return errors.New("aksi audit wajib diisi")
	}

	entry := &domain.AuditTrailEntry{
		PengajuanID:   input.PengajuanID,
		Aksi:          input.Aksi,
		StatusSebelum: input.StatusSebelum,
		StatusSesudah: input.StatusSesudah,
		Catatan:       input.Catatan,
		ActorID:       input.ActorID,
		ActorRole:     input.ActorRole,
		CreatedAt:     time.Now(),
	}

	if err := s.repo.Catat(ctx, entry); err != nil {
		return fmt.Errorf("gagal mencatat audit trail: %w", err)
	}
	return nil
}

func (s *auditService) AmbilRiwayatPengajuan(ctx context.Context, pengajuanID int64) ([]domain.AuditTrailEntry, error) {
	if pengajuanID <= 0 {
		return nil, domain.NewBusinessRuleError("VALIDATION_ERROR", "id pengajuan tidak valid")
	}
	return s.repo.AmbilRiwayatByPengajuan(ctx, pengajuanID)
}

func (s *auditService) AmbilSemua(ctx context.Context, limit, offset int) ([]domain.AuditTrailEntry, error) {
	return s.repo.AmbilSemua(ctx, limit, offset)
}
