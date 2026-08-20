package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/irgiys/tim1gow/backend/internal/domain"
	"github.com/irgiys/tim1gow/backend/internal/service"
)

// SkoringHandler menangani endpoint HTTP skoring kelayakan mikro (FR-06) dan
// perhitungan margin/nisbah (FR-07).
//
// Handler ini TIDAK memuat aturan bisnis: seluruh guard BR-03/05/06/07/08 hidup
// di internal/service (AGENTS.md Larangan 17). Tugasnya hanya parsing request,
// memanggil satu service, dan memetakan hasil/error ke HTTP lewat
// handleServiceError yang dipakai bersama seluruh API.
type SkoringHandler struct {
	skoring *service.SkoringService
	margin  *service.MarginService
}

// NewSkoringHandler membuat instance SkoringHandler baru.
func NewSkoringHandler(skoring *service.SkoringService, margin *service.MarginService) *SkoringHandler {
	return &SkoringHandler{skoring: skoring, margin: margin}
}

// skoringRequest adalah badan POST /api/pengajuan/{id}/skoring.
//
// Nilainya berasal dari data pengajuan, survei, dan hasil SLIK yang sudah
// tersimpan. Tidak ada NIK di sini: data pribadi tidak ikut mengalir ke
// lapisan perhitungan (BR-11).
type skoringRequest struct {
	AngsuranBulanan float64 `json:"angsuranBulanan"`
	OmzetHarian     float64 `json:"omzetHarian"`
	LamaUsahaBulan  int     `json:"lamaUsahaBulan"`
	Kolektibilitas  int     `json:"kolektibilitas"`
	NilaiSurvei     int     `json:"nilaiSurvei"`

	// Prasyarat BR-03. Dikirim eksplisit selama FR-03/FR-04/FR-05 belum
	// menyimpan keadaannya ke DB; begitu tabel dokumen/survei/hasil_slik ada,
	// nilai ini dibaca dari repository, bukan dari klien.
	SemuaDokumenVerified bool `json:"semuaDokumenVerified"`
	AdaSurveiValid       bool `json:"adaSurveiValid"`
	SlikSudahDijalankan  bool `json:"slikSudahDijalankan"`
}

// rincianResponse adalah bentuk JSON satu komponen skor. BR-08 mewajibkan
// rincian tiap komponen tampil ke ANL, bukan hanya skor akhirnya (AC-07).
type rincianResponse struct {
	Kode       string  `json:"kode"`
	Nama       string  `json:"nama"`
	SkorMentah float64 `json:"skorMentah"`
	Bobot      float64 `json:"bobot"`
	Kontribusi float64 `json:"kontribusi"`
}

type skoringResponse struct {
	PengajuanID         int64             `json:"pengajuanId"`
	SkorAkhir           int               `json:"skorAkhir"`
	Grade               int               `json:"grade"`
	TotalBobot          float64           `json:"totalBobot"`
	GradeMinimalDipaksa bool              `json:"gradeMinimalDipaksa"`
	Rincian             []rincianResponse `json:"rincian"`
}

// HitungSkoring menangani POST /api/pengajuan/{id}/skoring.
//
// Urutannya penting: BR-03 diperiksa SEBELUM menghitung. Skoring yang jalan
// tanpa dokumen VERIFIED / survei VALID / SLIK check menghasilkan angka yang
// tidak berdasar, jadi permintaannya ditolak 422 dengan rule BR-03 (AC-04).
func (h *SkoringHandler) HitungSkoring(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDPengajuan(w, r)
	if !ok {
		return
	}

	var req skoringRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "format JSON tidak valid", "")
		return
	}

	if err := h.skoring.PastikanBolehSkoring(service.PrasyaratSkoring{
		SemuaDokumenVerified: req.SemuaDokumenVerified,
		AdaSurveiValid:       req.AdaSurveiValid,
		SlikSudahDijalankan:  req.SlikSudahDijalankan,
	}); err != nil {
		handleServiceError(w, err)
		return
	}

	hasil, err := h.skoring.Hitung(domain.DataSkoring{
		PengajuanID:     id,
		AngsuranBulanan: req.AngsuranBulanan,
		OmzetHarian:     req.OmzetHarian,
		LamaUsahaBulan:  req.LamaUsahaBulan,
		Kolektibilitas:  req.Kolektibilitas,
		NilaiSurvei:     req.NilaiSurvei,
	})
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, keSkoringResponse(hasil))
}

// marginRequest adalah badan POST /api/pengajuan/{id}/margin.
type marginRequest struct {
	Akad  string  `json:"akad"`
	Grade int     `json:"grade"`
	Nilai float64 `json:"nilai"`
}

type marginResponse struct {
	PengajuanID int64  `json:"pengajuanId"`
	Akad        string `json:"akad"`
	Grade       int    `json:"grade"`

	Nilai       float64 `json:"nilai"`
	RentangMin  float64 `json:"rentangMin"`
	RentangMaks float64 `json:"rentangMaks"`
}

// TentukanMargin menangani POST /api/pengajuan/{id}/margin.
//
// Nilai di luar rentang grade DIBLOKIR (BR-06): responsnya 422 dengan
// rule BR-06, bukan 200 dengan peringatan. Tidak ada jalur "lanjutkan saja".
func (h *SkoringHandler) TentukanMargin(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDPengajuan(w, r)
	if !ok {
		return
	}

	var req marginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "format JSON tidak valid", "")
		return
	}
	if req.Grade <= 0 {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "grade wajib diisi", "")
		return
	}

	hasil, err := h.margin.Validasi(domain.Akad(req.Akad), req.Grade, req.Nilai)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, marginResponse{
		PengajuanID: id,
		Akad:        string(hasil.Akad),
		Grade:       hasil.Grade,
		Nilai:       hasil.Nilai,
		RentangMin:  hasil.RentangMin,
		RentangMaks: hasil.RentangMaks,
	})
}

// RentangMargin menangani GET /api/pengajuan/{id}/margin?akad=&grade=.
// Dipakai UI untuk menampilkan batas yang berlaku sebelum ANL mengisi angkanya,
// supaya batas yang tampil selalu berasal dari tabel parameter (AC-15).
func (h *SkoringHandler) RentangMargin(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDPengajuan(w, r)
	if !ok {
		return
	}

	grade, err := strconv.Atoi(r.URL.Query().Get("grade"))
	if err != nil || grade <= 0 {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "parameter grade tidak valid", "")
		return
	}
	akad := domain.Akad(r.URL.Query().Get("akad"))

	min, maks, err := h.margin.RentangUntukGrade(akad, grade)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, marginResponse{
		PengajuanID: id,
		Akad:        string(akad),
		Grade:       grade,
		RentangMin:  min,
		RentangMaks: maks,
	})
}

// parseIDPengajuan membaca {id} dari path dan membalas 400 bila tidak valid.
func parseIDPengajuan(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "id pengajuan tidak valid", "")
		return 0, false
	}
	return id, true
}

// keSkoringResponse memetakan hasil service ke bentuk JSON. Rincian selalu
// berupa array (bukan null) supaya klien tidak perlu menangani dua bentuk.
func keSkoringResponse(h domain.HasilSkoring) skoringResponse {
	rincian := make([]rincianResponse, 0, len(h.Rincian))
	for _, r := range h.Rincian {
		rincian = append(rincian, rincianResponse{
			Kode:       r.Kode,
			Nama:       r.Nama,
			SkorMentah: r.SkorMentah,
			Bobot:      r.Bobot,
			Kontribusi: r.Kontribusi,
		})
	}
	return skoringResponse{
		PengajuanID:         h.PengajuanID,
		SkorAkhir:           h.SkorAkhir,
		Grade:               h.Grade,
		TotalBobot:          h.TotalBobot,
		GradeMinimalDipaksa: h.GradeMinimalDipaksa,
		Rincian:             rincian,
	}
}
