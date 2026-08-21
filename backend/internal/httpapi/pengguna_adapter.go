package httpapi

import (
	"context"

	"github.com/irgiys/tim1gow/backend/internal/repository"
)

// adapterPengguna menjembatani repository.PenggunaRepository ke antarmuka
// sempit yang dibutuhkan handler (SumberPengguna) dan middleware
// (PemeriksaPengguna).
//
// Adapter ini ada supaya httpapi tidak memaksa repository memakai tipe milik
// httpapi, dan sebaliknya handler tidak bergantung pada struct baris database.
type adapterPengguna struct {
	repo repository.PenggunaRepository
}

func (a adapterPengguna) CariByEmailUntukLogin(ctx context.Context, email string) (*PenggunaLogin, error) {
	p, err := a.repo.CariByEmail(ctx, email)
	if err != nil || p == nil {
		return nil, err
	}
	return &PenggunaLogin{
		ID:           p.ID,
		Nama:         p.Nama,
		Email:        p.Email,
		PasswordHash: p.PasswordHash,
		Peran:        p.Peran,
		Aktif:        p.Aktif,
	}, nil
}

func (a adapterPengguna) PenggunaAktif(ctx context.Context, id int64) (bool, error) {
	return a.repo.PenggunaAktif(ctx, id)
}
