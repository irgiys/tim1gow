package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/irgiys/tim1gow/backend/internal/domain"
	"github.com/irgiys/tim1gow/backend/internal/service"
	"github.com/irgiys/tim1gow/backend/internal/slik"
)

type fakePemanggilSlik struct {
	hasil slik.Hasil
	err   error
}

func (f *fakePemanggilSlik) Inquiry(_ context.Context, _ string) (slik.Hasil, error) {
	return f.hasil, f.err
}

type fakeSlikRepo struct {
	disimpan []service.HasilSlik
}

func (f *fakeSlikRepo) Simpan(_ context.Context, h *service.HasilSlik) error {
	h.ID = int64(len(f.disimpan) + 1)
	f.disimpan = append(f.disimpan, *h)
	return nil
}

func (f *fakeSlikRepo) TerakhirSukses(_ context.Context, pengajuanID int64) (service.HasilSlik, error) {
	for i := len(f.disimpan) - 1; i >= 0; i-- {
		if f.disimpan[i].PengajuanID == pengajuanID && f.disimpan[i].StatusPanggilan == "SUKSES" {
			return f.disimpan[i], nil
		}
	}
	return service.HasilSlik{}, service.ErrTidakDitemukan
}

type fakeSlikPengajuanRepo struct {
	pengajuan map[int64]service.Pengajuan
}

func (f *fakeSlikPengajuanRepo) CariID(_ context.Context, id int64) (service.Pengajuan, error) {
	p, ok := f.pengajuan[id]
	if !ok {
		return service.Pengajuan{}, service.ErrTidakDitemukan
	}
	return p, nil
}

func (f *fakeSlikPengajuanRepo) Perbarui(_ context.Context, p *service.Pengajuan) error {
	if _, ok := f.pengajuan[p.ID]; !ok {
		return service.ErrTidakDitemukan
	}
	f.pengajuan[p.ID] = *p
	return nil
}

type fakeAuditSvc struct {
	entri []domain.CatatAuditInput
}

func (f *fakeAuditSvc) Catat(_ context.Context, input domain.CatatAuditInput) error {
	f.entri = append(f.entri, input)
	return nil
}

func (f *fakeAuditSvc) AmbilRiwayatPengajuan(_ context.Context, _ int64) ([]domain.AuditTrailEntry, error) {
	return nil, nil
}

func (f *fakeAuditSvc) AmbilSemua(_ context.Context, _, _ int) ([]domain.AuditTrailEntry, error) {
	return nil, nil
}

func ptrInt(v int) *int {
	return &v
}

func ptrInt64(v int64) *int64 {
	return &v
}

// TestAC05_Kolektibilitas4OtomatisRejectedSlik (AC-05 & Tabel 4.2).
// Nasabah dengan SLIK kolektibilitas 4 (dan 3, 5) otomatis berstatus
// REJECTED_SLIK tanpa melalui jalur approval.
func TestAC05_Kolektibilitas4OtomatisRejectedSlik(t *testing.T) {
	kasus := []struct {
		nama string
		kol  int
	}{
		{nama: "Kolektibilitas 3", kol: 3},
		{nama: "Kolektibilitas 4 (AC-05)", kol: 4},
		{nama: "Kolektibilitas 5", kol: 5},
	}

	for _, tc := range kasus {
		t.Run(tc.nama, func(t *testing.T) {
			tglData := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
			fakeClient := &fakePemanggilSlik{
				hasil: slik.Hasil{
					Status:               slik.StatusSukses,
					Kolektibilitas:       ptrInt(tc.kol),
					JumlahFasilitasAktif: ptrInt(1),
					TotalBakiDebet:       ptrInt64(15000000),
					TanggalData:          &tglData,
					ReferenceID:          "REF-TEST-01",
				},
			}
			fakeRepo := &fakeSlikRepo{}
			fakePjn := &fakeSlikPengajuanRepo{
				pengajuan: map[int64]service.Pengajuan{
					101: {ID: 101, NIK: "3201012345670001", Status: "VERIFYING"},
				},
			}
			fakeAud := &fakeAuditSvc{}

			svc := service.NewSlikService(service.OpsiSlikService{
				SlikRepo:        fakeRepo,
				Pengajuan:       fakePjn,
				Client:          fakeClient,
				Audit:           fakeAud,
				MasaBerlakuHari: 30,
			})

			hasil, err := svc.Jalankan(context.Background(), 101, 20)
			if err != nil {
				t.Fatalf("tidak mengharapkan error: %v", err)
			}

			if hasil.Kolektibilitas != tc.kol {
				t.Errorf("kolektibilitas = %d, ingin %d", hasil.Kolektibilitas, tc.kol)
			}
			if !hasil.Ditolak {
				t.Errorf("hasil.Ditolak = false, ingin true untuk kolektibilitas %d", tc.kol)
			}
			if hasil.StatusPengajuan != string(domain.StatusRejectedSlik) {
				t.Errorf("status = %q, ingin %q", hasil.StatusPengajuan, domain.StatusRejectedSlik)
			}

			// Pastikan tersimpan di DB pengajuan
			pjn, err := fakePjn.CariID(context.Background(), 101)
			if err != nil {
				t.Fatalf("gagal mencari pengajuan: %v", err)
			}
			if pjn.Status != string(domain.StatusRejectedSlik) {
				t.Errorf("status di repo = %q, ingin %q", pjn.Status, domain.StatusRejectedSlik)
			}

			// BR-10: Audit trail tercatat dengan aktor ANL dan perubahan status
			if len(fakeAud.entri) != 1 {
				t.Fatalf("jumlah entri audit = %d, ingin 1", len(fakeAud.entri))
			}
			if fakeAud.entri[0].Aksi != domain.AksiSlikCheck {
				t.Errorf("aksi audit = %q, ingin %q", fakeAud.entri[0].Aksi, domain.AksiSlikCheck)
			}
			if fakeAud.entri[0].StatusSebelum != domain.StatusPengajuan("VERIFYING") {
				t.Errorf("status sebelum = %q, ingin VERIFYING", fakeAud.entri[0].StatusSebelum)
			}
			if fakeAud.entri[0].StatusSesudah != domain.StatusRejectedSlik {
				t.Errorf("status sesudah = %q, ingin %q", fakeAud.entri[0].StatusSesudah, domain.StatusRejectedSlik)
			}
			if fakeAud.entri[0].ActorID != 20 {
				t.Errorf("actor id = %d, ingin 20", fakeAud.entri[0].ActorID)
			}

			// BR-11: NIK tidak boleh muncul di catatan audit
			if strings.Contains(fakeAud.entri[0].Catatan, "3201012345670001") {
				t.Errorf("catatan audit membocorkan NIK: %q", fakeAud.entri[0].Catatan)
			}
		})
	}
}

// TestAC06_Kolektibilitas2LanjutTetapiGradeMinimal3 (AC-06 & Tabel 4.2).
// Nasabah dengan SLIK kolektibilitas 2 dapat lanjut (SLIK_CHECKED), tetapi
// ditandai wajib grade minimal 3 dan wajib catatan analis.
func TestAC06_Kolektibilitas2LanjutTetapiGradeMinimal3(t *testing.T) {
	tglData := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	fakeClient := &fakePemanggilSlik{
		hasil: slik.Hasil{
			Status:               slik.StatusSukses,
			Kolektibilitas:       ptrInt(2),
			JumlahFasilitasAktif: ptrInt(2),
			TotalBakiDebet:       ptrInt64(5000000),
			TanggalData:          &tglData,
			ReferenceID:          "REF-TEST-02",
		},
	}
	fakeRepo := &fakeSlikRepo{}
	fakePjn := &fakeSlikPengajuanRepo{
		pengajuan: map[int64]service.Pengajuan{
			102: {ID: 102, NIK: "3201012345670002", Status: "VERIFYING"},
		},
	}
	fakeAud := &fakeAuditSvc{}

	svc := service.NewSlikService(service.OpsiSlikService{
		SlikRepo:        fakeRepo,
		Pengajuan:       fakePjn,
		Client:          fakeClient,
		Audit:           fakeAud,
		MasaBerlakuHari: 30,
	})

	hasil, err := svc.Jalankan(context.Background(), 102, 20)
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}

	if hasil.Kolektibilitas != 2 {
		t.Errorf("kolektibilitas = %d, ingin 2", hasil.Kolektibilitas)
	}
	if hasil.Ditolak {
		t.Errorf("hasil.Ditolak = true, ingin false untuk kolektibilitas 2")
	}
	if hasil.GradeMinimal != 3 {
		t.Errorf("grade minimal = %d, ingin 3", hasil.GradeMinimal)
	}
	if !hasil.WajibCatatanAnalis {
		t.Errorf("wajib catatan analis = false, ingin true")
	}
	if hasil.StatusPengajuan != string(domain.StatusSlikChecked) {
		t.Errorf("status = %q, ingin %q", hasil.StatusPengajuan, domain.StatusSlikChecked)
	}

	// Status pengajuan maju ke SLIK_CHECKED
	pjn, _ := fakePjn.CariID(context.Background(), 102)
	if pjn.Status != string(domain.StatusSlikChecked) {
		t.Errorf("status di repo = %q, ingin %q", pjn.Status, domain.StatusSlikChecked)
	}
}

// TestKolektibilitas1_LanjutNormal (Tabel 4.2).
func TestKolektibilitas1_LanjutNormal(t *testing.T) {
	tglData := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	fakeClient := &fakePemanggilSlik{
		hasil: slik.Hasil{
			Status:               slik.StatusSukses,
			Kolektibilitas:       ptrInt(1),
			JumlahFasilitasAktif: ptrInt(0),
			TotalBakiDebet:       ptrInt64(0),
			TanggalData:          &tglData,
			ReferenceID:          "REF-TEST-03",
		},
	}
	fakeRepo := &fakeSlikRepo{}
	fakePjn := &fakeSlikPengajuanRepo{
		pengajuan: map[int64]service.Pengajuan{
			103: {ID: 103, NIK: "3201012345670003", Status: "VERIFYING"},
		},
	}

	svc := service.NewSlikService(service.OpsiSlikService{
		SlikRepo:  fakeRepo,
		Pengajuan: fakePjn,
		Client:    fakeClient,
	})

	hasil, err := svc.Jalankan(context.Background(), 103, 20)
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}

	if hasil.Kolektibilitas != 1 {
		t.Errorf("kolektibilitas = %d, ingin 1", hasil.Kolektibilitas)
	}
	if hasil.Ditolak {
		t.Errorf("hasil.Ditolak = true, ingin false")
	}
	if hasil.GradeMinimal != 0 {
		t.Errorf("grade minimal = %d, ingin 0", hasil.GradeMinimal)
	}
	if hasil.WajibCatatanAnalis {
		t.Errorf("wajib catatan analis = true, ingin false")
	}
	if hasil.StatusPengajuan != string(domain.StatusSlikChecked) {
		t.Errorf("status = %q, ingin %q", hasil.StatusPengajuan, domain.StatusSlikChecked)
	}
}

// TestBR04_MasaBerlaku30Hari (BR-04).
// Hasil SLIK berlaku 30 hari dari tanggal_data.
func TestBR04_MasaBerlaku30Hari(t *testing.T) {
	tglData := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	expectedBerlaku := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)

	fakeClient := &fakePemanggilSlik{
		hasil: slik.Hasil{
			Status:               slik.StatusSukses,
			Kolektibilitas:       ptrInt(1),
			JumlahFasilitasAktif: ptrInt(0),
			TotalBakiDebet:       ptrInt64(0),
			TanggalData:          &tglData,
			ReferenceID:          "REF-TEST-04",
		},
	}
	fakeRepo := &fakeSlikRepo{}
	fakePjn := &fakeSlikPengajuanRepo{
		pengajuan: map[int64]service.Pengajuan{
			104: {ID: 104, NIK: "3201012345670004", Status: "VERIFYING"},
		},
	}

	svc := service.NewSlikService(service.OpsiSlikService{
		SlikRepo:        fakeRepo,
		Pengajuan:       fakePjn,
		Client:          fakeClient,
		MasaBerlakuHari: 30,
	})

	hasil, err := svc.Jalankan(context.Background(), 104, 20)
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if hasil.BerlakuSampai == nil {
		t.Fatal("hasil.BerlakuSampai nil, ingin terisi")
	}
	if !hasil.BerlakuSampai.Equal(expectedBerlaku) {
		t.Errorf("berlaku sampai = %v, ingin %v", *hasil.BerlakuSampai, expectedBerlaku)
	}

	// Pastikan tercatat di repository
	if len(fakeRepo.disimpan) != 1 {
		t.Fatalf("jumlah disimpan = %d, ingin 1", len(fakeRepo.disimpan))
	}
	if fakeRepo.disimpan[0].BerlakuSampai == nil {
		t.Fatal("repo berlaku sampai nil")
	}
	if !fakeRepo.disimpan[0].BerlakuSampai.Equal(expectedBerlaku) {
		t.Errorf("repo berlaku sampai = %v, ingin %v", *fakeRepo.disimpan[0].BerlakuSampai, expectedBerlaku)
	}
}

// TestSLIK_KegagalanHuluTidakDianggapBersih (Larangan 15).
// Kegagalan SLIK (503 / timeout) harus mengembalikan error ErrSlikTidakTersedia,
// status pengajuan TIDAK maju, dan percobaan tetap dicatat ke hasil_slik.
func TestSLIK_KegagalanHuluTidakDianggapBersih(t *testing.T) {
	kasus := []struct {
		nama      string
		status    slik.StatusPanggilan
		clientErr error
	}{
		{nama: "Service Unavailable", status: slik.StatusLayananTidakAda},
		{nama: "Timeout", status: slik.StatusTimeout},
		{nama: "HTTP Client Error", status: slik.StatusTimeout, clientErr: errors.New("connection refused")},
	}

	for _, tc := range kasus {
		t.Run(tc.nama, func(t *testing.T) {
			fakeClient := &fakePemanggilSlik{
				hasil: slik.Hasil{Status: tc.status},
				err:   tc.clientErr,
			}
			fakeRepo := &fakeSlikRepo{}
			fakePjn := &fakeSlikPengajuanRepo{
				pengajuan: map[int64]service.Pengajuan{
					105: {ID: 105, NIK: "3201012345670005", Status: "VERIFYING"},
				},
			}

			svc := service.NewSlikService(service.OpsiSlikService{
				SlikRepo:  fakeRepo,
				Pengajuan: fakePjn,
				Client:    fakeClient,
			})

			_, err := svc.Jalankan(context.Background(), 105, 20)
			if err == nil {
				t.Fatal("mengharapkan error, dapat nil")
			}
			if !errors.Is(err, service.ErrSlikTidakTersedia) {
				t.Errorf("error = %v, ingin membungkus ErrSlikTidakTersedia", err)
			}

			// Status pengajuan TIDAK berubah
			pjn, _ := fakePjn.CariID(context.Background(), 105)
			if pjn.Status != "VERIFYING" {
				t.Errorf("status pengajuan maju ke %q saat SLIK gagal", pjn.Status)
			}

			// Percobaan gagal TETAP dicatat di hasil_slik
			if len(fakeRepo.disimpan) != 1 {
				t.Fatalf("jumlah disimpan = %d, ingin 1", len(fakeRepo.disimpan))
			}
			if fakeRepo.disimpan[0].Kolektibilitas != nil {
				t.Errorf("kolektibilitas = %v, ingin nil untuk baris gagal", *fakeRepo.disimpan[0].Kolektibilitas)
			}
		})
	}
}

// TestSLIK_NIKTidakDitemukan.
func TestSLIK_NIKTidakDitemukan(t *testing.T) {
	fakeClient := &fakePemanggilSlik{
		hasil: slik.Hasil{Status: slik.StatusNIKTidakDitemukan},
	}
	fakeRepo := &fakeSlikRepo{}
	fakePjn := &fakeSlikPengajuanRepo{
		pengajuan: map[int64]service.Pengajuan{
			106: {ID: 106, NIK: "3201012345670006", Status: "VERIFYING"},
		},
	}

	svc := service.NewSlikService(service.OpsiSlikService{
		SlikRepo:  fakeRepo,
		Pengajuan: fakePjn,
		Client:    fakeClient,
	})

	_, err := svc.Jalankan(context.Background(), 106, 20)
	if err == nil {
		t.Fatal("mengharapkan error, dapat nil")
	}
	if !errors.Is(err, service.ErrNIKTidakDitemukanSlik) {
		t.Errorf("error = %v, ingin ErrNIKTidakDitemukanSlik", err)
	}

	pjn, _ := fakePjn.CariID(context.Background(), 106)
	if pjn.Status != "VERIFYING" {
		t.Errorf("status = %q, ingin VERIFYING", pjn.Status)
	}
}
