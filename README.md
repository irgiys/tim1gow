# iMitra — Sistem Originasi Pembiayaan Mikro Syariah

---

## 1. Tim

**Nama tim**: iMitra Tim 1

| Nama | Peran | Fokus FR | Akun GitHub |
|---|---|---|---|
| Luthfi | Tech Lead / Integrator | FR-08, FR-09 | https://github.com/zachnrrr |
| Irgiyansyah | Backend Engineer — domain & skoring + DevOps / Release | FR-06, FR-07, FR-13, CI & compose | https://github.com/irgiys/ |
| Yulio Zaki | Backend Engineer — auth & integrasi SLIK | FR-01, FR-05, mock SLIK | https://github.com/yuliozakik |
| Rayvaldo | Backend Engineer — pengajuan & dokumen | FR-02, FR-03, FR-04 | https://github.com/rayvaldoprawira |
| Aldi | AI Workflow Officer + Frontend Engineer | FR-03/FR-04/FR-08 (UI), FR-11 | https://github.com/aldiariq/ |
| Soleh | QA / Verification | FR-12, test AC-01…AC-15 | https://github.com/mshcode89 |

**Pembagian tanggung jawab non-koding** (satu berkas = satu pemilik tunggal; orang lain
mengusulkan lewat PR, supaya tidak ada konflik merge pada tabel markdown):

| Pemilik | Berkas yang dimiliki | Lapisan kode yang dimiliki |
|---|---|---|
| Luthfi (Tech Lead) | `AGENTS.md`, `docs/adr/`, memutus saat tim berdebat > 5 menit, merge PR | `internal/service/approval_service.go`, `audit_service.go` |
| Irgiyansyah | `docs/SDD-iMitra.md` (BAB 4 model data, BAB 5 endpoint), `docker-compose.yml`, `.github/workflows/ci.yml`, `.env.example` | `internal/service/skoring_service.go`, `margin_service.go`, tabel parameter, `backend/migrations/` |
| Yulio Zaki | `docs/SRS-iMitra.md` | `internal/httpapi/` middleware auth+peran, `internal/slik/`, `mock-slik/` |
| Rayvaldo | Kontrak API di SDD BAB 5 bersama Irgiyansyah | `internal/service/pengajuan_service.go`, `internal/repository/` |
| Aldi | `docs/AI-WORKFLOW.md`, `docs/AI-DEVLOG.md` (kontributor: semua anggota) | `frontend/app/`, `frontend/components/`, `frontend/lib/` |
| Soleh | `README.md`, `docs/DEMO-SCRIPT.md`, `docs/TRACEABILITY.md` | `*_test.go`, `backend/test/` |

Tim berisi 6 orang, jadi mengikuti bentuk **Tim 2** pada brief §10: peran DevOps / Release
dirangkap oleh salah satu Backend Engineer (Irgiyansyah), bukan oleh QA. Alasannya: DevOps
memiliki `docker-compose.yml`, `ci.yml`, dan migrasi — ketiganya paling sering rusak justru
karena perubahan di backend, jadi lebih murah dipegang orang yang menulis backend. QA
(Soleh) dibiarkan **murni sebagai penjaga gerbang**: kalau QA juga yang menulis CI, ia
menjadi pemeriksa atas pekerjaannya sendiri — pola yang persis kita larang di BR-09.
Semua peran, termasuk Tech Lead dan AI Workflow Officer, tetap ikut menulis kode.

**Pembagian frontend (`frontend/app/`, Next.js 14 App Router)**

Bentuk Tim 2 pada brief §10 hanya menyediakan satu Frontend Engineer, sedangkan UI iMitra
mencakup 6 peran. Kalau seluruh UI ditumpuk ke satu orang, ia menjadi penghambat semua FR
pada jam-jam terakhir. Karena itu pembagiannya **vertikal**: pemilik aturan bisnis sebuah FR
juga menulis UI untuk FR itu, memakai komponen bersama yang disiapkan Aldi. Aldi tetap
pemilik `frontend/` — ia yang menyiapkan fondasi lebih dulu dan me-review PR frontend orang
lain supaya gaya dan pemakaian komponen tetap konsisten.

| Route / bagian UI | Peran pemakai | Penulis | FR | Kapan |
|---|---|---|---|---|
| Fondasi: `app/layout.tsx`, `lib/apiClient.ts`, `lib/auth.ts`, guard peran sisi klien, komponen bersama (`Table`, `StatusBadge`, `FormField`, `ErrorAlert`) | semua | **Aldi** | — | **Kamis paling awal** — semua route lain menunggu ini |
| `app/login` + penyimpanan sesi/token | semua | **Aldi** | FR-01 | Kamis (walking skeleton) |
| `app/(ao)/pengajuan/baru` — form pengajuan | AO | **Rayvaldo** | FR-02 | Kamis (walking skeleton) |
| `app/(ao)/pengajuan` — daftar pengajuan milik AO | AO | **Rayvaldo** | FR-02 | Kamis (walking skeleton) |
| `app/(ao)/pengajuan/[id]/dokumen` — upload & re-upload satu dokumen | AO | **Rayvaldo** | FR-03 | Kamis–Jumat |
| `app/(ao)/pengajuan/[id]/survei` — form survei (koordinat, foto, omzet) | AO | **Aldi** | FR-04 | Kamis–Jumat |
| `app/(anl)/pengajuan/[id]/verifikasi` — verifikasi dokumen + kode alasan tolak | ANL | **Rayvaldo** | FR-03 | Jumat pagi |
| `app/(anl)/pengajuan/[id]/slik` — tombol SLIK check + tampilan hasil & jalur error | ANL | **Yulio Zaki** | FR-05 | Jumat pagi |
| `app/(anl)/pengajuan/[id]/skoring` — **rincian kontribusi tiap komponen** + form override | ANL | **Irgiyansyah** | FR-06 | Jumat pagi |
| `app/(anl)/pengajuan/[id]/margin` — hitung margin/nisbah + tampilan blokir BR-06 | ANL | **Irgiyansyah** | FR-07 | Jumat pagi |
| `app/(approval)/pengajuan/[id]` — kartu keputusan APPROVE / REJECT / RETURN + alasan | KCP, KC, KOM | **Luthfi** | FR-08 | Jumat pagi |
| `app/(anl)/pengajuan/[id]/audit` — tampilan audit trail (read-only) | ANL, approver | **Luthfi** | FR-09 | Jumat pagi |
| `app/dashboard` — pipeline per status, jumlah per tahap, filter per peran | semua | **Soleh** | FR-12 | Jumat, setelah P0 |
| `app/(adm)/parameter` — CRUD bobot skor, ambang approval, rentang margin | ADM | **Irgiyansyah** | FR-13 | Jumat, setelah P0 |
| Notifikasi in-app | semua | **Aldi** | FR-11 | Jumat, setelah P0 |

**Hak akses & review di GitHub**

| Nama | Akses repo | Peran di alur PR |
|---|---|---|
| Luthfi | Write | Tech Lead — approver & **satu-satunya yang me-merge** ke `main` |
| Irgiyansyah | Write | **Approver** (review + approve PR anggota lain), DevOps / Release — pemilik CI, compose, migrasi, dan yang men-tag `v1.0.0` |
| Yulio Zaki | Write | Reviewer bidang auth/SLIK, pemilik `docs/SRS-iMitra.md` |
| Rayvaldo | Write | Reviewer bidang pengajuan/dokumen |
| Aldi | Write | Reviewer seluruh PR `frontend/` |
| Soleh | Write | QA — **gerbang terakhir sebelum merge** (test & lint benar-benar lolos), tidak memiliki CI supaya tidak memeriksa pekerjaannya sendiri |
| Muhammad Harum Alrasyid (instruktur) | Write | Penilai; membuka issue saat cross-review Jumat 16.05 |

Batas yang berlaku untuk approver, mengikuti brief §8.2 dan `AGENTS.md` bagian 4.2:

- **Tidak ada yang menyetujui PR-nya sendiri**, termasuk Tech Lead dan approver. Ini cermin
  git dari BR-09 (maker ≠ checker) — kalau kontrol itu kita tegakkan di aplikasi, kita juga
  menegakkannya pada diri sendiri.
- Setiap PR butuh **minimal 1 approval dari anggota lain**, dan approval hangus kalau ada
  commit baru (`Dismiss stale approvals` aktif di branch protection).
- Approver **tidak boleh** meloloskan PR dengan CI merah atau test yang dilemahkan
  (`AGENTS.md` Larangan 7). Kalau test gagal, yang salah kode atau requirement-nya.
- Dua approver (Luthfi & Irgiyansyah) supaya PR tidak menganggur saat satu orang sedang
  fokus koding — bukan supaya review bisa dilewati.
- Kepemilikan per path didaftarkan di [`.github/CODEOWNERS`](.github/CODEOWNERS) sehingga
  GitHub otomatis meminta review dari orang yang paham konsekuensinya.

Aturan yang berlaku untuk seluruh frontend:

- **Tidak ada aturan bisnis di frontend.** Skoring, margin, routing approval, dan seluruh
  guard BR dihitung di `backend/internal/service/` (`AGENTS.md` bagian 3 dan Larangan 17).
  UI hanya menampilkan hasil dan pesan error dari API.
- **Menyembunyikan tombol bukan otorisasi.** Guard peran di UI sekadar kenyamanan; penolakan
  yang dinilai (AC-02) adalah 403 dari server (`AGENTS.md` Larangan 6).
- **Nomor referensi tidak dibangkitkan di frontend** (`AGENTS.md` Larangan 4).
- **NIK dan path foto tidak boleh masuk URL** — pakai id internal pengajuan (BR-11).
Pesan error diambil dari field `message` respons API, jangan disusun ulang di UI, supaya
  pesan yang menyebut kode BR (AC-04) tidak hilang.
- Semua panggilan API lewat `lib/apiClient.ts`; jangan ada `fetch()` telanjang di komponen.

**Perubahan peran selama hackathon**:

| Jam | Perubahan | Alasan |
|---|---|---|
| 2026-08-20 11:10 | Peran **DevOps / Release** pindah dari Soleh (QA) ke **Irgiyansyah**, beserta kepemilikan `ci.yml`, `docker-compose.yml`, `.env.example`, dan `backend/migrations/` | QA dibiarkan murni sebagai penjaga gerbang. Kalau QA juga yang menulis CI, ia menjadi pemeriksa atas pekerjaannya sendiri — pola yang persis kita larang di BR-09 (maker ≠ checker). Dicatat di `AGENTS.md` riwayat baris 11:10 |
| 2026-08-20 17:10 | Tabel di atas dan `docs/AI-WORKFLOW.md` disesuaikan agar mencerminkan perubahan 11:10 | Ketiga dokumen sebelumnya tidak sinkron: `AGENTS.md` sudah memindahkan DevOps ke Irgiyansyah sejak 11:10, tetapi README dan AI-WORKFLOW masih mencantumkan "CI & compose" sebagai tugas Soleh. Ditemukan saat audit QA atas tugas yang belum dikerjakan |

---

## 2. Cara Menjalankan

Langkah di bawah dijalankan dari clone bersih pada Windows 11 + Docker Desktop
(2026-08-21). Satu catatan jujur di depan supaya tidak menjadi kejutan: service
`frontend` **masih dikomentari** di `docker-compose.yml`, jadi `docker compose up`
menghidupkan backend, database, dan mock SLIK — bukan UI. Cara menjalankan UI ada di
bagian 2.2 langkah 4. Alasannya ada di bagian 5 (utang teknis).

### 2.1 Prasyarat

- **Docker Engine >= 24** dan **Docker Compose v2** (`docker compose version` —
  perhatikan spasi, bukan `docker-compose`). Ini satu-satunya prasyarat untuk backend,
  database, migrasi, seed, dan mock SLIK.
- **Node.js 20 LTS** + npm 10 — hanya diperlukan untuk menjalankan frontend, karena
  service-nya belum aktif di compose. Versi 20 dipakai supaya sama dengan
  `NODE_VERSION` di `.github/workflows/ci.yml`.
- Port yang harus bebas di host: **8080** (API), **3000** (UI), **5432** (Postgres),
  **9090** (mock SLIK). Semuanya dapat diubah lewat `.env`.
- Go **tidak** perlu dipasang. Backend dibangun di dalam container, dan bila ingin
  menjalankan test tanpa memasang Go, pakai perintah container di bagian 2.6.

### 2.2 Langkah

```bash
# 1. Clone
git clone https://github.com/irgiys/tim1gow.git
cd tim1gow

# 2. Siapkan environment (nilai default sudah cukup untuk demo)
cp .env.example .env

# 3. Hidupkan backend + database + mock SLIK.
#    Service `migrate` menjalankan migrasi lalu seed sendiri sebelum backend start,
#    jadi tidak ada langkah manual sesudah ini.
docker compose up -d --build

# 4. Jalankan frontend (service-nya belum aktif di compose — lihat bagian 5)
cd frontend
npm ci
npm run build
cp -r .next/static .next/standalone/.next/ && cp -r public .next/standalone/
cd .next/standalone && PORT=3000 HOSTNAME=127.0.0.1 node server.js
```

Dua jebakan pada langkah 4 yang kami temukan sendiri, ditulis di sini supaya tidak
terulang di mesin penilai:

- `npm start` / `next start` **tidak bekerja** dengan `output: 'standalone'` di
  `next.config.js` — Next mencetak peringatan lalu menyajikan route yang salah.
  Jalankan `node server.js` dari `.next/standalone`, sama seperti `frontend/Dockerfile`.
- `HOSTNAME=127.0.0.1` wajib. Tanpanya server bind ke hostname mesin
  (mis. `http://LAPTOP-XXXX:3000`) dan `http://localhost:3000` menolak koneksi walau
  log menulis "Ready".

Verifikasi bahwa semuanya benar-benar hidup:

```bash
docker compose ps                                  # backend/db/mock-slik: healthy
curl -s http://localhost:8080/readyz               # {"db":"ok","status":"ready"}
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:3000/login   # 200
```

### 2.3 Alamat layanan setelah jalan

Port mengikuti `.env.example` (`FRONTEND_PORT`, `BACKEND_PORT`, `MOCK_SLIK_PORT`, `DB_PORT`).

| Layanan | URL | Catatan |
|---|---|---|
| Frontend | http://localhost:3000 | Dijalankan manual (bagian 2.2 langkah 4). Route yang ada: `/login`, `/pengajuan`, `/approval` |
| Backend API | http://localhost:8080 | `/healthz` liveness, `/readyz` readiness (memeriksa DB). Seluruh endpoint di bawah `/api` |
| Mock SLIK | http://localhost:9090 | `POST /slik/inquiry` sesuai kontrak brief §6.1. Membaca `fixtures/nasabah-uji.csv` saat start |
| Database | `postgres://imitra_app:***@localhost:5432/imitra` | Password ada di `.env`. Akses cepat: `docker exec -it imitra-db psql -U imitra_app -d imitra` |

**CORS**: API hanya menerima origin yang terdaftar di `CORS_ALLOWED_ORIGINS`, dicocokkan
**persis**. Default memuat `http://localhost:3000` dan `http://127.0.0.1:3000` karena bagi
browser keduanya origin berbeda. Membuka UI dari alamat lain (mis. IP LAN) membuat login
gagal dengan preflight 403 tanpa error apa pun di log backend — tambahkan origin-nya di
`.env` lebih dulu.

### 2.4 Migrasi & seed

Pada `docker compose up`, service `migrate` menjalankan
`command: ["sh","-c","/app/migrate up && /app/seed"]` dan berhenti dengan kode 0. Backend
baru start sesudahnya. Jadi **untuk pemakaian normal tidak ada perintah yang perlu
dijalankan** — perintah di bawah hanya untuk menjalankan ulang secara manual.

```bash
# Migrasi ulang (golang-migrate, idempoten — aman diulang)
docker compose run --rm migrate sh -c "/app/migrate up"

# Seed ulang (idempoten; tabel parameter memakai ON CONFLICT DO NOTHING,
# jadi bobot yang diubah ADM saat demo TIDAK ter-reset — itu yang membuat AC-15 bisa dibuktikan)
docker compose run --rm migrate sh -c "/app/seed"

# Reset demo ke kondisi awal: buang volume database, lalu bangun ulang.
# Migrasi + seed berjalan otomatis, jadi ini satu-satunya perintah reset yang dibutuhkan.
docker compose down -v && docker compose up -d --build

# Verifikasi hasil reset
docker compose logs migrate --no-log-prefix | tail -5
docker exec imitra-db psql -U imitra_app -d imitra -c "SELECT count(*) FROM pengguna;"   # 6
```

`docker compose down -v` **menghapus data**. Tanpa `-v`, data pengajuan dari demo
sebelumnya tetap ada dan nomor referensi melanjutkan urutan (BR-12: nomor tidak pernah
dipakai ulang).


### 2.5 Akun demo

Keenam akun di bawah dibuat oleh `backend/cmd/seed` (idempoten, aman dijalankan ulang) dan
otomatis tersedia setelah `docker compose up` — service `migrate` menjalankan migrasi lalu seed
sebelum backend start, jadi tidak ada langkah manual.

Password di bawah bukan rahasia: akun ini hanya ada di lingkungan demo, dan nilainya berasal
dari `SEED_DEFAULT_PASSWORD` di `.env`. Tidak ada secret nyata (JWT secret, kredensial DB,
token API) yang ditulis di berkas ini.

**Login memakai EMAIL, bukan username.** Field JSON-nya `email`
(`backend/internal/httpapi/auth_handler.go`); mengirim `username` dijawab
`400 VALIDATION_ERROR "email dan password wajib diisi"`.

| Peran | Email | Password | Nama tampilan | Dipakai untuk AC |
|---|---|---|---|---|
| AO | `ao@imitra.test` | `Demo1234!` | Ayu Account Officer | AC-01, AC-02, AC-03 |
| ANL | `anl@imitra.test` | `Demo1234!` | Andi Analis Mikro | AC-03, AC-04, AC-06, AC-07, AC-08, AC-09 |
| KCP | `kcp@imitra.test` | `Demo1234!` | Kartika Kepala CP | AC-10, AC-11 |
| KC | `kc@imitra.test` | `Demo1234!` | Kurnia Kepala Cabang | AC-10 |
| KOM | `kom@imitra.test` | `Demo1234!` | Komite Pembiayaan | AC-10 |
| ADM | `adm@imitra.test` | `Demo1234!` | Admin Sistem | AC-15 |

Verifikasi cepat bahwa keenamnya benar-benar bisa login (dijalankan terhadap stack yang hidup):

```bash
for e in ao anl kcp kc kom adm; do
  printf '%-4s -> %s\n' "$e" "$(curl -s -X POST http://localhost:8080/api/auth/login \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$e@imitra.test\",\"password\":\"Demo1234!\"}" \
    | grep -o '"peran":"[^"]*"' | cut -d'"' -f4)"
done
```

Keluaran yang diharapkan: `ao -> AO`, `anl -> ANL`, `kcp -> KCP`, `kc -> KC`, `kom -> KOM`,
`adm -> ADM`. Baris yang kosong berarti login gagal — periksa apakah service `migrate` sudah
selesai (`docker compose logs migrate`).

**Catatan untuk penilai — apa yang sudah bisa dicoba lewat UI.** UI saat ini memiliki
`/login`, `/pengajuan` (AO/ANL/approver), dan `/approval` (KCP/KC/KOM/ANL). Masuk sebagai ADM
masih diarahkan ke `/parameter` yang **belum ada**, sehingga FR-13 hanya dapat diperiksa lewat
API. Dua batasan yang perlu diketahui sebelum mencoba:

- **Daftar pengajuan untuk ANL/approver masih kosong.** `GET /api/pengajuan` masih selalu
  memakai `DaftarMilikAO`, jadi hanya AO yang melihat isinya. Layar `/approval` karena itu
  membuka pengajuan **per id**, bukan lewat daftar — id-nya bisa diambil dari layar
  `/pengajuan` saat login sebagai AO, atau dari audit trail.
- **Skoring belum bisa dijalankan lewat UI.** Untuk mencapai status `SCORED` (prasyarat
  `ajukan-approval`), jalankan skoring lewat API. Alurnya: SLIK check (FR-05, endpointnya belum
  ada) → skoring → ajukan approval.

Status jujur per FR ada di tabel bagian 4.

### 2.6 Test & lint

Perintah di bawah sama dengan `.github/workflows/ci.yml` dan `AGENTS.md` bagian 7.
Backend dijalankan dari `backend/`, frontend dari `frontend/`.

```bash
# ---- Backend (Go 1.22) — dari ./backend ----
go mod download
gofmt -l .                      # harus TIDAK menghasilkan output
go vet ./...
golangci-lint run ./...
go test ./... -count=1

# ---- Frontend (Node 20) — dari ./frontend ----
npm ci
npm run lint
npm run build
```

**Tanpa memasang Go di host.** Go tidak wajib ada (bagian 2.1); versinya dikunci ke 1.22
oleh `go.mod`, jadi jalankan test di dalam container dengan versi yang sama:

```bash
docker run --rm -v "$PWD/backend:/src" -w /src golang:1.22-alpine \
  sh -c 'go build ./... && go test ./... -count=1'
```

**Catatan tentang `gofmt` di Windows.** Working copy repo ini memakai CRLF, sehingga
`gofmt -l .` di mesin Windows mendaftar hampir semua berkas — termasuk berkas yang tidak
pernah disentuh. Itu **bukan** bukti format bersih maupun kotor. Untuk memeriksanya
sungguhan, normalkan akhir baris lebih dulu di direktori sementara (repo tidak disentuh),
baru jalankan `gofmt`:

```bash
docker run --rm -v "$PWD/backend:/src" -w /src golang:1.22-alpine sh -c \
  'rm -rf /tmp/n && mkdir -p /tmp/n && cp -r internal cmd go.mod go.sum /tmp/n/ && cd /tmp/n && find . -name "*.go" -exec dos2unix -q {} + 2>/dev/null || find . -name "*.go" -exec sed -i "s/[[:cntrl:]]$//" {} + ; echo "berkas belum terformat: $(gofmt -l . | wc -l)"'
```

Hasil yang diharapkan: `berkas belum terformat: 0`.

Kami menuliskannya karena pernah keliru melaporkan `gofmt -l` sebagai "murni CRLF" tanpa
memeriksa — ternyata ada empat berkas yang benar-benar salah format, tenggelam di antara
40 berkas yang terdaftar hanya karena CRLF (lihat DEVLOG-05). Di CI hal ini tidak muncul:
runner Linux meng-checkout dengan LF, jadi `gofmt -l .` di sana bersih apa adanya.

Sebagian test integrasi backend membutuhkan database dan mock SLIK aktif. Cara paling
andal mereproduksi CI di mesin bersih:

```bash
docker compose up -d db mock-slik
cd backend && APP_ENV=test go run ./cmd/migrate up && go run ./cmd/seed && go test ./... -count=1
```

---

## 3. Arsitektur Singkat

Empat komponen berjalan di Docker Compose. **Browser** memanggil **Backend API** (Go 1.22 +
Chi) langsung — bukan lewat frontend sebagai proxy — sehingga API menegakkan CORS dengan
pencocokan origin persis. Di dalam backend alurnya searah: `httpapi` (handler + middleware
auth/peran) → `service` (seluruh aturan bisnis) → `repository` (GORM, satu-satunya lapisan
yang menyentuh SQL) → **PostgreSQL 16**. Panggilan SLIK keluar dari `service` ke **mock
SLIK** lewat HTTP, bukan sebagai fungsi lokal.

**Seluruh aturan bisnis ada di `backend/internal/service/`** — skoring (BR-07), rentang
margin (BR-06), routing approval berjenjang (BR-02), guard prasyarat skoring (BR-03), dan
maker ≠ checker (BR-09). Frontend tidak menghitung apa pun; ia menampilkan `message` dari
API apa adanya supaya pesan yang menyebut kode BR tidak hilang (AC-04). Angka ambang,
bobot, dan rentang **tidak** ada di kode: ketiganya baris di tabel `parameter_skoring`,
`ambang_approval`, dan `rentang_margin`, dibaca ulang setiap perhitungan sehingga ADM dapat
mengubahnya tanpa deploy ulang (AC-15).

**Otorisasi ditegakkan di server, dua lapis.** `MiddlewareAuth` memverifikasi JWT HS256 dan
menaruh identitas di context; `WajibPeran` membatasi setiap route ke peran yang berhak dan
menjawab **403** untuk akses lintas-peran (AC-02). Identitas aktor — yang dipakai untuk
BR-09 dan untuk kolom aktor di audit trail — **selalu** diambil dari token yang sudah
diverifikasi, tidak pernah dari header, query, atau badan request. Guard peran di UI hanya
kenyamanan tampilan. Skema database hanya berasal dari migrasi SQL (`golang-migrate`);
`AutoMigrate` dilarang.

Detail model data dan daftar endpoint ada di [`docs/SDD-iMitra.md`](docs/SDD-iMitra.md) —
tidak diduplikasi di sini.

```mermaid
graph TB
  subgraph host["Host (browser pengguna)"]
    B["Browser<br/>Next.js 14 App Router<br/>:3000"]
  end

  subgraph compose["Docker Compose — network: imitra"]
    subgraph be["Backend API — Go 1.22 + Chi · :8080"]
      MW["httpapi: MiddlewareCORS → MiddlewareAuth → WajibPeran<br/>(JWT HS256, identitas dari token)"]
      SVC["service: ATURAN BISNIS<br/>skoring · margin · approval · audit<br/>BR-01…BR-12"]
      REPO["repository (GORM)<br/>satu-satunya lapisan penyentuh SQL"]
      MW --> SVC --> REPO
    end
    DB[("PostgreSQL 16<br/>:5432<br/>skema dari migrasi SQL")]
    SLIK["mock-slik :9090<br/>POST /slik/inquiry<br/>fixtures/nasabah-uji.csv"]
    MIG["migrate (sekali jalan)<br/>golang-migrate up + seed"]
  end

  B -- "fetch + Bearer token<br/>CORS: origin dicocokkan persis" --> MW
  REPO --> DB
  SVC -- "HTTP, bukan fungsi lokal<br/>timeout/503/404 ditangani" --> SLIK
  MIG -- "jalan & selesai sebelum backend start" --> DB

  PARAM["parameter_skoring · ambang_approval · rentang_margin<br/>dibaca ulang setiap perhitungan (AC-15)"]
  DB -.-> PARAM
  PARAM -.-> SVC
```

**Stack yang dipilih**: Go 1.22 + Chi v5 + GORM 1.25 (query saja) + golang-migrate 4.17,
Next.js 14.2 App Router + React 18 + TypeScript 5.4, PostgreSQL 16-alpine, mock SLIK dengan
`net/http` stdlib.
Alasan pemilihan ada di [`docs/adr/0001-pilihan-stack.md`](docs/adr/0001-pilihan-stack.md).

**Aturan untuk AI agent**: [`AGENTS.md`](AGENTS.md)

---

## 4. Status Functional Requirements

### P0 — WAJIB (batas lulus fungsional)

Status di bawah dinilai dari **panggilan API terhadap stack yang berjalan**, bukan dari
lulusnya test unit. Kolom PR menyebut PR yang menyelesaikannya; `#11` masih menunggu review
pada saat baris ini ditulis.

| FR | Requirement | Prioritas | Status | PR |
|---|---|---|---|---|
| FR-01 | Autentikasi & Otorisasi Berbasis Peran | P0 | Selesai & teruji | #11 |
| FR-02 | Pengajuan Pembiayaan Mikro | P0 | Selesai & teruji | #11 |
| FR-03 | Upload & Verifikasi Dokumen | P0 | Sebagian | #11 |
| FR-04 | Survei Lapangan (OTS) | P0 | Selesai & teruji | #11 |
| FR-05 | SLIK Check | P0 | Sebagian | — |
| FR-06 | Skoring Kelayakan Mikro | P0 | Selesai & teruji | #5 |
| FR-07 | Perhitungan Margin / Nisbah | P0 | Selesai & teruji | #5 |
| FR-08 | Approval Berjenjang | P0 | Selesai & teruji | #6, #11 |
| FR-09 | Audit Trail | P0 | Selesai & teruji | #6, #11 |

Dua baris P0 yang **bukan** "Selesai & teruji", dirinci di bagian 5:

- **FR-05 SLIK Check** — mock SLIK berjalan dan punya test sendiri, tetapi backend belum
  memanggilnya: `POST /api/pengajuan/{id}/slik-check` mengembalikan **404** dan
  `internal/slik/` belum ada. Akibat berantai: BR-04 (hasil SLIK kedaluwarsa 30 hari) belum
  ditegakkan, dan status `SCORED` hanya dapat dicapai dengan mengubah status lewat SQL —
  sehingga **AC-05 dan AC-06 belum dapat dibuktikan lewat aplikasi**. Ini kekurangan P0
  paling berdampak yang kami sadari.
- **FR-03 Upload & Verifikasi Dokumen** — verifikasi, penolakan dengan kode alasan, dan
  re-upload satu dokumen (AC-03) bekerja penuh lewat API. Yang belum sesuai: endpoint upload
  menerima JSON `urlBerkas`, sedangkan SDD BAB 5 menetapkan `multipart/form-data`. Jadi
  berkas tidak benar-benar diunggah, hanya rujukannya yang dicatat.

**Catatan penting soal cakupan UI.** Delapan FR di atas berstatus jalan **lewat API**;
antarmukanya baru ada tiga halaman (`/login`, `/pengajuan`, `/approval`). FR-03, FR-04,
FR-06, FR-07, dan FR-09 karena itu hanya dapat didemokan lewat `curl`, bukan lewat browser.
Rinciannya ada di bagian 5.

### P1 — SEHARUSNYA (nilai penuh butuh ini)

| FR | Requirement | Prioritas | Status | PR |
|---|---|---|---|---|
| FR-10 | Pembiayaan Kelompok (Majelis) | P1 | Sebagian | #11 |
| FR-11 | Notifikasi Perubahan Status | P1 | Tidak dikerjakan | — |
| FR-12 | Dashboard Pipeline | P1 | Tidak dikerjakan | — |
| FR-13 | Parameter Terkonfigurasi | P1 | Sebagian | #5 |

### P2 — BOLEH (hanya kalau P0 dan P1 tuntas dan teruji)

Seluruh FR P2 **tidak dikerjakan**: P0 belum tuntas (FR-05), jadi mengerjakan P2 berarti
menambah permukaan yang belum teruji sementara batas lulus fungsional belum tercapai.
Alasannya dicatat di bagian 5.

| FR | Requirement | Prioritas | Status | PR |
|---|---|---|---|---|
| FR-14 | Simulasi angsuran murabahah & proyeksi bagi hasil musyarakah | P2 | Tidak dikerjakan | — |
| FR-15 | Ekspor daftar pengajuan ke CSV | P2 | Tidak dikerjakan | — |
| FR-16 | Mode draft offline untuk AO di lapangan | P2 | Tidak dikerjakan | — |
| FR-17 | Deteksi lokasi palsu (mock location) pada survei lapangan | P2 | Tidak dikerjakan | — |
| FR-18 | Laporan Turn-Around Time per tahap dan per petugas | P2 | Tidak dikerjakan | — |

Penelusuran rinci FR → endpoint → test → PR ada di [`docs/TRACEABILITY.md`](docs/TRACEABILITY.md).

---

## 5. Tidak Diimplementasikan dan Mengapa

> **Bagian ini wajib ada dan wajib terisi.** Ia bukan pengakuan kegagalan — ia bukti bahwa
> tim memutuskan secara sadar.

| FR / Bagian | Keputusan | Apa yang jalan | Apa yang tidak | Alasan | Diputuskan kapan |
|---|---|---|---|---|---|
| **FR-05** SLIK Check | Sebagian | Mock SLIK berjalan di compose, punya test sendiri (`mock-slik/main_test.go`), dan mengembalikan 200/404/503 sesuai kontrak brief §6.1 | `internal/slik/` client dan route `POST /api/pengajuan/{id}/slik-check` (**404**). BR-04 (kedaluwarsa 30 hari) belum ditegakkan. AC-05 & AC-06 belum dapat dibuktikan lewat aplikasi | Dua bug otorisasi yang lebih berbahaya muncul lebih dulu: prasyarat BR-03 dapat dipalsukan klien, dan identitas approver diambil dari header sehingga audit trail mencatat orang yang salah. Keduanya cacat kontrol di aplikasi perbankan, jadi kami dahulukan itu daripada menambah jalur baru | Jumat, setelah audit BR-10 selesai |
| **FR-03** upload berkas | Sebagian | Verifikasi dokumen, penolakan dengan kode alasan, re-upload satu dokumen tanpa kehilangan data lain (AC-03) — semuanya lewat API | Upload menerima JSON `urlBerkas`, bukan `multipart/form-data` seperti SDD BAB 5. Berkas tidak benar-benar tersimpan | Menyelesaikan AC-03 (perilaku yang dinilai) lebih dulu; mengubah ke multipart menyentuh kontrak SDD dan butuh keputusan Tech Lead, bukan diputuskan sendiri saat implementasi | Jumat, saat wiring handler |
| **FR-10** Pembiayaan Kelompok | Sebagian | Enum `TipeKelompok` dan tabel `pengajuan_anggota` sudah ada; pengajuan bertipe `KELOMPOK` dapat dibuat dan disimpan | Belum ada endpoint pengelolaan anggota, dan ambang approval belum dihitung dari **total plafon kelompok** seperti bagian 5.1 AGENTS.md. AC-14 belum diuji | Fondasi datanya sudah ada sejak migrasi 000004, tetapi menyelesaikannya berarti mengubah logika routing approval yang sudah teruji untuk P0. Risiko merusak AC-10/AC-11 lebih besar daripada nilai P1 yang didapat | Jumat, saat menilai sisa waktu |
| **FR-13** Parameter Terkonfigurasi | Sebagian | Ketiga tabel parameter ada, di-seed idempoten, dan **dibaca ulang dari DB setiap perhitungan** — mengubah baris `parameter_skoring` langsung mengubah hasil skoring tanpa restart (AC-15 terbukti lewat test dan lewat SQL) | Endpoint CRUD ADM (**404** semua) dan halaman `/parameter`. ADM login lalu mendarat di 404 | Inti AC-15 adalah "parameter berupa data, bukan konstanta" — itu sudah terbukti. CRUD-nya lapisan tipis di atasnya, dan P0 (FR-05) belum tuntas | Jumat, setelah P0 dinilai belum tuntas |
| **FR-11** Notifikasi | Dibuang | — | Seluruhnya (`/api/notifikasi` → 404) | Notifikasi tidak dirujuk AC mana pun. Menambahkannya berarti permukaan baru tanpa kriteria penerimaan, sementara FR-05 (P0, punya dua AC) masih terbuka | Jumat, Gate 3 |
| **FR-12** Dashboard Pipeline | Dibuang | — | Seluruhnya (`/api/dashboard` → 404) | Sama seperti FR-11: tanpa AC. Selain itu dashboard bergantung pada query lingkup per peran yang justru masih menjadi utang (lihat di bawah), jadi mengerjakannya sekarang akan menghasilkan dashboard yang menampilkan daftar kosong untuk ANL dan approver | Jumat, Gate 3 |
| **FR-14 – FR-18** (seluruh P2) | Dibuang | — | Seluruhnya | Brief menetapkan P2 hanya dikerjakan bila P0 **dan** P1 tuntas dan teruji. P0 belum tuntas, jadi mengerjakan P2 adalah kesalahan prioritas — bukan kekurangan waktu | Kamis, saat menyusun urutan kerja |
| **UI untuk FR-03/04/06/07/09** | Sebagian | Ketiga halaman yang ada (`/login`, `/pengajuan`, `/approval`) berfungsi penuh terhadap API sungguhan | Halaman `/pengajuan/[id]/dokumen`, `/survei`, `/verifikasi`, `/slik`, `/skoring`, `/margin`, `/audit`, `/parameter`, `/dashboard` | Kami memilih menutup lubang **otorisasi dan audit** di backend lebih dulu (lima bug, semuanya hanya muncul saat aplikasi dijalankan) daripada menambah halaman di atas backend yang identitas aktornya masih bisa dipalsukan. Konsekuensinya jujur: lima FR itu hanya dapat didemokan lewat `curl` | Jumat, setelah menemukan bug identitas |

**Utang teknis yang kami sadari**:

- **`GET /api/pengajuan` selalu memakai `DaftarMilikAO`.** ANL dan approver menerima daftar
  **kosong** (terverifikasi: ANL 0 baris, AO 5 baris), padahal SDD BAB 5 menetapkan ANL
  melihat semua dan approver melihat yang menunggu levelnya. Route sudah dibuka untuk
  mereka, query-nya belum ada. Akibat langsung: layar `/approval` harus membuka pengajuan
  **per id** alih-alih menampilkan daftar. Kami memilih membiarkannya terlihat apa adanya
  daripada menyembunyikannya dengan daftar palsu.
- **Service `frontend` masih dikomentari di `docker-compose.yml`** (baris ~213–240),
  sehingga `docker compose up` tidak menghidupkan UI dan penilai harus menjalankannya
  manual (bagian 2.2 langkah 4). `frontend/Dockerfile` sendiri sudah ada dan berfungsi.
  Belum diaktifkan karena mengubah compose adalah PR terpisah menurut AGENTS.md Larangan 14.
- **`golangci-lint` belum pernah dijalankan di lokal** — tidak terpasang di mesin
  pengembang, jadi kebersihannya hanya dibuktikan CI. `gofmt`, `go vet`, dan `go test`
  sudah bersih.
- **Test frontend belum ada.** Hanya `npm run lint` dan `npm run build` yang menjaga; tidak
  ada test komponen maupun end-to-end. Sesuai AGENTS.md bagian 2 test frontend belum wajib
  di P0, tetapi ini tetap utang.
- **Upload belum memvalidasi ukuran dan MIME sungguhan.** `UPLOAD_ALLOWED_MIME` ada di
  `.env` tetapi belum ditegakkan karena berkasnya belum benar-benar diunggah (lihat FR-03).
- **Penanganan timeout SLIK belum teruji dari sisi backend.** Mock SLIK dapat dipaksa 503
  dan `SLIK_TIMEOUT_MS` sudah ada di `.env`, tetapi karena client belum ada, jalur error itu
  belum pernah dilalui backend. Yang sudah pasti: kegagalan SLIK **tidak** boleh dianggap
  SLIK bersih (AGENTS.md Larangan 15) — aturannya sudah ditulis, penegakannya menyusul
  bersama FR-05.
- **Data demo bertumpuk di database.** Beberapa pengajuan uji dibuat lewat API selama
  pengujian manual, dan status sebagiannya diubah lewat SQL untuk mencapai `SCORED` (karena
  FR-05 belum ada). Untuk demo bersih, jalankan perintah reset di bagian 2.4.

---

## 6. Catatan AI Workflow

Tim ini memakai AI sebagai alat rekayasa. Jejaknya ada di tiga tempat:

| Dokumen | Isi |
|---|---|
| [`AGENTS.md`](AGENTS.md) | Aturan repo yang dibaca AI agent: stack, struktur, konvensi, aturan bisnis, larangan |
| [`docs/AI-WORKFLOW.md`](docs/AI-WORKFLOW.md) | Tool dan model apa untuk tugas apa, cara memberi konteks, pembagian AI vs manual |
| [`docs/AI-DEVLOG.md`](docs/AI-DEVLOG.md) | Jurnal pemakaian AI: minimal 10 entri, minimal 3 di antaranya kasus AI salah dan kami menangkapnya |

**Pola pemakaian AI paling berguna**: Menggunakan AI untuk otomatisasi eksplorasi kode, pembuatan boilerplates handler/service/test Go, serta konversi kontrak SDD menjadi test suite otomatis secara konsisten.

**Hal yang TIDAK diserahkan ke AI**: Penentuan aturan otorisasi perbankan, validasi identitas aktor di context JWT, dan keputusan pemisahan layer arsitektur. Semua verifikasi keamanan dan mutation test dilakukan manual oleh engineer.

**Keputusan arsitektur yang menolak saran AI**: ADR-0001 (Menolak saran AI untuk menggunakan `GORM AutoMigrate` dan library tambahan di luar stdlib, demi memastikan reproduksibilitas skema SQL via `golang-migrate` di lingkungan penilai).

---

## 7. Dokumen Lain

| Dokumen | Isi |
|---|---|
| [`docs/SRS-iMitra.md`](docs/SRS-iMitra.md) | Requirement ringkas turunan brief |
| [`docs/SDD-iMitra.md`](docs/SDD-iMitra.md) | Arsitektur, model data, daftar endpoint |
| [`docs/TRACEABILITY.md`](docs/TRACEABILITY.md) | FR → AC → endpoint → test → PR |
| [`docs/DEMO-SCRIPT.md`](docs/DEMO-SCRIPT.md) | Skrip demo AC-01 s.d. AC-15 beserta data uji |
| [`docs/adr/`](docs/adr/) | Architecture Decision Records (minimal 3) |
| [`fixtures/nasabah-uji.csv`](fixtures/nasabah-uji.csv) | Data uji wajib untuk mock SLIK |
| [`SETUP-SPRINT-0.md`](SETUP-SPRINT-0.md) | Checklist Sprint 0 — kerjakan ini lebih dulu |