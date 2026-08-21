package service

import (
	"context"
	"strings"
	"time"

	"github.com/irgiys/tim1gow/backend/internal/domain"
)

// SurveiService memuat aturan perekaman survei lapangan / OTS (FR-04).
type SurveiService struct {
	repo    SurveiRepository
	sekaran func() time.Time
	// audit mencatat jejak BR-10. Boleh nil pada test unit.
	audit   AuditService
}

func NewSurveiService(repo SurveiRepository) *SurveiService {
	return &SurveiService{repo: repo, sekaran: time.Now}
}

// DenganWaktu mengganti sumber waktu; dipakai test.
func (s *SurveiService) DenganWaktu(f func() time.Time) *SurveiService {
	s.sekaran = f
	return s
}

// DenganAudit memasang pencatat audit (BR-10). Kegagalan mencatat membuat
// operasi gagal, bukan diabaikan.
func (s *SurveiService) DenganAudit(a AuditService) *SurveiService {
	s.audit = a
	return s
}

// InputSurvei adalah hasil kunjungan lapangan yang direkam AO (FR-04).
type InputSurvei struct {
	PengajuanID    int64
	AOID           int64
	Latitude       float64
	Longitude      float64
	FotoURL        string
	OmzetHarian    int64
	LamaUsahaBulan int
	CatatanKondisi string
	Status         StatusSurvei
}

// Rekam menyimpan satu hasil survei lapangan.
//
// Koordinat, minimal satu foto, estimasi omzet harian, lama usaha, dan catatan
// kondisi wajib ada — kelimanya menjadi masukan skoring, sehingga survei yang
// tidak lengkap tidak boleh tersimpan sebagai VALID.
func (s *SurveiService) Rekam(ctx context.Context, in InputSurvei) (Survei, error) {
	if err := validasiSurvei(in); err != nil {
		return Survei{}, err
	}

	status := in.Status
	if status == "" {
		status = StatusSurveiValid
	}

	sv := Survei{
		PengajuanID:    in.PengajuanID,
		AOID:           in.AOID,
		Latitude:       in.Latitude,
		Longitude:      in.Longitude,
		FotoURL:        strings.TrimSpace(in.FotoURL),
		OmzetHarian:    in.OmzetHarian,
		LamaUsahaBulan: in.LamaUsahaBulan,
		CatatanKondisi: strings.TrimSpace(in.CatatanKondisi),
		Status:         status,
		DibuatPada:     s.sekaran(),
	}
	if err := s.repo.Simpan(ctx, &sv); err != nil {
		return Survei{}, err
	}

	// BR-10. Koordinat dan path foto TIDAK ikut ke catatan audit (BR-11);
	// yang dicatat hanya bahwa survei direkam beserta statusnya.
	if s.audit != nil {
		id := sv.PengajuanID
		if err := s.audit.Catat(ctx, domain.CatatAuditInput{
			PengajuanID: &id,
			Aksi:        domain.AksiRekamSurvei,
			Catatan:     "survei lapangan direkam dengan status " + string(sv.Status),
			ActorID:     in.AOID,
			ActorRole:   domain.PeranAO,
		}); err != nil {
			return Survei{}, err
		}
	}
	return sv, nil
}

// AdaSurveiValid melaporkan apakah pengajuan sudah punya minimal satu survei
// VALID. Dipakai sebagai masukan guard BR-03 di SkoringService.
func (s *SurveiService) AdaSurveiValid(ctx context.Context, pengajuanID int64) (bool, error) {
	return s.repo.AdaSurveiValid(ctx, pengajuanID)
}

func validasiSurvei(in InputSurvei) error {
	var kurang []string
	if in.Latitude == 0 && in.Longitude == 0 {
		kurang = append(kurang, "koordinat lokasi usaha")
	}
	if strings.TrimSpace(in.FotoURL) == "" {
		kurang = append(kurang, "minimal satu foto kondisi usaha")
	}
	if in.OmzetHarian <= 0 {
		kurang = append(kurang, "estimasi omzet harian")
	}
	if in.LamaUsahaBulan <= 0 {
		kurang = append(kurang, "lama usaha dalam bulan")
	}
	if strings.TrimSpace(in.CatatanKondisi) == "" {
		kurang = append(kurang, "catatan kondisi usaha")
	}
	if len(kurang) > 0 {
		return NewValidationError("survei belum lengkap: %s wajib diisi", strings.Join(kurang, ", "))
	}
	if in.Latitude < -90 || in.Latitude > 90 || in.Longitude < -180 || in.Longitude > 180 {
		return NewValidationError("koordinat lokasi usaha tidak valid")
	}
	if in.Status != "" && in.Status != StatusSurveiValid && in.Status != StatusSurveiTidakValid {
		return NewValidationError("status survei tidak dikenal")
	}
	return nil
}
