// Package slik adalah satu-satunya jalan keluar backend menuju layanan SLIK.
//
// Dipisah menjadi paket sendiri (AGENTS.md bagian 3) karena dua alasan: layanan
// SLIK dipanggil lewat HTTP nyata, bukan fungsi lokal (bagian 5.2), dan cara
// kegagalannya harus diterjemahkan sekali di satu tempat. Kalau penerjemahan itu
// tersebar di service atau handler, satu cabang yang lupa akan mengubah "SLIK
// mati" menjadi "SLIK bersih" — kegagalan yang justru dilarang keras oleh
// Larangan 15.
//
// Yang TIDAK dilakukan paket ini: memutuskan nasib pengajuan. Ia hanya melaporkan
// apa yang dijawab SLIK. Keputusan kol-3-5 ditolak, kol-2 dibatasi grade, dan
// masa berlaku 30 hari (BR-04) ada di internal/service.
//
// BR-11: NIK adalah data pribadi. Di paket ini NIK hanya boleh berada di badan
// request. Ia TIDAK BOLEH masuk ke URL, pesan error, atau log — karena itu tidak
// ada satu pun error di berkas ini yang membawa NIK, dan pemanggilnya memakai
// referenceId atau id pengajuan untuk korelasi.
package slik

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// StatusPanggilan mencerminkan kolom hasil_slik.status_panggilan pada migrasi
// 000004. Nilainya dibatasi CHECK constraint di database, jadi daftar di sini
// wajib sama persis — menambah nilai baru berarti menambah migrasi.
type StatusPanggilan string

const (
	StatusSukses            StatusPanggilan = "SUKSES"
	StatusNIKTidakDitemukan StatusPanggilan = "NIK_NOT_FOUND"
	StatusLayananTidakAda   StatusPanggilan = "SERVICE_UNAVAILABLE"
	StatusTimeout           StatusPanggilan = "TIMEOUT"
)

// Hasil adalah jawaban SLIK yang sudah diterjemahkan.
//
// Status SELALU terisi, termasuk saat panggilan gagal — itu yang membuat
// percobaan gagal tetap bisa dicatat sebagai bukti (migrasi 000004 sengaja
// mengizinkan kolektibilitas NULL untuk baris gagal). Kolektibilitas bertipe
// pointer supaya "tidak ada jawaban" tidak bisa tertukar dengan angka 1:
// nilai default itulah bentuk konkret dari pelanggaran Larangan 15.
type Hasil struct {
	Status               StatusPanggilan
	Kolektibilitas       *int
	JumlahFasilitasAktif *int
	TotalBakiDebet       *int64
	TanggalData          *time.Time
	ReferenceID          string
}

// Sukses menyatakan panggilan berhasil dan kolektibilitasnya dapat dipakai.
// Constraint chk_slik_sukses_punya_kolektibilitas menuntut keduanya sejalan,
// jadi pemeriksaannya digabung di sini alih-alih diulang di setiap pemanggil.
func (h Hasil) Sukses() bool {
	return h.Status == StatusSukses && h.Kolektibilitas != nil
}

// Client memanggil layanan SLIK lewat HTTP.
type Client struct {
	httpClient  *http.Client
	baseURL     string
	inquiryPath string
}

// Opsi mengikuti bentuk config.Config supaya pemanggil tidak perlu membaca env
// sendiri (AGENTS.md bagian 3: env dibaca sekali di internal/config).
type Opsi struct {
	BaseURL     string
	InquiryPath string
	Timeout     time.Duration
}

// NewClient membangun client SLIK.
//
// Timeout wajib punya nilai: tanpa batas waktu, satu panggilan SLIK yang
// menggantung akan menahan request ANL sampai proxy memutusnya, dan jejaknya
// tidak pernah tercatat sebagai TIMEOUT.
func NewClient(o Opsi) *Client {
	timeout := o.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	path := o.InquiryPath
	if path == "" {
		path = "/slik/inquiry"
	}
	return &Client{
		httpClient:  &http.Client{Timeout: timeout},
		baseURL:     strings.TrimRight(o.BaseURL, "/"),
		inquiryPath: path,
	}
}

// jawabanInquiry mengikuti kontrak AGENTS.md bagian 5.2 apa adanya.
type jawabanInquiry struct {
	NIK                  string `json:"nik"`
	Nama                 string `json:"nama"`
	Kolektibilitas       int    `json:"kolektibilitas"`
	JumlahFasilitasAktif int    `json:"jumlahFasilitasAktif"`
	TotalBakiDebet       int64  `json:"totalBakiDebet"`
	TanggalData          string `json:"tanggalData"`
	ReferenceID          string `json:"referenceId"`
}

type jawabanGalat struct {
	Error string `json:"error"`
}

// ErrKontrakSlik menandai jawaban yang tidak sesuai kontrak bagian 5.2 —
// misalnya 200 dengan kolektibilitas di luar 1..5, atau badan JSON rusak.
// Dibedakan dari SLIK-mati supaya bug integrasi tidak terlihat seperti
// gangguan layanan dan tersembunyi di balik retry.
var ErrKontrakSlik = errors.New("jawaban SLIK di luar kontrak")

// Inquiry menanyakan satu NIK ke layanan SLIK.
//
// Kontrak error yang penting bagi pemanggil: 404, 503, dan timeout BUKAN error
// Go — ketiganya jawaban sah yang dikembalikan sebagai Hasil dengan Status
// terisi, supaya service dapat mencatatnya ke hasil_slik. Yang dikembalikan
// sebagai error hanyalah hal yang tidak boleh terjadi diam-diam (kontrak
// dilanggar, request gagal dibentuk).
func (c *Client) Inquiry(ctx context.Context, nik string) (Hasil, error) {
	badan, err := json.Marshal(map[string]string{"nik": nik})
	if err != nil {
		return Hasil{}, fmt.Errorf("menyusun permintaan SLIK: %w", err)
	}

	// NIK ada di badan request, TIDAK di URL (BR-11).
	url := c.baseURL + c.inquiryPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(badan))
	if err != nil {
		return Hasil{}, fmt.Errorf("membentuk permintaan SLIK: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Timeout dan koneksi gagal dilaporkan sebagai status, bukan error,
		// supaya percobaannya tetap tercatat. Keduanya disatukan sebagai
		// TIMEOUT: dari sisi pengguna sama-sama "SLIK tidak menjawab", dan
		// status_panggilan hanya mengenal empat nilai.
		return Hasil{Status: StatusTimeout}, nil
	}
	defer func() { _ = resp.Body.Close() }()

	// Batasi badan yang dibaca: layanan hulu tidak boleh bisa membuat backend
	// kehabisan memori.
	isi, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return Hasil{Status: StatusTimeout}, nil
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return terjemahkanSukses(isi)

	case http.StatusNotFound:
		// Pesan error dari hulu TIDAK diteruskan mentah — ia memuat NIK.
		return Hasil{Status: StatusNIKTidakDitemukan}, nil

	case http.StatusServiceUnavailable:
		return Hasil{Status: StatusLayananTidakAda}, nil

	default:
		// 4xx/5xx lain berarti kontrak bagian 5.2 dilanggar. Tidak dianggap
		// SLIK bersih dan tidak ditelan (Larangan 15 & aturan 4.3).
		var galat jawabanGalat
		_ = json.Unmarshal(isi, &galat)
		return Hasil{}, fmt.Errorf("%w: status %d kode %q",
			ErrKontrakSlik, resp.StatusCode, galat.Error)
	}
}

// terjemahkanSukses memvalidasi jawaban 200 sebelum dipercaya.
func terjemahkanSukses(isi []byte) (Hasil, error) {
	var j jawabanInquiry
	if err := json.Unmarshal(isi, &j); err != nil {
		return Hasil{}, fmt.Errorf("%w: badan 200 bukan JSON yang dikenal", ErrKontrakSlik)
	}

	// 200 tanpa kolektibilitas sah adalah jawaban yang tidak dapat dipakai.
	// Menerimanya berarti menulis baris SUKSES yang ditolak
	// chk_slik_sukses_punya_kolektibilitas, atau lebih buruk: lolos ke skoring.
	if j.Kolektibilitas < 1 || j.Kolektibilitas > 5 {
		return Hasil{}, fmt.Errorf("%w: kolektibilitas %d di luar 1..5",
			ErrKontrakSlik, j.Kolektibilitas)
	}

	h := Hasil{
		Status:         StatusSukses,
		Kolektibilitas: &j.Kolektibilitas,
		ReferenceID:    j.ReferenceID,
	}
	if j.JumlahFasilitasAktif >= 0 {
		h.JumlahFasilitasAktif = &j.JumlahFasilitasAktif
	}
	if j.TotalBakiDebet >= 0 {
		h.TotalBakiDebet = &j.TotalBakiDebet
	}

	// tanggalData menentukan berlaku_sampai (BR-04). Kalau formatnya tidak
	// terbaca, dibiarkan nil supaya service yang memutuskan — lebih baik
	// daripada memakai hari ini dan memperpanjang masa berlaku secara diam-diam.
	if t, err := time.Parse("2006-01-02", j.TanggalData); err == nil {
		h.TanggalData = &t
	}

	return h, nil
}
