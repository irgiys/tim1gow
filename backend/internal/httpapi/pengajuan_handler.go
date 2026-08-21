package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/irgiys/tim1gow/backend/internal/domain"
	"github.com/irgiys/tim1gow/backend/internal/service"
)

// PengajuanHandler menangani endpoint pengajuan (FR-02), dokumen (FR-03), dan
// survei lapangan (FR-04).
//
// Handler ini TIDAK memuat aturan bisnis (AGENTS.md Larangan 17): validasi
// plafon (BR-01), nomor referensi (BR-12), aturan re-upload dan kode alasan
// tolak (AC-03), serta kelengkapan survei semuanya hidup di internal/service.
// Tugas handler hanya parsing request, mengambil identitas aktor dari context,
// memanggil service, dan memetakan hasil/error ke HTTP.
type PengajuanHandler struct {
	pengajuan *service.PengajuanService
	dokumen   *service.DokumenService
	survei    *service.SurveiService
}

// NewPengajuanHandler membuat instance PengajuanHandler baru.
func NewPengajuanHandler(
	pengajuan *service.PengajuanService,
	dokumen *service.DokumenService,
	survei *service.SurveiService,
) *PengajuanHandler {
	return &PengajuanHandler{pengajuan: pengajuan, dokumen: dokumen, survei: survei}
}

// ---------------------------------------------------------------------------
// Bentuk request/response
// ---------------------------------------------------------------------------

type anggotaRequest struct {
	NamaAnggota   string `json:"namaAnggota"`
	NIKAnggota    string `json:"nikAnggota"`
	PlafonAnggota int64  `json:"plafonAnggota"`
}

// buatPengajuanRequest adalah badan POST /api/pengajuan.
//
// aoID TIDAK ada di sini: pemiliknya diambil dari token, bukan dari klien.
// Kalau klien boleh menentukannya, AO dapat membuat pengajuan atas nama orang
// lain dan BR-09 (maker != checker) menjadi tidak berarti.
type buatPengajuanRequest struct {
	Tipe           string           `json:"tipe"`
	NamaNasabah    string           `json:"namaNasabah"`
	NIK            string           `json:"nik"`
	AlamatUsaha    string           `json:"alamatUsaha"`
	JenisUsaha     string           `json:"jenisUsaha"`
	JenisAkad      string           `json:"jenisAkad"`
	PlafonDiajukan int64            `json:"plafonDiajukan"`
	TenorBulan     int              `json:"tenorBulan"`
	Anggota        []anggotaRequest `json:"anggota"`
}

// pengajuanResponse sengaja TIDAK memuat NIK (BR-11). Klien memakai id
// internal atau nomor referensi untuk merujuk pengajuan.
type pengajuanResponse struct {
	ID              int64    `json:"id"`
	NomorReferensi  string   `json:"nomorReferensi"`
	Tipe            string   `json:"tipe"`
	NamaNasabah     string   `json:"namaNasabah"`
	AlamatUsaha     string   `json:"alamatUsaha"`
	JenisUsaha      string   `json:"jenisUsaha"`
	JenisAkad       string   `json:"jenisAkad"`
	PlafonDiajukan  int64    `json:"plafonDiajukan"`
	PlafonDisetujui *int64   `json:"plafonDisetujui,omitempty"`
	TenorBulan      int      `json:"tenorBulan"`
	Margin          *float64 `json:"marginAtauNisbah,omitempty"`
	Status          string   `json:"status"`
}

func kePengajuanResponse(p service.Pengajuan) pengajuanResponse {
	return pengajuanResponse{
		ID:              p.ID,
		NomorReferensi:  p.NomorReferensi,
		Tipe:            string(p.Tipe),
		NamaNasabah:     p.NamaNasabah,
		AlamatUsaha:     p.AlamatUsaha,
		JenisUsaha:      p.JenisUsaha,
		JenisAkad:       p.JenisAkad,
		PlafonDiajukan:  p.PlafonDiajukan,
		PlafonDisetujui: p.PlafonDisetujui,
		TenorBulan:      p.TenorBulan,
		Margin:          p.MarginAtauNisbah,
		Status:          p.Status,
	}
}

type dokumenResponse struct {
	ID              int64   `json:"id"`
	PengajuanID     int64   `json:"pengajuanId"`
	JenisDokumen    string  `json:"jenisDokumen"`
	Status          string  `json:"status"`
	AlasanPenolakan *string `json:"alasanPenolakan,omitempty"`
}

// keDokumenResponse sengaja TIDAK memuat url_berkas: path foto adalah data
// pribadi dan tidak boleh mengalir ke klien lewat daftar (BR-11). Berkasnya
// diakses lewat endpoint terautentikasi tersendiri.
func keDokumenResponse(d service.Dokumen) dokumenResponse {
	return dokumenResponse{
		ID:              d.ID,
		PengajuanID:     d.PengajuanID,
		JenisDokumen:    d.JenisDokumen,
		Status:          string(d.Status),
		AlasanPenolakan: d.AlasanPenolakan,
	}
}

// ---------------------------------------------------------------------------
// FR-02 — Pengajuan
// ---------------------------------------------------------------------------

// Buat menangani POST /api/pengajuan (AO).
func (h *PengajuanHandler) Buat(w http.ResponseWriter, r *http.Request) {
	ident, ok := identitasWajib(w, r)
	if !ok {
		return
	}

	var req buatPengajuanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "format JSON tidak valid", "")
		return
	}

	tipe := service.TipePengajuan(req.Tipe)
	if tipe == "" {
		tipe = service.TipeIndividu
	}

	anggota := make([]service.PengajuanAnggota, 0, len(req.Anggota))
	for _, a := range req.Anggota {
		anggota = append(anggota, service.PengajuanAnggota{
			NamaAnggota:   a.NamaAnggota,
			NIKAnggota:    a.NIKAnggota,
			PlafonAnggota: a.PlafonAnggota,
		})
	}

	hasil, err := h.pengajuan.Buat(r.Context(), service.InputPengajuan{
		AOID:           ident.PenggunaID,
		Tipe:           tipe,
		NamaNasabah:    req.NamaNasabah,
		NIK:            req.NIK,
		AlamatUsaha:    req.AlamatUsaha,
		JenisUsaha:     req.JenisUsaha,
		JenisAkad:      domain.Akad(req.JenisAkad),
		PlafonDiajukan: req.PlafonDiajukan,
		TenorBulan:     req.TenorBulan,
		Anggota:        anggota,
	})
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, kePengajuanResponse(hasil))
}

// Daftar menangani GET /api/pengajuan.
//
// AO hanya melihat pengajuan miliknya. Pembatasan itu ditegakkan di server
// dengan memakai identitas dari token, bukan dengan menyembunyikan baris di UI
// (AGENTS.md Larangan 6).
func (h *PengajuanHandler) Daftar(w http.ResponseWriter, r *http.Request) {
	ident, ok := identitasWajib(w, r)
	if !ok {
		return
	}

	daftar, err := h.pengajuan.Daftar(r.Context(), ident.PenggunaID, ident.Peran)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	out := make([]pengajuanResponse, 0, len(daftar))
	for _, p := range daftar {
		out = append(out, kePengajuanResponse(p))
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": out})
}

// Detail menangani GET /api/pengajuan/{id}.
func (h *PengajuanHandler) Detail(w http.ResponseWriter, r *http.Request) {
	ident, ok := identitasWajib(w, r)
	if !ok {
		return
	}
	id, ok := parseIDPengajuan(w, r)
	if !ok {
		return
	}

	p, err := h.pengajuan.Detail(r.Context(), id, ident.PenggunaID, ident.Peran)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, kePengajuanResponse(p))
}

// ---------------------------------------------------------------------------
// FR-03 — Dokumen
// ---------------------------------------------------------------------------

type uploadDokumenRequest struct {
	JenisDokumen string `json:"jenisDokumen"`
	URLBerkas    string `json:"urlBerkas"`
}

// UploadDokumen menangani POST /api/pengajuan/{id}/dokumen (AO).
//
// Re-upload satu dokumen yang ditolak TIDAK menghapus dokumen lain maupun data
// pengajuan (AC-03); aturan itu ditegakkan di DokumenService dan repository.
func (h *PengajuanHandler) UploadDokumen(w http.ResponseWriter, r *http.Request) {
	ident, ok := identitasWajib(w, r)
	if !ok {
		return
	}
	id, ok := parseIDPengajuan(w, r)
	if !ok {
		return
	}

	var req uploadDokumenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "format JSON tidak valid", "")
		return
	}

	d, err := h.dokumen.Upload(r.Context(), id, req.JenisDokumen, req.URLBerkas, ident.PenggunaID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, keDokumenResponse(d))
}

// DaftarDokumen menangani GET /api/pengajuan/{id}/dokumen.
func (h *PengajuanHandler) DaftarDokumen(w http.ResponseWriter, r *http.Request) {
	if _, ok := identitasWajib(w, r); !ok {
		return
	}
	id, ok := parseIDPengajuan(w, r)
	if !ok {
		return
	}

	daftar, err := h.dokumen.DaftarPerPengajuan(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	out := make([]dokumenResponse, 0, len(daftar))
	for _, d := range daftar {
		out = append(out, keDokumenResponse(d))
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": out})
}

type verifikasiDokumenRequest struct {
	Setujui    bool   `json:"setujui"`
	KodeAlasan string `json:"kodeAlasan"`
}

// VerifikasiDokumen menangani PATCH /api/pengajuan/{id}/dokumen/{dokId}/verifikasi (ANL).
//
// Penolakan wajib menyertakan kode alasan (AC-03). Aturan itu ditegakkan di
// service DAN oleh CHECK constraint di migrasi, bukan di handler ini.
func (h *PengajuanHandler) VerifikasiDokumen(w http.ResponseWriter, r *http.Request) {
	ident, ok := identitasWajib(w, r)
	if !ok {
		return
	}

	dokID, err := strconv.ParseInt(chi.URLParam(r, "dokId"), 10, 64)
	if err != nil || dokID <= 0 {
		respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "id dokumen tidak valid", "")
		return
	}

	var req verifikasiDokumenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "format JSON tidak valid", "")
		return
	}

	d, err := h.dokumen.Verifikasi(r.Context(), dokID, ident.PenggunaID, req.Setujui, req.KodeAlasan)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, keDokumenResponse(d))
}

// ---------------------------------------------------------------------------
// FR-04 — Survei lapangan
// ---------------------------------------------------------------------------

type rekamSurveiRequest struct {
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	FotoURL        string  `json:"fotoUrl"`
	OmzetHarian    int64   `json:"omzetHarian"`
	LamaUsahaBulan int     `json:"lamaUsahaBulan"`
	CatatanKondisi string  `json:"catatanKondisi"`
	Status         string  `json:"status"`
}

type surveiResponse struct {
	ID          int64  `json:"id"`
	PengajuanID int64  `json:"pengajuanId"`
	Status      string `json:"status"`
}

// RekamSurvei menangani POST /api/pengajuan/{id}/survei (AO).
func (h *PengajuanHandler) RekamSurvei(w http.ResponseWriter, r *http.Request) {
	ident, ok := identitasWajib(w, r)
	if !ok {
		return
	}
	id, ok := parseIDPengajuan(w, r)
	if !ok {
		return
	}

	var req rekamSurveiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "format JSON tidak valid", "")
		return
	}

	sv, err := h.survei.Rekam(r.Context(), service.InputSurvei{
		PengajuanID:    id,
		AOID:           ident.PenggunaID,
		Latitude:       req.Latitude,
		Longitude:      req.Longitude,
		FotoURL:        req.FotoURL,
		OmzetHarian:    req.OmzetHarian,
		LamaUsahaBulan: req.LamaUsahaBulan,
		CatatanKondisi: req.CatatanKondisi,
		Status:         service.StatusSurvei(req.Status),
	})
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, surveiResponse{
		ID:          sv.ID,
		PengajuanID: sv.PengajuanID,
		Status:      string(sv.Status),
	})
}

// ---------------------------------------------------------------------------
// Pembantu
// ---------------------------------------------------------------------------

// identitasWajib mengambil identitas terautentikasi dari context.
//
// Ketiadaan identitas dijawab 401, BUKAN diperlakukan sebagai pengguna anonim
// yang boleh lanjut. Handler yang menebak aktornya melanggar BR-10 (setiap
// perubahan wajib punya aktor).
func identitasWajib(w http.ResponseWriter, r *http.Request) (Identitas, bool) {
	ident, ok := IdentitasDari(r.Context())
	if !ok || ident.PenggunaID <= 0 {
		respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "autentikasi diperlukan", "")
		return Identitas{}, false
	}
	return ident, true
}
