package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/irgiys/tim1gow/backend/internal/domain"
)

// ApprovalRepository adalah antarmuka akses database untuk pengajuan dan approval yang dibutuhkan service.
type ApprovalRepository interface {
	AmbilPengajuan(ctx context.Context, id int64) (*domain.Pengajuan, error)
	UpdateStatusPengajuan(ctx context.Context, id int64, statusBaru domain.StatusPengajuan) error
	SimpanPengajuan(ctx context.Context, p *domain.Pengajuan) error
	SimpanKeputusan(ctx context.Context, k *domain.KeputusanApprovalRecord) error
	AmbilKeputusanByPengajuan(ctx context.Context, pengajuanID int64) ([]domain.KeputusanApprovalRecord, error)
}

// ApprovalService mengatur alur persetujuan berjenjang (FR-08) dan penegakan
// aturan bisnis approval (BR-01, BR-02, BR-05, BR-09, BR-10).
type ApprovalService interface {
	// AjukanKeApproval memvalidasi kelayakan dan memajukan status ke WAITING_APPROVAL_L1.
	AjukanKeApproval(ctx context.Context, pengajuanID int64, actorID int64, actorRole domain.Peran) error

	// PutuskanApproval mengeksekusi keputusan APPROVE/REJECT/RETURN oleh approver.
	PutuskanApproval(ctx context.Context, pengajuanID int64, req domain.ApprovalDecisionRequest, actorID int64, actorRole domain.Peran) (*domain.KeputusanApprovalRecord, error)

	// AmbilDetailApproval mengembalikan data pengajuan dan histori keputusan untuk layar approval.
	AmbilDetailApproval(ctx context.Context, pengajuanID int64) (*domain.PengajuanApprovalDetail, error)
}

type approvalService struct {
	approvalRepo  ApprovalRepository
	parameterRepo ParameterRepository
	auditService  AuditService
}

// NewApprovalService membuat instance ApprovalService baru.
func NewApprovalService(
	approvalRepo ApprovalRepository,
	parameterRepo ParameterRepository,
	auditService AuditService,
) ApprovalService {
	return &approvalService{
		approvalRepo:  approvalRepo,
		parameterRepo: parameterRepo,
		auditService:  auditService,
	}
}

func (s *approvalService) AjukanKeApproval(ctx context.Context, pengajuanID int64, actorID int64, actorRole domain.Peran) error {
	if pengajuanID <= 0 {
		return domain.NewBusinessRuleError("VALIDATION_ERROR", "id pengajuan tidak valid")
	}

	p, err := s.approvalRepo.AmbilPengajuan(ctx, pengajuanID)
	if err != nil {
		return fmt.Errorf("gagal mengambil pengajuan: %w", err)
	}
	if p == nil {
		return domain.NewBusinessRuleError("NOT_FOUND", "pengajuan tidak ditemukan")
	}

	// BR-05: Grade 5 tidak dapat diajukan ke approval; status menjadi REJECTED_SCORING
	if p.Grade == 5 {
		_ = s.approvalRepo.UpdateStatusPengajuan(ctx, p.ID, domain.StatusRejectedScoring)
		_ = s.auditService.Catat(ctx, domain.CatatAuditInput{
			PengajuanID:   &p.ID,
			Aksi:          domain.AksiAjukanApproval,
			StatusSebelum: p.Status,
			StatusSesudah: domain.StatusRejectedScoring,
			Catatan:       "penolakan otomatis skoring: grade 5 tidak dapat diajukan ke approval (BR-05)",
			ActorID:       actorID,
			ActorRole:     actorRole,
		})
		return domain.NewBusinessRuleError("BR-05", "grade 5 tidak dapat diajukan ke approval")
	}

	// Validasi status awal: pengajuan harus sudah diskor (SCORED) atau dikembalikan (RETURNED)
	if p.Status != domain.StatusScored && p.Status != domain.StatusReturned {
		return domain.NewBusinessRuleError("BUSINESS_RULE_VIOLATION",
			"pengajuan berstatus %s belum siap diajukan ke approval", p.Status)
	}

	// Validasi plafon dengan ambang approval (BR-01)
	ambang, ditemukan, err := s.parameterRepo.AmbangApproval(p.TotalPlafon)
	if err != nil {
		return fmt.Errorf("gagal membaca ambang approval: %w", err)
	}
	if !ditemukan || len(ambang.Level) == 0 {
		return domain.NewConfigError("ambang approval untuk total plafon Rp %d belum diatur", p.TotalPlafon)
	}

	statusSebelum := p.Status
	statusBaru := domain.StatusWaitingApprovalL1

	if err := s.approvalRepo.UpdateStatusPengajuan(ctx, p.ID, statusBaru); err != nil {
		return fmt.Errorf("gagal memperbarui status pengajuan: %w", err)
	}

	// BR-10: Catat ke audit trail
	_ = s.auditService.Catat(ctx, domain.CatatAuditInput{
		PengajuanID:   &p.ID,
		Aksi:          domain.AksiAjukanApproval,
		StatusSebelum: statusSebelum,
		StatusSesudah: statusBaru,
		Catatan:       fmt.Sprintf("pengajuan diajukan ke approval level 1 (%s)", ambang.Level[0]),
		ActorID:       actorID,
		ActorRole:     actorRole,
	})

	return nil
}

func (s *approvalService) PutuskanApproval(
	ctx context.Context,
	pengajuanID int64,
	req domain.ApprovalDecisionRequest,
	actorID int64,
	actorRole domain.Peran,
) (*domain.KeputusanApprovalRecord, error) {
	if pengajuanID <= 0 {
		return nil, domain.NewBusinessRuleError("VALIDATION_ERROR", "id pengajuan tidak valid")
	}

	req.Alasan = strings.TrimSpace(req.Alasan)
	req.Catatan = strings.TrimSpace(req.Catatan)

	// Validasi nilai keputusan
	if req.Keputusan != domain.KeputusanApprove &&
		req.Keputusan != domain.KeputusanReject &&
		req.Keputusan != domain.KeputusanReturn {
		return nil, domain.NewBusinessRuleError("VALIDATION_ERROR", "keputusan tidak valid, harus APPROVE, REJECT, atau RETURN")
	}

	p, err := s.approvalRepo.AmbilPengajuan(ctx, pengajuanID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil pengajuan: %w", err)
	}
	if p == nil {
		return nil, domain.NewBusinessRuleError("NOT_FOUND", "pengajuan tidak ditemukan")
	}

	// BR-09: Satu pengguna tidak boleh menjadi maker dan approver pada pengajuan yang sama
	if p.CreatedBy == actorID {
		return nil, domain.NewBusinessRuleError("BR-09",
			"pengguna pembuat pengajuan tidak dapat menyetujui pengajuannya sendiri")
	}

	// BR-02: Approval harus berurutan
	var expectedRole domain.Peran
	var levelNumber int
	switch p.Status {
	case domain.StatusWaitingApprovalL1:
		expectedRole = domain.PeranKCP
		levelNumber = 1
	case domain.StatusWaitingApprovalL2:
		expectedRole = domain.PeranKC
		levelNumber = 2
	case domain.StatusWaitingApprovalL3:
		expectedRole = domain.PeranKOM
		levelNumber = 3
	default:
		return nil, domain.NewBusinessRuleError("BUSINESS_RULE_VIOLATION",
			"pengajuan berstatus %s tidak sedang menunggu approval", p.Status)
	}

	if actorRole != expectedRole {
		// Pesan error spesifik sesuai BR-02
		if levelNumber == 1 && (actorRole == domain.PeranKC || actorRole == domain.PeranKOM) {
			return nil, domain.NewBusinessRuleError("BR-02",
				"approval harus berurutan: level 2 tidak dapat memutuskan sebelum level 1 memberi APPROVE")
		}
		if levelNumber == 2 && actorRole == domain.PeranKOM {
			return nil, domain.NewBusinessRuleError("BR-02",
				"approval harus berurutan: level 3 tidak dapat memutuskan sebelum level 2 memberi APPROVE")
		}
		return nil, domain.NewBusinessRuleError("FORBIDDEN",
			"peran %s tidak berwenang memutuskan approval pada tahap ini (diperlukan %s)", actorRole, expectedRole)
	}

	// Baca ambang approval dari database (Tabel 4.1)
	ambang, ditemukan, err := s.parameterRepo.AmbangApproval(p.TotalPlafon)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca ambang approval: %w", err)
	}
	if !ditemukan || len(ambang.Level) == 0 {
		return nil, domain.NewConfigError("ambang approval untuk total plafon Rp %d belum diatur", p.TotalPlafon)
	}

	statusSebelum := p.Status
	var statusSesudah domain.StatusPengajuan

	switch req.Keputusan {
	case domain.KeputusanReject:
		if req.Alasan == "" {
			return nil, domain.NewBusinessRuleError("VALIDATION_ERROR", "alasan wajib diisi saat menolak pengajuan")
		}
		statusSesudah = domain.StatusRejected

	case domain.KeputusanReturn:
		if req.Alasan == "" {
			return nil, domain.NewBusinessRuleError("VALIDATION_ERROR", "alasan wajib diisi saat mengembalikan pengajuan ke AO")
		}
		statusSesudah = domain.StatusReturned

	case domain.KeputusanApprove:
		// Tentukan apakah masih ada level selanjutnya
		// Cari indeks level saat ini di ambang.Level
		currentIdx := -1
		for i, l := range ambang.Level {
			if l == actorRole {
				currentIdx = i
				break
			}
		}

		if currentIdx == -1 {
			return nil, domain.NewBusinessRuleError("BUSINESS_RULE_VIOLATION",
				"peran %s tidak ada dalam daftar level approval untuk plafon Rp %d", actorRole, p.TotalPlafon)
		}

		// Jika masih ada level berikutnya
		if currentIdx < len(ambang.Level)-1 {
			nextLevel := ambang.Level[currentIdx+1]
			switch nextLevel {
			case domain.PeranKC:
				statusSesudah = domain.StatusWaitingApprovalL2
			case domain.PeranKOM:
				statusSesudah = domain.StatusWaitingApprovalL3
			default:
				return nil, domain.NewConfigError("urutan level berikutnya tidak dikenal: %s", nextLevel)
			}
		} else {
			// Level terakhir -> APPROVED
			statusSesudah = domain.StatusApproved
		}
	}

	// Simpan rekaman keputusan approval
	keputusanRecord := &domain.KeputusanApprovalRecord{
		PengajuanID: p.ID,
		Level:       actorRole,
		Keputusan:   req.Keputusan,
		Alasan:      req.Alasan,
		Catatan:     req.Catatan,
		ApproverID:  actorID,
	}

	if err := s.approvalRepo.SimpanKeputusan(ctx, keputusanRecord); err != nil {
		return nil, fmt.Errorf("gagal menyimpan keputusan approval: %w", err)
	}

	// Update status pengajuan
	if err := s.approvalRepo.UpdateStatusPengajuan(ctx, p.ID, statusSesudah); err != nil {
		return nil, fmt.Errorf("gagal memperbarui status pengajuan: %w", err)
	}

	// BR-10: Catat ke audit trail
	auditNote := fmt.Sprintf("keputusan level %s: %s", actorRole, req.Keputusan)
	if req.Alasan != "" {
		auditNote += fmt.Sprintf(" (alasan: %s)", req.Alasan)
	}
	if req.Catatan != "" {
		auditNote += fmt.Sprintf(" [catatan: %s]", req.Catatan)
	}

	_ = s.auditService.Catat(ctx, domain.CatatAuditInput{
		PengajuanID:   &p.ID,
		Aksi:          domain.AksiKeputusanApproval,
		StatusSebelum: statusSebelum,
		StatusSesudah: statusSesudah,
		Catatan:       auditNote,
		ActorID:       actorID,
		ActorRole:     actorRole,
	})

	return keputusanRecord, nil
}

func (s *approvalService) AmbilDetailApproval(ctx context.Context, pengajuanID int64) (*domain.PengajuanApprovalDetail, error) {
	p, err := s.approvalRepo.AmbilPengajuan(ctx, pengajuanID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil pengajuan: %w", err)
	}
	if p == nil {
		return nil, domain.NewBusinessRuleError("NOT_FOUND", "pengajuan tidak ditemukan")
	}

	riwayat, err := s.approvalRepo.AmbilKeputusanByPengajuan(ctx, pengajuanID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil riwayat keputusan: %w", err)
	}

	ambang, _, _ := s.parameterRepo.AmbangApproval(p.TotalPlafon)

	var levelSaatIni domain.Peran
	switch p.Status {
	case domain.StatusWaitingApprovalL1:
		levelSaatIni = domain.PeranKCP
	case domain.StatusWaitingApprovalL2:
		levelSaatIni = domain.PeranKC
	case domain.StatusWaitingApprovalL3:
		levelSaatIni = domain.PeranKOM
	}

	return &domain.PengajuanApprovalDetail{
		Pengajuan:       *p,
		RiwayatApproval: riwayat,
		LevelDiperlukan: ambang.Level,
		LevelSaatIni:    levelSaatIni,
	}, nil
}
