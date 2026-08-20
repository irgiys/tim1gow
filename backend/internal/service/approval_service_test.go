package service

import (
	"context"
	"testing"
	"time"

	"github.com/irgiys/tim1gow/backend/internal/domain"
)

// fakeApprovalRepo adalah mock repository memory untuk unit test approval.
type fakeApprovalRepo struct {
	pengajuan map[int64]*domain.Pengajuan
	keputusan map[int64][]domain.KeputusanApprovalRecord
}

func newFakeApprovalRepo() *fakeApprovalRepo {
	return &fakeApprovalRepo{
		pengajuan: make(map[int64]*domain.Pengajuan),
		keputusan: make(map[int64][]domain.KeputusanApprovalRecord),
	}
}

func (f *fakeApprovalRepo) AmbilPengajuan(_ context.Context, id int64) (*domain.Pengajuan, error) {
	p, ok := f.pengajuan[id]
	if !ok {
		return nil, nil
	}
	cp := *p
	return &cp, nil
}

func (f *fakeApprovalRepo) UpdateStatusPengajuan(_ context.Context, id int64, statusBaru domain.StatusPengajuan) error {
	if p, ok := f.pengajuan[id]; ok {
		p.Status = statusBaru
	}
	return nil
}

func (f *fakeApprovalRepo) SimpanPengajuan(_ context.Context, p *domain.Pengajuan) error {
	f.pengajuan[p.ID] = p
	return nil
}

func (f *fakeApprovalRepo) SimpanKeputusan(_ context.Context, k *domain.KeputusanApprovalRecord) error {
	f.keputusan[k.PengajuanID] = append(f.keputusan[k.PengajuanID], *k)
	return nil
}

func (f *fakeApprovalRepo) AmbilKeputusanByPengajuan(_ context.Context, pengajuanID int64) ([]domain.KeputusanApprovalRecord, error) {
	list := f.keputusan[pengajuanID]
	out := make([]domain.KeputusanApprovalRecord, len(list))
	copy(out, list)
	return out, nil
}

// fakeAuditRepo adalah mock repository memory untuk unit test audit trail.
type fakeAuditRepo struct {
	entries []domain.AuditTrailEntry
}

func newFakeAuditRepo() *fakeAuditRepo {
	return &fakeAuditRepo{entries: make([]domain.AuditTrailEntry, 0)}
}

func (f *fakeAuditRepo) Catat(_ context.Context, entry *domain.AuditTrailEntry) error {
	entry.ID = int64(len(f.entries) + 1)
	f.entries = append(f.entries, *entry)
	return nil
}

func (f *fakeAuditRepo) AmbilRiwayatByPengajuan(_ context.Context, pengajuanID int64) ([]domain.AuditTrailEntry, error) {
	var out []domain.AuditTrailEntry
	for _, e := range f.entries {
		if e.PengajuanID != nil && *e.PengajuanID == pengajuanID {
			out = append(out, e)
		}
	}
	return out, nil
}

func (f *fakeAuditRepo) AmbilSemua(_ context.Context, _, _ int) ([]domain.AuditTrailEntry, error) {
	out := make([]domain.AuditTrailEntry, len(f.entries))
	copy(out, f.entries)
	return out, nil
}

// Test AC-10: Verifikasi routing approval berjenjang berdasarkan nominal plafon (Tabel 4.1, BR-01, BR-02).
// - Rp 30.000.000 hanya butuh KCP -> langsung APPROVED
// - Rp 120.000.000 butuh KCP lalu KC; KC tidak bisa memutuskan sebelum KCP APPROVE
func TestApproval_AC10_RoutingBerjenjangDanUrutan(t *testing.T) {
	ctx := context.Background()

	// 1. Kasus Plafon Rp 30.000.000 (Tunggal: KCP)
	t.Run("Plafon 30jt - KCP Tunggal langsung APPROVED", func(t *testing.T) {
		appRepo := newFakeApprovalRepo()
		paramRepo := newFakeParameterRepo()
		auditRepo := newFakeAuditRepo()
		auditSvc := NewAuditService(auditRepo)
		appSvc := NewApprovalService(appRepo, paramRepo, auditSvc)

		makerID := int64(10) // AO
		p := &domain.Pengajuan{
			ID:             1,
			NomorReferensi: "IMT-20260820-0001",
			PlafonDiajukan: 30_000_000,
			Grade:          2,
			Status:         domain.StatusScored,
			AOID:           makerID,
			DibuatPada:     time.Now(),
		}
		_ = appRepo.SimpanPengajuan(ctx, p)

		// ANL mengajukan ke approval
		err := appSvc.AjukanKeApproval(ctx, p.ID, 20, domain.PeranANL)
		if err != nil {
			t.Fatalf("AjukanKeApproval gagal: %v", err)
		}

		pUpdated, _ := appRepo.AmbilPengajuan(ctx, p.ID)
		if pUpdated.Status != domain.StatusWaitingApprovalL1 {
			t.Fatalf("status diharapkan WAITING_APPROVAL_L1, dapat %s", pUpdated.Status)
		}

		// KCP Approve (actor ID berbeda dengan maker)
		kcpID := int64(30)
		dec, err := appSvc.PutuskanApproval(ctx, p.ID, domain.ApprovalDecisionRequest{
			Keputusan: domain.KeputusanApprove,
			Catatan:   "disetujui sesuai kelayakan",
		}, kcpID, domain.PeranKCP)
		if err != nil {
			t.Fatalf("KCP PutuskanApproval gagal: %v", err)
		}
		if dec.Keputusan != domain.KeputusanApprove {
			t.Fatalf("keputusan diharapkan APPROVE, dapat %s", dec.Keputusan)
		}

		pApproved, _ := appRepo.AmbilPengajuan(ctx, p.ID)
		if pApproved.Status != domain.StatusApproved {
			t.Fatalf("pengajuan 30jt setelah KCP approve diharapkan APPROVED, dapat %s", pApproved.Status)
		}
	})

	// 2. Kasus Plafon Rp 120.000.000 (Berjenjang 2: KCP -> KC)
	t.Run("Plafon 120jt - KC tidak bisa sebelum KCP, setelah KCP jadi WAITING_L2 lalu KC APPROVED", func(t *testing.T) {
		appRepo := newFakeApprovalRepo()
		paramRepo := newFakeParameterRepo()
		auditRepo := newFakeAuditRepo()
		auditSvc := NewAuditService(auditRepo)
		appSvc := NewApprovalService(appRepo, paramRepo, auditSvc)

		p := &domain.Pengajuan{
			ID:             2,
			NomorReferensi: "IMT-20260820-0002",
			PlafonDiajukan: 120_000_000,
			Grade:          2,
			Status:         domain.StatusScored,
			AOID:           10,
			DibuatPada:     time.Now(),
		}
		_ = appRepo.SimpanPengajuan(ctx, p)

		// Ajukan ke approval
		_ = appSvc.AjukanKeApproval(ctx, p.ID, 20, domain.PeranANL)

		// Penegakan BR-02: KC mencoba approve saat status masih WAITING_APPROVAL_L1 -> WAJIB DITOLAK
		kcID := int64(40)
		_, err := appSvc.PutuskanApproval(ctx, p.ID, domain.ApprovalDecisionRequest{
			Keputusan: domain.KeputusanApprove,
		}, kcID, domain.PeranKC)
		if err == nil {
			t.Fatal("KC seharusnya ditolak memutuskan saat pengajuan masih di Level 1 (BR-02)")
		}
		brErr, ok := err.(*domain.BusinessRuleError)
		if !ok || brErr.Rule != "BR-02" {
			t.Fatalf("error diharapkan BusinessRuleError BR-02, dapat %v", err)
		}

		// KCP Approve (Level 1)
		kcpID := int64(30)
		_, err = appSvc.PutuskanApproval(ctx, p.ID, domain.ApprovalDecisionRequest{
			Keputusan: domain.KeputusanApprove,
			Catatan:   "L1 setuju, teruskan ke KC",
		}, kcpID, domain.PeranKCP)
		if err != nil {
			t.Fatalf("KCP PutuskanApproval gagal: %v", err)
		}

		pL2, _ := appRepo.AmbilPengajuan(ctx, p.ID)
		if pL2.Status != domain.StatusWaitingApprovalL2 {
			t.Fatalf("pengajuan 120jt setelah KCP approve diharapkan WAITING_APPROVAL_L2, dapat %s", pL2.Status)
		}

		// KC Approve (Level 2)
		_, err = appSvc.PutuskanApproval(ctx, p.ID, domain.ApprovalDecisionRequest{
			Keputusan: domain.KeputusanApprove,
			Catatan:   "L2 setuju final",
		}, kcID, domain.PeranKC)
		if err != nil {
			t.Fatalf("KC PutuskanApproval gagal: %v", err)
		}

		pApproved, _ := appRepo.AmbilPengajuan(ctx, p.ID)
		if pApproved.Status != domain.StatusApproved {
			t.Fatalf("pengajuan 120jt setelah KC approve diharapkan APPROVED, dapat %s", pApproved.Status)
		}
	})

	// 3. Kasus Plafon Rp 300.000.000 (Berjenjang 3: KCP -> KC -> KOM)
	t.Run("Plafon 300jt - Butuh 3 Level berurutan (KCP -> KC -> KOM)", func(t *testing.T) {
		appRepo := newFakeApprovalRepo()
		paramRepo := newFakeParameterRepo()
		auditRepo := newFakeAuditRepo()
		auditSvc := NewAuditService(auditRepo)
		appSvc := NewApprovalService(appRepo, paramRepo, auditSvc)

		p := &domain.Pengajuan{
			ID:             3,
			NomorReferensi: "IMT-20260820-0003",
			PlafonDiajukan: 300_000_000,
			Grade:          1,
			Status:         domain.StatusScored,
			AOID:           10,
			DibuatPada:     time.Now(),
		}
		_ = appRepo.SimpanPengajuan(ctx, p)

		_ = appSvc.AjukanKeApproval(ctx, p.ID, 20, domain.PeranANL)

		// KOM mencoba memutuskan saat L1 -> Ditolak (BR-02)
		komID := int64(50)
		_, err := appSvc.PutuskanApproval(ctx, p.ID, domain.ApprovalDecisionRequest{
			Keputusan: domain.KeputusanApprove,
		}, komID, domain.PeranKOM)
		if err == nil {
			t.Fatal("KOM seharusnya ditolak saat L1 (BR-02)")
		}

		// KCP approve -> L2
		_, err = appSvc.PutuskanApproval(ctx, p.ID, domain.ApprovalDecisionRequest{Keputusan: domain.KeputusanApprove}, 30, domain.PeranKCP)
		if err != nil {
			t.Fatalf("KCP approve gagal: %v", err)
		}

		// KOM mencoba memutuskan saat L2 -> Ditolak (BR-02)
		_, err = appSvc.PutuskanApproval(ctx, p.ID, domain.ApprovalDecisionRequest{Keputusan: domain.KeputusanApprove}, komID, domain.PeranKOM)
		if err == nil {
			t.Fatal("KOM seharusnya ditolak saat L2 (BR-02)")
		}

		// KC approve -> L3
		_, err = appSvc.PutuskanApproval(ctx, p.ID, domain.ApprovalDecisionRequest{Keputusan: domain.KeputusanApprove}, 40, domain.PeranKC)
		if err != nil {
			t.Fatalf("KC approve gagal: %v", err)
		}

		pL3, _ := appRepo.AmbilPengajuan(ctx, p.ID)
		if pL3.Status != domain.StatusWaitingApprovalL3 {
			t.Fatalf("setelah KC approve diharapkan WAITING_APPROVAL_L3, dapat %s", pL3.Status)
		}

		// KOM approve -> APPROVED
		_, err = appSvc.PutuskanApproval(ctx, p.ID, domain.ApprovalDecisionRequest{Keputusan: domain.KeputusanApprove}, komID, domain.PeranKOM)
		if err != nil {
			t.Fatalf("KOM approve gagal: %v", err)
		}

		pFinal, _ := appRepo.AmbilPengajuan(ctx, p.ID)
		if pFinal.Status != domain.StatusApproved {
			t.Fatalf("setelah KOM approve diharapkan APPROVED, dapat %s", pFinal.Status)
		}
	})
}

// Test AC-11: Pembuat pengajuan (maker) tidak bisa menyetujui pengajuannya sendiri (BR-09).
func TestApproval_AC11_MakerChecker_BR09(t *testing.T) {
	ctx := context.Background()
	appRepo := newFakeApprovalRepo()
	paramRepo := newFakeParameterRepo()
	auditRepo := newFakeAuditRepo()
	auditSvc := NewAuditService(auditRepo)
	appSvc := NewApprovalService(appRepo, paramRepo, auditSvc)

	makerID := int64(99) // User yang merangkap atau membuat pengajuan
	p := &domain.Pengajuan{
		ID:             4,
		NomorReferensi: "IMT-20260820-0004",
		PlafonDiajukan: 30_000_000,
		Grade:          1,
		Status:         domain.StatusWaitingApprovalL1,
		AOID:           makerID,
		DibuatPada:     time.Now(),
	}
	_ = appRepo.SimpanPengajuan(ctx, p)

	// Maker mencoba approve pengajuannya sendiri -> WAJIB DITOLAK (BR-09)
	_, err := appSvc.PutuskanApproval(ctx, p.ID, domain.ApprovalDecisionRequest{
		Keputusan: domain.KeputusanApprove,
	}, makerID, domain.PeranKCP)

	if err == nil {
		t.Fatal("maker tidak boleh menyetujui pengajuannya sendiri (BR-09)")
	}

	brErr, ok := err.(*domain.BusinessRuleError)
	if !ok || brErr.Rule != "BR-09" {
		t.Fatalf("error diharapkan BusinessRuleError BR-09, dapat %v", err)
	}

	// Approver lain mencoba approve -> Berhasil (kasus pembanding)
	approverID := int64(100)
	_, err = appSvc.PutuskanApproval(ctx, p.ID, domain.ApprovalDecisionRequest{
		Keputusan: domain.KeputusanApprove,
	}, approverID, domain.PeranKCP)
	if err != nil {
		t.Fatalf("approver berbeda seharusnya berhasil, tapi error: %v", err)
	}
}

// Test BR-05: Pengajuan Grade 5 tidak boleh diajukan ke approval -> REJECTED_SCORING.
func TestApproval_BR05_Grade5Ditolak(t *testing.T) {
	ctx := context.Background()
	appRepo := newFakeApprovalRepo()
	paramRepo := newFakeParameterRepo()
	auditRepo := newFakeAuditRepo()
	auditSvc := NewAuditService(auditRepo)
	appSvc := NewApprovalService(appRepo, paramRepo, auditSvc)

	p := &domain.Pengajuan{
		ID:             5,
		NomorReferensi: "IMT-20260820-0005",
		PlafonDiajukan: 30_000_000,
		Grade:          5,
		Status:         domain.StatusScored,
		AOID:           10,
		DibuatPada:     time.Now(),
	}
	_ = appRepo.SimpanPengajuan(ctx, p)

	err := appSvc.AjukanKeApproval(ctx, p.ID, 20, domain.PeranANL)
	if err == nil {
		t.Fatal("grade 5 seharusnya ditolak diajukan ke approval (BR-05)")
	}

	brErr, ok := err.(*domain.BusinessRuleError)
	if !ok || brErr.Rule != "BR-05" {
		t.Fatalf("error diharapkan BusinessRuleError BR-05, dapat %v", err)
	}

	pUpdated, _ := appRepo.AmbilPengajuan(ctx, p.ID)
	if pUpdated.Status != domain.StatusRejectedScoring {
		t.Fatalf("status setelah penolakan grade 5 diharapkan REJECTED_SCORING, dapat %s", pUpdated.Status)
	}
}

// Test Keputusan REJECT & RETURN beserta validasi alasan wajib.
func TestApproval_KeputusanRejectDanReturn(t *testing.T) {
	ctx := context.Background()

	t.Run("REJECT tanpa alasan ditolak, dengan alasan status jadi REJECTED", func(t *testing.T) {
		appRepo := newFakeApprovalRepo()
		paramRepo := newFakeParameterRepo()
		auditRepo := newFakeAuditRepo()
		auditSvc := NewAuditService(auditRepo)
		appSvc := NewApprovalService(appRepo, paramRepo, auditSvc)

		p := &domain.Pengajuan{
			ID:             6,
			NomorReferensi: "IMT-20260820-0006",
			PlafonDiajukan: 30_000_000,
			Grade:          2,
			Status:         domain.StatusWaitingApprovalL1,
			AOID:           10,
			DibuatPada:     time.Now(),
		}
		_ = appRepo.SimpanPengajuan(ctx, p)

		// REJECT tanpa alasan -> Ditolak
		_, err := appSvc.PutuskanApproval(ctx, p.ID, domain.ApprovalDecisionRequest{
			Keputusan: domain.KeputusanReject,
			Alasan:    "",
		}, 30, domain.PeranKCP)
		if err == nil {
			t.Fatal("REJECT tanpa alasan seharusnya ditolak")
		}

		// REJECT dengan alasan -> Berhasil
		_, err = appSvc.PutuskanApproval(ctx, p.ID, domain.ApprovalDecisionRequest{
			Keputusan: domain.KeputusanReject,
			Alasan:    "Kapasitas usaha meragukan dan tidak memenuhi syarat",
		}, 30, domain.PeranKCP)
		if err != nil {
			t.Fatalf("REJECT dengan alasan gagal: %v", err)
		}

		pUpdated, _ := appRepo.AmbilPengajuan(ctx, p.ID)
		if pUpdated.Status != domain.StatusRejected {
			t.Fatalf("status diharapkan REJECTED, dapat %s", pUpdated.Status)
		}
	})

	t.Run("RETURN tanpa alasan ditolak, dengan alasan status jadi RETURNED", func(t *testing.T) {
		appRepo := newFakeApprovalRepo()
		paramRepo := newFakeParameterRepo()
		auditRepo := newFakeAuditRepo()
		auditSvc := NewAuditService(auditRepo)
		appSvc := NewApprovalService(appRepo, paramRepo, auditSvc)

		p := &domain.Pengajuan{
			ID:             7,
			NomorReferensi: "IMT-20260820-0007",
			PlafonDiajukan: 30_000_000,
			Grade:          2,
			Status:         domain.StatusWaitingApprovalL1,
			AOID:           10,
			DibuatPada:     time.Now(),
		}
		_ = appRepo.SimpanPengajuan(ctx, p)

		// RETURN tanpa alasan -> Ditolak
		_, err := appSvc.PutuskanApproval(ctx, p.ID, domain.ApprovalDecisionRequest{
			Keputusan: domain.KeputusanReturn,
			Alasan:    "",
		}, 30, domain.PeranKCP)
		if err == nil {
			t.Fatal("RETURN tanpa alasan seharusnya ditolak")
		}

		// RETURN dengan alasan -> Berhasil
		_, err = appSvc.PutuskanApproval(ctx, p.ID, domain.ApprovalDecisionRequest{
			Keputusan: domain.KeputusanReturn,
			Alasan:    "Foto tempat usaha kurang jelas, minta verifikasi ulang ke AO",
		}, 30, domain.PeranKCP)
		if err != nil {
			t.Fatalf("RETURN dengan alasan gagal: %v", err)
		}

		pUpdated, _ := appRepo.AmbilPengajuan(ctx, p.ID)
		if pUpdated.Status != domain.StatusReturned {
			t.Fatalf("status diharapkan RETURNED, dapat %s", pUpdated.Status)
		}
	})
}
