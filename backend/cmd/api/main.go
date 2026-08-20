// Command api adalah entry point server HTTP iMitra. Ia hanya melakukan
// bootstrap: baca config, susun router, jalankan server dengan graceful
// shutdown. Tanpa aturan bisnis di sini (AGENTS.md bagian 3).
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gorm.io/gorm"

	"github.com/irgiys/tim1gow/backend/internal/config"
	"github.com/irgiys/tim1gow/backend/internal/httpapi"
	"github.com/irgiys/tim1gow/backend/internal/repository/db"
)

func main() {
	cfg := config.Load()

	// Buka koneksi DB bila DATABASE_URL diset. Kegagalan koneksi = gagal boot;
	// tidak melanjutkan dengan DB yang rusak (AGENTS.md bagian 4.3).
	var gdb *gorm.DB
	if cfg.DatabaseURL != "" {
		var err error
		gdb, err = db.Open(cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("gagal koneksi database: %v", err)
		}
		log.Println("database terhubung")
	} else {
		log.Println("DATABASE_URL kosong — server jalan tanpa DB (/readyz akan 503)")
	}

	srv := &http.Server{
		Addr:              ":" + cfg.BackendPort,
		Handler:           httpapi.NewRouter(cfg, gdb),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Jalankan server di goroutine agar main bisa menunggu sinyal shutdown.
	go func() {
		log.Printf("iMitra API listening on :%s (env=%s)", cfg.BackendPort, cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Tunggu SIGINT/SIGTERM lalu shutdown dengan tenggat.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}
	log.Println("server stopped")
}
