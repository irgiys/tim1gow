// Package db membuka koneksi GORM ke PostgreSQL. Ini bagian dari lapisan akses
// data (AGENTS.md bagian 3): satu-satunya tempat yang membuka koneksi DB.
//
// CATATAN: GORM di sini HANYA untuk query/persistence. AutoMigrate DILARANG
// (AGENTS.md Larangan 16) — skema hanya berasal dari berkas migrasi SQL.
package db

import (
	"context"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open membuka pool koneksi GORM dari DATABASE_URL dan memverifikasinya dengan
// satu ping. Gagal ping = gagal boot; tidak ada koneksi diam-diam yang rusak.
func Open(dsn string) (*gorm.DB, error) {
	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		// Logger diam supaya query (yang bisa memuat parameter) tidak bocor ke
		// log pada level default — sejalan dengan BR-11.
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, err
	}
	return gdb, nil
}

// Ping memverifikasi koneksi masih hidup (dipakai endpoint readiness).
func Ping(ctx context.Context, gdb *gorm.DB) error {
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
