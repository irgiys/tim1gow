package service

import (
	"context"
	"time"
)

// Berkas ini meniru tabel `pengajuan`, `dokumen`, dan `survei` di memori.
// Barisnya dapat DIUBAH di tengah test — itulah yang membuktikan service membaca
// nilai dari data, bukan dari konstanta (AC-15, AGENTS.md Larangan 3).

type fakePengajuanRepo struct {
	baris     map[int64]Pengajuan
	anggota   map[int64][]PengajuanAnggota
	urutan    map[string]int
	idBerikut int64

	// jumlahTransaksi mencatat berapa kali service membuka transaksi, dipakai
	// memastikan pembuatan nomor referensi memang terjadi di dalam satu transaksi.
	jumlahTransaksi int
}

func newFakePengajuanRepo() *fakePengajuanRepo {
	return &fakePengajuanRepo{
		baris:   map[int64]Pengajuan{},
		anggota: map[int64][]PengajuanAnggota{},
		urutan:  map[string]int{},
	}
}

func (f *fakePengajuanRepo) Simpan(_ context.Context, p *Pengajuan) error {
	f.idBerikut++
	p.ID = f.idBerikut
	p.DibuatPada = time.Now()
	p.DiperbaruiPada = p.DibuatPada
	f.baris[p.ID] = *p
	return nil
}

func (f *fakePengajuanRepo) Perbarui(_ context.Context, p *Pengajuan) error {
	if _, ada := f.baris[p.ID]; !ada {
		return ErrTidakDitemukan
	}
	p.DiperbaruiPada = time.Now()
	f.baris[p.ID] = *p
	return nil
}

func (f *fakePengajuanRepo) CariID(_ context.Context, id int64) (Pengajuan, error) {
	p, ada := f.baris[id]
	if !ada {
		return Pengajuan{}, ErrTidakDitemukan
	}
	return p, nil
}

func (f *fakePengajuanRepo) DaftarMilikAO(_ context.Context, aoID int64) ([]Pengajuan, error) {
	var hasil []Pengajuan
	for _, p := range f.baris {
		if p.AOID == aoID {
			hasil = append(hasil, p)
		}
	}
	return hasil, nil
}

// AmbilNomorUrutHarian meniru counter harian terkunci: urutan naik per tanggal
// dan tidak pernah mengembalikan angka yang sama dua kali (BR-12).
func (f *fakePengajuanRepo) AmbilNomorUrutHarian(_ context.Context, tanggal time.Time) (int, error) {
	kunci := tanggal.Format("20060102")
	f.urutan[kunci]++
	return f.urutan[kunci], nil
}

func (f *fakePengajuanRepo) SimpanAnggota(_ context.Context, pengajuanID int64, a []PengajuanAnggota) error {
	f.anggota[pengajuanID] = a
	return nil
}

func (f *fakePengajuanRepo) DaftarAnggota(_ context.Context, pengajuanID int64) ([]PengajuanAnggota, error) {
	return f.anggota[pengajuanID], nil
}

func (f *fakePengajuanRepo) DalamTransaksi(ctx context.Context, fn func(tx PengajuanRepository) error) error {
	f.jumlahTransaksi++
	return fn(f)
}

// fakeBatasPlafonRepo meniru baris batas plafon di tabel parameter. Nilai awal
// mengikuti brief; test WAJIB mengubahnya untuk membuktikan angkanya dibaca dari
// data, bukan dari konstanta di service.
type fakeBatasPlafonRepo struct {
	minimum   int64
	maksimum  int64
	ditemukan bool
}

func newFakeBatasPlafonRepo() *fakeBatasPlafonRepo {
	return &fakeBatasPlafonRepo{minimum: 5_000_000, maksimum: 500_000_000, ditemukan: true}
}

func (f *fakeBatasPlafonRepo) BatasPlafon(_ context.Context) (int64, int64, bool, error) {
	return f.minimum, f.maksimum, f.ditemukan, nil
}

type fakeDokumenRepo struct {
	baris     map[int64]Dokumen
	idBerikut int64
}

func newFakeDokumenRepo() *fakeDokumenRepo {
	return &fakeDokumenRepo{baris: map[int64]Dokumen{}}
}

func (f *fakeDokumenRepo) Simpan(_ context.Context, d *Dokumen) error {
	f.idBerikut++
	d.ID = f.idBerikut
	f.baris[d.ID] = *d
	return nil
}

func (f *fakeDokumenRepo) Perbarui(_ context.Context, d *Dokumen) error {
	if _, ada := f.baris[d.ID]; !ada {
		return ErrTidakDitemukan
	}
	f.baris[d.ID] = *d
	return nil
}

func (f *fakeDokumenRepo) CariID(_ context.Context, id int64) (Dokumen, error) {
	d, ada := f.baris[id]
	if !ada {
		return Dokumen{}, ErrTidakDitemukan
	}
	return d, nil
}

func (f *fakeDokumenRepo) DaftarPerPengajuan(_ context.Context, pengajuanID int64) ([]Dokumen, error) {
	var hasil []Dokumen
	for _, d := range f.baris {
		if d.PengajuanID == pengajuanID {
			hasil = append(hasil, d)
		}
	}
	return hasil, nil
}

// CariAktif mengembalikan dokumen terakhir berjenis tertentu, meniru perilaku
// query yang mengambil baris terbaru.
func (f *fakeDokumenRepo) CariAktif(_ context.Context, pengajuanID int64, jenis string) (Dokumen, error) {
	var hasil Dokumen
	ketemu := false
	for _, d := range f.baris {
		if d.PengajuanID == pengajuanID && d.JenisDokumen == jenis && d.ID > hasil.ID {
			hasil, ketemu = d, true
		}
	}
	if !ketemu {
		return Dokumen{}, ErrTidakDitemukan
	}
	return hasil, nil
}

// fakeDokumenWajibRepo meniru daftar dokumen wajib di tabel parameter.
type fakeDokumenWajibRepo struct {
	jenis []string
}

func newFakeDokumenWajibRepo() *fakeDokumenWajibRepo {
	return &fakeDokumenWajibRepo{jenis: []string{
		JenisDokumenKTP,
		JenisDokumenKK,
		JenisDokumenSuratKeteranganUsaha,
	}}
}

func (f *fakeDokumenWajibRepo) JenisDokumenWajib(_ context.Context) ([]string, error) {
	return f.jenis, nil
}

type fakeSurveiRepo struct {
	baris     map[int64]Survei
	idBerikut int64
}

func newFakeSurveiRepo() *fakeSurveiRepo {
	return &fakeSurveiRepo{baris: map[int64]Survei{}}
}

func (f *fakeSurveiRepo) Simpan(_ context.Context, s *Survei) error {
	f.idBerikut++
	s.ID = f.idBerikut
	f.baris[s.ID] = *s
	return nil
}

func (f *fakeSurveiRepo) DaftarPerPengajuan(_ context.Context, pengajuanID int64) ([]Survei, error) {
	var hasil []Survei
	for _, s := range f.baris {
		if s.PengajuanID == pengajuanID {
			hasil = append(hasil, s)
		}
	}
	return hasil, nil
}

func (f *fakeSurveiRepo) AdaSurveiValid(_ context.Context, pengajuanID int64) (bool, error) {
	for _, s := range f.baris {
		if s.PengajuanID == pengajuanID && s.Status == StatusSurveiValid {
			return true, nil
		}
	}
	return false, nil
}
