// Operasi baca pengajuan dan dokumen yang dipakai endpoint FR-02/FR-03.
//
// Aturan lingkup akses ada DI SINI, bukan di handler (AGENTS.md Larangan 17):
// AO hanya boleh melihat pengajuan miliknya sendiri, dan penegakannya di
// server, bukan dengan menyembunyikan baris di UI (Larangan 6).
package service

import (
	"context"

	"github.com/irgiys/tim1gow/backend/internal/domain"
)

// Daftar mengembalikan daftar pengajuan sesuai peran:
// - AO hanya melihat pengajuan yang ia buat sendiri (Larangan 6).
// - Peran lain (ANL, KCP, KC, KOM, ADM) melihat seluruh pengajuan aktif.
func (s *PengajuanService) Daftar(ctx context.Context, penggunaID int64, peran domain.Peran) ([]Pengajuan, error) {
	if peran == domain.PeranAO {
		return s.DaftarMilikAO(ctx, penggunaID)
	}
	return s.repo.DaftarSemua(ctx)
}

// DaftarMilikAO mengembalikan pengajuan milik seorang AO, terbaru lebih dulu.
func (s *PengajuanService) DaftarMilikAO(ctx context.Context, aoID int64) ([]Pengajuan, error) {
	if aoID <= 0 {
		return nil, NewValidationError("identitas AO tidak diketahui")
	}
	return s.repo.DaftarMilikAO(ctx, aoID)
}

// Detail mengembalikan satu pengajuan dengan pemeriksaan lingkup akses.
//
// AO hanya boleh membuka pengajuan yang ia buat sendiri. Peran lain (ANL dan
// para approver) boleh membuka semuanya karena tugasnya memang memeriksa
// pengajuan orang lain.
//
// Pengajuan milik AO lain dijawab 404, bukan 403: membedakan keduanya
// memberi tahu penyerang bahwa sebuah id itu ada — kebocoran keberadaan data
// yang tidak perlu pada sistem perbankan.
func (s *PengajuanService) Detail(ctx context.Context, id, penggunaID int64, peran domain.Peran) (Pengajuan, error) {
	p, err := s.repo.CariID(ctx, id)
	if err != nil {
		return Pengajuan{}, err
	}

	if peran == domain.PeranAO && p.AOID != penggunaID {
		return Pengajuan{}, ErrTidakDitemukan
	}
	return p, nil
}

// DaftarPerPengajuan mengembalikan seluruh dokumen sebuah pengajuan (FR-03).
func (s *DokumenService) DaftarPerPengajuan(ctx context.Context, pengajuanID int64) ([]Dokumen, error) {
	return s.dok.DaftarPerPengajuan(ctx, pengajuanID)
}
