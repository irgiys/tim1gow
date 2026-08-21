package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/irgiys/tim1gow/backend/internal/auth"
	"github.com/irgiys/tim1gow/backend/internal/domain"
)

// kunciKonteks adalah tipe privat untuk key context, supaya paket lain tidak
// dapat menimpa nilai identitas dengan key string yang sama.
type kunciKonteks string

const kunciIdentitas kunciKonteks = "identitas"

// Identitas adalah pengguna yang terautentikasi pada satu request.
//
// Nilainya SELALU berasal dari token yang sudah diverifikasi, tidak pernah
// dari body atau query. Service memakainya sebagai aktor audit (BR-10) dan
// sebagai pembanding maker/checker (BR-09).
type Identitas struct {
	PenggunaID int64
	Peran      domain.Peran
}

// IdentitasDari mengambil identitas dari context. `ok` bernilai false bila
// request tidak melewati middleware auth.
func IdentitasDari(ctx context.Context) (Identitas, bool) {
	id, ok := ctx.Value(kunciIdentitas).(Identitas)
	return id, ok
}

// PemeriksaPengguna memeriksa apakah pengguna masih berhak memakai token.
//
// SDD BAB 7: pencabutan dilakukan lewat penonaktifan `pengguna.aktif`, yang
// diperiksa ulang SETIAP request. Tanpa ini, token pengguna yang sudah
// dinonaktifkan tetap berlaku sampai kedaluwarsa.
type PemeriksaPengguna interface {
	PenggunaAktif(ctx context.Context, id int64) (bool, error)
}

// MiddlewareAuth memverifikasi Bearer token dan menaruh identitas di context.
//
// Yang TIDAK dilakukan di sini: keputusan peran. Middleware ini hanya menjawab
// "siapa ini"; "boleh apa" ditentukan WajibPeran per route. Memisahkan keduanya
// membuat setiap route menyatakan perannya secara eksplisit, sehingga route
// baru tidak diam-diam terbuka untuk semua peran.
func MiddlewareAuth(secret []byte, pemeriksa PemeriksaPengguna) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := ambilBearer(r)
			if !ok {
				respondError(w, http.StatusUnauthorized, "UNAUTHORIZED",
					"token tidak ada atau format Authorization salah", "")
				return
			}

			klaim, err := auth.Verifikasi(secret, token, time.Now())
			if err != nil {
				// Pesan sengaja seragam: membedakan "kedaluwarsa" dan "tanda
				// tangan salah" ke klien memberi penyerang informasi gratis.
				respondError(w, http.StatusUnauthorized, "UNAUTHORIZED",
					"token tidak valid atau kedaluwarsa", "")
				return
			}

			// Pencabutan diperiksa tiap request (SDD BAB 7).
			if pemeriksa != nil {
				aktif, err := pemeriksa.PenggunaAktif(r.Context(), klaim.PenggunaID)
				if err != nil {
					respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR",
						"gagal memeriksa status pengguna", "")
					return
				}
				if !aktif {
					respondError(w, http.StatusUnauthorized, "UNAUTHORIZED",
						"akun tidak aktif", "")
					return
				}
			}

			ident := Identitas{PenggunaID: klaim.PenggunaID, Peran: domain.Peran(klaim.Peran)}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), kunciIdentitas, ident)))
		})
	}
}

// WajibPeran membatasi route hanya untuk peran yang disebut.
//
// AC-02 menguji ini secara langsung: panggilan API lintas peran wajib 403,
// bukan 200 dan bukan 404. Menyembunyikan tombol di UI bukan otorisasi
// (AGENTS.md Larangan 6).
func WajibPeran(diizinkan ...domain.Peran) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ident, ok := IdentitasDari(r.Context())
			if !ok {
				// Route terpasang WajibPeran tanpa MiddlewareAuth di depannya.
				// Ini salah pasang route, bukan kesalahan klien — tetapi
				// jawabannya tetap 401 supaya tidak membocorkan apa pun.
				respondError(w, http.StatusUnauthorized, "UNAUTHORIZED",
					"token tidak ada atau format Authorization salah", "")
				return
			}

			for _, p := range diizinkan {
				if ident.Peran == p {
					next.ServeHTTP(w, r)
					return
				}
			}

			respondError(w, http.StatusForbidden, "FORBIDDEN",
				"peran Anda tidak berwenang mengakses sumber daya ini", "")
		})
	}
}

// ambilBearer membaca header Authorization: Bearer <token>.
//
// Token TIDAK PERNAH dibaca dari query string: query ikut tercatat di log
// akses dan riwayat peramban.
func ambilBearer(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return "", false
	}
	// Skema diperlakukan case-insensitive sesuai RFC 7235.
	const prefix = "bearer "
	if len(h) <= len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return "", false
	}
	token := strings.TrimSpace(h[len(prefix):])
	if token == "" {
		return "", false
	}
	return token, true
}
