package auth

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"
)

var (
	secretUji = []byte("rahasia-untuk-test-saja-bukan-nilai-produksi")
	waktuUji  = time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
)

// TestTerbitkanLaluVerifikasi_Valid adalah kasus pembanding untuk seluruh test
// penolakan di bawah (AGENTS.md Larangan 18): tanpa ini, Verifikasi yang
// menolak SEMUA token akan meloloskan setiap test penolakan.
func TestTerbitkanLaluVerifikasi_Valid(t *testing.T) {
	token, err := Terbitkan(secretUji, 42, "ANL", 30*time.Minute, waktuUji)
	if err != nil {
		t.Fatalf("Terbitkan: %v", err)
	}

	klaim, err := Verifikasi(secretUji, token, waktuUji.Add(time.Minute))
	if err != nil {
		t.Fatalf("Verifikasi: %v", err)
	}
	if klaim.PenggunaID != 42 {
		t.Errorf("PenggunaID = %d, mau 42", klaim.PenggunaID)
	}
	if klaim.Peran != "ANL" {
		t.Errorf("Peran = %q, mau ANL", klaim.Peran)
	}
	if klaim.Kedaluwarsa != waktuUji.Add(30*time.Minute).Unix() {
		t.Errorf("Kedaluwarsa = %d, mau %d", klaim.Kedaluwarsa, waktuUji.Add(30*time.Minute).Unix())
	}
}

func TestVerifikasi_SecretBerbedaDitolak(t *testing.T) {
	token, err := Terbitkan(secretUji, 1, "AO", time.Hour, waktuUji)
	if err != nil {
		t.Fatalf("Terbitkan: %v", err)
	}

	if _, err := Verifikasi([]byte("secret-yang-salah"), token, waktuUji); !errors.Is(err, ErrTokenTidakValid) {
		t.Errorf("err = %v, mau ErrTokenTidakValid", err)
	}
}

// TestVerifikasi_PayloadDiubahDitolak: inti dari tanda tangan. Penyerang yang
// menaikkan perannya sendiri dari AO ke ADM harus ditolak.
func TestVerifikasi_PayloadDiubahDitolak(t *testing.T) {
	token, err := Terbitkan(secretUji, 1, "AO", time.Hour, waktuUji)
	if err != nil {
		t.Fatalf("Terbitkan: %v", err)
	}

	bagian := strings.Split(token, ".")
	payload, err := base64.RawURLEncoding.DecodeString(bagian[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	dipalsukan := strings.Replace(string(payload), `"peran":"AO"`, `"peran":"ADM"`, 1)
	if dipalsukan == string(payload) {
		t.Fatal("payload uji tidak berubah — test tidak menguji apa pun")
	}
	tokenPalsu := bagian[0] + "." + base64.RawURLEncoding.EncodeToString([]byte(dipalsukan)) + "." + bagian[2]

	if _, err := Verifikasi(secretUji, tokenPalsu, waktuUji); !errors.Is(err, ErrTokenTidakValid) {
		t.Errorf("peran yang dipalsukan diterima; err = %v, mau ErrTokenTidakValid", err)
	}
}

// TestVerifikasi_AlgNoneDitolak menutup kerentanan JWT paling terkenal:
// token dengan alg "none" dan tanpa tanda tangan.
func TestVerifikasi_AlgNoneDitolak(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"1","peran":"ADM","iat":1,"exp":99999999999}`))

	for nama, token := range map[string]string{
		"tanpa tanda tangan":     header + "." + payload + ".",
		"tanda tangan sembarang": header + "." + payload + "." + base64.RawURLEncoding.EncodeToString([]byte("apa saja")),
	} {
		if _, err := Verifikasi(secretUji, token, waktuUji); !errors.Is(err, ErrAlgoTidakDidukung) {
			t.Errorf("%s: err = %v, mau ErrAlgoTidakDidukung", nama, err)
		}
	}
}

// TestVerifikasi_Kedaluwarsa memakai pasangan asersi: token yang sama diterima
// sebelum exp dan ditolak setelahnya, sehingga yang teruji adalah batas
// waktunya, bukan token yang kebetulan rusak.
func TestVerifikasi_Kedaluwarsa(t *testing.T) {
	token, err := Terbitkan(secretUji, 7, "KCP", 30*time.Minute, waktuUji)
	if err != nil {
		t.Fatalf("Terbitkan: %v", err)
	}

	if _, err := Verifikasi(secretUji, token, waktuUji.Add(29*time.Minute)); err != nil {
		t.Fatalf("sebelum exp seharusnya valid: %v", err)
	}
	if _, err := Verifikasi(secretUji, token, waktuUji.Add(31*time.Minute)); !errors.Is(err, ErrTokenKedaluwarsa) {
		t.Errorf("setelah exp err = %v, mau ErrTokenKedaluwarsa", err)
	}
	// Tepat pada detik exp sudah dianggap kedaluwarsa (>=, bukan >).
	if _, err := Verifikasi(secretUji, token, waktuUji.Add(30*time.Minute)); !errors.Is(err, ErrTokenKedaluwarsa) {
		t.Errorf("tepat pada exp err = %v, mau ErrTokenKedaluwarsa", err)
	}
}

func TestVerifikasi_BentukTokenRusakDitolak(t *testing.T) {
	for nama, token := range map[string]string{
		"kosong":             "",
		"satu bagian":        "abc",
		"dua bagian":         "abc.def",
		"empat bagian":       "a.b.c.d",
		"base64 tidak sah":   "!!!.???.***",
		"payload bukan json": base64.RawURLEncoding.EncodeToString([]byte(headerHS256)) + ".Ym9rYW4tanNvbg." + "xxx",
	} {
		if _, err := Verifikasi(secretUji, token, waktuUji); err == nil {
			t.Errorf("%s: seharusnya ditolak, tetapi diterima", nama)
		}
	}
}

// TestVerifikasi_KlaimKosongDitolak: token bertanda tangan sah tetapi tanpa
// identitas tidak boleh lolos, karena middleware akan memakai PenggunaID 0
// sebagai aktor audit (melanggar BR-10).
func TestVerifikasi_KlaimKosongDitolak(t *testing.T) {
	bagian := encode([]byte(headerHS256)) + "." + encode([]byte(`{"sub":"0","peran":"","iat":1,"exp":99999999999}`))
	token := bagian + "." + encode(tandaTangan(secretUji, bagian))

	if _, err := Verifikasi(secretUji, token, waktuUji); !errors.Is(err, ErrTokenTidakValid) {
		t.Errorf("err = %v, mau ErrTokenTidakValid", err)
	}
}

// TestTokenTidakMemuatDataPribadi menjaga BR-11: token melewati header, log
// proxy, dan riwayat peramban, jadi isinya harus dianggap semi-publik.
func TestTokenTidakMemuatDataPribadi(t *testing.T) {
	token, err := Terbitkan(secretUji, 42, "AO", time.Hour, waktuUji)
	if err != nil {
		t.Fatalf("Terbitkan: %v", err)
	}

	payload, err := decode(strings.Split(token, ".")[1])
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, dilarang := range []string{"nik", "email", "password", "nama_nasabah"} {
		if strings.Contains(strings.ToLower(string(payload)), dilarang) {
			t.Errorf("payload token memuat %q — melanggar BR-11: %s", dilarang, payload)
		}
	}
}

func TestTerbitkan_SecretKosongDitolak(t *testing.T) {
	if _, err := Terbitkan(nil, 1, "AO", time.Hour, waktuUji); err == nil {
		t.Error("secret kosong seharusnya menghasilkan error")
	}
}
