# TRACEABILITY — FR → AC → Endpoint → Test → PR

**Tim**: iMitra Tim 1
**Terakhir diperbarui**: 2026-08-20 15.30 (Gate 2) — kolom BR & Ringkasan Risiko diverifikasi QA terhadap kode nyata

---

## Cara memakai tabel ini sebagai alat deteksi risiko

Tabel ini **alat kerja Anda sendiri**, bukan formalitas untuk penilai (brief §8.4). Ia dipakai
seperti ini:

1. **Baris tanpa "File test" adalah risiko, bukan kekurangan administrasi.** FR yang jalan
   tetapi tidak punya test berarti Anda tidak tahu ia masih jalan setelah PR berikutnya.
   Pada Jumat pagi, baris-baris inilah yang paling mungkin gagal saat demo.
2. **Baris tanpa "Endpoint" pada FR P0 berarti fitur itu belum ada**, seberapa pun ramai
   diskusinya di grup. Isi kolom ini dari daftar route yang benar-benar terdaftar, bukan dari
   rencana.
3. **AC tanpa test otomatis wajib punya baris di `DEMO-SCRIPT.md`.** Kalau tidak ada di
   keduanya, AC itu tidak diverifikasi oleh siapa pun.
4. **Isi selama bekerja, bukan di akhir.** Cara termurah: setiap kali membuka PR, perbarui
   baris FR yang disentuh — itu bagian dari checklist PR.
5. **Pakai ini di Gate 3 (Jumat 11.20)** untuk memutuskan FR mana yang dibuang. Keputusan
   membuang jauh lebih mudah kalau Anda bisa melihat mana yang punya test dan mana yang tidak.

**Cara memakainya 10 menit sebelum demo**: urutkan mental sebagai berikut — P0 tanpa test,
lalu P0 dengan test tetapi status belum Done, lalu P1. Latih demo untuk baris teratas
lebih dulu. Penilai akan meminta AC secara acak, termasuk jalur error.

**Nilai kolom Status yang diizinkan**: `Belum` · `In Progress` · `Done` · `Done (tanpa test)`
· `Dibuang`. Pada tag `v1.0.0` tidak boleh ada lagi `In Progress` — ubah menjadi `Dibuang`
atau `Done (tanpa test)` dan jelaskan di `README.md` bagian 5.

---

## Tabel Traceability

<!-- ISI: kolom Endpoint, File test, PR, dan Status. Kolom FR, Judul, Prioritas, dan
     "AC terkait" sudah pre-isi dari brief §3 dan §5 — jangan diubah, penilai mencocokkannya.
     Kolom Endpoint: daftar endpoint nyata, mis. POST /api/pengajuan, GET /api/pengajuan/{id}.
     Kolom File test: path berkas test, mis. backend/tests/skoring.test.ts.
     Kolom PR: nomor PR, mis. #14. -->

| FR | Judul | Prioritas | AC terkait | Endpoint | File test | PR | Status |
|---|---|---|---|---|---|---|---|
| FR-01 | Autentikasi & Otorisasi Berbasis Peran | P0 | AC-01, AC-02 |  |  |  |  |
| FR-02 | Pengajuan Pembiayaan Mikro | P0 | AC-01 |  |  |  |  |
| FR-03 | Upload & Verifikasi Dokumen | P0 | AC-03 |  |  |  |  |
| FR-04 | Survei Lapangan (OTS) | P0 | AC-04 |  |  |  |  |
| FR-05 | SLIK Check | P0 | AC-05, AC-06 |  |  |  |  |
| FR-06 | Skoring Kelayakan Mikro | P0 | AC-06, AC-07, AC-08 | *(belum; service siap, handler milik FR-06 UI)* | `backend/internal/service/skoring_service_test.go` | #5 | In Progress |
| FR-07 | Perhitungan Margin / Nisbah | P0 | AC-09 | *(belum; service siap)* | `backend/internal/service/margin_service_test.go` | #5 | In Progress |
| FR-08 | Approval Berjenjang | P0 | AC-10, AC-11 | `POST /api/pengajuan/{id}/ajukan-approval`, `POST /api/pengajuan/{id}/approval`, `GET /api/pengajuan/{id}/approval` | `internal/service/approval_service_test.go`, `internal/httpapi/approval_http_test.go` | #6 | Done |
| FR-09 | Audit Trail | P0 | AC-08, AC-12, AC-13 | `GET /api/pengajuan/{id}/audit`, `GET /api/audit` *(append-only, tanpa PUT/PATCH/DELETE)* | `internal/service/audit_service_test.go`, `internal/httpapi/audit_http_test.go` | #6 | Done |
| FR-10 | Pembiayaan Kelompok (Majelis) | P1 | AC-14 |  |  |  |  |
| FR-11 | Notifikasi Perubahan Status | P1 | — |  |  |  |  |
| FR-12 | Dashboard Pipeline | P1 | — |  |  |  |  |
| FR-13 | Parameter Terkonfigurasi | P1 | AC-15 | *(CRUD ADM belum; tabel & migrasi siap)* | `skoring_service_test.go` (`TestHitung_AC15_UbahBobotLangsungBerlaku`), `margin_service_test.go` (`TestValidasi_AC15_UbahRentangLangsungBerlaku`) | #5 | In Progress |
| FR-14 | Simulasi angsuran murabahah & proyeksi bagi hasil musyarakah | P2 | — |  |  |  |  |
| FR-15 | Ekspor daftar pengajuan ke CSV | P2 | — |  |  |  |  |
| FR-16 | Mode draft offline untuk AO di lapangan | P2 | — |  |  |  |  |
| FR-17 | Deteksi lokasi palsu (mock location) pada survei lapangan | P2 | — |  |  |  |  |
| FR-18 | Laporan Turn-Around Time per tahap dan per petugas | P2 | — |  |  |  |  |

FR-11 dan FR-12 tidak dirujuk langsung oleh AC mana pun. Itu bukan berarti keduanya tidak
diverifikasi — **tetapkan kriteria verifikasi Anda sendiri** untuk keduanya dan tulis di
`DEMO-SCRIPT.md`, karena penilai tetap akan melihatnya saat demo alur utama.

---

## Penelusuran Aturan Bisnis

<!-- Kolom "Ditegakkan di" dan "Test" diisi dari kode nyata, bukan dari rencana.
     Aturan pengisian yang kami sepakati (QA): sebuah baris hanya boleh berstatus "Done"
     kalau (a) berkas penegaknya benar-benar ada, DAN (b) ada nama fungsi test yang
     menguji aturan itu — bukan sekadar test yang kebetulan menyentuh berkas yang sama.
     Kalau salah satu tidak terpenuhi, tulis *(belum)* beserta alasannya. BR tanpa test
     adalah risiko terbesar di sistem perbankan, karena pelanggarannya tidak terlihat
     di jalur bahagia. -->

| BR | Ringkasan | Ditegakkan di | Test | Status |
|---|---|---|---|---|
| BR-01 | Plafon di luar Rp 5 juta – Rp 500 juta ditolak saat submit | *(belum)* — `pengajuan_service.go` **belum ada**; `approval_service.go` hanya mencocokkan plafon ke `ambang_approval`, bukan menegakkan batas BR-01 | *(belum)* — `approval_service_test.go` hanya memakai plafon 30jt/120jt/300jt (semua di dalam batas), tidak ada kasus 4jt / 600jt | **Belum** |
| BR-02 | Approval berurutan; level 2 menunggu `APPROVE` level 1 | `backend/internal/service/approval_service.go` | `internal/service/approval_service_test.go` (`TestApproval_AC10_RoutingBerjenjangDanUrutan`), `internal/httpapi/approval_http_test.go` | Done |
| BR-03 | Skoring butuh dokumen `VERIFIED` + survei `VALID` + SLIK sudah dijalankan | `backend/internal/service/skoring_service.go` | `internal/service/skoring_service_test.go` (`TestPastikanBolehSkoring_BR03`) | Done |
| BR-04 | Hasil SLIK berlaku 30 hari | *(belum)* — rencana `service/slik_service.go`, berkasnya belum ada |  | Belum |
| BR-05 | Grade 5 tidak dapat diajukan; `REJECTED_SCORING` | `backend/internal/service/approval_service.go`, `skoring_service.go` | `internal/service/approval_service_test.go` (`TestApproval_BR05_Grade5Ditolak`) | Done |
| BR-06 | Margin/nisbah di luar rentang grade diblokir | `backend/internal/service/margin_service.go` | `internal/service/margin_service_test.go` (`TestValidasi_AC09_MarginDiBawahBatasGrade1Diblokir`) | Done — tepi batas nisbah musyarakah (20,0 / 25,0 / 25,01) belum diuji |
| BR-07 | Skor akhir = Σ(skor × bobot) ÷ Σbobot, dibulatkan sekali di akhir | `backend/internal/service/skoring_service.go` | `internal/service/skoring_service_test.go` (`TestHitung_BR07_RumusSkorAkhir`) | Done |
| BR-08 | Rincian komponen ditampilkan dan disimpan | `backend/internal/service/skoring_service.go` (rincian dibangun) — **persistensi belum ada**: tabel `komponen_skor` tidak ada di `backend/migrations/` | `internal/service/skoring_service_test.go` (`TestHitung_AC07_...`) — hanya menguji rincian di memori, bukan tersimpan | **Sebagian** |
| BR-09 | Maker tidak boleh menjadi approver; ditegakkan di server | `backend/internal/service/approval_service.go` | `internal/service/approval_service_test.go` (`TestApproval_AC11_MakerChecker_BR09`), `internal/httpapi/approval_http_test.go` | Done |
| BR-10 | Setiap perubahan status punya aktor + timestamp | `backend/internal/service/audit_service.go`, `approval_service.go` | `internal/service/audit_service_test.go` (`TestAudit_AC12_RiwayatLengkapUrutWaktu`) | Done (jalur approval) — transisi FR-02…FR-05 belum ada, jadi belum semua transisi tercakup |
| BR-11 | NIK & foto dokumen tidak muncul di log, pesan error, atau URL | *(belum)* — helper log terpusat belum ada; saat ini hanya bergantung pada review PR | *(belum)* — `audit_service_test.go` tidak memuat satu pun assertion tentang NIK | **Belum** — pelanggarannya tidak terlihat di jalur bahagia |
| BR-12 | Nomor referensi `IMT-YYYYMMDD-NNNN` unik, tidak dipakai ulang | Constraint `UNIQUE` sudah ada (`migrations/000003…up.sql:6`); **pembangkit nomornya belum ada** (`pengajuan_service.go` belum ada) | *(belum)* — constraint DB bukan test; format & anti-reuse belum diuji | **Sebagian** |

---

## Ringkasan Risiko (perbarui di setiap gate)

<!-- Dihitung dari kedua tabel di atas pada setiap gate. Angka diambil dari status yang
     sudah diverifikasi terhadap kode, bukan dari perkiraan. -->

| Pertanyaan | Kamis 15.30 (Gate 2) | Jumat 11.20 (Gate 3) | Jumat 15.00 (code freeze) |
|---|---|---|---|
| Berapa FR P0 berstatus Done? | **2 dari 9** — FR-08, FR-09. FR-06 & FR-07 `In Progress` (service + test siap, endpoint belum). FR-01…FR-05 belum mulai |  |  |
| Berapa FR P0 tanpa file test? | **5 dari 9** — FR-01, FR-02, FR-03, FR-04, FR-05. Kelimanya juga belum ada kodenya, jadi ini risiko jadwal, bukan sekadar kekurangan test |  |  |
| Berapa BR tanpa test? | **4 dari 12** — BR-01, BR-04, BR-11, BR-12. Ditambah 2 `Sebagian`: BR-08 (rincian belum tersimpan), BR-10 (baru jalur approval) |  |  |
| Berapa AC yang sudah pernah dilatih di demo? | **0 dari 15** — kolom "Status latihan" di `DEMO-SCRIPT.md` masih kosong seluruhnya; jalur error E-1…E-5 belum satu pun disiapkan |  |  |
| Risiko terbesar saat ini | **Alur inti pengajuan belum ada.** FR-02 (pengajuan) dan FR-05 (SLIK) belum mulai, padahal FR-06/07/08 di hilirnya sudah selesai — artinya belum ada satu pun pengajuan yang bisa berjalan `DRAFT` → `APPROVED` untuk didemokan (AC-01, AC-12). Risiko kedua: **BR-11 tanpa penegakan otomatis** — NIK bisa lolos ke log tanpa ada yang menangkapnya, dan `mock-slik/` belum ada sehingga E-1/E-2 (yang brief §13 nyatakan pasti diuji penilai) belum bisa dilatih |  |  |
