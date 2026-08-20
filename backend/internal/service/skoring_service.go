package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/irgiys/tim1gow/backend/internal/domain"
)

// SkoringService menghitung skor kelayakan mikro (FR-06) dan menurunkan grade
// risiko dari skor itu. Lapisan ini tidak tahu apa pun tentang HTTP dan tidak
// membangun SQL (AGENTS.md bagian 3).
type SkoringService struct {
	param ParameterRepository

	// audit dipakai untuk merekam override grade (AC-08 / BR-10). Boleh nil
	// pada pemakaian yang hanya menghitung skor, supaya perhitungan murni
	// tetap bisa diuji tanpa dependensi audit.
	audit AuditService
}

func NewSkoringService(param ParameterRepository) *SkoringService {
	return &SkoringService{param: param}
}

// NewSkoringServiceWithAudit dipakai jalur yang mengubah keadaan (override
// grade): setiap perubahan wajib punya aktor dan timestamp (BR-10).
func NewSkoringServiceWithAudit(param ParameterRepository, audit AuditService) *SkoringService {
	return &SkoringService{param: param, audit: audit}
}

// PrasyaratSkoring adalah keadaan pengajuan yang menentukan boleh/tidaknya
// skoring dijalankan (BR-03).
type PrasyaratSkoring struct {
	SemuaDokumenVerified bool
	AdaSurveiValid       bool
	SlikSudahDijalankan  bool
}

// PastikanBolehSkoring menegakkan BR-03: skoring hanya boleh jalan kalau semua
// dokumen wajib VERIFIED, ada minimal satu survei VALID, dan SLIK check sudah
// dijalankan. Pesan errornya menyebut BR-03 karena AC-04 memeriksanya.
func (s *SkoringService) PastikanBolehSkoring(p PrasyaratSkoring) error {
	var kurang []string
	if !p.SemuaDokumenVerified {
		kurang = append(kurang, "seluruh dokumen wajib belum VERIFIED")
	}
	if !p.AdaSurveiValid {
		kurang = append(kurang, "belum ada survei lapangan berstatus VALID")
	}
	if !p.SlikSudahDijalankan {
		kurang = append(kurang, "SLIK check belum dijalankan")
	}
	if len(kurang) == 0 {
		return nil
	}
	pesan := "skoring belum dapat dijalankan: " + kurang[0]
	for _, k := range kurang[1:] {
		pesan += "; " + k
	}
	return domain.NewBusinessRuleError("BR-03", "%s", pesan)
}

// Hitung menghitung skor kelayakan 0-100 beserta rincian tiap komponen (BR-08)
// dan grade risikonya. Seluruh bobot dan ambang dibaca dari tabel parameter.
func (s *SkoringService) Hitung(d domain.DataSkoring) (domain.HasilSkoring, error) {
	var hasil domain.HasilSkoring

	komponen, err := s.param.KomponenSkor()
	if err != nil {
		return hasil, err
	}
	if len(komponen) == 0 {
		return hasil, domain.NewConfigError("tabel parameter_skoring belum diisi")
	}

	// Urutkan supaya rincian yang ditampilkan ke ANL stabil urutannya.
	sort.Slice(komponen, func(i, j int) bool { return komponen[i].Kode < komponen[j].Kode })

	var totalKontribusi, totalBobot float64
	rincian := make([]domain.RincianKomponenSkor, 0, len(komponen))

	for _, k := range komponen {
		if !k.Aktif {
			continue
		}
		skor, err := s.skorKomponen(k, d)
		if err != nil {
			return hasil, err
		}
		kontribusi := skor * k.Bobot
		rincian = append(rincian, domain.RincianKomponenSkor{
			Kode:       k.Kode,
			Nama:       k.Nama,
			SkorMentah: skor,
			Bobot:      k.Bobot,
			Kontribusi: kontribusi,
		})
		totalKontribusi += kontribusi
		totalBobot += k.Bobot
	}

	if totalBobot <= 0 {
		return hasil, domain.NewConfigError("total bobot komponen skor nol; parameter_skoring belum benar")
	}

	// BR-07: skor akhir = SUM(skor x bobot) / SUM(bobot), dibulatkan ke bilangan
	// bulat terdekat. Pembulatan dilakukan SEKALI di akhir, bukan per komponen.
	skorAkhir := int(math.Round(totalKontribusi / totalBobot))
	if skorAkhir < 0 {
		skorAkhir = 0
	}
	if skorAkhir > 100 {
		skorAkhir = 100
	}

	grade, err := s.GradeDariSkor(skorAkhir)
	if err != nil {
		return hasil, err
	}

	// Tabel 4.2: kolektibilitas 2 memaksa grade risiko minimal 3.
	dipaksa := false
	if d.Kolektibilitas == 2 && grade < 3 {
		grade = 3
		dipaksa = true
	}

	hasil = domain.HasilSkoring{
		PengajuanID:         d.PengajuanID,
		SkorAkhir:           skorAkhir,
		Grade:               grade,
		Rincian:             rincian,
		TotalBobot:          totalBobot,
		GradeMinimalDipaksa: dipaksa,
	}
	return hasil, nil
}

// GradeDariSkor menurunkan grade risiko dari skor akhir memakai rentang skor
// pada tabel rentang_margin — bukan rentang yang ditulis di kode.
func (s *SkoringService) GradeDariSkor(skor int) (int, error) {
	rentang, err := s.param.RentangMarginPerGrade()
	if err != nil {
		return 0, err
	}
	if len(rentang) == 0 {
		return 0, domain.NewConfigError("tabel rentang_margin belum diisi")
	}
	for _, r := range rentang {
		if skor >= r.SkorMin && skor <= r.SkorMaks {
			return r.Grade, nil
		}
	}
	return 0, domain.NewConfigError("tidak ada grade yang mencakup skor %d pada tabel rentang_margin", skor)
}

// PastikanBolehKeApproval menegakkan BR-05: grade 5 tidak dapat diajukan ke
// approval. Pemanggil mengubah status menjadi REJECTED_SCORING.
func (s *SkoringService) PastikanBolehKeApproval(grade int) error {
	r, ada, err := s.param.RentangMargin(grade)
	if err != nil {
		return err
	}
	if !ada {
		return domain.NewConfigError("rentang margin grade %d belum diatur", grade)
	}
	if !r.DapatDibiayai {
		return domain.NewBusinessRuleError("BR-05",
			"grade %d tidak dapat diajukan ke approval", grade)
	}
	return nil
}

// OverrideGrade adalah masukan permintaan override grade oleh ANL (AC-08).
type OverrideGrade struct {
	PengajuanID int64
	GradeSemula int
	GradeBaru   int
	Alasan      string
	ActorID     int64
	ActorRole   domain.Peran
}

// HasilOverrideGrade adalah keluaran override yang sudah tercatat.
type HasilOverrideGrade struct {
	PengajuanID int64
	GradeSemula int
	GradeBaru   int
	Alasan      string
}

// OverrideGrade menerapkan override grade oleh ANL dan MENCATATNYA ke audit
// trail bersama identitas pelakunya (AC-08, BR-10).
//
// Aturan yang ditegakkan di sini, bukan di handler (Larangan 17):
//   - Alasan wajib. Override tanpa alasan ditolak; AC-08 memintanya eksplisit.
//   - Grade harus ada di tabel rentang_margin, supaya ANL tidak bisa memasukkan
//     grade yang tidak dikenal sistem.
//   - Grade yang sama dengan semula bukan override, jadi ditolak.
//   - Pencatatan audit adalah bagian dari operasi: kalau audit gagal, override
//     dianggap GAGAL. Tidak ada perubahan tanpa jejak (BR-10) — ini sebabnya
//     error audit tidak diabaikan di sini.
func (s *SkoringService) OverrideGrade(ctx context.Context, in OverrideGrade) (HasilOverrideGrade, error) {
	var hasil HasilOverrideGrade

	if strings.TrimSpace(in.Alasan) == "" {
		return hasil, domain.NewBusinessRuleError("VALIDATION_ERROR",
			"alasan override wajib diisi")
	}
	if in.ActorID <= 0 || in.ActorRole == "" {
		return hasil, domain.NewBusinessRuleError("VALIDATION_ERROR",
			"identitas aktor wajib diketahui untuk override (BR-10)")
	}
	// Override adalah kewenangan Analis Mikro (AGENTS.md bagian 1).
	if in.ActorRole != domain.PeranANL {
		return hasil, domain.NewBusinessRuleError("FORBIDDEN",
			"hanya ANL yang dapat melakukan override grade")
	}
	if in.GradeBaru == in.GradeSemula {
		return hasil, domain.NewBusinessRuleError("VALIDATION_ERROR",
			"grade baru sama dengan grade semula; tidak ada yang di-override")
	}

	// Grade tujuan wajib dikenal tabel parameter — bukan divalidasi terhadap
	// rentang yang ditulis di kode.
	if _, ada, err := s.param.RentangMargin(in.GradeBaru); err != nil {
		return hasil, err
	} else if !ada {
		return hasil, domain.NewBusinessRuleError("VALIDATION_ERROR",
			"grade %d tidak dikenal pada tabel rentang_margin", in.GradeBaru)
	}

	if s.audit == nil {
		return hasil, domain.NewConfigError("audit service belum dipasang; override tidak dapat dicatat")
	}

	// Catatan sengaja hanya memuat grade dan alasan dari ANL — tanpa NIK,
	// nomor dokumen, atau path foto (BR-11).
	catatan := fmt.Sprintf("override grade %d -> %d; alasan: %s",
		in.GradeSemula, in.GradeBaru, strings.TrimSpace(in.Alasan))

	pengajuanID := in.PengajuanID
	if err := s.audit.Catat(ctx, domain.CatatAuditInput{
		PengajuanID: &pengajuanID,
		Aksi:        domain.AksiOverrideSkor,
		Catatan:     catatan,
		ActorID:     in.ActorID,
		ActorRole:   in.ActorRole,
	}); err != nil {
		return hasil, err
	}

	return HasilOverrideGrade{
		PengajuanID: in.PengajuanID,
		GradeSemula: in.GradeSemula,
		GradeBaru:   in.GradeBaru,
		Alasan:      strings.TrimSpace(in.Alasan),
	}, nil
}
