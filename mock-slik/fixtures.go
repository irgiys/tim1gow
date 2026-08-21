// Package main menjalankan stub layanan SLIK untuk iMitra.
//
// Kontrak endpoint berasal dari brief §6.1 dan AGENTS.md bagian 5.2, dan
// TIDAK boleh diubah:
//
//	POST /slik/inquiry
//	Request : { "nik": "3404xxxxxxxxxxxx" }
//	200     : { nik, nama, kolektibilitas, jumlahFasilitasAktif,
//	            totalBakiDebet, tanggalData, referenceId }
//	404     : { "error": "NIK_NOT_FOUND" }
//	503     : { "error": "SERVICE_UNAVAILABLE" }
//
// Berkas ini memuat pembacaan fixtures saja; HTTP-nya ada di main.go supaya
// logika parsing bisa diuji tanpa menghidupkan server.
package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Nasabah adalah satu baris fixtures/nasabah-uji.csv yang relevan untuk SLIK.
//
// Kolom omzet_harian, lama_usaha_bulan, dan skenario sengaja TIDAK dimuat:
// keduanya dipakai skoring di backend, bukan oleh SLIK. Mock ini hanya boleh
// mengembalikan field yang ada di kontrak §6.1 — mengirim lebih banyak
// membuat backend tergoda memakai data yang di dunia nyata tidak tersedia.
type Nasabah struct {
	NIK                  string
	Nama                 string
	Kolektibilitas       int
	JumlahFasilitasAktif int
	TotalBakiDebet       int64
}

// NIKPemicu503 memaksa respons 503 supaya jalur kegagalan dapat didemokan
// (brief §6.1: "penilai akan meminta ini"). NIK ini ada di fixtures dengan
// skenario "Pemicu respons 503".
const NIKPemicu503 = "3404000000000503"

// nilaiKosong menandai kolom yang di CSV ditulis "-" — dipakai baris pemicu
// 404 dan 503 yang memang tidak punya data kolektibilitas.
const nilaiKosong = "-"

// MuatFixtures membaca CSV nasabah uji menjadi peta ber-key NIK.
//
// Baris penanda skenario (kolom kolektibilitas berisi "-") sengaja TIDAK
// dimasukkan ke peta. Dua baris di fixtures berbentuk demikian: pemicu 404
// dan pemicu 503. Kalau keduanya ikut dimuat, NIK pemicu 404 justru ditemukan
// dan dijawab 200 dengan kolektibilitas 0 — nilai default diam-diam yang
// persis dilarang AGENTS.md bagian 4.3, dan membuat AC uji 404 tidak pernah
// benar-benar teruji. Nasabah tanpa kolektibilitas bukan data SLIK yang sah.
func MuatFixtures(r io.Reader) (map[string]Nasabah, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true

	baris, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("membaca csv: %w", err)
	}
	if len(baris) < 2 {
		return nil, fmt.Errorf("fixtures kosong: butuh header + minimal satu baris data")
	}

	// Kolom dicari lewat nama di header, bukan indeks tetap. Fixtures dimiliki
	// QA dan bisa bertambah kolom; mengunci indeks membuat mock diam-diam
	// membaca kolom yang salah alih-alih gagal dengan jelas.
	idx := map[string]int{}
	for i, nama := range baris[0] {
		idx[strings.TrimSpace(strings.ToLower(nama))] = i
	}
	for _, wajib := range []string{"nik", "nama", "kolektibilitas", "jumlah_fasilitas_aktif", "total_baki_debet"} {
		if _, ada := idx[wajib]; !ada {
			return nil, fmt.Errorf("kolom %q tidak ada di header fixtures", wajib)
		}
	}

	hasil := make(map[string]Nasabah, len(baris)-1)
	for nomor, rec := range baris[1:] {
		nik := strings.TrimSpace(rec[idx["nik"]])
		if nik == "" {
			continue
		}

		// Baris penanda skenario: kolektibilitas "-" berarti baris ini bukan
		// data nasabah, melainkan pemicu jalur error. Dilewati supaya NIK-nya
		// tidak pernah ditemukan (404), dan pemicu 503 ditangani di handler.
		if kolMentah := strings.TrimSpace(rec[idx["kolektibilitas"]]); kolMentah == "" || kolMentah == nilaiKosong {
			continue
		}

		if _, duplikat := hasil[nik]; duplikat {
			return nil, fmt.Errorf("baris %d: NIK duplikat di fixtures", nomor+2)
		}

		kol, err := angkaOpsional(rec[idx["kolektibilitas"]])
		if err != nil {
			return nil, fmt.Errorf("baris %d kolom kolektibilitas: %w", nomor+2, err)
		}
		fasilitas, err := angkaOpsional(rec[idx["jumlah_fasilitas_aktif"]])
		if err != nil {
			return nil, fmt.Errorf("baris %d kolom jumlah_fasilitas_aktif: %w", nomor+2, err)
		}
		baki, err := angkaOpsional(rec[idx["total_baki_debet"]])
		if err != nil {
			return nil, fmt.Errorf("baris %d kolom total_baki_debet: %w", nomor+2, err)
		}

		hasil[nik] = Nasabah{
			NIK:                  nik,
			Nama:                 strings.TrimSpace(rec[idx["nama"]]),
			Kolektibilitas:       int(kol),
			JumlahFasilitasAktif: int(fasilitas),
			TotalBakiDebet:       baki,
		}
	}
	return hasil, nil
}

// angkaOpsional memperlakukan "-" dan string kosong sebagai 0, selebihnya
// wajib berupa bilangan bulat. Nilai yang tidak dikenal menghasilkan error,
// bukan 0 diam-diam — fixtures rusak harus terlihat saat start, bukan
// menjadi kolektibilitas 0 yang menyesatkan saat demo.
func angkaOpsional(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == nilaiKosong {
		return 0, nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("nilai %q bukan bilangan bulat", s)
	}
	return n, nil
}

// MuatFixturesDariBerkas adalah pembungkus MuatFixtures untuk path berkas.
func MuatFixturesDariBerkas(path string) (map[string]Nasabah, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("membuka fixtures %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return MuatFixtures(f)
}
