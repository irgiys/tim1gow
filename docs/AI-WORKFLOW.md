# AI-WORKFLOW — Cara Tim Ini Bekerja dengan AI

**Tim**: iMitra Tim 1
**Pemilik berkas**: AI Workflow Officer — Aldi
**Terakhir diperbarui**: 2026-08-20 11:45

> Target panjang: satu sampai dua halaman. Ini dokumen **keputusan**, bukan laporan.
> Bedanya dengan `AI-DEVLOG.md`: devlog mencatat kejadian satu per satu, berkas ini mencatat
> pola — bagaimana tim memutuskan tool mana untuk tugas mana, dan apa yang tidak diserahkan
> ke AI. Isi kerangkanya di Sprint 0 (walaupun masih dugaan), lalu perbarui pada Jumat siang
> dengan apa yang **ternyata** terjadi. Perbedaan antara dugaan awal dan kenyataan justru
> bagian yang paling menarik untuk penilai — jangan dihapus, catat sebagai revisi.

**Cara membaca penanda di berkas ini**

| Penanda | Arti |
|---|---|
| *(belum disetor)* | Anggota yang bersangkutan belum menyetor 3 barisnya. Bukan berarti tidak dipakai — berarti kami belum punya datanya, dan kami tidak mengarang. |
| *(belum diamati)* | Tool dipakai, tetapi kami belum punya pengamatan yang cukup spesifik untuk dituliskan. Diisi saat istirahat makan. |
| Kolom "Terbukti?" | Sengaja kosong sampai Jumat. Mengisinya sekarang berarti mengarang. |

---

## 1. Tugas → Tool/Model → Alasan

Kolom **Penanggung jawab** mengikuti pembagian peran di `README.md` bagian 1 dan pemilik
lapisan kode di `AGENTS.md` bagian 3 — satu tugas satu orang, supaya tidak ada dua orang
mengedit berkas yang sama.

| Tugas | Penanggung jawab | Tool | Model | Alasan pemilihan | Terbukti? |
|---|---|---|---|---|---|
| Perancangan & kritik model data | Irgiyansyah (SDD BAB 4) | Hermes IDE | Claude Opus | Konteks cukup panjang untuk melampirkan `AGENTS.md` utuh + brief §10 + `SETUP-SPRINT-0.md` sekaligus, sehingga kritik model diuji terhadap AC-14 dengan angka, bukan terhadap prosa saja (lihat DEVLOG-01) |  |
| Menulis migrasi skema | Rayvaldo | *(belum disetor)* | *(belum disetor)* | *(belum disetor)* |  |
| Aturan bisnis (skoring, margin) | Irgiyansyah | *(belum disetor)* | *(belum disetor)* | *(belum disetor)* |  |
| Aturan bisnis (routing approval, audit) | Luthfi | *(belum disetor)* | *(belum disetor)* | *(belum disetor)* |  |
| CRUD & endpoint rutin | Rayvaldo | *(belum disetor)* | *(belum disetor)* | *(belum disetor)* |  |
| Komponen UI per peran (shell, auth, AO) | Aldi | Hermes IDE | Claude Opus | Agen bisa menjalankan `tsc`, `next build`, `eslint`, dan `curl` sendiri lalu membaca keluarannya, sehingga fondasi frontend diverifikasi hijau **sebelum** commit, bukan diserahkan ke CI |  |
| Integrasi HTTP ke mock SLIK + jalur error | Yulio Zaki | *(belum disetor)* | *(belum disetor)* | *(belum disetor)* |  |
| Menulis test dari AC | Soleh | *(belum disetor)* | *(belum disetor)* | *(belum disetor)* |  |
| Review kode / cari bug sebelum PR | Soleh (gerbang) + Luthfi (merge) | *(belum disetor)* | *(belum disetor)* | *(belum disetor)* |  |
| docker-compose & CI | Soleh | *(belum disetor)* | *(belum disetor)* | *(belum disetor)* |  |
| Seed & data uji | Irgiyansyah (parameter) + Yulio (SLIK) | *(belum disetor)* | *(belum disetor)* | *(belum disetor)* |  |
| Dokumentasi (SRS Yulio · SDD Irgi · ADR Luthfi) | Yulio Zaki / Irgiyansyah / Luthfi | Hermes IDE | Claude Opus | Patuh pada format tabel yang diminta dan bisa menghitung sendiri placeholder `<!-- ISI -->` per berkas, jadi tidak ada bagian yang terlewat karena lupa |  |
| Debugging galat runtime | pemilik lapisan yang bersangkutan | *(belum disetor)* | *(belum disetor)* | *(belum disetor)* |  |

---

## 2. Cara Kami Memberi Konteks ke AI

**Yang selalu dilampirkan** (dan mengapa):

| Yang dilampirkan | Untuk tugas apa | Kenapa |
|---|---|---|
| `AGENTS.md` | Semua tugas yang menghasilkan kode atau menyentuh struktur repo | Memuat stack + versi terkunci (bagian 2), tabel penempatan kode (bagian 3), dan 17 larangan. Tanpa ini agen menaruh aturan bisnis di handler dan menaikkan versi dependensi |
| Bagian brief yang relevan (mis. Tabel 4.3) | Skoring (FR-06), margin/nisbah (FR-07), routing approval (FR-08) | Angka ambang harus berasal dari brief lalu **disimpan sebagai data**. Melampirkan tabelnya membuat agen tahu nilainya tanpa punya alasan menuliskannya sebagai konstanta |
| AC terkait, apa adanya | Penulisan test, dan setiap FR sebelum kodenya ditulis | Test wajib diturunkan dari AC, bukan dari kode yang baru ditulis (`AGENTS.md` bagian 7 Definition of Done). Menempel AC apa adanya mencegah agen menulis test yang hanya mencerminkan implementasinya sendiri |
| Skema / berkas migrasi | Repository, query, seed | Agen tidak boleh menebak nama tabel/kolom. Salah nama kolom baru terlihat saat runtime, jauh setelah review |
| Berkas yang akan diubah, utuh | Semua perubahan pada berkas yang sudah ada | Potongan sebagian membuat agen menulis ulang bagian yang tidak diminta dan menghapus komentar aturan yang sudah ada |
| ADR yang sudah diputuskan | Usulan arsitektur apa pun | Agen tidak boleh mengusulkan hal yang bertentangan dengan ADR `Accepted` tanpa ADR baru (`AGENTS.md` bagian 1) |
| `README.md` bagian 1 (pembagian peran & pemilik berkas) | Semua tugas dokumentasi | Supaya agen tidak menyunting berkas milik orang lain dan memicu konflik merge pada tabel markdown |

**Yang sengaja TIDAK dilampirkan**:

- **Seluruh repo sekaligus.** Konteks terlalu luas membuat keluaran menyentuh berkas yang
  tidak diminta — termasuk `docker-compose.yml`, `ci.yml`, dan `AGENTS.md` yang menurut
  Larangan 14 hanya boleh diubah lewat PR terpisah.
- **Data nasabah di luar `fixtures/nasabah-uji.csv`.** Tidak ada NIK karangan di prompt,
  supaya tidak ada NIK karangan yang lolos ke seed, test, atau contoh di dokumen (BR-11).
- **Isi `.env` siapa pun.** Hanya `.env.example` dengan placeholder (Larangan 10).

**Batasan yang selalu kami sebut eksplisit di prompt**:

- "Parameter dibaca dari tabel `parameter_skoring` / `ambang_approval` / `rentang_margin`.
  Jangan hardcode, termasuk sebagai nilai default dan termasuk di dalam test."
- "Jangan tambah dependensi. Pakai yang sudah ada di `go.mod` / `package.json`; kalau memang
  perlu, usulkan dulu beserta alternatifnya dan tunggu persetujuan Tech Lead."
- "Jelaskan rencana dulu maksimal 10 baris. Jangan tulis kode sebelum saya setuju."
- "Maksimal ~200 baris per keluaran. Kalau lebih, pecah jadi tahap bernomor."
- "Kegagalan tidak boleh dianggap sukses: jangan `catch` kosong, jangan isi nilai default
  saat panggilan gagal, khususnya di jalur SLIK."
- "Jangan tulis NIK, nomor dokumen, atau path foto ke log, pesan error, atau URL. Pakai id
  internal pengajuan untuk korelasi."
- "Setelah selesai, jalankan lint + test + build dan tunjukkan keluaran aslinya. Jangan
  laporkan 'seharusnya lolos'."

**Pola prompt yang paling sering berhasil** — dua pola, keduanya sudah dipakai hari ini:

1. **"Turunkan dari berkas, jangan simpulkan dari prosa."** Bentuknya: *"Baca `X` dan `Y`
   lebih dulu, lalu jawab hanya dari apa yang tertulis di sana. Kalau sebuah fakta tidak ada
   di kedua berkas itu, katakan tidak ada — jangan menyimpulkan dari brief."* Pola ini lahir
   dari DEVLOG-01, di mana inferensi agen atas prosa brief terdengar kuat tetapi salah.
2. **"Bertahap dengan gerbang verifikasi."** Bentuknya: *"Kerjakan tahap 1 saja, jalankan
   lint + build, tunjukkan keluarannya, baru lanjut tahap 2."* Dipakai saat membangun fondasi
   frontend: scaffold → `lib/` → `components/` → route, dengan `tsc`/`eslint`/`next build`
   dijalankan di antaranya. Keluaran besar sekali jalan tidak bisa direview baris per baris
   dalam tekanan waktu.

**Pola prompt yang kami hentikan**: meminta agen "lengkapi semua placeholder di berkas ini"
tanpa menyebut bagian mana yang **belum punya datanya**. Hasilnya, agen mengisi seluruh tabel
termasuk kolom yang seharusnya menunggu pengalaman nyata (kolom "Terbukti?", pengamatan per
tool), dan isian itu terlihat rapi sehingga sulit dibedakan dari fakta. Penggantinya: sebutkan
eksplisit *"bagian yang belum ada datanya tandai `(belum disetor)`, jangan diisi"*.

---

## 3. Pembagian AI vs Manual

| Pekerjaan | AI / Manual / Campuran | Dasar keputusan |
|---|---|---|
| Model data & relasi | Campuran — manusia merancang, AI **mengkritik** | Kesalahan model data mahal dibatalkan: skema salah pada FR-02 merembet ke FR-10, skoring, dan routing approval. AI dipakai sebagai penantang ("apakah skema ini bisa menghasilkan Rp 240jt → 3 level dan Rp 180jt → 2 level hanya dari data tersimpan?"), bukan sebagai perancang |
| Migrasi | Campuran — AI menulis SQL, manusia membaca sebelum commit | Migrasi yang sudah masuk `main` tidak boleh diubah (Larangan 2), jadi biaya kesalahan tinggi dan permanen. AI mempercepat penulisan boilerplate `up`/`down`; manusia memverifikasi tipe kolom, constraint, dan indeks |
| Perhitungan skor (BR-07, BR-08) | Campuran — AI menulis struktur, **angka harapan test dari manusia** | Rumus Σ(skor×bobot)÷Σbobot mudah ditulis, tetapi titik pembulatannya menentukan grade — dan grade menentukan margin. Kalau angka harapan test juga dari AI, test hanya membuktikan AI konsisten dengan dirinya sendiri (brief §13 butir 7) |
| Validasi rentang margin (BR-06) | Campuran — AI menulis, manusia wajib memeriksa sumber angkanya | Kegagalan khas AI di sini sudah terdokumentasi di brief §14: menuliskan rentang sebagai object literal di dalam service, lengkap dengan test yang lolos. Hijaunya menipu karena test ikut menguji konstanta yang salah |
| Routing approval berjenjang (BR-01, BR-02) | Campuran — AI menulis, manusia memverifikasi urutan | AC-10 menuntut KC tidak bisa memutuskan sebelum KCP. Ini kondisi urutan yang mudah "hampir benar": kode yang memeriksa keberadaan approval L1 tanpa memeriksa **hasilnya** `APPROVE` akan lolos test yang dangkal |
| Pemisahan maker/checker (BR-09) | **Manual** | Kontrol perbankan inti. Ditegakkan di server, dan kegagalannya tidak selalu terlihat di UI. Ditulis dan direview manusia, lalu diuji oleh QA dari AC-11 |
| Middleware otorisasi | **Manual** | AC-02 menguji 403 secara langsung. AI cenderung menghasilkan guard yang benar di jalur bahagia tetapi longgar di kasus tepi (token kedaluwarsa, peran tidak dikenal, endpoint baru yang lupa didaftarkan). Konsekuensi kebocoran di sini paling mahal |
| Audit trail append-only | Campuran — AI menulis penulisan baris, manusia memastikan **tidak ada** jalur ubah/hapus | Sifat append-only dibuktikan dari **daftar route** (AC-13), bukan dari niat. AI bisa menambahkan endpoint "perbaiki catatan" karena terlihat berguna; itu justru pelanggaran Larangan 8 |
| Boilerplate CRUD | **AI** | Berulang, punya pola jelas, dan salahnya langsung terlihat di test integrasi. Di sinilah AI memberi penghematan waktu terbesar dengan risiko terendah |
| Form & tampilan | **AI** | Tidak memuat aturan bisnis (Larangan 17) — UI hanya menampilkan hasil dan pesan dari API. Kesalahan tampilan terlihat langsung di layar dan murah diperbaiki |
| Test dari AC | Campuran — AI menulis kerangka, **manusia menentukan nilai harapan** | Test wajib diturunkan dari AC, bukan dari kode. Kalau AI membaca implementasi lalu menulis test, test itu hanya mengunci perilaku yang ada, termasuk bug-nya |
| Review PR | Campuran — AI sebagai pembaca pertama, manusia yang memutuskan | AI berguna untuk menemukan pola terlarang secara mekanis (angka hardcode, `catch` kosong, NIK di log). Keputusan approve tetap manusia, dan approver tidak boleh yang meminta keluaran AI itu |

**Aturan tim tentang ukuran keluaran AI**: keluaran > 200 baris **tidak** di-merge sebelum
dibaca baris per baris oleh orang selain yang memintanya (`AGENTS.md` Larangan 13). Kalau
tugasnya memang besar, agen wajib mengajukan rencana bertahap lebih dulu dan menunggu
persetujuan — dan setiap tahap diverifikasi (lint/test/build) sebelum tahap berikutnya. Kami
memilih ini karena keluaran besar yang rapi adalah kombinasi paling berbahaya: terlihat
selesai, terlalu panjang untuk direview jujur dalam tekanan waktu.

**Aturan tim tentang siapa yang bertanggung jawab atas kode hasil AI**: **yang commit yang
bertanggung jawab, tanpa pengecualian.** Kalau Anda tidak bisa menjelaskan satu baris —
apa yang dilakukannya, mengapa ada, dan apa yang rusak kalau dihapus — baris itu tidak boleh
Anda commit. Hapus, atau pahami dulu. Jawaban "itu dari AI" tidak berlaku di review internal
kami, sama seperti tidak berlaku di penilaian (brief §12). Konsekuensi praktisnya: reviewer
berhak menunjuk baris acak di PR dan meminta penjelasan sebelum approve.

---

## 4. Yang Tidak Kami Percayakan ke AI

| Yang tidak diserahkan ke AI | Alasan | Bagaimana kami mengerjakannya |
|---|---|---|
| **Nilai harapan (expected value) di dalam test aturan bisnis** | Kalau angka harapan berasal dari AI, test hanya membuktikan AI konsisten dengan dirinya sendiri — bukan konsisten dengan brief. Mekanisme kegagalannya: AI menghitung skor dengan pembulatan di langkah tengah, menulis test dengan hasil hitungannya sendiri, dan test lolos. Grade bergeser satu tingkat, margin ikut bergeser, dan CI tetap hijau (persis kasus di brief §14) | QA menurunkan angka harapan langsung dari AC dan Tabel 4.3 brief, dihitung manual di kertas lebih dulu. Angka itu ditulis ke test sebelum implementasinya ada |
| **Penegakan otorisasi di server & pemisahan maker/checker (AC-02, BR-09)** | AI menghasilkan guard yang benar di jalur yang diminta, tetapi tidak punya gambaran menyeluruh tentang endpoint mana saja yang ada. Endpoint baru yang lupa didaftarkan tidak menghasilkan galat apa pun — ia hanya terbuka. Kalau ini lolos ke `main`, kebocoran otorisasi tidak terlihat di UI dan baru ketahuan saat penilai mencobanya | Middleware ditulis manusia. Setiap endpoint baru wajib menyertakan test 403 dari peran yang tidak berwenang, dan QA memeriksa daftar route utuh sebagai gerbang sebelum merge |
| **Apa pun yang menyentuh data pribadi (BR-11)** | AI menambahkan konteks ke pesan galat dan log karena itu praktik baik pada umumnya — dan konteks paling "berguna" yang tersedia justru NIK dan nama nasabah. Kegagalannya sunyi: tidak ada test yang gagal karena log terlalu informatif, dan kebocorannya permanen begitu masuk riwayat log | Log dan pesan error hanya memakai id internal pengajuan. Reviewer khusus mencari NIK/nomor dokumen/path foto pada setiap PR yang menyentuh logging, error handling, atau URL |
| **Keputusan arsitektur yang mahal dibatalkan** | Agen tidak menanggung biaya migrasi keputusan yang salah, jadi usulannya cenderung optimistis terhadap perubahan besar. Membatalkan pilihan ORM atau bentuk skema di jam ke-7 berarti kehilangan seluruh sisa waktu | Ditulis manusia sebagai ADR di `docs/adr/`, dengan minimal dua alternatif yang ditolak beserta alasannya. AI boleh mengkritik draf ADR, tidak boleh memutuskan |
| **Penentuan prioritas FR yang dibuang di Gate 3** | Ini keputusan tentang risiko dan sisa waktu tim — informasi yang tidak dimiliki agen. AI cenderung menyarankan menyelesaikan semuanya, yang justru menghasilkan banyak fitur setengah jadi (penalti −5 per fitur) | Diputuskan tim bersama pada Gate 3, ditulis di `README.md` bagian 5 dengan alasan rekayasa |

---

## 5. Perbandingan Tool Antar Anggota

> Tim ini memakai tool yang berbeda-beda: 9Router/OmniRouter (Claude dan LLM lain),
> VSCode + Copilot, Hermes IDE, Antigravity IDE. Brief §9.1 menyebut hal ini **menarik**,
> bukan masalah. Bagian ini adalah kesempatan mendapatkan nilai yang tidak bisa didapat tim
> lain: satu proyek nyata, satu domain, empat tool, dan pengamatan langsung dari orang yang
> memakainya selama 9 jam.
>
> Yang membuat bagian ini bernilai adalah **kekhususannya**. "Tool A bagus untuk backend"
> tidak bernilai. "Tool A mempertahankan konteks lintas berkas sehingga cocok untuk
> perubahan yang menyentuh service + repository + test sekaligus; tool B kehilangan konteks
> di berkas ketiga sehingga kami pakai hanya untuk perubahan satu berkas" bernilai.
>
> Cara mengisinya tanpa menghabiskan waktu: setiap anggota menulis 3 baris tentang tool-nya
> saat istirahat makan, AI Workflow Officer yang merangkum. Sepuluh menit total.

### 5.1 Siapa memakai apa

| Anggota | Tool utama | Model yang dipakai | Untuk bagian apa |
|---|---|---|---|
| Luthfi | *(belum disetor)* | *(belum disetor)* | `AGENTS.md`, ADR, `approval_service.go`, `audit_service.go`, merge PR |
| Irgiyansyah | Hermes IDE | Claude Opus | Pembagian peran & pemilik berkas (DEVLOG-01), SDD BAB 4–5, skoring & margin, tabel parameter |
| Yulio Zaki | *(belum disetor)* | *(belum disetor)* | SRS, middleware auth+peran, `internal/slik/`, `mock-slik/` |
| Rayvaldo | *(belum disetor)* | *(belum disetor)* | Pengajuan, dokumen, survei, `internal/repository/`, migrasi |
| Aldi | Hermes IDE | Claude Opus | Fondasi `frontend/` (`lib/apiClient.ts`, `lib/auth.ts`, komponen bersama, `app/login`), `AI-WORKFLOW.md`, `AI-DEVLOG.md` |
| Soleh | *(belum disetor)* | *(belum disetor)* | Test dari AC, `ci.yml`, `docker-compose.yml`, `README.md`, DEMO-SCRIPT |

### 5.2 Pengamatan per tool

| Tool | Paling kuat untuk | Paling lemah untuk | Kegagalan khas yang kami amati | Cara kami menyiasatinya |
|---|---|---|---|---|
| Hermes IDE | Pekerjaan yang perlu **membaca keadaan repo lalu memverifikasi dirinya sendiri**: bisa menjalankan `git log`, `tsc`, `eslint`, `next build`, dan `curl` lalu membaca keluaran aslinya. Fondasi frontend (13 berkas) selesai dengan lint/build/type-check hijau sebelum commit, bukan diserahkan ke CI | Fakta tentang **orang** dan kepemilikan berkas. Konteksnya luas, jadi ia menyimpulkan dari prosa brief padahal jawabannya ada di `git log` | (1) DEVLOG-01: menetapkan Tech Lead yang salah dari inferensi brief §1.1 yang terdengar kuat, dan menukar pemilik SRS/SDD — keduanya tidak terdeteksi pemeriksaan otomatis karena tabelnya rapi dan konsisten secara internal. (2) Satu koreksi nama hanya diperbaiki di berkas yang sedang dibuka, bukan di semua berkas yang menyebutnya | Setiap keluaran yang menyebut nama orang atau kepemilikan berkas dicek ke `git log` + tabel riwayat `AGENTS.md` lebih dulu. Untuk koreksi nama, dilacak dengan `grep` ke seluruh repo, bukan per berkas |
| 9Router / OmniRouter | *(belum diamati)* | *(belum diamati)* | *(belum diamati)* | *(belum diamati)* |
| VSCode + Copilot | *(belum diamati)* | *(belum diamati)* | *(belum diamati)* | *(belum diamati)* |
| Antigravity IDE | *(belum diamati)* | *(belum diamati)* | *(belum diamati)* | *(belum diamati)* |

### 5.3 Uji banding kecil (opsional, tetapi bernilai)

**Status**: **belum dilakukan.** Direncanakan pada istirahat makan Jumat, dibatasi 15 menit.

**Tugas yang akan dibandingkan**: penanganan SLIK 503 + timeout (FR-05). Dipilih karena
punya jawaban benar yang tegas dan tidak bergantung selera: panggilan yang gagal **harus**
menghentikan pengajuan, dan **tidak boleh** mengisi kolektibilitas dengan nilai default atau
menganggap SLIK bersih (`AGENTS.md` Larangan 15). Jawaban salah mudah dikenali, jadi uji ini
tidak berujung pada debat rasa.

**Konteks yang akan diberikan (identik ke keduanya)**: kontrak mock SLIK `AGENTS.md`
bagian 5.2, Tabel 4.2 keluaran kolektibilitas, dan AC-05. Tanpa melampirkan kode `internal/slik/`
yang sudah ada, supaya keduanya menjawab dari requirement, bukan dari implementasi.

| Tool | Keluaran | Benar / salah | Catatan |
|---|---|---|---|
| *(belum dilakukan)* |  |  |  |
| *(belum dilakukan)* |  |  |  |

**Kesimpulan yang kami pakai untuk sisa hackathon**: *(diisi setelah uji dilakukan; kalau
sampai code freeze tidak sempat, baris ini tetap ditulis "tidak dilakukan" — tidak dikarang)*

### 5.4 Apakah `AGENTS.md` terbaca oleh semua tool?

Diuji dengan cara yang disarankan: meminta agen menyebutkan satu larangan spesifik dari
`AGENTS.md` **tanpa** melampirkan berkasnya.

| Tool | Membaca otomatis? | Nama berkas yang dibacanya | Siasat kalau tidak terbaca |
|---|---|---|---|
| Hermes IDE | **Ya** | `AGENTS.md` dan `CLAUDE.md` — keduanya dimuat sebagai konteks proyek di awal sesi, terverifikasi karena agen menyebut Larangan 13 (batas ~200 baris), Larangan 14 (`docker-compose.yml`/`ci.yml`/`AGENTS.md` lewat PR terpisah), dan Larangan 17 (aturan bisnis tidak di `httpapi`) tanpa dilampirkan | Tidak perlu siasat |
| 9Router / OmniRouter | *(belum diuji)* | — | Kalau tidak terbaca: tempel bagian 2, 3, dan 6 `AGENTS.md` ke awal prompt — ketiganya yang paling sering dilanggar |
| VSCode + Copilot | *(belum diuji)* | Kandidat: `.github/copilot-instructions.md` — **berkas ini belum ada di repo** | Buat penunjuk satu baris `.github/copilot-instructions.md` berisi "Lihat AGENTS.md". Perlu PR terpisah karena `.github/` bukan efek samping tugas fitur |
| Antigravity IDE | *(belum diuji)* | Kandidat: `AGENTS.md` (konvensi bersama, lihat `CLAUDE.md`) | Lampirkan manual kalau ternyata tidak terbaca |

**Temuan yang perlu ditindaklanjuti**: `CLAUDE.md` sudah ada sebagai penunjuk ke `AGENTS.md`,
tetapi **`.github/copilot-instructions.md` belum ada**. Anggota yang memakai Copilot berarti
bekerja tanpa aturan repo sama sekali. Ini bukan masalah teori: 17 larangan di `AGENTS.md`
bagian 6 tidak akan terlihat oleh tool itu. Perlu satu PR kecil membuat penunjuk tersebut.

---

## 6. Revisi Cara Kerja Selama Hackathon

| Kapan | Yang diubah | Pemicu (DEVLOG-xx) |
|---|---|---|
| 2026-08-20 11.00 | Setiap keluaran agen yang menyebut **nama orang atau kepemilikan berkas** wajib dicek ke `git log` dan tabel riwayat `AGENTS.md` lebih dulu — tidak boleh disimpulkan dari prosa brief | DEVLOG-01 |
| 2026-08-20 11.00 | Satu koreksi nama/kepemilikan wajib dilacak dengan `grep` ke seluruh repo, bukan hanya diperbaiki di berkas yang sedang dibuka | DEVLOG-01 |
| 2026-08-20 11.45 | Prompt "lengkapi semua placeholder" dihentikan. Diganti dengan menyebut eksplisit bagian mana yang belum punya data dan wajib ditandai `(belum disetor)` | (dicatat di berkas ini, bagian 2) |
| 2026-08-20 11.45 | Tugas besar wajib dipecah bertahap dengan gerbang verifikasi (lint/build/test) di antara tahap, bukan satu keluaran besar sekali jalan | (pola dari pembangunan fondasi frontend) |
