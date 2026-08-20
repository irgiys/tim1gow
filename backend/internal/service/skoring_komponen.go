package service

import "github.com/irgiys/tim1gow/backend/internal/domain"

// skorKomponen menghitung skor mentah 0-100 satu komponen. Bobot TIDAK dipakai
// di sini — pembobotan dilakukan pemanggil supaya BR-07 hanya hidup di satu
// tempat, dan supaya pembulatan tidak terjadi dua kali.
func (s *SkoringService) skorKomponen(k domain.ParameterKomponenSkor, d domain.DataSkoring) (float64, error) {
	switch k.Kode {
	case domain.KomponenKapasitasBayar:
		return s.skorKapasitasBayar(k, d)
	case domain.KomponenRiwayatSlik:
		return s.skorRiwayatSlik(d)
	case domain.KomponenLamaUsaha:
		return skorLinearTurun(float64(d.LamaUsahaBulan), k.Batas1, k.Batas2), nil
	case domain.KomponenSurveiLapangan:
		return s.skorSurvei(d)
	default:
		// Komponen tak dikenal berarti parameter_skoring memuat baris yang belum
		// ada rumusnya. Berhenti, jangan diam-diam beri skor 0 (Larangan 3).
		return 0, domain.NewConfigError("komponen skor %q belum punya rumus", k.Kode)
	}
}

// skorKapasitasBayar: rasio angsuran bulanan terhadap kapasitas usaha.
// Kapasitas = omzet harian x hari kerja x margin usaha. Kedua pengali dibaca
// dari tabel parameter (brief §4.4 memakai 25 hari dan margin usaha 30 %,
// tetapi angkanya WAJIB berupa data supaya ADM bisa mengubahnya).
//
// rasio <= Batas1 -> skor penuh; rasio > Batas2 -> skor 0; linear di antaranya.
func (s *SkoringService) skorKapasitasBayar(k domain.ParameterKomponenSkor, d domain.DataSkoring) (float64, error) {
	hariKerja, ada, err := s.param.Umum(KunciHariKerjaPerBulan)
	if err != nil {
		return 0, err
	}
	if !ada {
		return 0, domain.NewConfigError("parameter %q belum diatur", KunciHariKerjaPerBulan)
	}
	marginUsaha, ada, err := s.param.Umum(KunciMarginUsaha)
	if err != nil {
		return 0, err
	}
	if !ada {
		return 0, domain.NewConfigError("parameter %q belum diatur", KunciMarginUsaha)
	}

	kapasitas := d.OmzetHarian * hariKerja * marginUsaha
	if kapasitas <= 0 {
		// Tanpa kapasitas usaha, rasio angsuran tidak terdefinisi. Skor 0 di sini
		// adalah hasil perhitungan yang benar, bukan nilai default penutup galat.
		return 0, nil
	}

	// Batas1 = rasio batas skor penuh (mis. 0,30), Batas2 = batas skor nol
	// (mis. 0,60). Urutan argumen penting: untuk rasio angsuran, makin KECIL
	// makin baik, jadi batas skor penuh adalah nilai yang lebih kecil.
	rasio := d.AngsuranBulanan / kapasitas
	return skorLinearTurun(rasio, k.Batas1, k.Batas2), nil
}

// skorRiwayatSlik membaca skor kolektibilitas dari tabel parameter. Kalau
// barisnya tidak ada, perhitungan BERHENTI — kegagalan membaca riwayat SLIK
// tidak boleh diperlakukan sebagai SLIK bersih (AGENTS.md Larangan 15).
func (s *SkoringService) skorRiwayatSlik(d domain.DataSkoring) (float64, error) {
	skor, ada, err := s.param.SkorRiwayatSlik(d.Kolektibilitas)
	if err != nil {
		return 0, err
	}
	if !ada {
		return 0, domain.NewConfigError(
			"skor riwayat SLIK untuk kolektibilitas %d belum diatur", d.Kolektibilitas)
	}
	return skor, nil
}

// skorSurvei mengubah penilaian ANL skala 1-5 menjadi skor 0-100.
// Pengalinya diturunkan dari skala itu sendiri (100/5), bukan angka 20 yang
// ditulis di kode, supaya tetap benar kalau skalanya berubah.
func (s *SkoringService) skorSurvei(d domain.DataSkoring) (float64, error) {
	const skalaMaks = 5
	if d.NilaiSurvei < 0 {
		return 0, domain.NewBusinessRuleError("BR-08",
			"nilai survei %d tidak sah", d.NilaiSurvei)
	}
	if d.NilaiSurvei > skalaMaks {
		return 0, domain.NewBusinessRuleError("BR-08",
			"nilai survei %d melebihi skala maksimum %d", d.NilaiSurvei, skalaMaks)
	}
	return float64(d.NilaiSurvei) * (100.0 / float64(skalaMaks)), nil
}

// skorLinearTurun memetakan sebuah nilai ke skor 0-100 secara linear:
// nilai <= batasSkorPenuh -> 100; nilai >= batasSkorNol -> 0.
//
// Fungsi ini juga menangani arah sebaliknya (batasSkorPenuh > batasSkorNol),
// yang dipakai komponen LAMA_USAHA: >= 36 bulan -> 100, < 6 bulan -> 0.
func skorLinearTurun(nilai, batasSkorPenuh, batasSkorNol float64) float64 {
	if batasSkorPenuh == batasSkorNol {
		if nilai >= batasSkorPenuh {
			return 100
		}
		return 0
	}

	if batasSkorPenuh > batasSkorNol {
		// Makin besar makin baik (mis. lama usaha dalam bulan).
		if nilai >= batasSkorPenuh {
			return 100
		}
		if nilai <= batasSkorNol {
			return 0
		}
		return (nilai - batasSkorNol) / (batasSkorPenuh - batasSkorNol) * 100
	}

	// Makin kecil makin baik (mis. rasio angsuran).
	if nilai <= batasSkorPenuh {
		return 100
	}
	if nilai >= batasSkorNol {
		return 0
	}
	return (batasSkorNol - nilai) / (batasSkorNol - batasSkorPenuh) * 100
}
