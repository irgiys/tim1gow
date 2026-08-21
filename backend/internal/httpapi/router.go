// Package httpapi berisi handler, route, dan middleware HTTP. Ia TIDAK memuat
// aturan bisnis (AGENTS.md Larangan 17) — hanya menghubungkan HTTP ke service.
package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"

	"github.com/irgiys/tim1gow/backend/internal/config"
	"github.com/irgiys/tim1gow/backend/internal/domain"
	"github.com/irgiys/tim1gow/backend/internal/repository"
	"github.com/irgiys/tim1gow/backend/internal/repository/db"
	"github.com/irgiys/tim1gow/backend/internal/service"
	"github.com/irgiys/tim1gow/backend/internal/slik"
)

// errorResponse adalah satu-satunya bentuk respons error untuk seluruh API
// (AGENTS.md bagian 4.3). Handler tidak menyusun JSON error manual.
type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Rule    string `json:"rule,omitempty"`
}

// respondJSON menulis payload JSON dengan status yang diberikan.
func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// respondError adalah helper terpusat untuk seluruh error API. Pesan tidak
// boleh memuat data pribadi (BR-11).
func respondError(w http.ResponseWriter, status int, code, message, rule string) {
	respondJSON(w, status, errorResponse{Error: code, Message: message, Rule: rule})
}

// NewRouter membangun router Chi lengkap dengan middleware dasar dan route.
// gdb boleh nil (mode tanpa DB); saat itu /readyz melaporkan 503.
func NewRouter(cfg config.Config, gdb *gorm.DB) http.Handler {
	var appH *ApprovalHandler
	var audH *AuditHandler
	var skoH *SkoringHandler
	var authH *AuthHandler
	var pjnH *PengajuanHandler
	var slikH *SlikHandler
	var pemeriksa PemeriksaPengguna

	if gdb != nil {
		paramRepo := repository.NewParameterRepository(gdb)
		approvalRepo := repository.NewApprovalRepository(gdb)
		auditRepo := repository.NewAuditRepository(gdb)

		auditSvc := service.NewAuditService(auditRepo)
		approvalSvc := service.NewApprovalService(approvalRepo, paramRepo, auditSvc)

		appH = NewApprovalHandler(approvalSvc)
		audH = NewAuditHandler(auditSvc)

		// FR-06 & FR-07. Keduanya membaca tabel parameter lewat repository yang
		// sama, jadi perubahan bobot/rentang oleh ADM langsung berlaku (AC-15).
		//
		// DenganPrasyarat memasang sumber keadaan BR-03. Tanpa ini endpoint
		// skoring menolak semua permintaan sebagai CONFIG_ERROR — disengaja:
		// guard yang kehilangan sumber datanya harus berhenti, bukan lolos.
		skoH = NewSkoringHandler(
			service.NewSkoringServiceWithAudit(paramRepo, auditSvc).
				DenganPrasyarat(repository.NewPrasyaratSkoringRepository(gdb)),
			service.NewMarginService(paramRepo),
		)

		// FR-01. Adapter memakai bcrypt asli; test memakai pembanding palsu.
		adapter := adapterPengguna{repo: repository.NewPenggunaRepository(gdb)}
		pemeriksa = adapter
		authH = NewAuthHandler(adapter, []byte(cfg.JWTSecret), cfg.JWTExpiresIn,
			repository.CocokkanPassword)

		// FR-02, FR-03, FR-04. Batas plafon dan daftar dokumen wajib dibaca
		// dari tabel parameter, bukan konstanta di kode (Larangan 3).
		//
		// DenganAudit dipasang di ketiganya: setiap perubahan keadaan wajib
		// punya aktor dan timestamp (BR-10). Tanpa ini pengajuan, dokumen, dan
		// survei berubah tanpa jejak, dan audit_trail tetap kosong walau
		// endpoint /audit-nya sendiri berfungsi.
		pjnH = NewPengajuanHandler(
			service.NewPengajuanService(
				repository.NewPengajuanRepository(gdb),
				repository.NewBatasPlafonRepository(gdb),
			).DenganAudit(auditSvc),
			service.NewDokumenService(
				repository.NewDokumenRepository(gdb),
				repository.NewDokumenWajibRepository(gdb),
			).DenganAudit(auditSvc),
			service.NewSurveiService(repository.NewSurveiRepository(gdb)).
				DenganAudit(auditSvc),
		)

		// FR-05. SLIK Check (mock HTTP / service).
		slikClient := slik.NewClient(slik.Opsi{
			BaseURL:     cfg.SlikBaseURL,
			InquiryPath: cfg.SlikInquiryPath,
			Timeout:     cfg.SlikTimeout,
		})
		slikSvc := service.NewSlikService(service.OpsiSlikService{
			SlikRepo:        repository.NewSlikRepository(gdb),
			Pengajuan:       repository.NewPengajuanRepository(gdb),
			Client:          slikClient,
			Audit:           auditSvc,
			MasaBerlakuHari: 30,
		})
		slikH = NewSlikHandler(slikSvc)
	}

	return NewRouterLengkap(cfg, gdb, appH, audH, skoH, authH, pjnH, slikH, pemeriksa)
}

// NewRouterWithHandlers membangun router Chi dengan handler yang disediakan (mudah di-test/mock).
func NewRouterWithHandlers(cfg config.Config, gdb *gorm.DB, appH *ApprovalHandler, audH *AuditHandler) http.Handler {
	return NewRouterWithAllHandlers(cfg, gdb, appH, audH, nil)
}

// NewRouterWithAllHandlers membangun router TANPA autentikasi.
//
// HANYA UNTUK TEST HANDLER. Dipakai test yang menguji perilaku aturan bisnis
// satu endpoint tanpa perlu menerbitkan token lebih dulu. Jalur produksi
// selalu lewat NewRouter, yang memasang MiddlewareAuth + WajibPeran.
// Penegakan peran itu sendiri diuji langsung di middleware_auth_test.go
// (TestAC02_PeranLainDitolak403).
func NewRouterWithAllHandlers(cfg config.Config, gdb *gorm.DB, appH *ApprovalHandler, audH *AuditHandler, skoH *SkoringHandler) http.Handler {
	return NewRouterLengkap(cfg, gdb, appH, audH, skoH, nil, nil, nil, nil)
}

// NewRouterLengkap adalah bentuk penuh: seluruh handler plus autentikasi.
//
// Autentikasi dipasang hanya bila `pemeriksa` tidak nil. Pemisahan ini membuat
// test handler tetap sederhana, sementara jalur produksi (NewRouter) selalu
// mengirimkan pemeriksa sehingga setiap route API terlindungi.
func NewRouterLengkap(cfg config.Config, gdb *gorm.DB, appH *ApprovalHandler, audH *AuditHandler,
	skoH *SkoringHandler, authH *AuthHandler, pjnH *PengajuanHandler, slikH *SlikHandler, pemeriksa PemeriksaPengguna) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	// CORS dipasang sebelum autentikasi: browser TIDAK mengirim header
	// Authorization pada preflight, jadi preflight yang sampai ke
	// MiddlewareAuth dijawab 401 dan request aslinya tidak pernah dikirim.
	// Tanpa ini cfg.CorsAllowedOrigins dibaca dari environment lalu dibuang.
	r.Use(MiddlewareCORS(cfg.CorsAllowedOrigins))

	// authAktif menentukan apakah lapisan autentikasi dipasang. Bernilai false
	// hanya pada router test handler (NewRouterWithAllHandlers); jalur produksi
	// lewat NewRouter selalu mengirim pemeriksa.
	authAktif := pemeriksa != nil

	// Pada router TEST saja, identitas aktor boleh berasal dari header
	// X-Actor-ID / X-Actor-Role supaya test handler dapat menguji BR-02, BR-09,
	// dan AC-08 tanpa menerbitkan token lebih dulu.
	//
	// Ini TIDAK PERNAH aktif di produksi: NewRouter selalu mengirim pemeriksa,
	// sehingga authAktif true dan blok ini dilewati. Jalur produksi membaca
	// identitas hanya dari token yang sudah diverifikasi — header X-Actor-*
	// dari klien diabaikan sepenuhnya.
	if !authAktif {
		r.Use(injeksiIdentitasUji)
	}

	// peran mengembalikan middleware pembatas peran, atau pass-through ketika
	// autentikasi tidak aktif. Tanpa ini, router test akan menolak semuanya
	// dengan 401 karena tidak ada identitas di context.
	peran := func(p ...domain.Peran) func(http.Handler) http.Handler {
		if !authAktif {
			return func(next http.Handler) http.Handler { return next }
		}
		return WajibPeran(p...)
	}

	// Liveness: proses hidup, tidak menyentuh dependensi.
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
			"env":    cfg.AppEnv,
		})
	})

	// Readiness: pastikan DB benar-benar bisa dihubungi.
	r.Get("/readyz", func(w http.ResponseWriter, req *http.Request) {
		if gdb == nil {
			respondError(w, http.StatusServiceUnavailable, "DB_NOT_CONFIGURED",
				"database belum dikonfigurasi", "")
			return
		}
		ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
		defer cancel()
		if err := db.Ping(ctx, gdb); err != nil {
			respondError(w, http.StatusServiceUnavailable, "DB_UNAVAILABLE",
				"database tidak dapat dihubungi", "")
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"status": "ready", "db": "ok"})
	})

	// Route API
	r.Route("/api", func(api chi.Router) {
		// Login bersifat publik: ia justru yang menerbitkan token.
		if authH != nil {
			api.Post("/auth/login", authH.Login)
		}

		// Semua route di bawah ini WAJIB membawa token. Peran diperiksa per
		// endpoint sesuai SDD BAB 5 — AC-02 menguji ini lewat panggilan API
		// langsung, bukan lewat UI (AGENTS.md Larangan 6).
		api.Group(func(aman chi.Router) {
			if authAktif {
				aman.Use(MiddlewareAuth([]byte(cfg.JWTSecret), pemeriksa))
			}

			if authH != nil {
				aman.Get("/auth/me", authH.Saya)
			}

			if appH != nil {
				// ANL mengajukan ke jalur approval; approver memutuskan.
				aman.With(peran(domain.PeranANL)).
					Post("/pengajuan/{id}/ajukan-approval", appH.AjukanApproval)
				aman.With(peran(domain.PeranKCP, domain.PeranKC, domain.PeranKOM)).
					Post("/pengajuan/{id}/approval", appH.PutuskanApproval)
				aman.With(peran(domain.PeranAO, domain.PeranANL,
					domain.PeranKCP, domain.PeranKC, domain.PeranKOM)).
					Get("/pengajuan/{id}/approval", appH.DetailApproval)
			}

			if audH != nil {
				// FR-09 & AC-12 / AC-13: Audit Trail hanya ada method GET (append-only)
				aman.With(peran(domain.PeranAO, domain.PeranANL, domain.PeranKCP,
					domain.PeranKC, domain.PeranKOM, domain.PeranADM)).
					Get("/pengajuan/{id}/audit", audH.RiwayatPengajuan)
				aman.With(peran(domain.PeranADM, domain.PeranANL)).
					Get("/audit", audH.SemuaAudit)
			}

			if pjnH != nil {
				// FR-02 — Pengajuan. Membuat pengajuan adalah wewenang AO;
				// pemiliknya diambil dari token, bukan dari badan request.
				aman.With(peran(domain.PeranAO)).
					Post("/pengajuan", pjnH.Buat)

				// Daftar & detail dibuka untuk AO dan seluruh pemeriksa.
				// Pembatasan "AO hanya miliknya" ditegakkan di service
				// (pengajuan_baca.go), bukan dengan menyembunyikan baris di UI
				// (AGENTS.md Larangan 6).
				aman.With(peran(domain.PeranAO, domain.PeranANL,
					domain.PeranKCP, domain.PeranKC, domain.PeranKOM)).
					Get("/pengajuan", pjnH.Daftar)
				aman.With(peran(domain.PeranAO, domain.PeranANL,
					domain.PeranKCP, domain.PeranKC, domain.PeranKOM)).
					Get("/pengajuan/{id}", pjnH.Detail)

				// FR-03 — Dokumen. AO mengunggah, ANL memverifikasi.
				// Pemisahan peran ini adalah bentuk maker != checker (BR-09)
				// pada tahap dokumen: pengunggah tidak boleh memverifikasi
				// berkasnya sendiri.
				aman.With(peran(domain.PeranAO)).
					Post("/pengajuan/{id}/dokumen", pjnH.UploadDokumen)
				aman.With(peran(domain.PeranAO, domain.PeranANL,
					domain.PeranKCP, domain.PeranKC, domain.PeranKOM)).
					Get("/pengajuan/{id}/dokumen", pjnH.DaftarDokumen)
				aman.With(peran(domain.PeranANL)).
					Patch("/pengajuan/{id}/dokumen/{dokId}/verifikasi", pjnH.VerifikasiDokumen)

				// FR-04 — Survei lapangan. Hanya AO yang merekam OTS.
				aman.With(peran(domain.PeranAO)).
					Post("/pengajuan/{id}/survei", pjnH.RekamSurvei)
			}

			if slikH != nil {
				// FR-05 — SLIK check (ANL)
				aman.With(peran(domain.PeranANL)).
					Post("/pengajuan/{id}/slik", slikH.JalankanSLIK)
			}

			if skoH != nil {
				// Seluruh tahap skoring & margin adalah wewenang ANL (SDD BAB 5).
				aman.With(peran(domain.PeranANL)).Group(func(anl chi.Router) {
					// FR-06 skoring kelayakan; BR-03 dicek sebelum hitung, rincian
					// komponen ikut di respons (BR-08 / AC-07).
					anl.Post("/pengajuan/{id}/skoring", skoH.HitungSkoring)
					// AC-08: override grade oleh ANL, alasan wajib, tercatat di audit.
					anl.Patch("/pengajuan/{id}/skoring/override", skoH.OverrideGrade)
					// FR-07 margin/nisbah; di luar rentang grade -> 422 BR-06 (AC-09).
					anl.Post("/pengajuan/{id}/margin", skoH.TentukanMargin)
					anl.Get("/pengajuan/{id}/margin", skoH.RentangMargin)
				})
			}

			if slikH != nil {
				// FR-05 — SLIK Check adalah wewenang ANL (SDD BAB 5, FR-05).
				aman.With(peran(domain.PeranANL)).Group(func(anl chi.Router) {
					anl.Post("/pengajuan/{id}/slik", slikH.JalankanSLIK)
					anl.Post("/pengajuan/{id}/slik-check", slikH.JalankanSLIK)
				})
			}
		})
	})

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "sumber daya tidak ditemukan", "")
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "metode HTTP tidak diizinkan", "")
	})

	return r
}
