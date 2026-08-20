# =============================================================================
#  backend/Dockerfile — iMitra API (Go 1.22 + Chi)
# =============================================================================
#  Multi-stage: build statis di stage `builder`, lalu jalankan di alpine tipis.
#  Dua binary dibangun sekaligus karena keduanya dipakai container ini:
#    - api     : server HTTP (cmd/api)
#    - migrate  : runner golang-migrate (cmd/migrate), dijalankan sebelum api
#
#  wget disertakan di stage runtime karena healthcheck compose memakainya
#  (penyebab umum container "selalu unhealthy" adalah image tanpa wget/curl).
# =============================================================================

FROM golang:1.22-alpine AS builder

WORKDIR /src

# Unduh dependensi lebih dulu supaya layer ini ter-cache selama go.mod/go.sum
# tidak berubah — build ulang setelah ubah kode jadi jauh lebih cepat.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 → binary statis, bisa jalan di alpine tanpa glibc.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# -----------------------------------------------------------------------------

FROM alpine:3.20

# wget: dipakai healthcheck. tzdata: supaya TZ=Asia/Jakarta berlaku benar pada
# timestamp audit trail (BR-10).
RUN apk add --no-cache wget tzdata ca-certificates

WORKDIR /app

COPY --from=builder /out/api /app/api
COPY --from=builder /out/migrate /app/migrate

# cmd/migrate membuka sumber migrasi dengan path relatif "file://migrations",
# jadi berkas SQL harus ada relatif terhadap WORKDIR.
COPY migrations /app/migrations

# Direktori upload dokumen (di-mount volume oleh compose).
RUN mkdir -p /app/uploads

# Jalan sebagai non-root: kalau UPLOAD_DIR bisa ditulis root saja, upload gagal.
RUN addgroup -S imitra && adduser -S imitra -G imitra && \
    chown -R imitra:imitra /app
USER imitra

EXPOSE 8080

CMD ["/app/api"]
