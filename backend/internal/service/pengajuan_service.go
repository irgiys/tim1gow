package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/irgiys/tim1gow/backend/internal/domain"
)

// PengajuanService memuat aturan bisnis pengajuan pembiayaan mikro (FR-02).
// Lapisan ini tidak tahu apa pun tentang HTTP dan tidak membangun SQL
// (AGENTS.md bagian 3).
type PengajuanService struct {
	repo    PengajuanRepository
	batas   BatasPlafonRepository
	sekaran func() time.Time
	// audit mencatat jejak BR-10. Boleh nil pada test unit yang hanya
	// memeriksa aturan plafon/nomor referensi.
	audit AuditService
}

// ValidationError adalah kegagalan validasi masukan, dipetakan handler ke HTTP
// 400 (AGENTS.md bagian 4.3). Ini BUKAN pelanggaran aturan bisnis — BR-xx
// memakai domain.BusinessRuleError dan dipetakan ke 422.
//
// Tipe ini hidup di sini, bukan di paket domain, karena berkas domain/errors.go
// dimiliki anggota lain. Kalau Tech Lead memutuskan tipe ini milik domain,
// pemindahannya lewat PR terpisah.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return "validasi: " + e.Message }

// NewValidationError membuat kegagalan validasi. Pesannya TIDAK BOLEH memuat
// NIK, nomor dokumen, atau path foto (BR-11).
func NewValidationError(format string, args ...any) *ValidationError {
	return &ValidationError{Message: fmt.Sprintf(format, args...)}
}

// BatasPlafonRepository membaca batas plafon minimum dan maksimum dari tabel
// parameter. Angkanya TIDAK ditulis sebagai konstanta di kode maupun di test
// (AGENTS.md Larangan 3) supaya ADM dapat mengubahnya tanpa deploy ulang.
type BatasPlafonRepository interface {
	BatasPlafon(ctx context.Context) (minimum, maksimum int64, ditemukan bool, err error)
}

// NewPengajuanService membuat service dengan sumber waktu bawaan.
func NewPengajuanService(repo PengajuanRepository, batas BatasPlafonRepository) *PengajuanService {
	return &PengajuanService{repo: repo, batas: batas, sekaran: time.Now}
}

// DenganAudit memasang pencatat audit (BR-10). Dipasang di wiring produksi;
// test unit yang hanya memeriksa aturan plafon/nomor referensi boleh
// mengabaikannya, sama seperti pola NewSkoringServiceWithAudit.
//
// Ketika audit terpasang, kegagalan mencatat membuat operasi GAGAL — bukan
// diabaikan. Pengajuan yang tersimpan tanpa jejak aktor melanggar BR-10.
func (s *PengajuanService) DenganAudit(a AuditService) *PengajuanService {
	s.audit = a
	return s
}

// DenganWaktu mengganti sumber waktu; dipakai test agar tanggal pada nomor
// referensi dapat ditentukan.
func (s *PengajuanService) DenganWaktu(f func() time.Time) *PengajuanService {
	s.sekaran = f
	return s
}

// InputPengajuan adalah data yang dikirim AO saat membuat pengajuan (FR-02).
type InputPengajuan struct {
	AOID           int64
	Tipe           TipePengajuan
	NamaNasabah    string
	NIK            string
	AlamatUsaha    string
	JenisUsaha     string
	JenisAkad      domain.Akad
	PlafonDiajukan int64
	TenorBulan     int
	Anggota        []PengajuanAnggota
}

// Buat membuat pengajuan baru berstatus DRAFT beserta nomor referensinya.
//
// Nomor referensi dibangkitkan di dalam transaksi yang sama dengan penyimpanan
// barisnya, memakai counter harian terkunci, sehingga tidak pernah dipakai ulang
// walaupun dua AO menekan simpan bersamaan (BR-12).
func (s *PengajuanService) Buat(ctx context.Context, in InputPengajuan) (Pengajuan, error) {
	if err := s.validasiInput(in); err != nil {
		return Pengajuan{}, err
	}

	total := in.PlafonDiajukan
	if in.Tipe == TipeKelompok {
		total = totalPlafonAnggota(in.Anggota)
	}
	if err := s.PastikanPlafonDalamBatas(ctx, total); err != nil {
		return Pengajuan{}, err
	}

	p := Pengajuan{
		Tipe:           in.Tipe,
		AOID:           in.AOID,
		NamaNasabah:    strings.TrimSpace(in.NamaNasabah),
		NIK:            strings.TrimSpace(in.NIK),
		AlamatUsaha:    strings.TrimSpace(in.AlamatUsaha),
		JenisUsaha:     strings.TrimSpace(in.JenisUsaha),
		JenisAkad:      string(in.JenisAkad),
		PlafonDiajukan: total,
		TenorBulan:     in.TenorBulan,
		Status:         string(domain.StatusDraft),
	}

	err := s.repo.DalamTransaksi(ctx, func(tx PengajuanRepository) error {
		tanggal := s.sekaran()
		urut, err := tx.AmbilNomorUrutHarian(ctx, tanggal)
		if err != nil {
			return err
		}
		p.NomorReferensi = FormatNomorReferensi(tanggal, urut)

		if err := tx.Simpan(ctx, &p); err != nil {
			return err
		}
		if in.Tipe == TipeKelompok {
			return tx.SimpanAnggota(ctx, p.ID, in.Anggota)
		}
		return nil
	})
	if err != nil {
		return Pengajuan{}, err
	}

	// BR-10: pembuatan pengajuan adalah perubahan keadaan, jadi wajib punya
	// aktor dan timestamp. Dicatat SETELAH transaksi berhasil supaya audit
	// tidak pernah menyebut pengajuan yang gagal disimpan.
	if err := s.catatAudit(ctx, p.ID, domain.AksiBuatPengajuan, "",
		string(domain.StatusDraft), "pengajuan dibuat "+p.NomorReferensi,
		in.AOID, domain.PeranAO); err != nil {
		return Pengajuan{}, err
	}
	return p, nil
}

// catatAudit merekam satu jejak perubahan keadaan (BR-10).
//
// Ketika audit belum dipasang (test unit), pencatatan dilewati. Di jalur
// produksi audit SELALU dipasang lewat DenganAudit, dan kegagalan mencatat
// diteruskan sebagai error — bukan ditelan — supaya tidak ada perubahan
// keadaan yang kehilangan jejak aktornya.
//
// Catatan tidak boleh memuat NIK, nomor dokumen, atau path foto (BR-11);
// pemanggil memakai nomor referensi atau id internal.
func (s *PengajuanService) catatAudit(ctx context.Context, pengajuanID int64,
	aksi domain.AksiAudit, sebelum, sesudah, catatan string,
	aktorID int64, aktorPeran domain.Peran) error {
	if s.audit == nil {
		return nil
	}
	id := pengajuanID
	return s.audit.Catat(ctx, domain.CatatAuditInput{
		PengajuanID:   &id,
		Aksi:          aksi,
		StatusSebelum: domain.StatusPengajuan(sebelum),
		StatusSesudah: domain.StatusPengajuan(sesudah),
		Catatan:       catatan,
		ActorID:       aktorID,
		ActorRole:     aktorPeran,
	})
}

// PastikanPlafonDalamBatas menegakkan BR-01: plafon di bawah batas minimum atau
// di atas batas maksimum ditolak, dengan pesan yang menjelaskan batasnya.
// Kedua batas dibaca dari tabel parameter setiap kali dipanggil.
func (s *PengajuanService) PastikanPlafonDalamBatas(ctx context.Context, plafon int64) error {
	minimum, maksimum, ditemukan, err := s.batas.BatasPlafon(ctx)
	if err != nil {
		return err
	}
	if !ditemukan {
		return domain.NewConfigError("batas plafon belum diatur di tabel parameter")
	}
	if plafon < minimum || plafon > maksimum {
		return domain.NewBusinessRuleError("BR-01",
			"plafon %s di luar batas yang diizinkan (%s s.d. %s)",
			formatRupiah(plafon), formatRupiah(minimum), formatRupiah(maksimum))
	}
	return nil
}

// FormatNomorReferensi menyusun nomor referensi IMT-YYYYMMDD-NNNN (BR-12).
// Nomor ini hanya dibangkitkan di server (AGENTS.md Larangan 4).
func FormatNomorReferensi(tanggal time.Time, urut int) string {
	return fmt.Sprintf("IMT-%s-%04d", tanggal.Format("20060102"), urut)
}

func (s *PengajuanService) validasiInput(in InputPengajuan) error {
	var kosong []string
	if strings.TrimSpace(in.NamaNasabah) == "" {
		kosong = append(kosong, "nama nasabah")
	}
	if strings.TrimSpace(in.NIK) == "" {
		kosong = append(kosong, "NIK")
	}
	if strings.TrimSpace(in.AlamatUsaha) == "" {
		kosong = append(kosong, "alamat usaha")
	}
	if strings.TrimSpace(in.JenisUsaha) == "" {
		kosong = append(kosong, "jenis usaha")
	}
	if len(kosong) > 0 {
		return NewValidationError("%s wajib diisi", strings.Join(kosong, ", "))
	}
	if in.JenisAkad != domain.AkadMurabahah && in.JenisAkad != domain.AkadMusyarakah {
		return NewValidationError("jenis akad tidak dikenal")
	}
	if in.TenorBulan <= 0 {
		return NewValidationError("tenor wajib lebih dari 0 bulan")
	}
	if in.Tipe == TipeKelompok && len(in.Anggota) == 0 {
		return NewValidationError("pengajuan kelompok wajib memiliki minimal satu anggota")
	}
	return nil
}

func totalPlafonAnggota(anggota []PengajuanAnggota) int64 {
	var total int64
	for _, a := range anggota {
		total += a.PlafonAnggota
	}
	return total
}

// formatRupiah menulis nominal dengan pemisah ribuan titik, mis. 5.000.000.
func formatRupiah(n int64) string {
	s := fmt.Sprintf("%d", n)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	return "Rp " + b.String()
}
