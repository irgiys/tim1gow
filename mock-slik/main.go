package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// permintaanInquiry adalah badan request POST /slik/inquiry.
type permintaanInquiry struct {
	NIK string `json:"nik"`
}

// jawabanInquiry adalah respons 200. Nama field JSON memakai camelCase
// persis seperti kontrak brief §6.1 — jangan diubah menjadi snake_case
// walaupun sisa repo memakai snake_case untuk kolom database.
type jawabanInquiry struct {
	NIK                  string `json:"nik"`
	Nama                 string `json:"nama"`
	Kolektibilitas       int    `json:"kolektibilitas"`
	JumlahFasilitasAktif int    `json:"jumlahFasilitasAktif"`
	TotalBakiDebet       int64  `json:"totalBakiDebet"`
	TanggalData          string `json:"tanggalData"`
	ReferenceID          string `json:"referenceId"`
}

// jawabanGalat adalah bentuk respons 404 dan 503. Kontrak hanya menetapkan
// field `error`, jadi tidak ada field tambahan di sini.
type jawabanGalat struct {
	Error string `json:"error"`
}

// Server memegang data fixtures dan sumber waktu.
//
// `sekarang` disuntikkan supaya test dapat mengunci tanggalData tanpa
// bergantung pada jam mesin.
type Server struct {
	nasabah  map[string]Nasabah
	sekarang func() time.Time
}

// NewServer membuat Server dengan sumber waktu bawaan.
func NewServer(nasabah map[string]Nasabah) *Server {
	return &Server{nasabah: nasabah, sekarang: time.Now}
}

// Handler mendaftarkan seluruh route mock.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/slik/inquiry", s.inquiry)
	// /health dipakai healthcheck docker-compose. Tanpa ini container selalu
	// unhealthy walaupun layanannya hidup.
	mux.HandleFunc("/health", s.health)
	return mux
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.tulisGalat(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED")
		return
	}
	s.tulisJSON(w, http.StatusOK, map[string]any{
		"status":         "ok",
		"jumlah_nasabah": len(s.nasabah),
	})
}

// inquiry menangani POST /slik/inquiry sesuai kontrak §6.1.
//
// Urutan pemeriksaan disengaja: NIK pemicu 503 diperiksa SEBELUM pencarian
// fixtures, supaya jalur "layanan tidak tersedia" bisa didemokan kapan pun
// tanpa mengubah data.
func (s *Server) inquiry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.tulisGalat(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED")
		return
	}

	var req permintaanInquiry
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		s.tulisGalat(w, http.StatusBadRequest, "BAD_REQUEST")
		return
	}

	nik := strings.TrimSpace(req.NIK)
	if nik == "" {
		s.tulisGalat(w, http.StatusBadRequest, "BAD_REQUEST")
		return
	}

	// Jalur 503 dapat dipaksa dua cara: NIK khusus dari fixtures, atau query
	// ?paksa=503 supaya penilai bisa menguji NIK mana pun.
	if nik == NIKPemicu503 || r.URL.Query().Get("paksa") == "503" {
		s.tulisGalat(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
		return
	}

	n, ada := s.nasabah[nik]
	if !ada {
		s.tulisGalat(w, http.StatusNotFound, "NIK_NOT_FOUND")
		return
	}

	now := s.sekarang()
	s.tulisJSON(w, http.StatusOK, jawabanInquiry{
		NIK:                  n.NIK,
		Nama:                 n.Nama,
		Kolektibilitas:       n.Kolektibilitas,
		JumlahFasilitasAktif: n.JumlahFasilitasAktif,
		TotalBakiDebet:       n.TotalBakiDebet,
		TanggalData:          now.Format("2006-01-02"),
		// referenceId cukup unik untuk korelasi log. TIDAK memuat NIK:
		// nomor referensi ikut tercatat di log backend, dan NIK adalah data
		// pribadi yang dilarang masuk log (BR-11).
		ReferenceID: fmt.Sprintf("SLIK-%d", now.UnixNano()),
	})
}

func (s *Server) tulisJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("gagal menulis respons: %v", err)
	}
}

func (s *Server) tulisGalat(w http.ResponseWriter, status int, kode string) {
	s.tulisJSON(w, status, jawabanGalat{Error: kode})
}

func main() {
	path := os.Getenv("FIXTURES_PATH")
	if path == "" {
		path = "../fixtures/nasabah-uji.csv"
	}

	nasabah, err := MuatFixturesDariBerkas(path)
	if err != nil {
		// Gagal keras. Mock yang hidup dengan 0 nasabah akan menjawab 404
		// untuk semua NIK, dan itu terlihat seperti bug di backend.
		log.Fatalf("mock-slik: %v", err)
	}
	log.Printf("mock-slik: memuat %d nasabah dari %s", len(nasabah), path)

	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           NewServer(nasabah).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("mock-slik: mendengarkan di :%s", port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("mock-slik: %v", err)
	}
}
