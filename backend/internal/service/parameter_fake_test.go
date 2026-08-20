package service

import "github.com/irgiys/tim1gow/backend/internal/domain"

// fakeParameterRepo meniru tabel parameter di database. Baris-barisnya dapat
// DIUBAH di tengah test — itulah yang membuktikan service benar-benar membaca
// nilai dari data, bukan dari konstanta (AC-15).
//
// Nilai awal di bawah SENGAJA mengikuti seed brief §4.4/Tabel 4.3, karena test
// harus meniru keadaan database nyata. Yang dilarang adalah service memakai
// angka ini sebagai default; test tetap wajib mengubahnya untuk membuktikan
// nilainya dibaca dari data.
type fakeParameterRepo struct {
	komponen []domain.ParameterKomponenSkor
	skorSlik map[int]float64
	umum     map[string]float64
	rentang  []domain.RentangMargin
	ambang   []domain.AmbangApproval

	// jumlahBacaKomponen mencatat berapa kali tabel dibaca, dipakai memastikan
	// service tidak meng-cache parameter antar pemanggilan.
	jumlahBacaKomponen int
}

func newFakeParameterRepo() *fakeParameterRepo {
	return &fakeParameterRepo{
		komponen: []domain.ParameterKomponenSkor{
			{Kode: domain.KomponenKapasitasBayar, Nama: "Kapasitas bayar", Bobot: 35, Batas1: 0.30, Batas2: 0.60, Aktif: true},
			{Kode: domain.KomponenRiwayatSlik, Nama: "Riwayat SLIK", Bobot: 25, Aktif: true},
			{Kode: domain.KomponenLamaUsaha, Nama: "Lama usaha", Bobot: 20, Batas1: 36, Batas2: 6, Aktif: true},
			{Kode: domain.KomponenSurveiLapangan, Nama: "Hasil survei lapangan", Bobot: 20, Aktif: true},
		},
		skorSlik: map[int]float64{1: 100, 2: 40},
		umum: map[string]float64{
			KunciHariKerjaPerBulan: 25,
			KunciMarginUsaha:       0.30,
		},
		rentang: []domain.RentangMargin{
			{Grade: 1, SkorMin: 85, SkorMaks: 100, MarginMin: 11.0, MarginMaks: 13.0, NisbahMin: 20, NisbahMaks: 25, DapatDibiayai: true},
			{Grade: 2, SkorMin: 70, SkorMaks: 84, MarginMin: 13.0, MarginMaks: 15.5, NisbahMin: 25, NisbahMaks: 30, DapatDibiayai: true},
			{Grade: 3, SkorMin: 55, SkorMaks: 69, MarginMin: 15.5, MarginMaks: 18.0, NisbahMin: 30, NisbahMaks: 35, DapatDibiayai: true},
			{Grade: 4, SkorMin: 40, SkorMaks: 54, MarginMin: 18.0, MarginMaks: 21.0, NisbahMin: 35, NisbahMaks: 40, DapatDibiayai: true},
			{Grade: 5, SkorMin: 0, SkorMaks: 39, DapatDibiayai: false},
		},
		ambang: []domain.AmbangApproval{
			{PlafonMin: 5000000, PlafonMaks: 50000000, Level: []domain.Peran{domain.PeranKCP}},
			{PlafonMin: 50000001, PlafonMaks: 200000000, Level: []domain.Peran{domain.PeranKCP, domain.PeranKC}},
			{PlafonMin: 200000001, PlafonMaks: 500000000, Level: []domain.Peran{domain.PeranKCP, domain.PeranKC, domain.PeranKOM}},
		},
	}
}

func (f *fakeParameterRepo) KomponenSkor() ([]domain.ParameterKomponenSkor, error) {
	f.jumlahBacaKomponen++
	out := make([]domain.ParameterKomponenSkor, len(f.komponen))
	copy(out, f.komponen)
	return out, nil
}

func (f *fakeParameterRepo) SkorRiwayatSlik(kol int) (float64, bool, error) {
	s, ok := f.skorSlik[kol]
	return s, ok, nil
}

func (f *fakeParameterRepo) Umum(kunci string) (float64, bool, error) {
	v, ok := f.umum[kunci]
	return v, ok, nil
}

func (f *fakeParameterRepo) RentangMarginPerGrade() ([]domain.RentangMargin, error) {
	out := make([]domain.RentangMargin, len(f.rentang))
	copy(out, f.rentang)
	return out, nil
}

func (f *fakeParameterRepo) RentangMargin(grade int) (domain.RentangMargin, bool, error) {
	for _, r := range f.rentang {
		if r.Grade == grade {
			return r, true, nil
		}
	}
	return domain.RentangMargin{}, false, nil
}

func (f *fakeParameterRepo) AmbangApproval(totalPlafon int64) (domain.AmbangApproval, bool, error) {
	for _, a := range f.ambang {
		if totalPlafon >= a.PlafonMin && totalPlafon <= a.PlafonMaks {
			return a, true, nil
		}
	}
	return domain.AmbangApproval{}, false, nil
}

func (f *fakeParameterRepo) SemuaAmbangApproval() ([]domain.AmbangApproval, error) {
	out := make([]domain.AmbangApproval, len(f.ambang))
	copy(out, f.ambang)
	return out, nil
}

// ubahBobot meniru ADM mengubah bobot satu komponen lewat FR-13.
func (f *fakeParameterRepo) ubahBobot(kode string, bobot float64) {
	for i := range f.komponen {
		if f.komponen[i].Kode == kode {
			f.komponen[i].Bobot = bobot
			return
		}
	}
}

// ubahBatasMargin meniru ADM mengubah rentang margin satu grade.
func (f *fakeParameterRepo) ubahBatasMargin(grade int, min, maks float64) {
	for i := range f.rentang {
		if f.rentang[i].Grade == grade {
			f.rentang[i].MarginMin = min
			f.rentang[i].MarginMaks = maks
			return
		}
	}
}
