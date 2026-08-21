-- Migrasi 000004: Skema inti yang belum ada — pengguna, kelengkapan pengajuan,
-- dokumen, survei, hasil SLIK, hasil skoring, dan notifikasi.
--
-- KENAPA MIGRASI BARU, BUKAN MENGUBAH 000003:
-- 000003 sudah di-merge ke main, jadi ia tidak boleh disentuh (AGENTS.md
-- Larangan 2). Tabel `pengajuan` di sana sengaja sempit karena hanya melayani
-- FR-08/FR-09. Migrasi ini melengkapinya ke bentuk SDD BAB 4.1.
--
-- PENAMAAN KOLOM:
-- SDD BAB 4.1 adalah kontrak yang disepakati, jadi kolom mengikuti SDD:
-- `ao_id` (bukan created_by), `plafon_diajukan` (bukan total_plafon),
-- `dibuat_pada`/`diperbarui_pada` (bukan created_at/updated_at).
-- Kolom lama dari 000003 DIGANTI NAMA, bukan diduplikasi: dua nama untuk satu
-- konsep dilarang AGENTS.md bagian 4.1, dan kolom kembar akan berbeda isi
-- begitu ada dua jalur tulis.

-- =============================================================================
--  pengguna (FR-01)
-- =============================================================================
-- Dibuat lebih dulu karena tabel lain memasang FK ke sini.
CREATE TABLE IF NOT EXISTS pengguna (
    id            BIGSERIAL PRIMARY KEY,
    nama          VARCHAR(150) NOT NULL,
    email         VARCHAR(150) UNIQUE NOT NULL,
    -- bcrypt menghasilkan 60 karakter; 255 memberi ruang bila cost/algoritma
    -- berubah. Password plaintext tidak pernah disimpan (SDD BAB 7).
    password_hash VARCHAR(255) NOT NULL,
    peran         VARCHAR(8) NOT NULL CHECK (peran IN ('AO','ANL','KCP','KC','KOM','ADM')),
    -- Nonaktifkan tanpa menghapus baris: audit trail merujuk pengguna lewat FK
    -- dan baris yang hilang membuat jejak audit menggantung (BR-10).
    aktif         BOOLEAN NOT NULL DEFAULT TRUE,
    dibuat_pada   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_pengguna_peran ON pengguna(peran) WHERE aktif;

-- =============================================================================
--  pengajuan — lengkapi kolom yang belum ada
-- =============================================================================
ALTER TABLE pengajuan
    RENAME COLUMN created_by TO ao_id;
ALTER TABLE pengajuan
    RENAME COLUMN total_plafon TO plafon_diajukan;
ALTER TABLE pengajuan
    RENAME COLUMN created_at TO dibuat_pada;
ALTER TABLE pengajuan
    RENAME COLUMN updated_at TO diperbarui_pada;

ALTER INDEX IF EXISTS idx_pengajuan_created_by RENAME TO idx_pengajuan_ao_id;

ALTER TABLE pengajuan
    ADD COLUMN IF NOT EXISTS tipe               VARCHAR(9) NOT NULL DEFAULT 'INDIVIDU'
        CHECK (tipe IN ('INDIVIDU','KELOMPOK')),
    ADD COLUMN IF NOT EXISTS nama_nasabah       VARCHAR(150) NOT NULL DEFAULT '',
    -- NIK disimpan karena SLIK check membutuhkannya, tetapi tidak pernah boleh
    -- muncul di log, pesan error, atau URL (BR-11).
    ADD COLUMN IF NOT EXISTS nik                VARCHAR(16) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS alamat_usaha       VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS jenis_usaha        VARCHAR(100) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS jenis_akad         VARCHAR(11) NOT NULL DEFAULT 'MURABAHAH'
        CHECK (jenis_akad IN ('MURABAHAH','MUSYARAKAH')),
    ADD COLUMN IF NOT EXISTS plafon_disetujui   BIGINT CHECK (plafon_disetujui >= 0),
    ADD COLUMN IF NOT EXISTS tenor_bulan        SMALLINT NOT NULL DEFAULT 12 CHECK (tenor_bulan > 0),
    -- Nullable sampai FR-07 dijalankan; NULL berarti "belum dihitung", bukan 0.
    ADD COLUMN IF NOT EXISTS margin_atau_nisbah NUMERIC(6,2),
    ADD COLUMN IF NOT EXISTS omzet_harian       BIGINT CHECK (omzet_harian >= 0),
    ADD COLUMN IF NOT EXISTS lama_usaha_bulan   SMALLINT CHECK (lama_usaha_bulan >= 0);

-- DEFAULT di atas hanya alat bantu agar ALTER pada tabel berisi tidak gagal.
-- Dilepas supaya baris baru wajib mengisi nilainya secara sadar dari service.
ALTER TABLE pengajuan
    ALTER COLUMN tipe         DROP DEFAULT,
    ALTER COLUMN nama_nasabah DROP DEFAULT,
    ALTER COLUMN nik          DROP DEFAULT,
    ALTER COLUMN alamat_usaha DROP DEFAULT,
    ALTER COLUMN jenis_usaha  DROP DEFAULT,
    ALTER COLUMN jenis_akad   DROP DEFAULT,
    ALTER COLUMN tenor_bulan  DROP DEFAULT;

-- FK maker dipasang setelah kolom ada. ON DELETE RESTRICT: pengguna yang pernah
-- membuat pengajuan tidak boleh terhapus, karena BR-09 membandingkan maker
-- dengan approver dan BR-10 menuntut aktor selalu dapat ditelusuri.
ALTER TABLE pengajuan
    ADD CONSTRAINT fk_pengajuan_ao FOREIGN KEY (ao_id)
        REFERENCES pengguna(id) ON DELETE RESTRICT;

-- =============================================================================
--  FK aktor pada tabel yang sudah ada (BR-09, BR-10)
-- =============================================================================
ALTER TABLE keputusan_approval
    ADD CONSTRAINT fk_keputusan_approver FOREIGN KEY (approver_id)
        REFERENCES pengguna(id) ON DELETE RESTRICT;

ALTER TABLE audit_trail
    ADD CONSTRAINT fk_audit_actor FOREIGN KEY (actor_id)
        REFERENCES pengguna(id) ON DELETE RESTRICT;

-- =============================================================================
--  pengajuan_anggota (FR-10, AC-14)
-- =============================================================================
CREATE TABLE IF NOT EXISTS pengajuan_anggota (
    id             BIGSERIAL PRIMARY KEY,
    pengajuan_id   BIGINT NOT NULL REFERENCES pengajuan(id) ON DELETE CASCADE,
    nama_anggota   VARCHAR(150) NOT NULL,
    nik_anggota    VARCHAR(16) NOT NULL,
    plafon_anggota BIGINT NOT NULL CHECK (plafon_anggota >= 0),
    status_anggota VARCHAR(8) NOT NULL DEFAULT 'AKTIF'
        CHECK (status_anggota IN ('AKTIF','DITOLAK')),
    dibuat_pada    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_anggota_pengajuan ON pengajuan_anggota(pengajuan_id);

-- =============================================================================
--  dokumen (FR-03, AC-03)
-- =============================================================================
CREATE TABLE IF NOT EXISTS dokumen (
    id               BIGSERIAL PRIMARY KEY,
    pengajuan_id     BIGINT NOT NULL REFERENCES pengajuan(id) ON DELETE CASCADE,
    jenis_dokumen    VARCHAR(30) NOT NULL,
    -- Path relatif, bukan URL publik: berkas hanya boleh diakses lewat endpoint
    -- terautentikasi, dan path foto tidak boleh bocor ke log/URL (BR-11).
    url_berkas       VARCHAR(255) NOT NULL,
    status           VARCHAR(10) NOT NULL DEFAULT 'UPLOADED'
        CHECK (status IN ('UPLOADED','VERIFIED','REJECTED')),
    alasan_penolakan VARCHAR(255),
    diverifikasi_oleh BIGINT REFERENCES pengguna(id) ON DELETE RESTRICT,
    diverifikasi_pada TIMESTAMPTZ,
    dibuat_pada      TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- AC-03 menuntut kode alasan saat menolak. Constraint ini membuat aturan
    -- itu tidak bisa dilewati lewat jalur tulis mana pun, bukan hanya lewat
    -- service yang kebetulan memeriksanya.
    CONSTRAINT chk_dokumen_alasan_saat_ditolak
        CHECK (status <> 'REJECTED' OR alasan_penolakan IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_dokumen_pengajuan ON dokumen(pengajuan_id);

-- Satu jenis dokumen aktif per pengajuan. Re-upload MENGGANTI baris yang ada,
-- sehingga data pengajuan lain tidak ikut hilang (FR-03).
CREATE UNIQUE INDEX IF NOT EXISTS uq_dokumen_pengajuan_jenis
    ON dokumen(pengajuan_id, jenis_dokumen);

-- =============================================================================
--  survei (FR-04)
-- =============================================================================
CREATE TABLE IF NOT EXISTS survei (
    id            BIGSERIAL PRIMARY KEY,
    pengajuan_id  BIGINT NOT NULL REFERENCES pengajuan(id) ON DELETE CASCADE,
    ao_id         BIGINT NOT NULL REFERENCES pengguna(id) ON DELETE RESTRICT,
    latitude      NUMERIC(9,6) NOT NULL,
    longitude     NUMERIC(9,6) NOT NULL,
    foto_url      VARCHAR(255) NOT NULL,
    catatan       TEXT,
    -- Penilaian kondisi usaha skala 1-5 (komponen SURVEI_LAPANGAN pada skoring).
    nilai_kondisi SMALLINT CHECK (nilai_kondisi BETWEEN 1 AND 5),
    status        VARCHAR(11) NOT NULL DEFAULT 'DRAFT'
        CHECK (status IN ('DRAFT','VALID','TIDAK_VALID')),
    dibuat_pada   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_survei_pengajuan ON survei(pengajuan_id);

-- =============================================================================
--  hasil_slik (FR-05, BR-04)
-- =============================================================================
CREATE TABLE IF NOT EXISTS hasil_slik (
    id                     BIGSERIAL PRIMARY KEY,
    pengajuan_id           BIGINT NOT NULL REFERENCES pengajuan(id) ON DELETE CASCADE,
    -- Nullable: panggilan yang GAGAL tetap dicatat sebagai bukti percobaan,
    -- dan kolektibilitas kosong lebih jujur daripada nilai default yang
    -- membuat SLIK gagal terlihat seperti SLIK bersih (AGENTS.md Larangan 15).
    kolektibilitas         SMALLINT CHECK (kolektibilitas BETWEEN 1 AND 5),
    jumlah_fasilitas_aktif SMALLINT CHECK (jumlah_fasilitas_aktif >= 0),
    total_baki_debet       BIGINT CHECK (total_baki_debet >= 0),
    tanggal_data           DATE,
    reference_id           VARCHAR(50),
    status_panggilan       VARCHAR(20) NOT NULL
        CHECK (status_panggilan IN ('SUKSES','NIK_NOT_FOUND','SERVICE_UNAVAILABLE','TIMEOUT')),
    -- tanggal_data + 30 hari (BR-04). Disimpan sebagai kolom, bukan dihitung di
    -- kode, supaya masa berlaku dapat diperiksa lewat query.
    berlaku_sampai         DATE,
    dicek_oleh             BIGINT REFERENCES pengguna(id) ON DELETE RESTRICT,
    dibuat_pada            TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Panggilan sukses wajib membawa kolektibilitas. Tanpa constraint ini,
    -- baris SUKSES berkolektibilitas NULL akan lolos ke skoring.
    CONSTRAINT chk_slik_sukses_punya_kolektibilitas
        CHECK (status_panggilan <> 'SUKSES' OR kolektibilitas IS NOT NULL)
);

-- Riwayat disimpan seluruhnya (SLIK ulang setelah 30 hari menambah baris baru),
-- jadi indeks diurutkan menurun untuk mengambil hasil terbaru.
CREATE INDEX IF NOT EXISTS idx_slik_pengajuan ON hasil_slik(pengajuan_id, dibuat_pada DESC);

-- =============================================================================
--  hasil_skoring & rincian_komponen_skor (FR-06, BR-07, BR-08)
-- =============================================================================
CREATE TABLE IF NOT EXISTS hasil_skoring (
    id              BIGSERIAL PRIMARY KEY,
    pengajuan_id    BIGINT NOT NULL UNIQUE REFERENCES pengajuan(id) ON DELETE CASCADE,
    skor_akhir      SMALLINT NOT NULL CHECK (skor_akhir BETWEEN 0 AND 100),
    grade           SMALLINT NOT NULL CHECK (grade BETWEEN 1 AND 5),
    dihitung_oleh   BIGINT NOT NULL REFERENCES pengguna(id) ON DELETE RESTRICT,
    -- Override ANL (AC-08). Grade hasil hitung tetap disimpan di `grade`;
    -- kolom ini merekam siapa yang mengubah dan alasannya.
    grade_override  SMALLINT CHECK (grade_override BETWEEN 1 AND 5),
    override_oleh   BIGINT REFERENCES pengguna(id) ON DELETE RESTRICT,
    alasan_override VARCHAR(255),
    dibuat_pada     TIMESTAMPTZ NOT NULL DEFAULT now(),
    diperbarui_pada TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- AC-08: alasan WAJIB saat override. Ditegakkan di database, bukan hanya
    -- di service, supaya tidak ada jalur tulis yang bisa melewatinya.
    CONSTRAINT chk_skoring_override_wajib_beralasan
        CHECK (grade_override IS NULL OR (override_oleh IS NOT NULL AND alasan_override IS NOT NULL))
);

CREATE TABLE IF NOT EXISTS rincian_komponen_skor (
    id               BIGSERIAL PRIMARY KEY,
    hasil_skoring_id BIGINT NOT NULL REFERENCES hasil_skoring(id) ON DELETE CASCADE,
    kode_komponen    VARCHAR(32) NOT NULL REFERENCES parameter_skoring(kode) ON DELETE RESTRICT,
    nilai_input      NUMERIC(14,4) NOT NULL,
    skor_komponen    NUMERIC(6,2) NOT NULL,
    -- Snapshot bobot SAAT dihitung. Bobot yang diubah ADM kemudian tidak boleh
    -- mengubah rincian hasil skoring lama (BR-08).
    bobot_dipakai    NUMERIC(6,2) NOT NULL,

    CONSTRAINT uq_rincian_per_komponen UNIQUE (hasil_skoring_id, kode_komponen)
);

CREATE INDEX IF NOT EXISTS idx_rincian_hasil ON rincian_komponen_skor(hasil_skoring_id);

-- =============================================================================
--  notifikasi (FR-11)
-- =============================================================================
CREATE TABLE IF NOT EXISTS notifikasi (
    id           BIGSERIAL PRIMARY KEY,
    pengguna_id  BIGINT NOT NULL REFERENCES pengguna(id) ON DELETE CASCADE,
    pengajuan_id BIGINT REFERENCES pengajuan(id) ON DELETE CASCADE,
    -- Pesan tidak boleh memuat NIK atau path foto (BR-11); pakai nomor referensi.
    pesan        VARCHAR(255) NOT NULL,
    dibaca       BOOLEAN NOT NULL DEFAULT FALSE,
    dibuat_pada  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_notifikasi_pengguna
    ON notifikasi(pengguna_id, dibuat_pada DESC) WHERE NOT dibaca;

-- =============================================================================
--  nomor_referensi_counter (BR-12)
-- =============================================================================
-- Counter harian untuk format IMT-YYYYMMDD-NNNN. Service menaikkannya di dalam
-- transaksi dengan SELECT ... FOR UPDATE, sehingga nomor tidak pernah dipakai
-- ulang walaupun pengajuannya kemudian ditolak.
CREATE TABLE IF NOT EXISTS nomor_referensi_counter (
    tanggal     DATE PRIMARY KEY,
    urutan_akhir INTEGER NOT NULL DEFAULT 0 CHECK (urutan_akhir >= 0)
);
