package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/irgiys/tim1gow/backend/internal/domain"
	"github.com/irgiys/tim1gow/backend/internal/slik"
)

// HasilSlik adalah satu baris tabel `hasil_slik` (FR-05, BR-04).
//
// Kolektibilitas dan tanggal bertipe pointer mengikuti skema migrasi 000004:
// panggilan GAGAL tetap disimpan sebagai bukti percobaan, dan kolomnya kosong.
// Kolom kosong lebih jujur daripada nilai default yang membuat SLIK gagal
// tampak seperti SLIK bersih (Larangan 15).
type HasilSlik struct {
	ID                   int64
	PengajuanID          int64
	Kolektibilitas       *int
	JumlahFasilitasAktif *int
	TotalBakiDebet       *int64
	TanggalData          *time.Time
	ReferenceID          string
	StatusPanggilan      string
	BerlakuSampai        *time.Time
	DicekOleh            *int64
	DibuatPada           time.Time
}

// SlikRepository adalah akses data tabel `hasil_slik` (FR-05).
//
// Interface didefinisikan di paket service mengikuti pola repository lain di
// paket ini (ParameterRepository, PrasyaratSkoringRepository) supaya tidak
// terjadi import cycle service <-> repository.
type SlikRepository interface {
	// Simpan menyisipkan satu baris hasil SLIK dan mengisi ID-nya.
	Simpan(ctx context.Context, h *HasilSlik) error

	// TerakhirSukses mengembalikan hasil SUKSES terbaru sebuah pengajuan.
	// ErrTidakDitemukan bila belum pernah ada panggilan sukses.
	TerakhirSukses(ctx context.Context, pengajuanID int64) (HasilSlik, error)
}

// SlikPengajuanRepository adalah bagian PengajuanRepository yang dibutuhkan
// jalur SLIK. Dipersempit supaya test tidak perlu mengimplementasikan seluruh
// PengajuanRepository hanya untuk menguji satu alur.
type SlikPengajuanRepository interface {
	CariID(ctx context.Context, id int64) (Pengajuan, error)
	Perbarui(ctx context.Context, p *Pengajuan) error
}

// PemanggilSlik adalah kontrak client SLIK yang dipakai service.
//
// Diabstraksi sebagai interface, bukan *slik.Client konkret, supaya jalur
// error (503, timeout) bisa diuji tanpa menyalakan server.
type PemanggilSlik interface {
	Inquiry(ctx context.Context, nik string) (slik.Hasil, error)
}

// SlikService menjalankan SLIK check dan menerjemahkan hasilnya menjadi
// keputusan alur pengajuan (FR-05, BR-04).
//
// Pembagian tanggung jawab yang dijaga berkas ini:
//   - internal/slik  : bicara HTTP, melaporkan apa jawaban SLIK.
//   - berkas ini     : memutuskan nasib pengajuan dari jawaban itu.
//
// Aturan yang ditegakkan di sini:
//   - BR-04: hasil berlaku 30 hari, dihitung dari tanggal_data.
//   - Kol 3/4/5 -> REJECTED_SLIK tanpa melalui approval (bagian 5.1).
//   - Kol 2     -> lanjut, tetapi ditandai wajib grade minimal 3 + catatan analis.
//   - BR-10: setiap perubahan status punya aktor dan timestamp.
//   - Larangan 15: panggilan gagal TIDAK pernah menjadi SLIK bersih.
type SlikService struct {
	slikRepo  SlikRepository
	pengajuan SlikPengajuanRepository
	client    PemanggilSlik
	audit     AuditService

	// masaBerlakuHari adalah 30 sesuai BR-04. Disimpan sebagai field, bukan
	// konstanta tertanam, supaya test masa berlaku tidak perlu menunggu 30
	// hari nyata. Nilainya berasal dari pemanggil (config), bukan dari brief
	// yang disalin ke kode.
	masaBerlakuHari int

	// sekarang dapat diganti di test supaya BR-04 dapat diuji deterministik.
	sekarang func() time.Time
}

// OpsiSlikService mengumpulkan dependensi SlikService.
type OpsiSlikService struct {
	SlikRepo        SlikRepository
	Pengajuan       SlikPengajuanRepository
	Client          PemanggilSlik
	Audit           AuditService
	MasaBerlakuHari int
}

// NewSlikService membangun SlikService.
func NewSlikService(o OpsiSlikService) *SlikService {
	hari := o.MasaBerlakuHari
	if hari <= 0 {
		hari = 30 // BR-04
	}
	return &SlikService{
		slikRepo:        o.SlikRepo,
		pengajuan:       o.Pengajuan,
		client:          o.Client,
		audit:           o.Audit,
		masaBerlakuHari: hari,
		sekarang:        time.Now,
	}
}

// ErrSlikTidakTersedia menandai SLIK tidak dapat dihubungi (503/timeout).
// Handler memetakannya ke 502: backend gagal memakai dependensi hulu, dan
// pengajuan TIDAK boleh lanjut (aturan 4.3).
var ErrSlikTidakTersedia = errors.New("layanan SLIK tidak tersedia")

// ErrNIKTidakDitemukanSlik menandai NIK tidak terdaftar di SLIK.
var ErrNIKTidakDitemukanSlik = errors.New("NIK tidak ditemukan di SLIK")

// HasilCheck adalah ringkasan yang dikembalikan ke pemanggil setelah SLIK check.
type HasilCheck struct {
	Kolektibilitas       int
	JumlahFasilitasAktif *int
	TotalBakiDebet       *int64
	TanggalData          *time.Time
	BerlakuSampai        *time.Time
	StatusPengajuan      string

	// GradeMinimal terisi 3 untuk kol-2 (bagian 5.1). Nol berarti tidak ada
	// batasan grade dari jalur SLIK.
	GradeMinimal int

	// WajibCatatanAnalis true untuk kol-2 (bagian 5.1).
	WajibCatatanAnalis bool

	// Ditolak true kalau kolektibilitas 3/4/5 menolak pengajuan otomatis.
	Ditolak bool
}

// Jalankan melakukan SLIK check untuk sebuah pengajuan (FR-05).
//
// Urutan yang disengaja: setiap percobaan dicatat ke hasil_slik SEBELUM
// keputusan dikembalikan — termasuk percobaan yang gagal. Kalau pencatatan
// ditaruh setelah cabang sukses saja, kegagalan SLIK tidak akan punya jejak
// dan AC-06 tidak bisa dibuktikan.
func (s *SlikService) Jalankan(ctx context.Context, pengajuanID, anlID int64) (HasilCheck, error) {
	pjn, err := s.pengajuan.CariID(ctx, pengajuanID)
	if err != nil {
		return HasilCheck{}, err
	}

	// NIK diambil dari database dan diteruskan ke client lewat badan request.
	// Ia tidak pernah masuk ke log, pesan error, atau URL di jalur ini (BR-11).
	jawaban, err := s.client.Inquiry(ctx, pjn.NIK)
	if err != nil {
		// Kontrak SLIK dilanggar (mis. 500, badan tak dikenal). Tetap dicatat
		// sebagai percobaan, lalu diteruskan sebagai kegagalan hulu.
		_ = s.catat(ctx, pengajuanID, anlID, slik.Hasil{Status: slik.StatusTimeout}, nil)
		return HasilCheck{}, fmt.Errorf("%w: %v", ErrSlikTidakTersedia, err)
	}

	// Masa berlaku hanya bermakna untuk hasil yang benar-benar punya data.
	var berlakuSampai *time.Time
	if jawaban.Sukses() && jawaban.TanggalData != nil {
		b := jawaban.TanggalData.AddDate(0, 0, s.masaBerlakuHari)
		berlakuSampai = &b
	}

	if err := s.catat(ctx, pengajuanID, anlID, jawaban, berlakuSampai); err != nil {
		return HasilCheck{}, err
	}

	switch jawaban.Status {
	case slik.StatusLayananTidakAda, slik.StatusTimeout:
		// Larangan 15: TIDAK dianggap bersih, status pengajuan tidak maju.
		return HasilCheck{}, ErrSlikTidakTersedia

	case slik.StatusNIKTidakDitemukan:
		return HasilCheck{}, ErrNIKTidakDitemukanSlik

	case slik.StatusSukses:
		return s.putuskan(ctx, pjn, anlID, jawaban, berlakuSampai)

	default:
		// Status tak dikenal tidak boleh lolos secara diam-diam.
		return HasilCheck{}, fmt.Errorf("%w: status panggilan %q tidak dikenal",
			ErrSlikTidakTersedia, jawaban.Status)
	}
}

// putuskan menerjemahkan kolektibilitas menjadi keputusan alur (bagian 5.1).
func (s *SlikService) putuskan(ctx context.Context, pjn Pengajuan, anlID int64,
	jawaban slik.Hasil, berlakuSampai *time.Time) (HasilCheck, error) {

	kol := *jawaban.Kolektibilitas
	hasil := HasilCheck{
		Kolektibilitas:       kol,
		JumlahFasilitasAktif: jawaban.JumlahFasilitasAktif,
		TotalBakiDebet:       jawaban.TotalBakiDebet,
		TanggalData:          jawaban.TanggalData,
		BerlakuSampai:        berlakuSampai,
	}

	statusSebelum := pjn.Status

	switch {
	case kol >= 3:
		// Penolakan otomatis, tanpa melalui approval (bagian 5.1).
		hasil.Ditolak = true
		pjn.Status = string(domain.StatusRejectedSlik)

	case kol == 2:
		// Lanjut, tetapi grade risiko minimal 3 dan wajib catatan analis.
		hasil.GradeMinimal = 3
		hasil.WajibCatatanAnalis = true
		pjn.Status = string(domain.StatusSlikChecked)

	default: // kol == 1
		pjn.Status = string(domain.StatusSlikChecked)
	}

	if err := s.pengajuan.Perbarui(ctx, &pjn); err != nil {
		return HasilCheck{}, err
	}
	hasil.StatusPengajuan = pjn.Status

	// BR-10: perubahan status wajib punya aktor, timestamp, dan sebab.
	// Catatan menyebut kolektibilitas (bukan data pribadi) supaya alasan
	// perubahan status terbaca di audit trail tanpa membocorkan NIK (BR-11).
	catatan := fmt.Sprintf("SLIK check: kolektibilitas %d", kol)
	if hasil.Ditolak {
		catatan += " — penolakan otomatis"
	}
	if err := s.catatAudit(ctx, pjn.ID, anlID, statusSebelum, pjn.Status, catatan); err != nil {
		return HasilCheck{}, err
	}

	return hasil, nil
}

// catat menyimpan satu baris hasil_slik, termasuk untuk panggilan gagal.
func (s *SlikService) catat(ctx context.Context, pengajuanID, anlID int64,
	jawaban slik.Hasil, berlakuSampai *time.Time) error {

	if s.slikRepo == nil {
		return nil
	}
	aktor := anlID
	h := HasilSlik{
		PengajuanID:          pengajuanID,
		Kolektibilitas:       jawaban.Kolektibilitas,
		JumlahFasilitasAktif: jawaban.JumlahFasilitasAktif,
		TotalBakiDebet:       jawaban.TotalBakiDebet,
		TanggalData:          jawaban.TanggalData,
		ReferenceID:          jawaban.ReferenceID,
		StatusPanggilan:      string(jawaban.Status),
		BerlakuSampai:        berlakuSampai,
		DicekOleh:            &aktor,
		DibuatPada:           s.sekarang(),
	}
	return s.slikRepo.Simpan(ctx, &h)
}

// catatAudit merekam jejak SLIK check (BR-10). Dilewati bila audit belum
// dipasang (test unit murni); di produksi kegagalannya diteruskan.
func (s *SlikService) catatAudit(ctx context.Context, pengajuanID, aktorID int64,
	statusSebelum, statusSesudah, catatan string) error {

	if s.audit == nil {
		return nil
	}
	id := pengajuanID
	return s.audit.Catat(ctx, domain.CatatAuditInput{
		PengajuanID:   &id,
		Aksi:          domain.AksiSlikCheck,
		StatusSebelum: domain.StatusPengajuan(statusSebelum),
		StatusSesudah: domain.StatusPengajuan(statusSesudah),
		Catatan:       catatan,
		ActorID:       aktorID,
		ActorRole:     domain.PeranANL,
	})
}
