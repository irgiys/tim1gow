// Package auth berisi pembuatan dan verifikasi JWT HS256 untuk iMitra.
//
// KENAPA DITULIS SENDIRI, BUKAN MEMAKAI PUSTAKA:
// Menambah dependensi baru butuh persetujuan Tech Lead (AGENTS.md Larangan 1).
// JWT HS256 hanya memerlukan HMAC-SHA256 dan base64url dari stdlib, sehingga
// implementasi sendiri di sini menghindari dependensi tambahan sekaligus
// sejalan dengan alasan tim memilih Chi: middleware ditulis sendiri supaya
// mudah dijelaskan saat demo (AGENTS.md bagian 2).
//
// Cakupannya sengaja sempit: HANYA HS256, hanya klaim yang iMitra pakai.
// Paket ini tidak berusaha menjadi pustaka JWT umum.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Galat verifikasi token. Handler memetakan semuanya ke HTTP 401 — pemanggil
// tidak perlu tahu bedanya, tetapi test perlu bisa membedakannya.
var (
	ErrTokenTidakValid   = errors.New("token tidak valid")
	ErrTokenKedaluwarsa  = errors.New("token kedaluwarsa")
	ErrAlgoTidakDidukung = errors.New("algoritma token tidak didukung")
)

// Klaim adalah isi payload JWT iMitra.
//
// Field sengaja minimal: hanya identitas dan peran. NIK, email, atau data
// pribadi lain TIDAK ikut — token muncul di header, log proxy, dan riwayat
// peramban, sehingga isinya harus dianggap semi-publik (BR-11).
type Klaim struct {
	PenggunaID int64  `json:"sub,string"`
	Peran      string `json:"peran"`
	// Unix timestamp. Dinamai mengikuti RFC 7519 supaya token tetap dapat
	// dibaca perkakas standar saat debugging.
	DiterbitkanPada int64 `json:"iat"`
	Kedaluwarsa     int64 `json:"exp"`
}

// header JWT untuk HS256. Konstan, jadi tidak perlu di-encode berulang.
const headerHS256 = `{"alg":"HS256","typ":"JWT"}`

// Terbitkan membuat token bertanda tangan untuk pengguna dan peran tertentu.
//
// `sekarang` diterima sebagai argumen, bukan dibaca dari time.Now() di dalam,
// supaya test dapat menguji kedaluwarsa tanpa menunggu waktu nyata.
func Terbitkan(secret []byte, penggunaID int64, peran string, masaBerlaku time.Duration, sekarang time.Time) (string, error) {
	if len(secret) == 0 {
		return "", errors.New("JWT secret kosong")
	}

	klaim := Klaim{
		PenggunaID:      penggunaID,
		Peran:           peran,
		DiterbitkanPada: sekarang.Unix(),
		Kedaluwarsa:     sekarang.Add(masaBerlaku).Unix(),
	}

	payload, err := json.Marshal(klaim)
	if err != nil {
		return "", fmt.Errorf("menyusun klaim: %w", err)
	}

	bagian := encode([]byte(headerHS256)) + "." + encode(payload)
	return bagian + "." + encode(tandaTangan(secret, bagian)), nil
}

// Verifikasi memeriksa tanda tangan dan masa berlaku, lalu mengembalikan klaim.
//
// Urutan pemeriksaan penting: tanda tangan diperiksa SEBELUM klaim dibaca.
// Payload yang belum terverifikasi adalah masukan dari penyerang, jadi ia
// tidak boleh dipakai untuk keputusan apa pun — termasuk keputusan "token ini
// sudah kedaluwarsa".
func Verifikasi(secret []byte, token string, sekarang time.Time) (*Klaim, error) {
	if len(secret) == 0 {
		return nil, errors.New("JWT secret kosong")
	}

	bagian := strings.Split(token, ".")
	if len(bagian) != 3 {
		return nil, ErrTokenTidakValid
	}

	headerJSON, err := decode(bagian[0])
	if err != nil {
		return nil, ErrTokenTidakValid
	}
	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, ErrTokenTidakValid
	}
	// Menolak "none" dan algoritma lain secara eksplisit. Ini kelas kerentanan
	// JWT yang paling terkenal: menerima alg dari token berarti penyerang
	// memilih sendiri cara tokennya diverifikasi.
	if header.Alg != "HS256" {
		return nil, ErrAlgoTidakDidukung
	}

	tandaDiterima, err := decode(bagian[2])
	if err != nil {
		return nil, ErrTokenTidakValid
	}
	tandaSeharusnya := tandaTangan(secret, bagian[0]+"."+bagian[1])
	// hmac.Equal: perbandingan waktu-konstan, bukan bytes.Equal.
	if !hmac.Equal(tandaDiterima, tandaSeharusnya) {
		return nil, ErrTokenTidakValid
	}

	payload, err := decode(bagian[1])
	if err != nil {
		return nil, ErrTokenTidakValid
	}
	var klaim Klaim
	if err := json.Unmarshal(payload, &klaim); err != nil {
		return nil, ErrTokenTidakValid
	}

	if klaim.PenggunaID <= 0 || klaim.Peran == "" {
		return nil, ErrTokenTidakValid
	}
	if klaim.Kedaluwarsa <= 0 || sekarang.Unix() >= klaim.Kedaluwarsa {
		return nil, ErrTokenKedaluwarsa
	}

	return &klaim, nil
}

func tandaTangan(secret []byte, bagian string) []byte {
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(bagian))
	return m.Sum(nil)
}

// JWT memakai base64url tanpa padding (RFC 7515 bagian 2).
func encode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func decode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}
