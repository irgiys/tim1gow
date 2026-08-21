package httpapi

import (
	"net/http"
	"time"

	"github.com/irgiys/tim1gow/backend/internal/service"
)

// SlikHandler menangani endpoint HTTP SLIK check (FR-05, AC-05, AC-06).
//
// Handler ini TIDAK memuat aturan bisnis: evaluasi kolektibilitas 3/4/5 ditolak
// otomatis, kolektibilitas 2 dibatasi grade min 3, dan masa berlaku 30 hari
// (BR-04) semuanya hidup di internal/service. SlikService (AGENTS.md Larangan 17).
// Tugas handler hanya parsing request, memanggil service, dan memetakan hasil
// ke JSON.
type SlikHandler struct {
	slik *service.SlikService
}

// NewSlikHandler membuat instance SlikHandler baru.
func NewSlikHandler(slik *service.SlikService) *SlikHandler {
	return &SlikHandler{slik: slik}
}

// slikResponse adalah bentuk respons sukses POST /api/pengajuan/{id}/slik.
//
// BR-11: NIK sengaja TIDAK dimuat di sini karena NIK adalah data pribadi dan
// tidak boleh mengalir ke klien tanpa keperluan khusus. Klien memakai id
// internal pengajuan.
type slikResponse struct {
	Kolektibilitas       int     `json:"kolektibilitas"`
	JumlahFasilitasAktif *int    `json:"jumlahFasilitasAktif"`
	TotalBakiDebet       *int64  `json:"totalBakiDebet"`
	TanggalData          *string `json:"tanggalData,omitempty"`
	BerlakuSampai        *string `json:"berlakuSampai,omitempty"`
	Status               string  `json:"status"`
}

// JalankanSLIK menangani POST /api/pengajuan/{id}/slik (ANL).
//
// Urutan penanganan:
//  1. Verifikasi identitas aktor (hanya ANL lewat middleware WajibPeran)
//  2. Ambil id pengajuan dari path
//  3. Jalankan service.Jalankan
//  4. Sukses: 200 dengan slikResponse
//  5. Gagal hulu / timeout: 502 Bad Gateway (AGENTS.md 4.3)
//  6. NIK tidak ditemukan: 404 Not Found
func (h *SlikHandler) JalankanSLIK(w http.ResponseWriter, r *http.Request) {
	ident, ok := identitasWajib(w, r)
	if !ok {
		return
	}

	id, ok := parseIDPengajuan(w, r)
	if !ok {
		return
	}

	hasil, err := h.slik.Jalankan(r.Context(), id, ident.PenggunaID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	var tglDataStr *string
	if hasil.TanggalData != nil {
		s := hasil.TanggalData.Format(time.RFC3339)
		tglDataStr = &s
	}

	var berlakuStr *string
	if hasil.BerlakuSampai != nil {
		s := hasil.BerlakuSampai.Format(time.RFC3339)
		berlakuStr = &s
	}

	respondJSON(w, http.StatusOK, slikResponse{
		Kolektibilitas:       hasil.Kolektibilitas,
		JumlahFasilitasAktif: hasil.JumlahFasilitasAktif,
		TotalBakiDebet:       hasil.TotalBakiDebet,
		TanggalData:          tglDataStr,
		BerlakuSampai:        berlakuStr,
		Status:               hasil.StatusPengajuan,
	})
}
