// Command migrate menjalankan migrasi skema golang-migrate (up/down).
//
// Skema database HANYA berasal dari berkas *.up.sql / *.down.sql di
// backend/migrations/ (AGENTS.md Larangan 16). Runner ini tidak pernah memakai
// gorm.AutoMigrate.
//
// Pemakaian:
//
//	go run ./cmd/migrate up          # terapkan semua migrasi
//	go run ./cmd/migrate down 1      # mundur 1 langkah
//	go run ./cmd/migrate version     # tampilkan versi saat ini
//
// Membaca DATABASE_URL (atau DATABASE_URL_TEST saat APP_ENV=test).
package main

import (
	"errors"
	"log"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/irgiys/tim1gow/backend/internal/config"
)

func main() {
	cfg := config.Load()
	dsn := cfg.DatabaseURL
	if cfg.AppEnv == "test" && cfg.DatabaseURLTest != "" {
		dsn = cfg.DatabaseURLTest
	}
	if dsn == "" {
		log.Fatal("DATABASE_URL kosong; set di environment atau .env")
	}

	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatal("perintah wajib: up | down [N] | version")
	}

	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		log.Fatalf("gagal membuka migrasi: %v", err)
	}
	defer func() { _, _ = m.Close() }()

	switch args[0] {
	case "up":
		runErr(m.Up())
		log.Println("migrasi up selesai")
	case "down":
		if len(args) >= 2 {
			n, convErr := strconv.Atoi(args[1])
			if convErr != nil {
				log.Fatalf("argumen down harus angka: %v", convErr)
			}
			runErr(m.Steps(-n))
		} else {
			runErr(m.Down())
		}
		log.Println("migrasi down selesai")
	case "version":
		v, dirty, verErr := m.Version()
		if verErr != nil {
			log.Fatalf("gagal membaca versi: %v", verErr)
		}
		log.Printf("versi migrasi: %d (dirty=%v)", v, dirty)
	default:
		log.Fatalf("perintah tidak dikenal: %s", args[0])
	}
}

// runErr menganggap ErrNoChange sebagai sukses (tidak ada migrasi baru), tetapi
// menghentikan proses pada error lain.
func runErr(err error) {
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("migrasi gagal: %v", err)
	}
}
