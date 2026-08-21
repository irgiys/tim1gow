package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/irgiys/tim1gow/backend/internal/auth"
	"github.com/irgiys/tim1gow/backend/internal/domain"
)

// PenggunaLogin adalah data pengguna yang dibutuhkan handler login.
//
// Bentuk ini dipakai supaya httpapi tidak bergantung pada tipe repository,
// sehingga test dapat memasang implementasi palsu.
type PenggunaLogin struct {
	ID           int64
	Nama         string
	Email        string
	PasswordHash string
	Peran        domain.Peran
	Aktif        bool
}

// SumberPengguna adalah kebutuhan handler login terhadap lapisan data.
type SumberPengguna interface {
	CariByEmailUntukLogin(ctx context.Context, email string) (*PenggunaLogin, error)
}

// AuthHandler melayani FR-01: login dan identitas pengguna saat ini.
type AuthHandler struct {
	sumber      SumberPengguna
	secret      []byte
	masaBerlaku time.Duration
	// cocokkan disuntikkan supaya test tidak perlu menghitung bcrypt asli
	// (bcrypt mahal secara sengaja; 6 test x ~60ms menambah waktu CI).
	cocokkan func(hash, password string) bool
	sekarang func() time.Time
}

// NewAuthHandler membuat handler login.
func NewAuthHandler(sumber SumberPengguna, secret []byte, masaBerlaku time.Duration,
	cocokkan func(hash, password string) bool) *AuthHandler {
	return &AuthHandler{
		sumber:      sumber,
		secret:      secret,
		masaBerlaku: masaBerlaku,
		cocokkan:    cocokkan,
		sekarang:    time.Now,
	}
}

type permintaanLogin struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type jawabanLogin struct {
	Token     string       `json:"token"`
	Peran     domain.Peran `json:"peran"`
	Nama      string       `json:"nama"`
	ID        int64        `json:"id"`
	BerlakuKe int64        `json:"berlaku_sampai"`
}

// Login menukar email + password dengan JWT (AC-01).
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req permintaanLogin
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR",
			"format permintaan tidak valid; field email dan password wajib diisi", "")
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR",
			"email dan password wajib diisi", "")
		return
	}

	p, err := h.sumber.CariByEmailUntukLogin(r.Context(), email)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR",
			"gagal memproses login", "")
		return
	}

	// Pesan yang SAMA untuk "email tidak terdaftar" dan "password salah".
	// Membedakannya membocorkan email mana yang terdaftar.
	const pesanGagal = "email atau password salah"

	if p == nil {
		// Tetap hitung bcrypt terhadap hash dummy supaya waktu respons untuk
		// email yang tidak ada tidak jauh lebih cepat (user enumeration).
		h.cocokkan("$2a$10$invalidinvalidinvalidinvalidinvalidinvalidinvalidinvalidinv", req.Password)
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", pesanGagal, "")
		return
	}
	if !h.cocokkan(p.PasswordHash, req.Password) {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", pesanGagal, "")
		return
	}
	if !p.Aktif {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "akun tidak aktif", "")
		return
	}

	sekarang := h.sekarang()
	token, err := auth.Terbitkan(h.secret, p.ID, string(p.Peran), h.masaBerlaku, sekarang)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR",
			"gagal menerbitkan token", "")
		return
	}

	respondJSON(w, http.StatusOK, jawabanLogin{
		Token:     token,
		Peran:     p.Peran,
		Nama:      p.Nama,
		ID:        p.ID,
		BerlakuKe: sekarang.Add(h.masaBerlaku).Unix(),
	})
}

// Saya mengembalikan identitas dari token yang sedang dipakai.
func (h *AuthHandler) Saya(w http.ResponseWriter, r *http.Request) {
	ident, ok := IdentitasDari(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "token tidak ada", "")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"id":    ident.PenggunaID,
		"peran": string(ident.Peran),
	})
}
