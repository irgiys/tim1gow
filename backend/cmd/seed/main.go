// Command seed menyiapkan data awal yang dibutuhkan demo dan test.
//
// PEMBAGIAN TUGAS DENGAN MIGRASI — dibaca dulu sebelum menambah isi di sini:
//
// Nilai tabel parameter (parameter_skoring, parameter_riwayat_slik,
// parameter_umum, rentang_margin, ambang_approval) di-seed oleh migrasi
// 000002_seed_parameter.up.sql, bukan oleh runner ini. Angka-angka itu TIDAK
// diduplikasi ke kode Go dengan sengaja: dua sumber kebenaran untuk ambang
// yang sama adalah cara termurah menghasilkan bug yang tidak terlihat, dan
// AGENTS.md Larangan 3 melarang menuliskan ambang/bobot sebagai konstanta di
// kode — termasuk di seed.
//
// Karena itu tugas runner ini ada dua:
//
//  1. VERIFIKASI bahwa tabel parameter benar-benar terisi. Kalau kosong, ia
//     berhenti dengan galat yang menyebut tabelnya. Parameter kosong berarti
//     skoring dan margin akan gagal saat demo (SkoringService mengembalikan
//     ConfigError, bukan memakai nilai default — AGENTS.md Larangan 3), dan
//     lebih baik ketahuan di sini daripada di depan penilai.
//
//  2. SEED data demo yang bukan skema dan bukan parameter: akun demo per peran
//     (AO/ANL/KCP/KC/KOM/ADM) di tabel `pengguna`, dibuat setelah migrasi
//
//  000004. Password diambil dari SEED_DEFAULT_PASSWORD dan di-hash bcrypt —
//     tidak pernah di-hardcode dan tidak pernah disimpan sebagai plaintext.
//
// IDEMPOTEN (brief §7.2 butir 5): aman dijalankan berulang. Runner ini tidak
// pernah memakai ON CONFLICT DO UPDATE untuk tabel parameter, supaya perubahan
// bobot yang dibuat ADM saat demo tidak ter-reset diam-diam pada restart —
// kalau ter-reset, AC-15 justru tidak bisa dibuktikan (AGENTS.md Larangan 19).
//
// Pemakaian:
//
//	go run ./cmd/seed          # verifikasi + seed
//	go run ./cmd/seed -verify  # hanya verifikasi, tidak menulis apa pun
package main

import (
	"flag"
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/irgiys/tim1gow/backend/internal/config"
	"github.com/irgiys/tim1gow/backend/internal/repository/db"
)

func main() {
	verifyOnly := flag.Bool("verify", false, "hanya verifikasi tabel parameter, tanpa menulis data")
	flag.Parse()

	cfg := config.Load()
	dsn := cfg.DatabaseURL
	if cfg.AppEnv == "test" && cfg.DatabaseURLTest != "" {
		dsn = cfg.DatabaseURLTest
	}
	if dsn == "" {
		log.Fatal("DATABASE_URL kosong; set di environment atau .env")
	}

	gdb, err := db.Open(dsn)
	if err != nil {
		log.Fatalf("gagal koneksi database: %v", err)
	}

	if err := verifikasiParameter(gdb); err != nil {
		log.Fatalf("verifikasi parameter gagal: %v", err)
	}
	log.Println("verifikasi tabel parameter: OK")

	if *verifyOnly {
		return
	}

	n, err := seedPengguna(gdb, cfg)
	if err != nil {
		log.Fatalf("seed pengguna gagal: %v", err)
	}
	if n < 0 {
		log.Println("seed pengguna: dilewati — tabel `pengguna` belum ada (FR-01 belum termigrasi)")
	} else {
		log.Printf("seed pengguna: %d baris disiapkan (idempoten)", n)
	}

	log.Println("seed selesai")
}

// tabelParameterWajib adalah tabel yang HARUS berisi minimal satu baris supaya
// skoring, margin, dan routing approval bisa jalan. Daftar ini sengaja hanya
// berisi NAMA tabel — tidak ada nilai ambang di dalamnya.
var tabelParameterWajib = []string{
	"parameter_skoring",
	"parameter_riwayat_slik",
	"parameter_umum",
	"rentang_margin",
	"ambang_approval",
}

// verifikasiParameter memastikan setiap tabel parameter terisi. Tabel kosong
// dilaporkan sebagai galat, bukan diperbaiki diam-diam dengan nilai default.
func verifikasiParameter(gdb *gorm.DB) error {
	var kosong []string
	for _, tabel := range tabelParameterWajib {
		var jumlah int64
		// Nama tabel berasal dari slice konstan di atas, bukan dari input
		// pengguna, jadi interpolasi di sini tidak membuka jalan injeksi.
		q := fmt.Sprintf("SELECT COUNT(*) FROM %s", tabel)
		if err := gdb.Raw(q).Scan(&jumlah).Error; err != nil {
			return fmt.Errorf("membaca %s: %w", tabel, err)
		}
		if jumlah == 0 {
			kosong = append(kosong, tabel)
		}
	}
	if len(kosong) > 0 {
		return fmt.Errorf("tabel parameter berikut kosong: %v — jalankan `go run ./cmd/migrate up` lebih dulu", kosong)
	}
	return nil
}

// akunDemo adalah daftar akun per peran untuk demo dan test manual.
//
// Ini BUKAN parameter aturan bisnis (tidak ada ambang/bobot di sini), jadi
// tidak melanggar Larangan 3. Password sengaja tidak ada di struct ini —
// nilainya satu untuk semua akun, dari SEED_DEFAULT_PASSWORD.
var akunDemo = []struct {
	Nama  string
	Email string
	Peran string
}{
	{"Ayu Account Officer", "ao@imitra.test", "AO"},
	{"Andi Analis Mikro", "anl@imitra.test", "ANL"},
	{"Kartika Kepala CP", "kcp@imitra.test", "KCP"},
	{"Kurnia Kepala Cabang", "kc@imitra.test", "KC"},
	{"Komite Pembiayaan", "kom@imitra.test", "KOM"},
	{"Admin Sistem", "adm@imitra.test", "ADM"},
}

// seedPengguna mengisi akun demo per peran. Mengembalikan -1 bila tabel
// `pengguna` belum ada, supaya runner ini tetap berguna (dan tetap exit 0)
// sebelum FR-01 termigrasi — compose memanggilnya dengan
// service_completed_successfully, jadi exit code non-nol akan memblokir backend.
func seedPengguna(gdb *gorm.DB, cfg config.Config) (int, error) {
	var ada bool
	err := gdb.Raw(`SELECT EXISTS (
		SELECT 1 FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name = 'pengguna'
	)`).Scan(&ada).Error
	if err != nil {
		return 0, fmt.Errorf("memeriksa tabel pengguna: %w", err)
	}
	if !ada {
		return -1, nil
	}

	if cfg.SeedDefaultPassword == "" {
		return 0, fmt.Errorf("SEED_DEFAULT_PASSWORD kosong; wajib diisi untuk membuat akun demo")
	}

	// Hash dihitung sekali untuk semua akun: bcrypt itu mahal secara sengaja,
	// dan setiap akun memakai password yang sama.
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.SeedDefaultPassword), cfg.PasswordHashCost)
	if err != nil {
		return 0, fmt.Errorf("menghitung hash password: %w", err)
	}

	var dibuat int
	for _, a := range akunDemo {
		// ON CONFLICT DO NOTHING, bukan DO UPDATE (Larangan 19): password yang
		// diubah ADM saat demo tidak boleh ter-reset diam-diam saat restart.
		res := gdb.Exec(`
			INSERT INTO pengguna (nama, email, password_hash, peran, aktif)
			VALUES (?, ?, ?, ?, TRUE)
			ON CONFLICT (email) DO NOTHING`,
			a.Nama, a.Email, string(hash), a.Peran)
		if res.Error != nil {
			// Email BUKAN data pribadi nasabah, tetapi tetap tidak diikutkan ke
			// pesan galat supaya pola log seragam dengan BR-11.
			return dibuat, fmt.Errorf("menyisipkan akun peran %s: %w", a.Peran, res.Error)
		}
		dibuat += int(res.RowsAffected)
	}
	return dibuat, nil
}
