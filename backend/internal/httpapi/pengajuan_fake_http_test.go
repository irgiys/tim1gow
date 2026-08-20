package httpapi

import (
	"context"
	"time"

	"github.com/irgiys/tim1gow/backend/internal/service"
)

// Berkas ini hanya memuat tiruan repository untuk test route FR-02/FR-03/FR-04.
// Tidak ada assertion di sini; tiruannya sengaja dipisah supaya berkas test
// tetap ringkas dan yang diuji tetap terlihat.
//
// Batas plafon dan daftar dokumen wajib disimpan sebagai FIELD yang bisa diubah
// di tengah test, bukan konstanta — angka ambang tidak boleh hidup di kode
// maupun di test (AGENTS.md Larangan 3).

type fakePengajuanRepoHTTP struct {
	baris     map[int64]service.Pengajuan
	anggota   map[int64][]service.PengajuanAnggota
	urutan    map[string]int
	idBerikut int64
	err       error
}

func newFakePengajuanRepoHTTP() *fakePengajuanRepoHTTP {
	return &fakePengajuanRepoHTTP{
		baris:   map[int64]service.Pengajuan{},
		anggota: map[int64][]service.PengajuanAnggota{},
		urutan:  map[string]int{},
	}
}

func (f *fakePengajuanRepoHTTP) Simpan(_ context.Context, p *service.Pengajuan) error {
	if f.err != nil {
		return f.err
	}
	f.idBerikut++
	p.ID = f.idBerikut
	f.baris[p.ID] = *p
	return nil
}

func (f *fakePengajuanRepoHTTP) Perbarui(_ context.Context, p *service.Pengajuan) error {
	if _, ada := f.baris[p.ID]; !ada {
		return service.ErrTidakDitemukan
	}
	f.baris[p.ID] = *p
	return nil
}

func (f *fakePengajuanRepoHTTP) CariID(_ context.Context, id int64) (service.Pengajuan, error) {
	if f.err != nil {
		return service.Pengajuan{}, f.err
	}
	p, ada := f.baris[id]
	if !ada {
		return service.Pengajuan{}, service.ErrTidakDitemukan
	}
	return p, nil
}

func (f *fakePengajuanRepoHTTP) DaftarMilikAO(_ context.Context, aoID int64) ([]service.Pengajuan, error) {
	if f.err != nil {
		return nil, f.err
	}
	hasil := make([]service.Pengajuan, 0, len(f.baris))
	for _, p := range f.baris {
		if p.AOID == aoID {
			hasil = append(hasil, p)
		}
	}
	return hasil, nil
}

func (f *fakePengajuanRepoHTTP) AmbilNomorUrutHarian(_ context.Context, tanggal time.Time) (int, error) {
	kunci := tanggal.Format("20060102")
	f.urutan[kunci]++
	return f.urutan[kunci], nil
}

func (f *fakePengajuanRepoHTTP) SimpanAnggota(_ context.Context, pengajuanID int64, a []service.PengajuanAnggota) error {
	f.anggota[pengajuanID] = a
	return nil
}

func (f *fakePengajuanRepoHTTP) DaftarAnggota(_ context.Context, pengajuanID int64) ([]service.PengajuanAnggota, error) {
	return f.anggota[pengajuanID], nil
}

func (f *fakePengajuanRepoHTTP) DalamTransaksi(ctx context.Context, fn func(tx service.PengajuanRepository) error) error {
	return fn(f)
}

// fakeBatasPlafonHTTP meniru baris batas plafon di tabel parameter.
type fakeBatasPlafonHTTP struct {
	minimum   int64
	maksimum  int64
	ditemukan bool
}

func (f *fakeBatasPlafonHTTP) BatasPlafon(context.Context) (int64, int64, bool, error) {
	return f.minimum, f.maksimum, f.ditemukan, nil
}

type fakeDokumenRepoHTTP struct {
	baris     map[int64]service.Dokumen
	idBerikut int64
}

func newFakeDokumenRepoHTTP() *fakeDokumenRepoHTTP {
	return &fakeDokumenRepoHTTP{baris: map[int64]service.Dokumen{}}
}

func (f *fakeDokumenRepoHTTP) Simpan(_ context.Context, d *service.Dokumen) error {
	f.idBerikut++
	d.ID = f.idBerikut
	f.baris[d.ID] = *d
	return nil
}

func (f *fakeDokumenRepoHTTP) Perbarui(_ context.Context, d *service.Dokumen) error {
	if _, ada := f.baris[d.ID]; !ada {
		return service.ErrTidakDitemukan
	}
	f.baris[d.ID] = *d
	return nil
}

func (f *fakeDokumenRepoHTTP) CariID(_ context.Context, id int64) (service.Dokumen, error) {
	d, ada := f.baris[id]
	if !ada {
		return service.Dokumen{}, service.ErrTidakDitemukan
	}
	return d, nil
}

func (f *fakeDokumenRepoHTTP) DaftarPerPengajuan(_ context.Context, pengajuanID int64) ([]service.Dokumen, error) {
	hasil := make([]service.Dokumen, 0, len(f.baris))
	for _, d := range f.baris {
		if d.PengajuanID == pengajuanID {
			hasil = append(hasil, d)
		}
	}
	return hasil, nil
}

func (f *fakeDokumenRepoHTTP) CariAktif(_ context.Context, pengajuanID int64, jenis string) (service.Dokumen, error) {
	var hasil service.Dokumen
	ketemu := false
	for _, d := range f.baris {
		if d.PengajuanID == pengajuanID && d.JenisDokumen == jenis && d.ID > hasil.ID {
			hasil, ketemu = d, true
		}
	}
	if !ketemu {
		return service.Dokumen{}, service.ErrTidakDitemukan
	}
	return hasil, nil
}

// fakeDokumenWajibHTTP meniru daftar jenis dokumen wajib di tabel parameter.
type fakeDokumenWajibHTTP struct {
	jenis []string
}

func (f *fakeDokumenWajibHTTP) JenisDokumenWajib(context.Context) ([]string, error) {
	return f.jenis, nil
}

type fakeSurveiRepoHTTP struct {
	baris     map[int64]service.Survei
	idBerikut int64
}

func newFakeSurveiRepoHTTP() *fakeSurveiRepoHTTP {
	return &fakeSurveiRepoHTTP{baris: map[int64]service.Survei{}}
}

func (f *fakeSurveiRepoHTTP) Simpan(_ context.Context, s *service.Survei) error {
	f.idBerikut++
	s.ID = f.idBerikut
	f.baris[s.ID] = *s
	return nil
}

func (f *fakeSurveiRepoHTTP) DaftarPerPengajuan(_ context.Context, pengajuanID int64) ([]service.Survei, error) {
	hasil := make([]service.Survei, 0, len(f.baris))
	for _, s := range f.baris {
		if s.PengajuanID == pengajuanID {
			hasil = append(hasil, s)
		}
	}
	return hasil, nil
}

func (f *fakeSurveiRepoHTTP) AdaSurveiValid(_ context.Context, pengajuanID int64) (bool, error) {
	for _, s := range f.baris {
		if s.PengajuanID == pengajuanID && s.Status == service.StatusSurveiValid {
			return true, nil
		}
	}
	return false, nil
}
