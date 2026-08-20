package service

import "github.com/irgiys/tim1gow/backend/internal/domain"

// MarginService menghitung dan memvalidasi margin murabahah / nisbah bank
// musyarakah (FR-07). Rentang yang disetujui dibaca dari tabel rentang_margin.
type MarginService struct {
	param ParameterRepository
}

func NewMarginService(param ParameterRepository) *MarginService {
	return &MarginService{param: param}
}

// HasilMargin adalah keluaran perhitungan margin/nisbah beserta rentang yang
// dipakai memvalidasinya, supaya ANL melihat dasar keputusannya.
type HasilMargin struct {
	Akad        domain.Akad
	Grade       int
	Nilai       float64
	RentangMin  float64
	RentangMaks float64
}

// Validasi menegakkan BR-05 dan BR-06 untuk sebuah nilai margin/nisbah yang
// diusulkan. Nilai di luar rentang DIBLOKIR — tidak ada jalur "lanjutkan saja",
// dan tidak ada bentuk keluaran berupa peringatan.
func (m *MarginService) Validasi(akad domain.Akad, grade int, nilai float64) (HasilMargin, error) {
	var hasil HasilMargin

	r, ada, err := m.param.RentangMargin(grade)
	if err != nil {
		return hasil, err
	}
	if !ada {
		return hasil, domain.NewConfigError("rentang margin grade %d belum diatur", grade)
	}

	// BR-05 diperiksa lebih dulu: grade yang tidak dibiayai tidak punya rentang
	// yang sah, jadi memvalidasi angkanya tidak ada artinya.
	if !r.DapatDibiayai {
		return hasil, domain.NewBusinessRuleError("BR-05",
			"grade %d tidak dapat diajukan ke approval", grade)
	}

	min, maks, err := rentangUntukAkad(akad, r)
	if err != nil {
		return hasil, err
	}

	hasil = HasilMargin{
		Akad:        akad,
		Grade:       grade,
		Nilai:       nilai,
		RentangMin:  min,
		RentangMaks: maks,
	}

	if nilai < min || nilai > maks {
		// Pesan sengaja tanpa data pribadi (BR-11) dan menyebut kode BR (AC-04).
		return hasil, domain.NewBusinessRuleError("BR-06",
			"nilai %.2f%% di luar rentang grade %d (%.2f%%-%.2f%%)", nilai, grade, min, maks)
	}
	return hasil, nil
}

// RentangUntukGrade mengembalikan rentang yang disetujui untuk sebuah grade dan
// akad, dipakai UI untuk menampilkan batas sebelum ANL mengisi angkanya.
func (m *MarginService) RentangUntukGrade(akad domain.Akad, grade int) (min, maks float64, err error) {
	r, ada, err := m.param.RentangMargin(grade)
	if err != nil {
		return 0, 0, err
	}
	if !ada {
		return 0, 0, domain.NewConfigError("rentang margin grade %d belum diatur", grade)
	}
	if !r.DapatDibiayai {
		return 0, 0, domain.NewBusinessRuleError("BR-05",
			"grade %d tidak dibiayai", grade)
	}
	return rentangUntukAkad(akad, r)
}

func rentangUntukAkad(akad domain.Akad, r domain.RentangMargin) (min, maks float64, err error) {
	switch akad {
	case domain.AkadMurabahah:
		return r.MarginMin, r.MarginMaks, nil
	case domain.AkadMusyarakah:
		return r.NisbahMin, r.NisbahMaks, nil
	default:
		return 0, 0, domain.NewBusinessRuleError("BR-06", "jenis akad %q tidak dikenal", akad)
	}
}
