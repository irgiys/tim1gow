#!/bin/sh
# Mutation testing untuk guard BR-03 (FR-06).
#
# Setiap mutan mensimulasikan satu cara guard prasyarat bisa rusak. Test yang
# baik WAJIB merah untuk semuanya. Mutan yang lolos berarti test kita hijau
# untuk alasan yang salah — persis yang terjadi pada percobaan pertama, ketika
# TestHTTP_BR03_SumberPrasyaratHilangMenolak hanya memeriksa kode status 500
# dan tertipu oleh ConfigError dari hilir.
#
# Pakai: dijalankan di dalam container golang:1.22-alpine dengan repo di /repo.
set -eu

SRC=internal/service/prasyarat_skoring.go
LULUS=0
GAGAL=0

jalankan_mutan() {
	nama="$1"
	shift
	rm -rf /mut
	cp -r /repo /mut
	cd /mut/backend

	"$@"

	if go test ./internal/httpapi/ -run BR03 -count=1 >/tmp/out 2>&1; then
		echo "LOLOS  $nama  <-- MASALAH: test tidak menangkap mutan ini"
		LULUS=$((LULUS + 1))
	else
		if grep -q "build failed" /tmp/out; then
			echo "BUILD  $nama  <-- mutan tidak valid, bukan bukti apa-apa"
			GAGAL=$((GAGAL + 1))
		else
			echo "MERAH  $nama  (tertangkap)"
		fi
	fi
}

# M1: sumber prasyarat hilang tetapi permintaan diloloskan (fail-open).
m1() {
	sed -i 's|return keadaan, domain.NewConfigError(|return keadaan, nil; _ = domain.NewConfigError(|' "$SRC"
}

# M2: hasil guard BR-03 diabaikan — error ditelan, alur lanjut.
m2() {
	sed -i 's|^\t\treturn keadaan, err$|\t\t_ = err|' "$SRC"
}

# M3: kegagalan membaca keadaan dianggap keadaan kosong yang lolos.
m3() {
	sed -i 's|^\tkeadaan, err := s.prasyarat.KeadaanPrasyaratSkoring(ctx, pengajuanID)$|\tkeadaan, err := s.prasyarat.KeadaanPrasyaratSkoring(ctx, pengajuanID); err = nil; keadaan = KeadaanPrasyarat{true, true, true, 1}|' "$SRC"
}

jalankan_mutan "M1 fail-open saat sumber prasyarat nil" m1
jalankan_mutan "M2 hasil guard BR-03 ditelan" m2
jalankan_mutan "M3 kegagalan baca keadaan dianggap lolos" m3

echo
echo "mutan lolos (buruk): $LULUS ; mutan tidak valid: $GAGAL"
[ "$LULUS" -eq 0 ]
