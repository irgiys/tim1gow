package httpapi

import (
	"net/http"
	"strconv"
	"strings"
)

// MiddlewareCORS menjawab preflight dan memasang header CORS untuk origin yang
// terdaftar di CORS_ALLOWED_ORIGINS.
//
// Tanpa middleware ini config.CorsAllowedOrigins dibaca dari environment lalu
// dibuang: preflight OPTIONS dijawab 405 oleh MethodNotAllowed router, sehingga
// halaman login (Client Component yang fetch dari BROWSER, lintas origin
// :3000 -> :8080) tidak pernah dapat memanggil API. Backend sehat, tetapi tidak
// ada satu pun alur UI yang dapat diselesaikan.
//
// Ditulis dengan net/http saja, tanpa menambah dependensi (AGENTS.md Larangan 1).
//
// Keputusan keamanan yang disengaja:
//
//   - Origin dicocokkan PERSIS dengan daftar yang diizinkan. Tidak ada
//     pencocokan prefix/suffix: "http://localhost:3000.penyerang.com" tidak
//     boleh lolos hanya karena berawalan sama.
//   - Wildcard "*" TIDAK didukung. Ini API perbankan yang memakai Authorization
//     header; mengizinkan semua origin membuat token dapat dipakai dari
//     halaman mana pun.
//   - Origin yang tidak dikenal tidak diberi header CORS sama sekali. Request
//     tetap diproses (curl dan panggilan server-to-server tidak terpengaruh),
//     tetapi browser yang menegakkan CORS akan memblokirnya sendiri.
//   - Header Vary: Origin dipasang supaya cache/proxy tidak menyajikan respons
//     ber-header origin A kepada origin B.
func MiddlewareCORS(allowedOrigins []string) func(http.Handler) http.Handler {
	// Disalin ke map sekali saat wiring, bukan di setiap request.
	diizinkan := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if o = strings.TrimSpace(o); o != "" {
			diizinkan[o] = struct{}{}
		}
	}

	const (
		metodeDiizinkan = "GET, POST, PATCH, PUT, DELETE, OPTIONS"
		headerDiizinkan = "Authorization, Content-Type"
		// Preflight boleh di-cache browser 10 menit. Cukup pendek supaya
		// perubahan konfigurasi tidak tertahan lama saat demo.
		maxAgeDetik = 600
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Bukan request lintas origin: tidak ada yang perlu ditambahkan.
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Respons bergantung pada Origin, apa pun hasil pencocokannya.
			w.Header().Add("Vary", "Origin")

			if _, ok := diizinkan[origin]; !ok {
				// Origin asing: JANGAN pasang header CORS. Preflight-nya pun
				// ditolak, supaya browser tidak melanjutkan request aslinya.
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			// Preflight berhenti di sini: 204, tanpa menyentuh handler.
			// Ia tidak boleh melewati MiddlewareAuth, karena browser TIDAK
			// mengirim Authorization pada preflight — kalau diteruskan,
			// jawabannya 401 dan request aslinya tidak pernah dikirim.
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", metodeDiizinkan)
				w.Header().Set("Access-Control-Allow-Headers", headerDiizinkan)
				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(maxAgeDetik))
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
