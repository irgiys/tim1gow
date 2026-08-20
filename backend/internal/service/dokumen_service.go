package service

import (
	"context"
	"strings"
	"time"

	"github.com/irgiys/tim1gow/backend/internal/domain"
)

// DokumenService memuat aturan upload dan verifikasi dokumen (FR-03).
type DokumenService struct {
	dok     DokumenRepository
	wajib   DokumenWajibRepository
	sekaran func() time.Time
}

// DokumenWajibRepository membaca daftar jenis dokumen yang wajib dilengkapi.
// Daftarnya berupa data supaya ADM dapat mengubahnya tanpa deploy ulang
// (AGENTS.md Larangan 3).
type DokumenWajibRepository interface {
	JenisDokumenWajib(ctx context.Context) ([]string, error)
}

func NewDokumenService(dok DokumenRepository, wajib DokumenWajibRepository) *DokumenService {
	return &DokumenService{dok: dok, wajib: wajib, sekaran: time.Now}
}

// DenganWaktu mengganti sumber waktu; dipakai test.
func (s *DokumenService) DenganWaktu(f func() time.Time) *DokumenService {
	s.sekaran = f
	return s
}

// Upload menyimpan satu dokumen berstatus UPLOADED.
//
// Saat AO mengunggah ulang berkas yang sebelumnya REJECTED, hanya jenis dokumen
// itu yang tergantikan; dokumen lain dan data pengajuan tidak tersentuh (AC-03).
// Dokumen yang sudah VERIFIED tidak boleh ditimpa.
func (s *DokumenService) Upload(ctx context.Context, pengajuanID int64, jenis, urlBerkas string) (Dokumen, error) {
	jenis = strings.TrimSpace(jenis)
	if jenis == "" {
		return Dokumen{}, NewValidationError("jenis dokumen wajib diisi")
	}
	if strings.TrimSpace(urlBerkas) == "" {
		return Dokumen{}, NewValidationError("berkas dokumen wajib diunggah")
	}

	lama, err := s.dok.CariAktif(ctx, pengajuanID, jenis)
	switch {
	case err == nil && lama.Status == StatusDokumenVerified:
		return Dokumen{}, domain.NewBusinessRuleError("BR-03",
			"dokumen %s sudah diverifikasi dan tidak dapat diunggah ulang", jenis)
	case err != nil && err != ErrTidakDitemukan:
		return Dokumen{}, err
	}

	d := Dokumen{
		PengajuanID:  pengajuanID,
		JenisDokumen: jenis,
		URLBerkas:    urlBerkas,
		Status:       StatusDokumenUploaded,
		DibuatPada:   s.sekaran(),
	}
	if err := s.dok.Simpan(ctx, &d); err != nil {
		return Dokumen{}, err
	}
	return d, nil
}

// Verifikasi menandai satu dokumen VERIFIED atau REJECTED oleh ANL (FR-03).
// Penolakan wajib menyertakan kode alasan; tanpa itu keputusan ditolak.
func (s *DokumenService) Verifikasi(ctx context.Context, dokumenID, anlID int64, setujui bool, kodeAlasan string) (Dokumen, error) {
	d, err := s.dok.CariID(ctx, dokumenID)
	if err != nil {
		return Dokumen{}, err
	}

	kodeAlasan = strings.TrimSpace(kodeAlasan)
	if !setujui && kodeAlasan == "" {
		return Dokumen{}, NewValidationError("penolakan dokumen wajib menyertakan kode alasan")
	}

	waktu := s.sekaran()
	d.DiverifikasiOleh = &anlID
	d.DiverifikasiPada = &waktu
	if setujui {
		d.Status = StatusDokumenVerified
		d.AlasanPenolakan = nil
	} else {
		d.Status = StatusDokumenRejected
		d.AlasanPenolakan = &kodeAlasan
	}

	if err := s.dok.Perbarui(ctx, &d); err != nil {
		return Dokumen{}, err
	}
	return d, nil
}

// SemuaDokumenWajibVerified melaporkan apakah seluruh jenis dokumen wajib sudah
// berstatus VERIFIED. Hasilnya dipakai sebagai masukan guard BR-03 di
// SkoringService; service ini tidak memutuskan boleh/tidaknya skoring.
func (s *DokumenService) SemuaDokumenWajibVerified(ctx context.Context, pengajuanID int64) (bool, error) {
	wajib, err := s.wajib.JenisDokumenWajib(ctx)
	if err != nil {
		return false, err
	}
	if len(wajib) == 0 {
		return false, domain.NewConfigError("daftar dokumen wajib belum diatur di tabel parameter")
	}

	daftar, err := s.dok.DaftarPerPengajuan(ctx, pengajuanID)
	if err != nil {
		return false, err
	}

	terverifikasi := make(map[string]bool, len(daftar))
	for _, d := range daftar {
		if d.Status == StatusDokumenVerified {
			terverifikasi[d.JenisDokumen] = true
		}
	}
	for _, j := range wajib {
		if !terverifikasi[j] {
			return false, nil
		}
	}
	return true, nil
}
