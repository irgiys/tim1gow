#!/bin/sh
# =============================================================================
#  01-create-test-database.sh — iMitra
# =============================================================================
#
#  Dijalankan otomatis oleh image postgres saat volume data masih KOSONG
#  (mekanisme /docker-entrypoint-initdb.d). Karena itu skrip ini tidak berjalan
#  ulang pada `docker compose up` berikutnya — untuk memaksanya jalan lagi:
#      docker compose down -v && docker compose up
#
#  Tugasnya HANYA membuat database test yang kosong, supaya
#  `go test ./test/...` (DATABASE_URL_TEST di .env.example) tidak menghapus
#  data demo di database utama.
#
#  Skrip ini TIDAK membuat tabel. Skema hanya berasal dari berkas migrasi
#  golang-migrate di backend/migrations/ (AGENTS.md Larangan 2 & 16). Jangan
#  pernah menambahkan CREATE TABLE di sini: skema yang dibuat di berkas ini
#  tidak akan pernah ter-migrasi dan akan menyimpang dari database demo.
#
#  Dipakai shell (bukan .sql) karena hanya di sini $POSTGRES_DB / $POSTGRES_USER
#  tersedia sebagai variabel; berkas .sql di initdb dijalankan tanpa variabel itu.
#
#  CREATE DATABASE tidak mendukung IF NOT EXISTS, jadi dipakai pola \gexec supaya
#  skrip tetap aman kalau dijalankan ulang manual.
# =============================================================================

set -eu

TEST_DB="${POSTGRES_DB}_test"

echo "[db-init] memastikan database test '${TEST_DB}' ada"

psql --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
     --no-password --set ON_ERROR_STOP=1 <<EOSQL
SELECT format('CREATE DATABASE %I OWNER %I', '${TEST_DB}', '${POSTGRES_USER}')
WHERE NOT EXISTS (
  SELECT 1 FROM pg_database WHERE datname = '${TEST_DB}'
)
\gexec
EOSQL

echo "[db-init] selesai"
