package httpapi

import (
	"context"
	"net/http"
	"strconv"

	"github.com/irgiys/tim1gow/backend/internal/domain"
)

// injeksiIdentitasUji menaruh identitas dari header X-Actor-ID / X-Actor-Role
// ke context.
//
// HANYA DIPASANG DI ROUTER TEST (NewRouterWithHandlers /
// NewRouterWithAllHandlers), yaitu ketika `pemeriksa` bernilai nil. Jalur
// produksi lewat NewRouter selalu mengirim pemeriksa, sehingga middleware ini
// tidak pernah ikut terpasang dan header X-Actor-* dari klien diabaikan.
//
// Kenapa ini ada: test handler untuk BR-02 (urutan approval), BR-09
// (maker != checker), dan AC-08 (override grade tercatat dengan identitas ANL)
// perlu berpura-pura menjadi aktor tertentu tanpa menerbitkan token lebih
// dulu. Sebelumnya kebutuhan itu dipenuhi oleh handler produksi yang membaca
// header secara langsung dengan fallback ke (id=1, ANL) — artinya identitas
// dapat dipalsukan siapa pun, dan tanpa header setiap aktor dianggap id=1
// sehingga audit trail mencatat orang yang salah.
//
// Memindahkan pembacaan header ke sini membuat kebutuhan test terpenuhi tanpa
// melubangi jalur produksi.
func injeksiIdentitasUji(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idStr := r.Header.Get("X-Actor-ID")
		peranStr := r.Header.Get("X-Actor-Role")

		// Tanpa header sama sekali, request diteruskan apa adanya. Handler yang
		// membutuhkan aktor akan menjawab 401 — bukan menebak aktor.
		if idStr == "" && peranStr == "" {
			next.ServeHTTP(w, r)
			return
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			next.ServeHTTP(w, r)
			return
		}

		ident := Identitas{PenggunaID: id, Peran: domain.Peran(peranStr)}
		next.ServeHTTP(w, r.WithContext(
			context.WithValue(r.Context(), kunciIdentitas, ident)))
	})
}
