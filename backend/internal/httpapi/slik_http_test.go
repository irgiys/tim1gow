package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/irgiys/tim1gow/backend/internal/config"
	"github.com/irgiys/tim1gow/backend/internal/domain"
	"github.com/irgiys/tim1gow/backend/internal/service"
	"github.com/irgiys/tim1gow/backend/internal/slik"
)

type fakePemanggilSlikHTTP struct {
	hasil slik.Hasil
	err   error
}

func (f *fakePemanggilSlikHTTP) Inquiry(_ context.Context, _ string) (slik.Hasil, error) {
	return f.hasil, f.err
}

type fakeSlikRepoHTTP struct {
	disimpan []service.HasilSlik
}

func (f *fakeSlikRepoHTTP) Simpan(_ context.Context, h *service.HasilSlik) error {
	h.ID = int64(len(f.disimpan) + 1)
	f.disimpan = append(f.disimpan, *h)
	return nil
}

func (f *fakeSlikRepoHTTP) TerakhirSukses(_ context.Context, pengajuanID int64) (service.HasilSlik, error) {
	for i := len(f.disimpan) - 1; i >= 0; i-- {
		if f.disimpan[i].PengajuanID == pengajuanID && f.disimpan[i].StatusPanggilan == "SUKSES" {
			return f.disimpan[i], nil
		}
	}
	return service.HasilSlik{}, service.ErrTidakDitemukan
}

func ptrIntHTTP(v int) *int {
	return &v
}

func ptrInt64HTTP(v int64) *int64 {
	return &v
}

func susunRouterSlikUji(client service.PemanggilSlik, pjnRepo *fakePengajuanRepoHTTP) http.Handler {
	slikRepo := &fakeSlikRepoHTTP{}
	slikSvc := service.NewSlikService(service.OpsiSlikService{
		SlikRepo:        slikRepo,
		Pengajuan:       pjnRepo,
		Client:          client,
		MasaBerlakuHari: 30,
	})
	slikH := NewSlikHandler(slikSvc)

	return NewRouterLengkap(
		config.Config{AppEnv: "test", JWTSecret: string(secretMw)},
		nil, nil, nil, nil, nil, nil, slikH, pemeriksaPalsu{aktif: true},
	)
}

// TestRoute_FR05_AC05_Kolektibilitas4_OtomatisRejectedSlik (AC-05).
// Panggilan SLIK mengembalikan kolektibilitas 4 -> HTTP 200, status REJECTED_SLIK,
// tidak memuat NIK di respons (BR-11).
func TestRoute_FR05_AC05_Kolektibilitas4_OtomatisRejectedSlik(t *testing.T) {
	tglData := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	fakeClient := &fakePemanggilSlikHTTP{
		hasil: slik.Hasil{
			Status:               slik.StatusSukses,
			Kolektibilitas:       ptrIntHTTP(4),
			JumlahFasilitasAktif: ptrIntHTTP(1),
			TotalBakiDebet:       ptrInt64HTTP(15_000_000),
			TanggalData:          &tglData,
			ReferenceID:          "REF-SLIK-004",
		},
	}
	pjnRepo := newFakePengajuanRepoHTTP()
	pjnRepo.baris[1] = service.Pengajuan{
		ID:             1,
		NomorReferensi: "IMT-20260821-0001",
		NamaNasabah:    "Nasabah Kol 4",
		NIK:            "3201012345670004",
		Status:         "VERIFYING",
	}

	router := susunRouterSlikUji(fakeClient, pjnRepo)

	rec := kirim(t, router, http.MethodPost, "/api/pengajuan/1/slik", domain.PeranANL, 20, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, ingin 200 (body=%s)", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}

	if resp["status"] != string(domain.StatusRejectedSlik) {
		t.Errorf("status = %v, ingin %s", resp["status"], domain.StatusRejectedSlik)
	}
	if kol, ok := resp["kolektibilitas"].(float64); !ok || int(kol) != 4 {
		t.Errorf("kolektibilitas = %v, ingin 4", resp["kolektibilitas"])
	}

	// BR-11: NIK tidak boleh muncul di respons
	if strings.Contains(rec.Body.String(), "3201012345670004") {
		t.Errorf("respons memuat NIK nasabah (BR-11): %s", rec.Body.String())
	}

	// Status di database pengajuan diperbarui
	if pjnRepo.baris[1].Status != string(domain.StatusRejectedSlik) {
		t.Errorf("status di repo = %s, ingin %s", pjnRepo.baris[1].Status, domain.StatusRejectedSlik)
	}
}

// TestRoute_FR05_AC06_Kolektibilitas2_LanjutSlikChecked (AC-06).
func TestRoute_FR05_AC06_Kolektibilitas2_LanjutSlikChecked(t *testing.T) {
	tglData := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	fakeClient := &fakePemanggilSlikHTTP{
		hasil: slik.Hasil{
			Status:               slik.StatusSukses,
			Kolektibilitas:       ptrIntHTTP(2),
			JumlahFasilitasAktif: ptrIntHTTP(2),
			TotalBakiDebet:       ptrInt64HTTP(5_000_000),
			TanggalData:          &tglData,
			ReferenceID:          "REF-SLIK-002",
		},
	}
	pjnRepo := newFakePengajuanRepoHTTP()
	pjnRepo.baris[2] = service.Pengajuan{
		ID:             2,
		NomorReferensi: "IMT-20260821-0002",
		NamaNasabah:    "Nasabah Kol 2",
		NIK:            "3201012345670002",
		Status:         "VERIFYING",
	}

	router := susunRouterSlikUji(fakeClient, pjnRepo)

	rec := kirim(t, router, http.MethodPost, "/api/pengajuan/2/slik", domain.PeranANL, 20, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, ingin 200 (body=%s)", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["status"] != string(domain.StatusSlikChecked) {
		t.Errorf("status = %v, ingin %s", resp["status"], domain.StatusSlikChecked)
	}
	if pjnRepo.baris[2].Status != string(domain.StatusSlikChecked) {
		t.Errorf("status di repo = %s, ingin %s", pjnRepo.baris[2].Status, domain.StatusSlikChecked)
	}
}

// TestRoute_FR05_Kolektibilitas1_LanjutNormal.
func TestRoute_FR05_Kolektibilitas1_LanjutNormal(t *testing.T) {
	tglData := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	fakeClient := &fakePemanggilSlikHTTP{
		hasil: slik.Hasil{
			Status:               slik.StatusSukses,
			Kolektibilitas:       ptrIntHTTP(1),
			JumlahFasilitasAktif: ptrIntHTTP(0),
			TotalBakiDebet:       ptrInt64HTTP(0),
			TanggalData:          &tglData,
			ReferenceID:          "REF-SLIK-001",
		},
	}
	pjnRepo := newFakePengajuanRepoHTTP()
	pjnRepo.baris[3] = service.Pengajuan{
		ID:             3,
		NomorReferensi: "IMT-20260821-0003",
		NamaNasabah:    "Nasabah Kol 1",
		NIK:            "3201012345670003",
		Status:         "VERIFYING",
	}

	router := susunRouterSlikUji(fakeClient, pjnRepo)

	rec := kirim(t, router, http.MethodPost, "/api/pengajuan/3/slik", domain.PeranANL, 20, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, ingin 200 (body=%s)", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["status"] != string(domain.StatusSlikChecked) {
		t.Errorf("status = %v, ingin %s", resp["status"], domain.StatusSlikChecked)
	}
}

// TestRoute_FR05_LayananSlikTidakTersedia_502 (AGENTS.md 4.3).
// Ketika mock SLIK mengembalikan 503 / timeout, backend merespons 502 Bad Gateway
// dengan error SLIK_UNAVAILABLE dan pengajuan tidak maju.
func TestRoute_FR05_LayananSlikTidakTersedia_502(t *testing.T) {
	fakeClient := &fakePemanggilSlikHTTP{
		hasil: slik.Hasil{Status: slik.StatusLayananTidakAda},
		err:   errors.New("mock slik 503 service unavailable"),
	}
	pjnRepo := newFakePengajuanRepoHTTP()
	pjnRepo.baris[4] = service.Pengajuan{
		ID:             4,
		NomorReferensi: "IMT-20260821-0004",
		NamaNasabah:    "Nasabah Uji Error",
		NIK:            "3201012345670004",
		Status:         "VERIFYING",
	}

	router := susunRouterSlikUji(fakeClient, pjnRepo)

	rec := kirim(t, router, http.MethodPost, "/api/pengajuan/4/slik", domain.PeranANL, 20, nil)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, ingin 502 (body=%s)", rec.Code, rec.Body.String())
	}

	var resp errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if resp.Error != "SLIK_UNAVAILABLE" {
		t.Errorf("error code = %q, ingin SLIK_UNAVAILABLE", resp.Error)
	}

	// Status pengajuan TIDAK berubah
	if pjnRepo.baris[4].Status != "VERIFYING" {
		t.Errorf("status di repo maju ke %s saat SLIK gagal", pjnRepo.baris[4].Status)
	}
}

// TestRoute_FR05_HanyaBolehANL_PeranLainDitolak403 (AC-02 & SDD BAB 5).
func TestRoute_FR05_HanyaBolehANL_PeranLainDitolak403(t *testing.T) {
	fakeClient := &fakePemanggilSlikHTTP{
		hasil: slik.Hasil{
			Status:         slik.StatusSukses,
			Kolektibilitas: ptrIntHTTP(1),
		},
	}
	pjnRepo := newFakePengajuanRepoHTTP()
	pjnRepo.baris[1] = service.Pengajuan{ID: 1, Status: "VERIFYING"}

	router := susunRouterSlikUji(fakeClient, pjnRepo)

	peranDitolak := []domain.Peran{
		domain.PeranAO,
		domain.PeranKCP,
		domain.PeranKC,
		domain.PeranKOM,
		domain.PeranADM,
	}

	for _, p := range peranDitolak {
		t.Run(string(p), func(t *testing.T) {
			rec := kirim(t, router, http.MethodPost, "/api/pengajuan/1/slik", p, 10, nil)
			if rec.Code != http.StatusForbidden {
				t.Errorf("peran %s: status = %d, ingin 403", p, rec.Code)
			}
		})
	}
}
