package repository

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/irgiys/tim1gow/backend/internal/domain"
)

// Pengguna adalah satu baris tabel `pengguna` (migrasi 000004).
//
// PasswordHash sengaja tidak punya tag json: struct ini tidak boleh diserialkan
// apa adanya ke respons API.
type Pengguna struct {
	ID           int64        `gorm:"column:id;primaryKey"`
	Nama         string       `gorm:"column:nama"`
	Email        string       `gorm:"column:email"`
	PasswordHash string       `gorm:"column:password_hash"`
	Peran        domain.Peran `gorm:"column:peran"`
	Aktif        bool         `gorm:"column:aktif"`
}

func (Pengguna) TableName() string { return "pengguna" }

// PenggunaRepository adalah akses data untuk autentikasi (FR-01).
type PenggunaRepository interface {
	// CariByEmail mengembalikan nil tanpa error bila pengguna tidak ada —
	// handler login memperlakukan "tidak ada" dan "password salah" sama.
	CariByEmail(ctx context.Context, email string) (*Pengguna, error)
	// PenggunaAktif dipakai middleware untuk memeriksa pencabutan tiap request.
	PenggunaAktif(ctx context.Context, id int64) (bool, error)
}

type gormPenggunaRepo struct{ db *gorm.DB }

// NewPenggunaRepository membuat implementasi GORM untuk PenggunaRepository.
func NewPenggunaRepository(db *gorm.DB) PenggunaRepository {
	return &gormPenggunaRepo{db: db}
}

func (r *gormPenggunaRepo) CariByEmail(ctx context.Context, email string) (*Pengguna, error) {
	var p Pengguna
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *gormPenggunaRepo) PenggunaAktif(ctx context.Context, id int64) (bool, error) {
	var aktif bool
	err := r.db.WithContext(ctx).
		Model(&Pengguna{}).
		Select("aktif").
		Where("id = ?", id).
		Scan(&aktif).Error
	if err != nil {
		return false, err
	}
	return aktif, nil
}

// CocokkanPassword membandingkan password dengan hash bcrypt.
//
// Diletakkan di sini supaya pemanggil tidak pernah menyentuh PasswordHash
// secara langsung.
func CocokkanPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
