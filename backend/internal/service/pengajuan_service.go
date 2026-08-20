package service

import (
	"fmt"
	"time"

	"github.com/irgiys/tim1gow/backend/internal/domain"
)

// NomorReferensiRepository menerbitkan nomor urut pengajuan per tanggal.
//
// Kontraknya sengaja "maju terus": urutan tidak pernah dihitung ulang dari
// baris pengajuan yang masih hidup, karena BR-12 melarang nomor dipakai ulang
// walaupun pengajuan pemiliknya ditolak.
type NomorReferensiRepository interface {
	UrutanBerikutnya(tanggal string) (int, error)
	CatatNomor(nomor string) error
}

// PengajuanService memuat aturan bisnis pengajuan pembiayaan (FR-02).
// Lapisan ini tidak tahu apa pun tentang HTTP dan tidak membangun SQL
// (AGENTS.md bagian 3).
type PengajuanService struct {
	nomor NomorReferensiRepository
	param ParameterRepository
}

func NewPengajuanService(nomor NomorReferensiRepository, param ParameterRepository) *PengajuanService {
	return &PengajuanService{nomor: nomor, param: param}
}

// formatNomorReferensi adalah format wajib dari AGENTS.md bagian 4.1 dan
// AC-01: IMT-YYYYMMDD-NNNN. Formatnya tidak boleh diubah (Larangan 4).
const formatNomorReferensi = "IMT-%s-%04d"

// BuatNomorReferensi menerbitkan nomor referensi baru untuk tanggal tertentu
// (BR-12). Nomor dibangkitkan di server, bukan di frontend (Larangan 4).
//
// Tanggal diterima sebagai parameter, bukan diambil dari time.Now() di dalam,
// supaya perilakunya dapat diuji tanpa bergantung pada jam mesin.
func (s *PengajuanService) BuatNomorReferensi(tanggal time.Time) (string, error) {
	hari := tanggal.Format("20060102")

	urutan, err := s.nomor.UrutanBerikutnya(hari)
	if err != nil {
		return "", err
	}

	nomor := fmt.Sprintf(formatNomorReferensi, hari, urutan)

	if err := s.nomor.CatatNomor(nomor); err != nil {
		return "", err
	}
	return nomor, nil
}

// PastikanPlafonValid menegakkan BR-01: plafon di luar batas ditolak saat
// submit, dengan pesan yang menjelaskan batasnya.
//
// Kedua batas dibaca dari tabel parameter_umum setiap kali dipanggil — tidak
// di-cache dan tidak punya nilai default — supaya perubahan ADM langsung
// berlaku (FR-13) dan kegagalan konfigurasi tidak diam-diam meloloskan
// pengajuan (AGENTS.md Larangan 3 & 15).
func (s *PengajuanService) PastikanPlafonValid(totalPlafon int64) error {
	minimum, ada, err := s.param.Umum(KunciPlafonMinimum)
	if err != nil {
		return err
	}
	if !ada {
		return domain.NewConfigError("batas plafon minimum (%s) belum diatur", KunciPlafonMinimum)
	}

	maksimum, ada, err := s.param.Umum(KunciPlafonMaksimum)
	if err != nil {
		return err
	}
	if !ada {
		return domain.NewConfigError("batas plafon maksimum (%s) belum diatur", KunciPlafonMaksimum)
	}

	if float64(totalPlafon) < minimum || float64(totalPlafon) > maksimum {
		// Pesan menyebut kedua batas supaya AO tahu rentang yang berlaku.
		// Tanpa data pribadi apa pun (BR-11).
		return domain.NewBusinessRuleError("BR-01",
			"plafon %s di luar batas yang diizinkan: minimum Rp %s dan maksimum Rp %s",
			formatRupiah(totalPlafon),
			formatRupiah(int64(minimum)),
			formatRupiah(int64(maksimum)))
	}
	return nil
}

// formatRupiah menuliskan nominal dengan pemisah ribuan titik, mengikuti cara
// angka ditulis di brief dan di UI (mis. 5000000 -> "5.000.000").
func formatRupiah(n int64) string {
	if n < 0 {
		return "-" + formatRupiah(-n)
	}

	digit := fmt.Sprintf("%d", n)
	if len(digit) <= 3 {
		return digit
	}

	// Sisipkan titik setiap 3 digit dari kanan.
	depan := len(digit) % 3
	if depan == 0 {
		depan = 3
	}
	hasil := digit[:depan]
	for i := depan; i < len(digit); i += 3 {
		hasil += "." + digit[i:i+3]
	}
	return hasil
}
