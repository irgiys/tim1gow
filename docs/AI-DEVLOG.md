# AI-DEVLOG — Jurnal Pemakaian AI

**Tim**: iMitra Tim 1
**Pemilik berkas**: AI Workflow Officer — Aldi
**Kontributor**: seluruh anggota tim

---

## Kenapa berkas ini yang dinilai paling tinggi

Menurut brief §9.3, ini **artefak paling bernilai** dalam hackathon, dan §12 menempatkan
"Disiplin rekayasa berbantuan AI" pada bobot terbesar (25 poin).

Alasannya sederhana. Aplikasi yang jalan hanya membuktikan bahwa kode Anda benar hari ini.
Devlog membuktikan bahwa **Anda tahu mengapa kode itu benar** — bahwa Anda memberi AI
spesifikasi, memverifikasi keluarannya terhadap acceptance criteria, dan menangkap
kesalahannya sebelum kesalahan itu masuk ke `main`. Kemampuan itulah yang Anda bawa kembali
ke pekerjaan hari Senin; aplikasi ini tidak.

Konsekuensi praktisnya: **kalau Anda memakai AI dengan baik tetapi tidak mencatatnya,
secara penilaian itu sama dengan tidak melakukannya.**

Dua sanksi dan dua bonus yang langsung terkait berkas ini:

| Temuan | Nilai |
|---|---|
| Devlog ditulis seluruhnya dalam 2 jam terakhir | **−8** |
| Entri yang menunjukkan AI salah secara **halus**, dan tim menangkapnya lewat test (bukan kebetulan) | **+3 per entri, maks +6** |
| Tidak ada satu pun entri kegagalan | Penilai menyimpulkan Anda tidak memverifikasi, atau tidak jujur. Keduanya merugikan |
| Minimal 3 entri sudah ada di Gate 2 (Kamis 15.30) | Syarat kelulusan Gate 2 |

---

## Aturan pengisian

1. **Minimal 10 entri.** Kurang dari itu dianggap tidak lengkap.
2. **Minimal 3 entri berupa kasus AI salah dan Anda menangkapnya.** Dalam 9 jam koding
   dengan AI, sesuatu pasti salah — itu normal. Menangkapnya adalah keahlian yang dinilai.
   Entri kegagalan yang paling bernilai adalah yang **halus**: keluaran yang tampak benar,
   test-nya hijau, dan tetap salah.
3. **Tersebar di dua hari.** Target: minimal 4 entri pada Kamis (3 di antaranya sebelum
   Gate 2 pukul 15.30) dan sisanya pada Jumat. Penilai membaca timestamp commit, bukan
   hanya isi entri. Sepuluh entri yang muncul dalam satu commit pada Jumat 14.50 dikenai −8.
4. **Commit devlog bersamaan dengan commit kode**, bukan dikumpulkan. Satu entri = beberapa
   menit, ditulis saat kejadiannya masih segar.
5. **Semua anggota menyetor entri**, bukan hanya AI Workflow Officer. Kolom "Oleh" yang
   berisi satu nama untuk 10 entri menandakan sembilan orang lain tidak memakai AI secara
   sadar — atau tidak mencatatnya.
6. **Jujur.** Entri yang seluruhnya berisi keberhasilan lebih merugikan daripada entri yang
   mencatat prompt yang gagal tiga kali.
7. **Rujuk artefak nyata**: nomor PR, nama berkas test, ID AC, ID BR. Entri tanpa rujukan
   tidak bisa diverifikasi penilai.

---

## Format entri (wajib, dari brief §9.3)

Salin blok di bawah untuk setiap entri baru. Jangan hilangkan field mana pun; kalau suatu
field tidak berlaku, tulis "tidak ada" beserta alasan singkat.

```markdown
### [DEVLOG-NN] <judul singkat> (FR-xx)
- **Waktu**: YYYY-MM-DD HH:MM
- **Oleh**: <nama>
- **Tool/Model**: <mis. 9Router -> Claude Opus>
- **Tugas**: <apa yang diminta, sebutkan FR/BR/AC terkait>
- **Cara memberi konteks**: <berkas apa yang dilampirkan, bagian brief mana, batasan apa
  yang disebut eksplisit>
- **Keluaran AI**: <apa yang dihasilkan, berapa besar>
- **Yang salah**: <kalau ada. Kalau tidak ada, tulis "tidak ada" dan sebutkan apa yang
  Anda periksa untuk memastikannya>
- **Cara verifikasi**: <langkah konkret. "Saya baca dan kelihatan benar" bukan verifikasi>
- **Tindakan**: <apa yang Anda ubah: prompt, kode, test, AGENTS.md>
- **Pelajaran**: <aturan yang Anda ambil dari kejadian ini. Kalau pelajarannya bersifat
  aturan repo, tambahkan ke AGENTS.md dan sebutkan di sini>
```

---

## Dua contoh entri terisi penuh (teladan, bukan untuk diisi ulang)

Dua entri di bawah adalah contoh standar yang diharapkan. **Hapus keduanya sebelum
`v1.0.0`** — atau biarkan, tetapi jangan hitung sebagai bagian dari 10 entri Anda.
Perhatikan tingkat kedetailannya: nomor PR, nama berkas, angka nyata, dan langkah verifikasi
yang bisa diulang orang lain.

### [CONTOH-A] Kritik model data nasabah perorangan vs kelompok (FR-02, FR-10) — kasus sukses

- **Waktu**: 2026-08-20 10:12
- **Oleh**: Rizky (Tech Lead)
- **Tool/Model**: 9Router → Claude Opus (chat, tanpa akses tulis ke repo)
- **Tugas**: Sebelum menulis migrasi pertama, meminta AI **mengkritik** rancangan model data
  kami — bukan membuatnya. Fokus: bagaimana satu pengajuan bisa mewakili nasabah perorangan
  maupun kelompok (majelis) 3–10 anggota, dengan level approval ditentukan dari total plafon
  kelompok (FR-10, AC-14).
- **Cara memberi konteks**: melampirkan brief §1.3 dan §4.1, ditambah draf skema kami dalam
  bentuk DDL (5 tabel: `pengajuan`, `nasabah`, `dokumen`, `survei`, `hasil_slik`). Prompt
  dibuka dengan batasan tegas: "jangan tulis kode, jangan usulkan tabel baru dulu; sebutkan
  3 kelemahan terbesar rancangan ini terhadap AC-14 dan BR-01, urut dari yang paling mahal
  kalau baru ditemukan besok."
- **Keluaran AI**: tiga kritik, tanpa kode. Yang paling berguna: rancangan kami menyimpan
  `plafon_diajukan` hanya di tabel `pengajuan`, sehingga untuk kelompok tidak ada tempat
  menyimpan plafon per anggota — dan AC-14 mensyaratkan penolakan satu anggota
  Rp 60.000.000 mengurangi total dari Rp 240.000.000 menjadi Rp 180.000.000. Tanpa plafon
  per anggota, angka itu tidak bisa dihitung ulang, hanya bisa ditulis manual.
- **Yang salah**: tidak ada pada keluaran itu sendiri. Satu usulan tambahannya — menyimpan
  kolom turunan `level_approval_diperlukan` di tabel `pengajuan` — kami tolak, karena nilai
  turunan yang disimpan akan basi persis pada skenario AC-14. Penolakan ini dicatat di
  `docs/adr/0002-plafon-per-anggota.md`.
- **Cara verifikasi**: menulis ulang AC-14 sebagai tabel angka di papan tulis (4 anggota,
  60+60+60+60 = 240 → 3 level; satu ditolak → 180 → 2 level), lalu menelusuri apakah skema
  usulan bisa menghasilkan kedua angka itu **hanya** dari data yang tersimpan. Skema lama
  gagal di langkah kedua.
- **Tindakan**: menambahkan tabel `pengajuan_anggota` (plafon per anggota + status per
  anggota), memindahkan `total_plafon` menjadi nilai yang dihitung saat dibaca, bukan kolom
  tersimpan. Migrasi awal ditulis setelah ini, bukan sebelumnya — jadi tidak ada migrasi
  yang perlu dibatalkan. Ditulis di PR #3.
- **Pelajaran**: AI jauh lebih berguna sebagai pengkritik rancangan daripada sebagai pembuat
  rancangan, dan kritik terbaik keluar ketika kami melampirkan **acceptance criteria berupa
  angka**, bukan deskripsi fitur. Sejak entri ini, setiap keputusan model data kami uji
  dengan cara: "AC mana yang membuktikan skema ini cukup?" Aturan ini masuk ke `AGENTS.md`
  bagian 3.

### [CONTOH-B] Pembulatan skor kelayakan menggeser grade (FR-06, BR-07) — kasus AI salah, halus

- **Waktu**: 2026-08-21 09:48
- **Oleh**: Dewi (Backend Engineer)
- **Tool/Model**: VSCode + Copilot (inline, mode agent pada satu berkas service)
- **Tugas**: Implementasi perhitungan skor kelayakan 0–100 dari empat komponen berbobot
  (§4.4) dan penurunan grade 1–5 dari rentang skor (Tabel 4.3), sesuai BR-07.
- **Cara memberi konteks**: melampirkan tabel §4.4 dan Tabel 4.3 sebagai komentar di atas
  fungsi, plus repository parameter yang sudah ada, plus catatan bahwa bobot dibaca dari
  tabel `parameter_skoring` (bukan konstanta).
- **Keluaran AI**: satu fungsi `hitungSkorKelayakan` (± 60 baris) plus 6 unit test. Bobot
  memang dibaca dari database — larangan di `AGENTS.md` bagian 6 butir 3 dipatuhi. Semua
  test hijau pada percobaan pertama.
- **Yang salah**: AI **membulatkan skor setiap komponen ke bilangan bulat sebelum dikalikan
  bobot**, lalu membulatkan lagi hasil akhirnya. BR-07 mensyaratkan pembulatan **hanya pada
  skor akhir**. Selisihnya biasanya 0 atau 1 poin, jadi tidak terlihat — kecuali tepat di
  batas grade. Pada data uji Slamet Riyadi (kapasitas bayar 67,3; SLIK kol-1 → 100; lama
  usaha 60 bulan → 100; survei kondisi usaha 4 × 20 = 80) hasil yang benar adalah
  (67,3×35 + 100×25 + 100×20 + 80×20) ÷ 100 = 84,555 → **85 → grade 1**. Versi AI
  membulatkan 67,3 menjadi 67 lebih dulu, sehingga menghasilkan 84,45 → **84 → grade 2**.
  Akibatnya rentang margin yang divalidasi ikut
  bergeser dari 11,0–13,0 % menjadi 13,0–15,5 %, sehingga AC-09 (margin 10,0 % harus
  diblokir untuk grade 1) diuji pada grade yang salah. Enam test buatan AI semuanya lolos,
  karena semuanya memakai nilai komponen bulat — jadi cabang yang salah tidak pernah
  tereksekusi. Hijaunya menipu.
- **Cara verifikasi**: QA menulis satu test dari AC secara terpisah, memakai baris data
  Slamet Riyadi dari `fixtures/nasabah-uji.csv` dan **menghitung skor manual di kalkulator
  lebih dulu**, bukan menyalin keluaran fungsi sebagai nilai harapan. Test itu gagal:
  harapan 85, hasil 84. Setelah itu kami tambahkan test batas untuk setiap ambang grade
  (39/40, 54/55, 69/70, 84/85) di `<!-- berkas test Anda -->` — dua di antaranya juga gagal.
- **Tindakan**: menghapus pembulatan per komponen, menyimpan skor komponen sebagai desimal
  di kolom rincian (BR-08 mewajibkan rincian disimpan, dan rincian yang sudah dibulatkan
  tidak bisa dipakai auditor untuk merekonstruksi angka akhir), dan membulatkan sekali saja
  di akhir. Test buatan AI yang menegaskan perilaku salah dihapus, bukan disesuaikan.
  Larangan "jangan bulatkan nilai antara; pembulatan hanya sekali di akhir sesuai BR-07"
  ditambahkan ke `AGENTS.md` bagian 6. PR #21.
- **Pelajaran**: test yang dibuat AI menguji asumsi AI, bukan requirement kami. Dua aturan
  tim sejak entri ini: (1) test untuk aturan bisnis ditulis dari AC dengan nilai harapan
  dihitung manual lebih dulu; (2) setiap aturan yang punya ambang wajib punya test tepat di
  batas atas dan batas bawahnya. Bug ini tidak akan pernah muncul di jalur bahagia demo —
  ia hanya muncul pada satu nasabah, dan nasabah itu mendapat margin yang lebih mahal
  daripada haknya.

---

## Entri Tim

<!-- ISI: sepuluh blok di bawah. Isi berurutan sesuai waktu kejadian, bukan sesuai nomor
     yang Anda sukai. Kalau ternyata perlu lebih dari 10, lanjutkan dengan DEVLOG-11, dst.
     Beri tanda pada judul entri kegagalan supaya penilai mudah menemukannya, mis.
     "### [DEVLOG-04] ... (FR-07) — kasus AI salah".
     Ingat: minimal 3 entri kegagalan, dan minimal 3 entri sudah ada sebelum Gate 2. -->

### [DEVLOG-01] Pembagian peran, pemilik berkas, dan pembagian frontend (Sprint 0) — kasus AI salah
- **Waktu**: 2026-08-20 10:30 – 11:00
- **Oleh**: Irgiyansyah
- **Tool/Model**: Hermes IDE (agen dengan akses baca/tulis repo)
- **Tugas**: Menurunkan pembagian peran 6 anggota dan pemilik tiap berkas dokumentasi dari
  aturan yang sudah ada di `AGENTS.md` — bukan mengarang struktur tim baru. Sasarannya:
  setiap berkas markdown punya satu pemilik supaya tidak ada konflik merge pada tabel, dan
  setiap FR punya satu penanggung jawab (brief §10, syarat Gate 1).
- **Cara memberi konteks**: melampirkan `AGENTS.md` utuh, ditambah `01-BRIEF-Hackathon-iMitra.md`
  §10 dan `SETUP-SPRINT-0.md`. Batasan yang disebut eksplisit: pembagian harus **diturunkan
  dari bagian 3 (struktur `internal/service/`) dan bagian 5 (kolom "Ditegakkan di")** berkas
  `AGENTS.md`, sehingga satu berkas service hanya punya satu pemilik. Agen juga diminta
  menghitung placeholder `<!-- ISI -->` per berkas lebih dulu, bukan menebak mana yang kosong.
- **Keluaran AI**: tabel peran 6 orang + tabel pemilik berkas/lapisan kode di `README.md`
  bagian 1, kolom penanggung jawab di `docs/AI-WORKFLOW.md` bagian 1, header pemilik di 6
  berkas `docs/`, tabel pembagian `frontend/app/` per route (14 baris), tabel hak akses &
  approver, dan pengisian `.github/CODEOWNERS` per path. Tiga commit: `429fe68`, `e5b2f91`,
  `f9a731b`.
- **Yang salah**: dua kesalahan, keduanya **halus karena keluarannya terlihat masuk akal**.
  (1) Pada jawaban pertama agen menetapkan Rayvaldo sebagai Tech Lead dengan alasan yang
  terdengar kuat — brief §1.1 menyebut dia penyusun SRS/SDD iLoan Commercial. Itu inferensi
  dari brief, bukan dari keadaan repo: tabel riwayat `AGENTS.md` menunjukkan **Luthfi** yang
  mengisi bagian 2–7, dan berkas itu menyatakan pemiliknya Tech Lead. (2) Agen menetapkan
  pemilik `SRS` = Irgiyansyah dan `SDD` = Yulio Zaki, tertukar dari kesepakatan tim
  (`SRS` = Yulio Zaki, `SDD` = Irgiyansyah). Keduanya tidak terdeteksi oleh pemeriksaan
  otomatis apa pun — tabelnya rapi dan konsisten secara internal.
- **Cara verifikasi**: menjalankan `git log --oneline` dan membaca tabel "Riwayat perubahan"
  di `AGENTS.md` untuk mencari **bukti siapa yang benar-benar menyentuh berkas**, bukan
  mengandalkan penalaran agen atas brief. Untuk kesalahan kedua, mencocokkan nama pemilik di
  `README.md` bagian 1 dengan kesepakatan tim, lalu memeriksa bahwa perbaikannya ikut
  diterapkan di **tiga tempat** — tabel README, header `**Penyusun**` di kedua berkas, dan
  baris "Perancangan & kritik model data" di `AI-WORKFLOW.md` — karena satu koreksi nama
  menyentuh lebih dari satu berkas dan agen sempat hanya memperbaiki README.
- **Tindakan**: Tech Lead dikoreksi menjadi Luthfi; kepemilikan SRS/SDD ditukar di ketiga
  lokasi (commit `e5b2f91`). Pembagian frontend diubah dari "semua UI ke satu Frontend
  Engineer" menjadi vertikal — pemilik aturan bisnis sebuah FR juga menulis UI-nya, karena
  bentuk Tim 2 brief §10 hanya menyediakan satu Frontend Engineer untuk UI 6 peran. Ketiga
  perubahan dicatat di tabel riwayat `AGENTS.md`.
- **Pelajaran**: kalau sebuah fakta bisa dibaca dari **keadaan repo** (`git log`, tabel
  riwayat, isi berkas), jangan biarkan agen menyimpulkannya dari prosa brief — inferensi yang
  terdengar kuat justru paling sulit dibantah saat dibaca ulang. Sejak entri ini, setiap
  keluaran agen yang menyebut **nama orang atau kepemilikan berkas** wajib dicek terhadap
  `git log` lebih dulu. Konsekuensi kedua: satu koreksi nama harus dilacak ke semua berkas
  yang menyebutnya, karena agen cenderung memperbaiki hanya berkas yang sedang dibuka.

### [DEVLOG-02] Argumen ambang kapasitas bayar tertukar (FR-06, BR-07) — kasus AI salah, halus
- **Waktu**: 2026-08-20 12:40 – 13:15
- **Oleh**: Irgiyansyah
- **Tool/Model**: Hermes IDE (agen dengan akses baca/tulis repo)
- **Tugas**: Implementasi `skoring_service` (FR-06) dan `margin_service` (FR-07) beserta
  test dari AC-06, AC-07, AC-09, AC-15. Syarat yang disebut eksplisit: seluruh bobot, ambang,
  dan rentang **dibaca dari tabel parameter**, tidak boleh muncul sebagai konstanta di kode
  maupun di dalam test (`AGENTS.md` Larangan 3).
- **Cara memberi konteks**: melampirkan `AGENTS.md` bagian 3, 4.1, 4.3, dan 5 (termasuk tabel
  komponen skor brief §4.4), ditambah AC-06/07/09/15 apa adanya. Batasan tambahan: fungsi
  perhitungan tidak boleh tahu tentang HTTP, dan test AC-15 wajib **mengubah baris parameter
  di tengah test** lalu memastikan hasilnya berubah.
- **Keluaran AI**: `skoring_service.go`, `skoring_komponen.go`, `margin_service.go`, plus 13
  test. Struktur lapisannya benar dan seluruh angka memang dibaca dari repository parameter.
- **Yang salah**: pada `skorKapasitasBayar`, argumen ambang dibalik —
  `skorLinearTurun(rasio, k.Batas2, k.Batas1)` alih-alih `(rasio, k.Batas1, k.Batas2)`.
  Akibatnya rasio angsuran yang **bagus** (0,267 — jauh di bawah batas 0,30) justru diberi
  skor 0, dan rasio buruk diberi skor penuh. Ini halus karena: fungsi `skorLinearTurun`
  sendiri benar, tidak ada error, kompilasi lolos, `go vet` lolos, dan skor akhir tetap
  keluar sebagai angka 0–100 yang tampak masuk akal. Kalau hanya diperiksa "apakah skornya
  wajar", ini lolos.
- **Cara verifikasi**: yang menangkapnya adalah test AC-06 — bukan pembacaan kode. Test itu
  sengaja memuat **kasus pembanding**: data yang sama dijalankan dengan kolektibilitas 1,
  lalu dipastikan grade-nya lebih baik dari 3. Ketika pembanding ikut menghasilkan grade 3,
  test gagal dengan pesan *"data pembanding menghasilkan grade 3; kasus uji tidak membuktikan
  apa pun"* — menandakan bukan aturan kolektibilitasnya yang salah, melainkan skor dasarnya
  terlalu rendah. Dari situ komponen dilacak satu per satu. Test yang hanya memeriksa
  `grade >= 3` untuk kolektibilitas 2 akan **lolos** dan bug ini masuk ke `main`.
- **Tindakan**: urutan argumen diperbaiki dan diberi komentar yang menjelaskan arah
  perbandingannya ("untuk rasio angsuran, makin kecil makin baik, jadi batas skor penuh
  adalah nilai yang lebih kecil"), karena inilah tepat sumber kebingungannya. Seluruh 13 test
  dijalankan ulang di container `golang:1.22-alpine` — lolos, `gofmt` bersih, `go vet` bersih.
  Sebelum itu `go vet` juga menangkap empat pesan test yang memuat `%` mentah sehingga dibaca
  sebagai verb format; diperbaiki di commit yang sama.
- **Pelajaran**: untuk fungsi berambang, **satu test per kasus tidak cukup** — wajib ada
  kasus pembanding yang membuktikan test itu bisa gagal. Aturan yang kami ambil: setiap test
  aturan bisnis yang memeriksa "nilai X ditolak" harus disertai kasus "nilai Y diterima",
  dan sebaliknya. Tanpa itu, fungsi yang selalu menolak akan meloloskan seluruh test
  penolakan. Pelajaran kedua: `go vet` dijalankan sebelum `go test`, bukan sesudah —
  ia menangkap kesalahan format yang membuat pesan kegagalan test menyesatkan.

### [DEVLOG-03] Penegakan maker != approver (BR-09), urutan berjenjang (BR-02), dan audit trail append-only (FR-08, FR-09)
- **Waktu**: 2026-08-20 14:45 – 15:15
- **Oleh**: Luthfi (Tech Lead)
- **Tool/Model**: Antigravity IDE (Gemini 3.7 Flash)
- **Tugas**: Implementasi fungsionalitas FR-08 (Approval Berjenjang) dan FR-09 (Audit Trail) beserta database migration (000003), repository GORM, service layer dengan penegakan BR-01/02/05/09/10/11/12, HTTP API handlers, dan test otomatis untuk AC-08, AC-10, AC-11, AC-12, AC-13.
- **Cara memberi konteks**: Melampirkan `AGENTS.md` bagian 3–5, `docs/SDD-iMitra.md` BAB 3.2, 4, 5, skema ambang approval di tabel `ambang_approval`, dan acceptance criteria brief §5 (AC-08/10/11/12/13). Menegaskan aturan server-side check untuk BR-09 (`maker != approver`), sequential routing BR-02, serta immutability AC-13.
- **Keluaran AI**: Migrasi `000003_pengajuan_approval_audit.up.sql` / `down.sql`, domain model `approval.go` dan `audit.go`, repository `approval_repository.go`, `audit_repository.go`, `parameter_repository_db.go`, service `approval_service.go` dan `audit_service.go`, handler `approval_handler.go` dan `audit_handler.go`, serta test suite `approval_service_test.go`, `audit_service_test.go`, `approval_http_test.go`, `audit_http_test.go`.
- **Yang salah**: Saat perancangan pertama, terjadi import cycle antara package `service` dan `repository` karena `service` mengimpor `repository` sementara `repository` mengimpor `service.ParameterRepository`.
- **Cara verifikasi**: Menjalankan `go test -v ./...` yang langsung mendeteksi `import cycle not allowed` pada saat kompilasi.
- **Tindakan**: Menerapkan idiomatic Go *Dependency Inversion Principle* di mana interface repository (`ApprovalRepository`, `AuditRepository`, `ParameterRepository`) didefinisikan di package konsumen (`service`), dan package `repository` yang mengimplementasikannya. Seluruh unit dan integration test (7 test approval, 2 test audit, 2 test HTTP approval, 2 test HTTP audit append-only) dijalankan ulang dan 100% PASS. Migrasi 000003 berhasil diterapkan pada database Aiven PostgreSQL instance.
- **Pelajaran**: Dalam layout Clean Architecture di Go, service mendefinisikan interface dependensinya sendiri; package luar (repository) yang mengimplementasikannya. Ini mencegah circular dependency dan membuat service mudah diuji dengan mock/fake tanpa ketergantungan DB nyata.

### [DEVLOG-04] Empat BR ditandai "Done" tanpa test yang mengujinya (BR-01, BR-08, BR-11, BR-12) — kasus AI salah
- **Waktu**: 2026-08-20 15:20 – 16:40
- **Oleh**: Soleh (QA / Verification)
- **Tool/Model**: Hermes IDE (Claude Opus, agen dengan akses baca/tulis repo)
- **Tugas**: Dua hal. (1) Mengisi kolom BR pada `docs/TRACEABILITY.md` dan tabel Ringkasan
  Risiko untuk Gate 2. (2) Menutup BR yang belum punya test, dimulai dari yang bisa diuji
  murni di `service` tanpa DB maupun HTTP.
- **Cara memberi konteks**: melampirkan `AGENTS.md` bagian 5 (daftar BR + lokasi penegakan),
  Larangan 3, 15, 18, 19, dan bagian "Cara memakai tabel ini sebagai alat deteksi risiko"
  pada `TRACEABILITY.md`. Batasan yang disebut eksplisit: sebuah baris hanya boleh berstatus
  `Done` kalau berkas penegaknya **benar-benar ada** DAN ada nama fungsi test yang menguji
  aturan itu.
- **Keluaran AI**: audit repo, koreksi tabel BR, pengisian Ringkasan Risiko Gate 2, lalu
  8 fungsi test baru + `pengajuan_service.go` + migrasi `000004`.
- **Yang salah**: dua hal, dan yang pertama bukan kesalahan sesi ini.
  1. **Tabel BR yang sudah diisi PR #8 memuat empat klaim `Done` tanpa bukti.** BR-01 menunjuk
     `pengajuan_service.go` yang **tidak ada satu pun di repo**; kolom test-nya menunjuk
     `approval_service_test.go`, yang ternyata hanya memakai plafon 30jt/120jt/300jt —
     semuanya **di dalam** batas, jadi tidak ada satu pun kasus yang menguji BR-01. BR-11
     ditandai `Done` dengan test `audit_service_test.go` yang tidak memuat satu pun assertion
     tentang NIK. BR-12 menunjuk berkas migrasi sebagai "test" — constraint DB bukan test.
     BR-08 ditandai `Done` padahal tabel `komponen_skor` tidak ada di migrasi mana pun,
     sementara BR-08 mewajibkan rincian **disimpan**, bukan sekadar dikembalikan di memori.
  2. **Kesalahan agen di sesi ini**: rencana awalnya "mengisi kolom BR yang kosong" disusun
     dari `main` lokal yang **ketinggalan 2 commit**. Tabel itu sebenarnya sudah terisi. Kalau
     tidak di-`fetch` lebih dulu, hasil kerjanya akan menimpa pekerjaan orang lain.
- **Cara verifikasi**: setiap klaim `Done` dicek satu per satu ke kode, bukan dibaca dari
  tabel — `ls` berkas penegaknya, lalu `grep` nama fungsi test dan isi assertion-nya. Itu yang
  memunculkan keempatnya. Untuk pekerjaan (2), test ditulis lebih dulu dan **dipastikan GAGAL**
  (`undefined: NewPengajuanService`) sebelum implementasi ditulis, supaya terbukti test-nya
  benar-benar menguji sesuatu. Verifikasi akhir dijalankan nyata, bukan diasumsikan:
  `go test ./internal/service/... -count=1` hijau (47 subtest), `go vet` bersih, `gofmt` bersih.
  Go tidak tersedia di mesin QA, jadi Go 1.22.12 portable dipasang ke direktori temp — versinya
  disamakan dengan `go 1.22` di `go.mod`, bukan versi terbaru.
- **Tindakan**: keempat baris diturunkan statusnya beserta alasannya (`b9adbe8`). BR-01 dan
  BR-12 lalu benar-benar ditutup (`ef8f775`): 8 test, masing-masing dua arah sesuai Larangan 18
  (4jt/4.999.999 **ditolak** dan 5jt/500jt **diterima**; batas diuji tepat di tepinya). Batas
  plafon tidak ditulis sebagai konstanta melainkan dibaca dari `parameter_umum` (Larangan 3),
  di-seed lewat migrasi **baru** `000004` — bukan mengubah `000002` yang sudah di-merge
  (Larangan 2) — dengan `ON CONFLICT DO NOTHING` (Larangan 19). Aturan pengisian tabel BR
  ditulis eksplisit di komentar berkas supaya tidak terulang.
- **Pelajaran**: **kolom "Test" pada tabel traceability adalah klaim, bukan bukti.** Menunjuk
  nama berkas test tidak membuktikan apa pun — yang membuktikan adalah nama fungsi test dan
  isi assertion-nya. Tiga dari empat kesalahan di atas berbentuk sama: kolom test diisi berkas
  yang *ada* dan *hijau*, tetapi tidak menguji aturan yang diklaim. Ini justru berbahaya karena
  CI hijau memberi rasa aman: tim masuk Jumat dengan keyakinan 12/12 BR terlindungi, padahal
  4 di antaranya tidak diuji siapa pun. Pelajaran kedua, untuk agen: **`git fetch` sebelum
  menyusun rencana kerja pada repo yang dikerjakan 6 orang paralel** — rencana yang disusun
  dari basis usang menghasilkan pekerjaan yang menimpa milik orang lain.

### [DEVLOG-05] `<!-- ISI: judul -->` (FR-`<!-- ISI -->`)
- **Waktu**:
- **Oleh**:
- **Tool/Model**:
- **Tugas**:
- **Cara memberi konteks**:
- **Keluaran AI**:
- **Yang salah**:
- **Cara verifikasi**:
- **Tindakan**:
- **Pelajaran**:

### [DEVLOG-06] `<!-- ISI: judul -->` (FR-`<!-- ISI -->`)
- **Waktu**:
- **Oleh**:
- **Tool/Model**:
- **Tugas**:
- **Cara memberi konteks**:
- **Keluaran AI**:
- **Yang salah**:
- **Cara verifikasi**:
- **Tindakan**:
- **Pelajaran**:

### [DEVLOG-07] `<!-- ISI: judul -->` (FR-`<!-- ISI -->`)
- **Waktu**:
- **Oleh**:
- **Tool/Model**:
- **Tugas**:
- **Cara memberi konteks**:
- **Keluaran AI**:
- **Yang salah**:
- **Cara verifikasi**:
- **Tindakan**:
- **Pelajaran**:

### [DEVLOG-08] `<!-- ISI: judul -->` (FR-`<!-- ISI -->`)
- **Waktu**:
- **Oleh**:
- **Tool/Model**:
- **Tugas**:
- **Cara memberi konteks**:
- **Keluaran AI**:
- **Yang salah**:
- **Cara verifikasi**:
- **Tindakan**:
- **Pelajaran**:

### [DEVLOG-09] `<!-- ISI: judul -->` (FR-`<!-- ISI -->`)
- **Waktu**:
- **Oleh**:
- **Tool/Model**:
- **Tugas**:
- **Cara memberi konteks**:
- **Keluaran AI**:
- **Yang salah**:
- **Cara verifikasi**:
- **Tindakan**:
- **Pelajaran**:

### [DEVLOG-10] `<!-- ISI: judul -->` (FR-`<!-- ISI -->`)
- **Waktu**:
- **Oleh**:
- **Tool/Model**:
- **Tugas**:
- **Cara memberi konteks**:
- **Keluaran AI**:
- **Yang salah**:
- **Cara verifikasi**:
- **Tindakan**:
- **Pelajaran**:

---

## Rekapitulasi (isi di akhir hari kedua, sebelum code freeze)

<!-- ISI: rekap singkat. Ini yang dibaca penilai lebih dulu sebelum masuk ke entri. -->

| Hal | Isi |
|---|---|
| Total entri | `<!-- ISI -->` |
| Entri pada Kamis / Jumat | `<!-- ISI -->` / `<!-- ISI -->` |
| Entri kegagalan AI (nomor) | `<!-- ISI: mis. DEVLOG-03, DEVLOG-06, DEVLOG-09 -->` |
| Anggota yang menyetor entri | `<!-- ISI: dari total anggota -->` |
| Perubahan `AGENTS.md` yang dipicu devlog | `<!-- ISI: nomor DEVLOG -> perubahan -->` |
| Satu pelajaran terpenting untuk pekerjaan sehari-hari | `<!-- ISI -->` |
