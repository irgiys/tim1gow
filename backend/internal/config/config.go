// Package config membaca konfigurasi dari environment di satu tempat lalu
// di-inject ke lapisan bawah (AGENTS.md bagian 3: jangan os.Getenv tersebar).
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config memuat seluruh setelan runtime backend.
type Config struct {
	AppEnv             string
	BackendPort        string
	DatabaseURL        string
	DatabaseURLTest    string
	SlikBaseURL        string
	SlikInquiryPath    string
	SlikTimeout        time.Duration
	CorsAllowedOrigins []string
	// SeedDefaultPassword dipakai cmd/seed untuk akun demo. Nilainya berasal
	// dari environment, tidak pernah di-hardcode (AGENTS.md Larangan 10).
	SeedDefaultPassword string
	// PasswordHashCost adalah cost bcrypt. Nilai kecil mempercepat seed dan
	// test; produksi memakai nilai lebih tinggi lewat environment.
	PasswordHashCost int
	// JWTSecret menandatangani token HS256. Wajib diisi lewat environment;
	// tidak ada nilai default supaya token tidak pernah ditandatangani dengan
	// secret yang bisa ditebak (AGENTS.md Larangan 10).
	JWTSecret string
	// JWTExpiresIn adalah masa berlaku access token.
	JWTExpiresIn time.Duration
}

// Load membaca konfigurasi dari environment. Tidak ada nilai secret yang
// di-hardcode; hanya default yang aman untuk pengembangan lokal.
func Load() Config {
	loadDotEnv()
	timeoutMs := getInt("SLIK_TIMEOUT_MS", 3000)
	return Config{
		AppEnv:          getEnv("APP_ENV", "development"),
		BackendPort:     getEnv("BACKEND_PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", ""),
		DatabaseURLTest: getEnv("DATABASE_URL_TEST", ""),
		SlikBaseURL:     getEnv("SLIK_BASE_URL", "http://localhost:9090"),
		SlikInquiryPath: getEnv("SLIK_INQUIRY_PATH", "/slik/inquiry"),
		SlikTimeout:     time.Duration(timeoutMs) * time.Millisecond,
		CorsAllowedOrigins: splitCSV(
			getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"),
		),
		// Tanpa default: seed akun demo harus gagal terang-terangan kalau
		// variabelnya belum diset, bukan memakai password yang bisa ditebak.
		SeedDefaultPassword: getEnv("SEED_DEFAULT_PASSWORD", ""),
		PasswordHashCost:    getInt("PASSWORD_HASH_COST", 10),
		JWTSecret:           getEnv("JWT_SECRET", ""),
		JWTExpiresIn:        getDurasi("JWT_EXPIRES_IN", 8*time.Hour),
	}
}

// loadDotEnv mencoba membaca .env dari direktori kerja atau direktori induk (repo root).
// Hanya mengisi variabel yang belum diset di environment.
func loadDotEnv() {
	paths := []string{".env", "../.env", "../../.env"}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			// Hapus quote pembungkus jika ada
			if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
				v = v[1 : len(v)-1]
			}
			if _, exists := os.LookupEnv(k); !exists {
				_ = os.Setenv(k, v)
			}
		}
		break
	}
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// getDurasi membaca durasi bergaya Go ("8h", "30m"). Nilai yang tidak dapat
// diurai memakai default, bukan 0 — masa berlaku token 0 detik akan membuat
// setiap login langsung kedaluwarsa dan sangat membingungkan saat demo.
func getDurasi(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(strings.TrimSpace(v)); err == nil && d > 0 {
			return d
		}
	}
	return def
}
