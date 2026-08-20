# =============================================================================
#  mock-slik/Dockerfile — stub layanan SLIK (Go 1.22, stdlib)
# =============================================================================
#  Modul Go terpisah dari backend (AGENTS.md bagian 2): tidak punya dependensi
#  eksternal sama sekali, jadi tidak ada `go mod download` di sini.
#
#  Fixtures TIDAK disalin ke dalam image. Berkas nasabah-uji.csv di-mount
#  read-only oleh docker-compose dari ./fixtures, supaya data uji punya satu
#  sumber kebenaran dan QA dapat mengubahnya tanpa build ulang image.
#
#  wget disertakan karena healthcheck compose memakainya — image tanpa wget
#  membuat container selalu unhealthy walaupun layanannya hidup.
# =============================================================================

FROM golang:1.22-alpine AS builder

WORKDIR /src

COPY go.mod ./
COPY *.go ./

# CGO_ENABLED=0 → binary statis, bisa jalan di alpine tanpa glibc.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/mock-slik .

# -----------------------------------------------------------------------------

FROM alpine:3.20

# wget: healthcheck. tzdata: supaya TZ=Asia/Jakarta berlaku benar pada
# field tanggalData respons inquiry.
RUN apk add --no-cache wget tzdata ca-certificates

WORKDIR /app

COPY --from=builder /out/mock-slik /app/mock-slik

RUN addgroup -S imitra && adduser -S imitra -G imitra && \
    chown -R imitra:imitra /app
USER imitra

EXPOSE 9090

CMD ["/app/mock-slik"]
