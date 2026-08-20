package service

import (
	"math"
	"sort"

	"github.com/irgiys/tim1gow/backend/internal/domain"
)

// SkoringService menghitung skor kelayakan mikro (FR-06) dan menurunkan grade
// risiko dari skor itu. Lapisan ini tidak tahu apa pun tentang HTTP dan tidak
// membangun SQL (AGENTS.md bagian 3).
type SkoringService struct {
	param ParameterRepository
}

func NewSkoringService(param ParameterRepository) *SkoringService {
	return &SkoringService{param: param}
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
