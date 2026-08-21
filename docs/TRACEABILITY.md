# TRACEABILITY — FR → AC → Endpoint → Test → PR

**Tim**: iMitra Tim 1
**Terakhir diperbarui**: 2026-08-20 13:20

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
| FR-02 | Pengajuan Pembiayaan Mikro | P0 | AC-01 | `POST /api/pengajuan`, `GET /api/pengajuan`, `GET /api/pengajuan/{id}` | `internal/service/pengajuan_service_test.go`, `internal/httpapi/pengajuan_http_test.go` (`TestRoute_FR02_...`) | #11 | Done |
| FR-03 | Upload & Verifikasi Dokumen | P0 | AC-03 | `POST /api/pengajuan/{id}/dokumen`, `GET /api/pengajuan/{id}/dokumen`, `PATCH /api/pengajuan/{id}/dokumen/{dokId}/verifikasi` | `internal/service/dokumen_service_test.go`, `internal/httpapi/pengajuan_http_test.go` (`TestRoute_FR03_...`), `internal/httpapi/dokumen_reupload_http_test.go` (`TestRoute_FR03_AC03_ReuploadDokumenDitolakTidakMenyentuhDokumenLain`, `TestRoute_FR03_AC03_DokumenVerifiedTidakBisaDiunggahUlang`) | #11 | Done |
| FR-04 | Survei Lapangan (OTS) | P0 | AC-04 | `POST /api/pengajuan/{id}/survei` | `internal/service/survei_service_test.go`, `internal/httpapi/pengajuan_http_test.go` (`TestRoute_FR04_SurveiHanyaAODanWajibLengkap`) | #11 | Done |
| FR-05 | SLIK Check | P0 | AC-05, AC-06 | `POST /api/pengajuan/{id}/slik`, `POST /api/pengajuan/{id}/slik-check` | `internal/service/slik_service_test.go` (`TestAC05_...`, `TestAC06_...`, `TestBR04_...`), `internal/httpapi/slik_http_test.go` (`TestRoute_FR05_...`) | #11 | Done |
| FR-06 | Skoring Kelayakan Mikro | P0 | AC-06, AC-07, AC-08 | `POST /api/pengajuan/{id}/skoring`, `PATCH /api/pengajuan/{id}/skoring/override` | `internal/service/skoring_service_test.go`, `internal/httpapi/skoring_http_test.go` (`TestHTTP_AC07_...`, `TestHTTP_AC06_...`, `TestHTTP_AC04_...`), `internal/httpapi/skoring_override_http_test.go` (`TestHTTP_AC08_OverrideGrade2Ke3TercatatDiAuditTrail`) | #5 | Done |
| FR-07 | Perhitungan Margin / Nisbah | P0 | AC-09 | `POST /api/pengajuan/{id}/margin`, `GET /api/pengajuan/{id}/margin` | `internal/service/margin_service_test.go`, `internal/httpapi/skoring_http_test.go` (`TestHTTP_AC09_MarginDiBawahBatasGrade1Diblokir`) | #5 | Done |
| FR-08 | Approval Berjenjang | P0 | AC-10, AC-11 | `POST /api/pengajuan/{id}/ajukan-approval`, `POST /api/pengajuan/{id}/approval`, `GET /api/pengajuan/{id}/approval` | `internal/service/approval_service_test.go`, `internal/httpapi/approval_http_test.go` | #6 | Done |
| FR-09 | Audit Trail | P0 | AC-08, AC-12, AC-13 | `GET /api/pengajuan/{id}/audit`, `GET /api/audit` *(append-only, tanpa PUT/PATCH/DELETE)* | `internal/service/audit_service_test.go`, `internal/httpapi/audit_http_test.go` | #6 | Done |
| FR-10 | Pembiayaan Kelompok (Majelis) | P1 | AC-14 |  |  |  |  |
| FR-11 | Notifikasi Perubahan Status | P1 | — |  |  |  |  |
| FR-12 | Dashboard Pipeline | P1 | — |  |  |  |  |
| FR-13 | Parameter Terkonfigurasi | P1 | AC-15 | *(CRUD ADM belum; nilainya dibaca live oleh endpoint skoring & margin)* | `skoring_service_test.go`, `margin_service_test.go`, `internal/httpapi/skoring_http_test.go` (`TestHTTP_AC15_UbahBobotLangsungBerlakuTanpaRestart`, `TestHTTP_AC15_UbahRentangMarginLangsungBerlaku`) | #5 | In Progress |
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

<!-- ISI: kolom "Ditegakkan di" dan "Test". BR tanpa test adalah risiko terbesar di sistem
     perbankan, karena pelanggarannya tidak terlihat di jalur bahagia.
     Kolom "Ditegakkan di" harus sama dengan yang tertulis di AGENTS.md bagian 5. -->

| BR | Ringkasan | Ditegakkan di | Test | Status |
|---|---|---|---|---|
| BR-01 | Plafon di luar Rp 5 juta – Rp 500 juta ditolak saat submit | `backend/internal/service/pengajuan_service.go`, `approval_service.go` | `internal/service/approval_service_test.go` | Done |
| BR-02 | Approval berurutan; level 2 menunggu `APPROVE` level 1 | `backend/internal/service/approval_service.go` | `internal/service/approval_service_test.go` (`TestApproval_AC10_RoutingBerjenjangDanUrutan`), `internal/httpapi/approval_http_test.go` | Done |
| BR-03 | Skoring butuh dokumen `VERIFIED` + survei `VALID` + SLIK sudah dijalankan | `backend/internal/service/skoring_service.go` | `internal/service/skoring_service_test.go` (`TestPastikanBolehSkoring_BR03`) | Done |
| BR-04 | Hasil SLIK berlaku 30 hari | `backend/internal/service/slik_service.go` | `internal/service/slik_service_test.go` (`TestBR04_MasaBerlaku30Hari`) | Done |
| BR-05 | Grade 5 tidak dapat diajukan; `REJECTED_SCORING` | `backend/internal/service/approval_service.go`, `skoring_service.go` | `internal/service/approval_service_test.go` (`TestApproval_BR05_Grade5Ditolak`) | Done |
| BR-06 | Margin/nisbah di luar rentang grade diblokir | `backend/internal/service/margin_service.go` | `internal/service/margin_service_test.go` (`TestValidasi_AC09_MarginDiBawahBatasGrade1Diblokir`) | Done |
| BR-07 | Skor akhir = Σ(skor × bobot) ÷ Σbobot, dibulatkan sekali di akhir | `backend/internal/service/skoring_service.go` | `internal/service/skoring_service_test.go` (`TestHitung_BR07_RumusSkorAkhir`) | Done |
| BR-08 | Rincian komponen ditampilkan dan disimpan | `backend/internal/service/skoring_service.go` | `internal/service/skoring_service_test.go` (`TestHitung_AC07_RincianKeempatKomponenTersedia`) | Done |
| BR-09 | Maker tidak boleh menjadi approver; ditegakkan di server | `backend/internal/service/approval_service.go` | `internal/service/approval_service_test.go` (`TestApproval_AC11_MakerChecker_BR09`), `internal/httpapi/approval_http_test.go` | Done |
| BR-10 | Setiap perubahan status punya aktor + timestamp | `backend/internal/service/audit_service.go`, `approval_service.go` | `internal/service/audit_service_test.go` (`TestAudit_AC12_RiwayatLengkapUrutWaktu`) | Done |
| BR-11 | NIK & foto dokumen tidak muncul di log, pesan error, atau URL | Lintas lapisan (`httpapi`, `audit_service`) | `internal/service/audit_service_test.go` | Done |
| BR-12 | Nomor referensi `IMT-YYYYMMDD-NNNN` unik, tidak dipakai ulang | `pengajuan_service.go` + DB constraint | `backend/migrations/000003_pengajuan_approval_audit.up.sql` | Done |

---

## Ringkasan Risiko (perbarui di setiap gate)

<!-- ISI: hitung dari tabel di atas. Tiga baris ini adalah versi paling berguna dari seluruh
     dokumen ini, dan yang paling cepat menunjukkan posisi tim kepada diri sendiri. -->

| Pertanyaan | Kamis 15.30 (Gate 2) | Jumat 11.20 (Gate 3) | Jumat 15.00 (code freeze) |
|---|---|---|---|
| Berapa FR P0 berstatus Done? |  |  |  |
| Berapa FR P0 tanpa file test? |  |  |  |
| Berapa BR tanpa test? |  |  |  |
| Berapa AC yang sudah pernah dilatih di demo? |  |  |  |
| Risiko terbesar saat ini |  |  |  |
