package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/irgiys/tim1gow/backend/internal/config"
	"github.com/irgiys/tim1gow/backend/internal/domain"
	"github.com/irgiys/tim1gow/backend/internal/service"
)

type fakeApprovalRepoForHTTP struct {
	pengajuan map[int64]*domain.Pengajuan
	keputusan map[int64][]domain.KeputusanApprovalRecord
}

func newFakeApprovalRepoForHTTP() *fakeApprovalRepoForHTTP {
	return &fakeApprovalRepoForHTTP{
		pengajuan: make(map[int64]*domain.Pengajuan),
		keputusan: make(map[int64][]domain.KeputusanApprovalRecord),
	}
}

func (f *fakeApprovalRepoForHTTP) AmbilPengajuan(_ context.Context, id int64) (*domain.Pengajuan, error) {
	p, ok := f.pengajuan[id]
	if !ok {
		return nil, nil
	}
	cp := *p
	return &cp, nil
}

func (f *fakeApprovalRepoForHTTP) UpdateStatusPengajuan(_ context.Context, id int64, statusBaru domain.StatusPengajuan) error {
	if p, ok := f.pengajuan[id]; ok {
		p.Status = statusBaru
	}
	return nil
}

func (f *fakeApprovalRepoForHTTP) SimpanPengajuan(_ context.Context, p *domain.Pengajuan) error {
	f.pengajuan[p.ID] = p
	return nil
}

func (f *fakeApprovalRepoForHTTP) SimpanKeputusan(_ context.Context, k *domain.KeputusanApprovalRecord) error {
	f.keputusan[k.PengajuanID] = append(f.keputusan[k.PengajuanID], *k)
	return nil
}

func (f *fakeApprovalRepoForHTTP) AmbilKeputusanByPengajuan(_ context.Context, pengajuanID int64) ([]domain.KeputusanApprovalRecord, error) {
	list := f.keputusan[pengajuanID]
	out := make([]domain.KeputusanApprovalRecord, len(list))
	copy(out, list)
	return out, nil
}

type fakeParameterRepoForHTTP struct {
	ambang []domain.AmbangApproval
}

func (f *fakeParameterRepoForHTTP) KomponenSkor() ([]domain.ParameterKomponenSkor, error) {
	return nil, nil
}

func (f *fakeParameterRepoForHTTP) SkorRiwayatSlik(_ int) (float64, bool, error) {
	return 0, false, nil
}

func (f *fakeParameterRepoForHTTP) Umum(_ string) (float64, bool, error) {
	return 0, false, nil
}

func (f *fakeParameterRepoForHTTP) RentangMarginPerGrade() ([]domain.RentangMargin, error) {
	return nil, nil
}

func (f *fakeParameterRepoForHTTP) RentangMargin(_ int) (domain.RentangMargin, bool, error) {
	return domain.RentangMargin{}, false, nil
}

func (f *fakeParameterRepoForHTTP) AmbangApproval(totalPlafon int64) (domain.AmbangApproval, bool, error) {
	for _, a := range f.ambang {
		if totalPlafon >= a.PlafonMin && totalPlafon <= a.PlafonMaks {
			return a, true, nil
		}
	}
	return domain.AmbangApproval{}, false, nil
}

func (f *fakeParameterRepoForHTTP) SemuaAmbangApproval() ([]domain.AmbangApproval, error) {
	return f.ambang, nil
}

// Test HTTP AC-11: Maker != Checker menghasilkan HTTP 422 dengan field "rule": "BR-09".
func TestApprovalHTTP_AC11_MakerChecker_BR09(t *testing.T) {
	appRepo := newFakeApprovalRepoForHTTP()
	auditRepo := &fakeAuditRepoForHTTP{}
	paramRepo := &fakeParameterRepoForHTTP{
		ambang: []domain.AmbangApproval{
			{PlafonMin: 5000000, PlafonMaks: 50000000, Level: []domain.Peran{domain.PeranKCP}},
		},
	}

	makerID := int64(10)
	p := &domain.Pengajuan{
		ID:             1,
		NomorReferensi: "IMT-20260820-0001",
		TotalPlafon:    30_000_000,
		Grade:          1,
		Status:         domain.StatusWaitingApprovalL1,
		CreatedBy:      makerID,
		CreatedAt:      time.Now(),
	}
	_ = appRepo.SimpanPengajuan(context.Background(), p)

	auditSvc := service.NewAuditService(auditRepo)
	appSvc := service.NewApprovalService(appRepo, paramRepo, auditSvc)
	appH := NewApprovalHandler(appSvc)

	cfg := config.Config{AppEnv: "test"}
	handler := NewRouterWithHandlers(cfg, nil, appH, nil)

	body, _ := json.Marshal(domain.ApprovalDecisionRequest{
		Keputusan: domain.KeputusanApprove,
	})

	// Maker mencoba approve -> HTTP 422 dengan rule: "BR-09"
	req := httptest.NewRequest(http.MethodPost, "/api/pengajuan/1/approval", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Actor-ID", "10")
	req.Header.Set("X-Actor-Role", "KCP")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status diharapkan 422 Unprocessable Entity, dapat %d", w.Code)
	}

	var res errorResponse
	_ = json.NewDecoder(w.Body).Decode(&res)
	if res.Rule != "BR-09" {
		t.Fatalf("field rule diharapkan 'BR-09', dapat '%s'", res.Rule)
	}
}

// Test HTTP AC-10 / BR-02: Level 2 mencoba approve sebelum Level 1 -> HTTP 422 dengan field "rule": "BR-02".
func TestApprovalHTTP_AC10_SequentialApproval_BR02(t *testing.T) {
	appRepo := newFakeApprovalRepoForHTTP()
	auditRepo := &fakeAuditRepoForHTTP{}
	paramRepo := &fakeParameterRepoForHTTP{
		ambang: []domain.AmbangApproval{
			{PlafonMin: 50000001, PlafonMaks: 200000000, Level: []domain.Peran{domain.PeranKCP, domain.PeranKC}},
		},
	}

	p := &domain.Pengajuan{
		ID:             2,
		NomorReferensi: "IMT-20260820-0002",
		TotalPlafon:    120_000_000,
		Grade:          2,
		Status:         domain.StatusWaitingApprovalL1,
		CreatedBy:      10,
		CreatedAt:      time.Now(),
	}
	_ = appRepo.SimpanPengajuan(context.Background(), p)

	auditSvc := service.NewAuditService(auditRepo)
	appSvc := service.NewApprovalService(appRepo, paramRepo, auditSvc)
	appH := NewApprovalHandler(appSvc)

	cfg := config.Config{AppEnv: "test"}
	handler := NewRouterWithHandlers(cfg, nil, appH, nil)

	body, _ := json.Marshal(domain.ApprovalDecisionRequest{
		Keputusan: domain.KeputusanApprove,
	})

	// KC mencoba approve pengajuan yang masih L1 -> HTTP 422 dengan rule: "BR-02"
	req := httptest.NewRequest(http.MethodPost, "/api/pengajuan/2/approval", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Actor-ID", "40")
	req.Header.Set("X-Actor-Role", "KC")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status diharapkan 422 Unprocessable Entity, dapat %d", w.Code)
	}

	var res errorResponse
	_ = json.NewDecoder(w.Body).Decode(&res)
	if res.Rule != "BR-02" {
		t.Fatalf("field rule diharapkan 'BR-02', dapat '%s'", res.Rule)
	}
}
