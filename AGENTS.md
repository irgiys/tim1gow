# AGENTS.md — Aturan untuk AI Agent di Repo iMitra

> ## WAJIB DIBACA SEBELUM MENGISI
>
> **1. Berkas ini WAJIB di-commit sebelum commit fitur pertama.** Penilai akan menjalankan
> `git log --reverse --oneline` dan memeriksa urutannya. Kalau commit fitur pertama muncul
> lebih dulu, nilai aspek "Disiplin rekayasa berbantuan AI" (bobot 25) turun.
>
> **2. Berkas ini WAJIB berevolusi.** Kalau isinya sama pada Kamis 09.00 dan Jumat 15.00,
> artinya tidak ada satu pun pelajaran dari 9 jam kerja yang masuk ke sini. Setiap kali
> AI melanggar sesuatu, larangannya ditambahkan ke berkas ini — itu mekanisme belajarnya.
> Targetkan minimal 4 commit yang menyentuh berkas ini, tersebar di kedua hari.
> Commit-nya berupa `docs(agents): larang X setelah kejadian DEVLOG-05`, bukan satu commit
> besar di akhir.
>
> **3. Berkas ini adalah satu-satunya sumber aturan.** `CLAUDE.md`, `.cursorrules`, dan
> `.github/copilot-instructions.md` hanya menunjuk ke sini. Jangan menyalin isinya
> ke tempat lain — salinan akan langsung usang.
>
> **4. Pemilik berkas: Tech Lead.** Siapa pun boleh mengusulkan perubahan lewat PR,
> tetapi Tech Lead yang memutuskan.
>
> ### Cara membaca penanda di berkas ini
>
> | Penanda | Arti |
> |---|---|
> | `<!-- ISI: ... -->` | Placeholder. Wajib Anda ganti dengan isi nyata. |
> | Bagian 5 (Aturan Bisnis) | **Sudah pre-isi dari brief.** Jangan ubah nilai ambangnya; lengkapi saja kolom lokasi penegakan. |
> | Bagian 6 (Larangan) | Sudah pre-isi dengan larangan dasar. Tambahkan larangan Anda sendiri di bawahnya. |

**Riwayat perubahan berkas ini** (isi setiap kali berubah — ini bukti evolusi):

| Tanggal & jam | Oleh | Perubahan | Dipicu oleh |
|---|---|---|---|
| 2026-08-20 09:50 | Luthfi | Versi awal — kerangka dari template brief | Sprint 0 |
| 2026-08-20 10:20 | Luthfi | Isi bagian 2–7: stack (Go + Chi + GORM + golang-migrate, Next.js, Postgres 16), struktur direktori, konvensi, lokasi penegakan BR, perintah test/lint | Sprint 0 · ADR-0001 |
|  |  |  |  |
|  |  |  |  |

---

## 1. Konteks Proyek

**iMitra** adalah sistem originasi pembiayaan mikro syariah untuk Bank Syariah Nasional.
Alurnya: pengajuan oleh Account Officer → verifikasi dokumen → survei lapangan → SLIK check
→ skoring kelayakan → perhitungan margin/nisbah → approval berjenjang → audit trail.

Ini aplikasi perbankan. Konsekuensinya bagi agent:

- Aturan bisnis bukan saran. Angka ambang berasal dari brief dan dari tabel parameter di
  database, bukan dari asumsi model.
- Setiap perubahan status wajib punya aktor dan timestamp. Tidak ada mutasi diam-diam.
- Data pribadi (NIK, foto dokumen) tidak boleh keluar ke log, pesan error, atau URL.
- Otorisasi ditegakkan di server. Menyembunyikan tombol di UI bukan otorisasi.

**Aktor sistem** (kode peran ini dipakai persis seperti ini di kode, database, dan UI):

| Kode | Aktor | Wewenang |
|---|---|---|
| `AO` | Account Officer Mikro | Buat/ubah pengajuan miliknya, upload dokumen, rekam survei, lihat status |
| `ANL` | Analis Mikro | Verifikasi dokumen, SLIK check, skoring & override, hitung margin, ajukan ke approval |
| `KCP` | Kepala Cabang Pembantu | Approval level 1 |
| `KC` | Kepala Cabang | Approval level 2 |
| `KOM` | Komite Pembiayaan | Approval level 3 |
| `ADM` | Admin | Kelola pengguna, parameter skoring, ambang approval, rentang margin |

**Di luar lingkup — jangan dibangun, jangan disarankan**: disbursement, akuntansi, jadwal
angsuran aktual, penagihan, restrukturisasi, integrasi nyata ke Core Banking atau SLIK
produksi, aplikasi mobile native, SSO/Active Directory nyata, multi-tenant, multi-currency,
multi-bahasa. Kalau agent menawarkan salah satunya, tolak — itu scope creep dan dinilai
sebagai kesalahan prioritas.

**Dokumen rujukan yang wajib agent hormati** (lampirkan yang relevan saat memberi konteks):

- `docs/SRS-iMitra.md` — requirement
- `docs/SDD-iMitra.md` — arsitektur, model data, daftar endpoint
- `docs/adr/` — keputusan arsitektur yang sudah diambil. Agent tidak boleh mengusulkan
  hal yang bertentangan dengan ADR yang sudah `Accepted` tanpa ADR baru yang membatalkannya.

---

## 2. Stack & Versi

Semua versi di bawah **wajib sama** dengan tabel Keputusan di `docs/adr/0001-pilihan-stack.md`.
Kalau salah satu berbeda, salah satunya usang — perbaiki lewat PR, jangan biarkan dua sumber.

| Lapisan | Teknologi | Versi | Catatan |
|---|---|---|---|
| Bahasa backend | Go | 1.22.x | `go.mod` menetapkan `go 1.22`; jangan naikkan tanpa persetujuan Tech Lead |
| Framework backend | Chi (`github.com/go-chi/chi/v5`) | 5.0.x | Router tipis di atas `net/http`; middleware auth/peran ditulis sendiri (mudah dijelaskan di demo) |
| Bahasa/framework frontend | Next.js (App Router) + React + TypeScript | Next 14.2.x · React 18.3.x · TS 5.4.x | `app/` router, bukan `pages/`. Server Components untuk fetch, Client Components untuk form |
| Database | PostgreSQL | 16-alpine | Sama dengan image di `docker-compose.yml` |
| ORM / query layer | GORM (`gorm.io/gorm` + `gorm.io/driver/postgres`) | 1.25.x | Hanya untuk query/persistence. **`AutoMigrate` DILARANG** — skema hanya dari migrasi SQL |
| Tool migrasi | golang-migrate (`github.com/golang-migrate/migrate/v4`) | 4.17.x | Migrasi berupa berkas `NNNNNN_nama.up.sql` / `.down.sql`. Bukan `db.sql` restore manual (brief §7.2 butir 4) |
| Test runner | `go test` + testify (`github.com/stretchr/testify`) | testify 1.9.x | Backend & mock SLIK. Frontend: belum wajib test di P0 |
| Linter / formatter | golangci-lint + `gofmt` (backend); ESLint + Prettier (frontend) | golangci-lint 1.59.x · ESLint 8.x (config Next) | `gofmt -l` harus kosong; `golangci-lint run` bersih |
| Mock SLIK | Go (`net/http` stdlib) | Go 1.22.x | Satu toolchain dengan backend → lebih sedikit yang harus di-build penilai. Baca `fixtures/nasabah-uji.csv` saat start |
| Runtime | Docker Compose | v2 (compose spec, tanpa kunci `version`) | `docker compose up` dari clone bersih harus menghidupkan seluruh sistem |

**Batasan versi yang tidak boleh diubah agent**: jangan menaikkan versi Go, Next.js, GORM,
atau golang-migrate — migrasi, `go.mod`, dan `package.json` sudah dikunci ke versi ini.
Jangan mengganti golang-migrate dengan `gorm.AutoMigrate`; skema database **hanya** berasal
dari berkas migrasi SQL agar reproducible di mesin penilai. Penambahan dependensi baru
mengikuti Larangan #1 (persetujuan Tech Lead lebih dulu).

---

## 3. Struktur Direktori & Di Mana Kode Baru Diletakkan

Struktur ini mengikuti layout Go yang lazim (`cmd/` + `internal/`) dengan pemisahan lapisan
handler → service → repository. Agent **wajib** menaruh kode sesuai tabel di bawah, bukan
membuat direktori baru tiap kali.

```
/
├── backend/                     # API Go (Chi)
│   ├── cmd/
│   │   ├── api/                 # main.go — bootstrap server, wiring dependency
│   │   ├── migrate/             # runner golang-migrate (up/down)
│   │   └── seed/                # runner seed idempoten
│   ├── internal/
│   │   ├── domain/              # entitas + enum + error tipe (tanpa HTTP, tanpa SQL)
│   │   ├── service/             # ATURAN BISNIS: skoring, margin, routing approval, SLIK gate
│   │   ├── repository/          # akses DB (GORM). Satu-satunya lapisan yang menyentuh DB
│   │   ├── httpapi/             # handler/route + middleware (auth, peran, request-id)
│   │   ├── slik/                # HTTP client ke mock SLIK + penanganan error/timeout
│   │   └── config/              # pembacaan env (satu tempat, di-inject ke bawah)
│   ├── migrations/              # NNNNNN_*.up.sql / *.down.sql (golang-migrate)
│   ├── testdata/                # fixtures test backend
│   └── Dockerfile
├── frontend/                    # Next.js 14 (App Router, TS)
│   ├── app/                     # route per peran (mis. app/(ao)/pengajuan/...)
│   ├── components/              # komponen UI reusable
│   ├── lib/                     # api client, helper auth, tipe bersama
│   └── Dockerfile
├── mock-slik/                   # stub SLIK Go (net/http) + Dockerfile
├── docs/
├── fixtures/                    # nasabah-uji.csv (dibaca mock-slik saat start, read-only)
└── docker-compose.yml
```

**Aturan penempatan (agent wajib mengikuti ini, bukan menebak)**:

| Jenis kode | Lokasi | Jangan taruh di |
|---|---|---|
| Aturan bisnis / perhitungan (skoring, margin, routing approval) | `backend/internal/service/` | controller (`httpapi`), komponen UI, middleware |
| Endpoint / route handler | `backend/internal/httpapi/` (route + handler) | — |
| Akses database / repository | `backend/internal/repository/` | service, controller (`httpapi`) |
| Migrasi skema | `backend/migrations/` (`*.up.sql` / `*.down.sql`) | mana pun selain direktori migrasi; jangan pakai `AutoMigrate` |
| Seed data | `backend/cmd/seed/` (idempoten, aman dijalankan ulang) | migrasi, service |
| Test unit | berdampingan dengan kode: `*_test.go` di paket yang sama (mis. `internal/service/`) | direktori terpisah yang jauh dari kode |
| Test integrasi / API | `backend/internal/httpapi/*_test.go` atau `backend/test/` (paket `_test`) | test unit yang menyentuh DB nyata |
| Komponen UI | `frontend/components/` dan route di `frontend/app/` | logika aturan bisnis (semuanya di backend) |
| Pemanggil HTTP ke mock SLIK (client + penanganan error) | `backend/internal/slik/` | dipanggil langsung dari controller (`httpapi`) atau repository |
| Konfigurasi / pembacaan env | `backend/internal/config/` (satu struct, di-load sekali) | tersebar di seluruh kode (`os.Getenv` di mana-mana) |

**Aturan lapisan**: alur wajib `httpapi (handler) → service → repository → DB`. Handler tidak
boleh mengakses database langsung; selalu lewat service → repository. **Aturan bisnis
(`service`) tidak boleh tahu tentang HTTP** (tanpa `http.Request`/`http.ResponseWriter`) dan
tidak boleh membangun SQL sendiri. `slik` client dipanggil dari `service`, bukan dari handler.
`repository` tidak berisi aturan bisnis — hanya baca/tulis data. Enum dan tipe error hidup di
`domain` supaya semua lapisan memakai definisi yang sama.

---

## 4. Konvensi

### 4.1 Penamaan

Konsisten dan tertulis. Agent meniru apa pun yang ada di sini — jadi ikuti persis.

| Objek | Konvensi | Contoh |
|---|---|---|
| Tabel database | `snake_case`, jamak, istilah domain Bahasa Indonesia | `pengajuan`, `dokumen`, `survei`, `hasil_skoring`, `audit_trail` |
| Kolom database | `snake_case` | `nomor_referensi`, `total_plafon`, `created_at`, `created_by` |
| Kelas / tipe (Go) | `PascalCase` exported, satu entitas satu tipe | `Pengajuan`, `HasilSkoring`, `SlikResult`, `MarginRange` |
| Fungsi / method (Go) | `PascalCase` (exported) / `camelCase` (unexported), kata kerja | `HitungSkor`, `RouteApproval`, `validatePlafon` |
| Berkas (Go) | `snake_case.go`; test `*_test.go` | `skoring_service.go`, `skoring_service_test.go` |
| Berkas/komponen (frontend) | Komponen React `PascalCase.tsx`; util `camelCase.ts` | `PengajuanForm.tsx`, `apiClient.ts` |
| Endpoint | `/api/<sumberdaya-jamak>[/{id}[/<aksi>]]`, kebab-case, sumber daya Bahasa Indonesia | `POST /api/pengajuan`, `POST /api/pengajuan/{id}/skoring`, `POST /api/dokumen/{id}/verifikasi` |
| Enum status (kode & DB) | `SCREAMING_SNAKE_CASE`, nilai persis seperti daftar di bawah | `DRAFT`, `REJECTED_SLIK`, `WAITING_APPROVAL_L1` |
| Branch | `feat/FR-NN-slug`, `fix/FR-NN-slug` | `feat/FR-06-skoring`, `fix/FR-03-reupload` |

**Bahasa dalam kode**: istilah domain memakai **Bahasa Indonesia** (`pengajuan`, `survei`,
`plafon`, `nisbah`, `skoring`, `dokumen`, `anggota`, `approval`), sisanya (kata teknis umum,
kata kerja generik) **Bahasa Inggris** (`create`, `handler`, `repository`, `service`,
`request`, `response`). **Dilarang** memakai dua istilah untuk konsep yang sama — pilih
`pengajuan`, jangan campur dengan `application`/`loan`; pilih `survei`, jangan `survey`.
Sekali sebuah konsep dinamai, nama itu dipakai di DB, kode backend, endpoint, dan UI.

**Status pengajuan (enum wajib)**: nilai berikut berasal dari brief dan tidak boleh diganti
namanya: `DRAFT`, `REJECTED_SLIK`, `REJECTED_SCORING`, `APPROVED`. Status dokumen: `VERIFIED`,
`REJECTED`. Status survei: `VALID`. Keputusan approval: `APPROVE`, `REJECT`, `RETURN`.

Status transisi tambahan yang dipakai iMitra (wajib juga tercatat di `docs/SDD-iMitra.md`
sebelum dipakai di kode) — **agent tidak boleh menambah nilai baru di luar daftar ini tanpa
memperbarui bagian ini DAN SDD**:

- Alur pengajuan: `DRAFT` → `SUBMITTED` → `VERIFYING` → `SLIK_CHECKED` → `SCORED`
  → `WAITING_APPROVAL_L1` → `WAITING_APPROVAL_L2` → `WAITING_APPROVAL_L3` → `APPROVED`
- Cabang penolakan/kembali: `REJECTED_SLIK`, `REJECTED_SCORING`, `RETURNED` (dari `RETURN`
  approval, kembali ke AO), `REJECTED` (penolakan approval final).
- Status dokumen: `PENDING` → `VERIFIED` | `REJECTED`. Status survei: `DRAFT` → `VALID`.

Diagram transisi lengkap + guard-nya wajib ada di `docs/SDD-iMitra.md`. Menambah/mengubah
satu nilai enum = perbarui daftar ini, SDD, dan migrasi (bila enum di DB).

**Format nomor referensi pengajuan**: `IMT-YYYYMMDD-NNNN` (contoh: `IMT-20260820-0007`).
Unik dan tidak pernah dipakai ulang, termasuk untuk pengajuan yang ditolak (BR-12).
Agent tidak boleh mengubah format ini atau membangkitkannya di sisi frontend.

### 4.2 Commit

Conventional commits, dengan ID FR di dalam scope:

```
feat(FR-06): hitung skor kelayakan dari parameter tersimpan
fix(FR-03): pertahankan data pengajuan saat re-upload satu dokumen
test(FR-07): tambah kasus AC-09 margin di bawah batas grade 1
docs(agents): larang hardcode rentang margin setelah DEVLOG-07
chore(ci): sesuaikan workflow ke stack terpilih
```

Aturan tambahan:

- Setiap PR menyebut issue-nya: `Closes #12`.
- Satu issue = satu branch = satu PR.
- Agent tidak boleh membuat commit atas nama orang lain, dan tidak boleh melakukan
  `git push` ke `main`. `main` dilindungi.
- Kalau perubahan berasal dari sesi AI, PR wajib menyebut nomor entri devlog
  (`DEVLOG-xx`) di bagian AI pada template PR.

### 4.3 Error Handling

Semua error API memakai satu bentuk respons JSON yang sama, dibangun oleh helper terpusat di
`backend/internal/httpapi/` (mis. `respondError`). Handler tidak menyusun JSON error manual.

Bentuk respons error yang seragam untuk seluruh API:

```json
{
  "error": "KODE_KONSTAN",
  "message": "pesan singkat untuk pengguna, tanpa data pribadi",
  "rule": "BR-03"
}
```

- `error`: kode konstan `SCREAMING_SNAKE_CASE` (mis. `FORBIDDEN`, `VALIDATION_ERROR`,
  `BUSINESS_RULE_VIOLATION`, `SLIK_UNAVAILABLE`, `NOT_FOUND`), stabil dan bisa dicek di test.
- `message`: kalimat untuk pengguna, **tanpa NIK / nomor dokumen / path foto** (BR-11).
- `rule`: opsional, diisi kode BR saat error berasal dari pelanggaran aturan bisnis.

| Situasi | Kode HTTP | Catatan |
|---|---|---|
| Belum login / token tidak valid | 401 | |
| Login tetapi peran tidak berwenang | **403** | AC-02 menguji ini secara langsung. Bukan 200, bukan 404 |
| Validasi input gagal | 400 | Sebutkan field yang salah di `message` (tanpa data pribadi) |
| Pelanggaran aturan bisnis (BR-xx) | **422** | Pilihan tetap untuk seluruh repo. `message` wajib menyebut kode BR; isi juga field `rule` |
| Sumber daya tidak ada | 404 | |
| Mock SLIK tidak tersedia / timeout / 503 | **502** | Backend gagal memakai dependensi hulu. **Tidak boleh** dianggap SLIK bersih; pengajuan tidak lanjut |
| Galat tak terduga | 500 | Tanpa stack trace ke klien |

Aturan yang tidak boleh dilanggar agent:

- **Jangan menelan exception.** `catch` yang kosong atau yang hanya mencatat log lalu
  melanjutkan alur normal dilarang, khususnya di jalur SLIK.
- **Jangan pakai nilai default diam-diam.** Kalau SLIK gagal, jangan mengisi
  kolektibilitas dengan `1` atau `null` lalu melanjutkan. Pengajuan berhenti dan
  statusnya mencerminkan kegagalan itu.
- **Jangan menulis NIK, nomor dokumen, atau path foto ke log dan pesan error** (BR-11).
  Pakai id internal pengajuan untuk korelasi.
- Pesan error yang berasal dari pelanggaran aturan bisnis wajib menyebut kode BR-nya,
  karena AC-04 secara eksplisit meminta pesan yang menyebut BR-03.

### 4.4 Contoh Baik vs Buruk (standar untuk seluruh repo)

Contoh ini memakai pseudocode agar berlaku untuk stack apa pun. Yang dinilai bukan
sintaksnya, tetapi di mana keputusan diambil dan dari mana angkanya datang.

**BURUK — jangan terima keluaran agent seperti ini:**

```
fungsi hitungMargin(plafon, tenor, grade):
    # rentang ditulis langsung di kode
    rentang = { 1: [11.0, 13.0], 2: [13.0, 15.5], 3: [15.5, 18.0] }
    margin = ...
    jika margin di luar rentang[grade]:
        catatLog("margin di luar rentang untuk NIK " + nasabah.nik)   # membocorkan NIK
        kembalikan { ok: true, peringatan: "margin di luar rentang" }  # tetap lanjut
```

Empat pelanggaran sekaligus:
1. Rentang margin di-hardcode, padahal wajib berupa data yang bisa diubah ADM (FR-13, BR-06).
2. NIK masuk ke log (BR-11).
3. Pelanggaran aturan hanya jadi peringatan, padahal wajib memblokir (BR-06).
4. Tidak ada grade 4 dan 5, sehingga grade 5 lolos padahal harus ditolak (BR-05).

**BAIK — standar yang kita pakai:**

```
fungsi hitungMargin(pengajuan, grade):
    jika grade == 5:
        lempar PelanggaranAturan("BR-05", "grade 5 tidak dapat diajukan ke approval")

    rentang = repositoriParameter.ambilRentangMargin(grade, akad)   # dari database
    jika rentang tidak ada:
        lempar KesalahanKonfigurasi("rentang margin grade " + grade + " belum diatur")

    margin = ...
    jika margin < rentang.min atau margin > rentang.maks:
        lempar PelanggaranAturan("BR-06",
            "margin " + margin + "% di luar rentang grade " + grade)   # tanpa data pribadi

    kembalikan margin
```

Dan test-nya ditulis dari AC-09, bukan dari kode di atas — termasuk satu test yang
**mengubah baris rentang di database lebih dulu** lalu memastikan hasilnya berubah.
Test yang hanya memanggil fungsi dengan nilai tetap tidak membuktikan bahwa parameter
benar-benar dibaca dari data.

---

## 5. Aturan Bisnis yang Tidak Boleh Dilanggar

> **Bagian ini sudah pre-isi dari brief §4 dan tidak boleh diubah nilainya.** Ia ada di sini
> supaya bisa dilampirkan ke agent sebagai satu blok. Yang Anda lengkapi hanya kolom
> **"Ditegakkan di"** — path berkas tempat aturan itu benar-benar hidup. Kolom itu sekaligus
> berfungsi sebagai deteksi dini: BR tanpa lokasi penegakan berarti aturan itu belum ada
> di kode mana pun.

| BR | Aturan (ringkas) | Ditegakkan di |
|---|---|---|
| **BR-01** | Plafon < Rp 5.000.000 atau > Rp 500.000.000 ditolak saat submit, dengan pesan yang menjelaskan batas | `backend/internal/service/pengajuan_service.go` (submit) |
| **BR-02** | Approval harus berurutan: level 2 tidak dapat memutuskan sebelum level 1 memberi `APPROVE` | `backend/internal/service/approval_service.go` |
| **BR-03** | Skoring baru boleh jalan jika semua dokumen wajib `VERIFIED` **dan** ada minimal satu survei `VALID` **dan** SLIK check sudah dijalankan | `backend/internal/service/skoring_service.go` (guard sebelum hitung) |
| **BR-04** | Hasil SLIK berlaku 30 hari; lewat itu pengajuan ditandai perlu SLIK ulang | `backend/internal/service/slik_service.go` |
| **BR-05** | Grade 5 tidak dapat diajukan ke approval; status menjadi `REJECTED_SCORING` | `backend/internal/service/skoring_service.go`, `approval_service.go` |
| **BR-06** | Margin/nisbah di luar rentang grade-nya **diblokir**, bukan diberi peringatan. Tidak ada jalur "lanjutkan saja" | `backend/internal/service/margin_service.go` |
| **BR-07** | Skor akhir = Σ (skor komponen × bobot) ÷ Σ bobot, dibulatkan ke bilangan bulat terdekat | `backend/internal/service/skoring_service.go` |
| **BR-08** | Rincian tiap komponen skor wajib ditampilkan ke ANL **dan disimpan** bersama hasil skoring, bukan hanya angka akhir | `backend/internal/service/skoring_service.go` + tabel `komponen_skor` (repository) |
| **BR-09** | Satu pengguna tidak boleh menjadi maker dan approver pada pengajuan yang sama; ditegakkan di **server** | `backend/internal/service/approval_service.go` (cek `created_by` ≠ approver) |
| **BR-10** | Setiap perubahan status wajib punya aktor dan timestamp; tidak ada perubahan "oleh sistem" tanpa jejak sebab | `backend/internal/service/audit_service.go` (dipanggil setiap transisi) |
| **BR-11** | NIK dan foto dokumen adalah data pribadi: tidak boleh muncul di log aplikasi, pesan error, atau URL | Lintas lapisan: helper log di `internal/config`/`httpapi`, review di setiap PR |
| **BR-12** | Nomor referensi `IMT-YYYYMMDD-NNNN` unik dan tidak pernah dipakai ulang, termasuk untuk pengajuan yang ditolak | `backend/internal/service/pengajuan_service.go` + constraint `UNIQUE` di migrasi + tabel/sequence `nomor_referensi` |

### 5.1 Tabel parameter — wajib sebagai data, bukan konstanta

Ketiga tabel berikut **wajib tersimpan di database** dan bisa diubah ADM tanpa deploy ulang
(FR-13, AC-15). Agent dilarang menuliskan angka-angka ini sebagai konstanta di dalam kode,
termasuk sebagai nilai default, termasuk di dalam test.

**Ambang approval** (dinilai dari **total plafon**; untuk kelompok/majelis: total plafon kelompok):

| Total plafon | Level | Jenis |
|---|---|---|
| Rp 5.000.000 – Rp 50.000.000 | KCP | Tunggal |
| > Rp 50.000.000 – Rp 200.000.000 | KCP → KC | Berjenjang 2 |
| > Rp 200.000.000 – Rp 500.000.000 | KCP → KC → KOM | Berjenjang 3 |

**Keluaran kolektibilitas SLIK:**

| Kolektibilitas | Keluaran sistem |
|---|---|
| 1 | Lanjut normal |
| 2 | Lanjut, **tetapi grade risiko minimal 3** dan wajib catatan analis |
| 3, 4, 5 | **Penolakan otomatis**, status `REJECTED_SLIK`, tanpa melalui approval |

**Rentang margin / nisbah per grade:**

| Grade | Rentang skor | Margin murabahah (p.a.) | Nisbah bank musyarakah |
|---|---|---|---|
| 1 — Sangat baik | 85–100 | 11,0 % – 13,0 % | 20 % – 25 % |
| 2 — Baik | 70–84 | 13,0 % – 15,5 % | 25 % – 30 % |
| 3 — Cukup | 55–69 | 15,5 % – 18,0 % | 30 % – 35 % |
| 4 — Perlu perhatian | 40–54 | 18,0 % – 21,0 % | 35 % – 40 % |
| 5 — Berisiko tinggi | < 40 | Tidak dibiayai | Tidak dibiayai |

**Komponen skor kelayakan** (bobot wajib bisa diubah ADM):

| Komponen | Bobot | Cara hitung |
|---|---|---|
| Kapasitas bayar | 35 | Rasio angsuran bulanan terhadap (omzet harian × 25 hari × margin usaha 30 %). ≤ 30 % → skor penuh; > 60 % → skor 0; linear di antaranya |
| Riwayat SLIK | 25 | Kol-1 → 100; Kol-2 → 40; Kol-3-5 → tidak sampai tahap ini |
| Lama usaha | 20 | ≥ 36 bulan → 100; < 6 bulan → 0; linear di antaranya |
| Hasil survei lapangan | 20 | Penilaian ANL atas kondisi usaha, skala 1–5, dikali 20 |

**Nama tabel parameter di database Anda**:

| Isi | Nama tabel |
|---|---|
| Bobot & aturan komponen skor | `parameter_skoring` |
| Ambang approval per plafon | `ambang_approval` |
| Rentang margin/nisbah per grade | `rentang_margin` |

Ketiga tabel ini di-*seed* dari `backend/cmd/seed/` (bukan hardcode di service), dan wajib
punya endpoint CRUD ADM (FR-13). Service membaca nilainya lewat repository setiap kali
menghitung — bukan meng-cache di konstanta. Test aturan bisnis (skoring/margin/approval)
wajib **mengubah baris tabel ini lebih dulu** lalu memverifikasi hasilnya berubah (AC-15),
bukan menyalin angka brief ke dalam test.

### 5.2 Kontrak mock SLIK (tidak boleh diubah agent)

```
POST /slik/inquiry
Request  : { "nik": "3404xxxxxxxxxxxx" }
200      : { "nik", "nama", "kolektibilitas", "jumlahFasilitasAktif",
             "totalBakiDebet", "tanggalData", "referenceId" }
404      : { "error": "NIK_NOT_FOUND" }
503      : { "error": "SERVICE_UNAVAILABLE" }
```

Dipanggil **via HTTP**, bukan sebagai fungsi lokal. Wajib menangani timeout, 503, dan 404.
Mock harus bisa dipaksa mengembalikan 503 supaya jalur error bisa didemokan — data uji
di `fixtures/nasabah-uji.csv` sudah menyediakan NIK pemicunya.

---

## 6. Larangan Eksplisit untuk Agent

> Pre-isi di bawah adalah dasar. **Tambahkan larangan baru setiap kali agent melakukan
> kesalahan yang sama dua kali** — itu inti dari berkas ini. Rujuk nomor DEVLOG-nya
> supaya jelas larangan ini datang dari pengalaman, bukan dari salinan template.

Agent **tidak boleh**:

1. Menambah dependensi/pustaka baru tanpa persetujuan Tech Lead. Kalau perlu, usulkan
   dulu beserta alasan dan alternatifnya; jangan langsung ubah manifest paket.
2. Mengubah atau menghapus migrasi yang sudah di-merge ke `main`. Perubahan skema
   selalu berupa migrasi baru.
3. Menuliskan angka ambang, bobot, atau rentang dari bagian 5 sebagai konstanta di kode
   (termasuk sebagai nilai default dan di dalam test).
4. Mengubah format nomor referensi `IMT-YYYYMMDD-NNNN`, atau membangkitkannya di frontend.
5. Menambah nilai enum status baru tanpa memperbarui bagian 4.1 dan `docs/SDD-iMitra.md`.
6. Melakukan otorisasi hanya di frontend. Setiap endpoint memeriksa peran di server.
7. Menghapus atau melemahkan test yang gagal supaya CI hijau. Test yang gagal berarti
   kode atau requirement yang salah, bukan test-nya.
8. Membuat endpoint yang bisa `UPDATE` atau `DELETE` baris audit trail. Audit trail
   append-only (FR-09, AC-13).
9. Menulis NIK, nomor dokumen, atau path foto ke log, pesan error, atau URL (BR-11).
10. Membuat berkas `.env`, menaruh nilai secret nyata di berkas apa pun, atau menulis
    kredensial di kode. Hanya `.env.example` dengan nilai placeholder.
11. Melakukan `git push` ke `main`, `git push --force`, atau merge PR-nya sendiri.
12. Membangun apa pun dari daftar "di luar lingkup" di bagian 1, walaupun terasa mudah.
13. Menghasilkan lebih dari ~200 baris kode dalam satu keluaran. Kalau tugasnya besar,
    ajukan rencana bertahap lebih dulu, tunggu persetujuan, baru tulis kode.
14. Mengubah `docker-compose.yml`, `ci.yml`, atau `AGENTS.md` sebagai efek samping dari
    tugas fitur. Ketiganya diubah lewat PR terpisah.
15. Menganggap kegagalan SLIK sebagai SLIK bersih, atau mengisi kolektibilitas dengan
    nilai default saat panggilan gagal.

<!-- ISI: larangan tambahan dari pengalaman tim. Format:
     16. <larangan> — ditambahkan setelah DEVLOG-xx, karena <apa yang terjadi>. -->

16. Memakai `gorm.AutoMigrate` atau membuat/mengubah skema dari kode Go. Skema **hanya** dari
    berkas migrasi SQL di `backend/migrations/` (golang-migrate). GORM hanya untuk query.
17. Menaruh aturan bisnis (skoring, margin, routing approval, guard BR) di `httpapi` (handler),
    middleware, `repository`, atau komponen frontend. Semuanya di `backend/internal/service/`.

---

## 7. Perintah Test & Lint

Perintah di bawah wajib **identik** dengan `.github/workflows/ci.yml` dan `README.md` bagian 2.6.
Kalau agent diminta "jalankan test", inilah yang ia jalankan. Backend Go dijalankan dari
`backend/`, frontend dari `frontend/`. Untuk lingkungan lengkap (DB + mock SLIK) pakai
`docker compose`.

```bash
# ---- Backend (Go) — dijalankan dari ./backend ----

# Instalasi dependensi
go mod download

# Migrasi (lingkungan test) — golang-migrate via runner cmd/migrate
go run ./cmd/migrate up            # membaca DATABASE_URL_TEST saat APP_ENV=test

# Seed data uji (idempoten)
go run ./cmd/seed

# Test unit (aturan bisnis: skoring, margin, approval)
go test ./internal/service/... -count=1

# Test integrasi / API (butuh DB test + mock SLIK aktif)
go test ./internal/httpapi/... ./test/... -count=1

# Semua test backend
go test ./... -count=1

# Lint
golangci-lint run ./...

# Format (harus tidak menghasilkan output; kalau ada, format belum bersih)
gofmt -l .

# ---- Frontend (Next.js) — dijalankan dari ./frontend ----
npm ci
npm run lint
npm run build

# ---- Semua sekaligus, sama seperti yang dijalankan CI ----
# Cara paling andal untuk mereproduksi CI di mesin bersih:
docker compose up -d db mock-slik
# (backend) dari ./backend:
APP_ENV=test go run ./cmd/migrate up && go run ./cmd/seed && go test ./... -count=1 && golangci-lint run ./... && test -z "$(gofmt -l .)"
# (frontend) dari ./frontend:
npm ci && npm run lint && npm run build
```

**Aturan Definition of Done untuk agent**: perubahan dianggap selesai hanya jika lint bersih,
seluruh test lolos, dan ada minimal satu test yang berasal dari AC terkait — bukan test yang
diturunkan dari kode yang baru saja ditulis.

**Sebelum membuka PR**, pastikan:

- Test dan lint lolos secara lokal, bukan hanya "seharusnya lolos".
- `docs/TRACEABILITY.md` diperbarui untuk FR yang disentuh.
- Ada entri `docs/AI-DEVLOG.md` kalau AI dipakai, dan nomornya disebut di deskripsi PR.
- Tabel status FR di `README.md` diperbarui kalau statusnya berubah.
