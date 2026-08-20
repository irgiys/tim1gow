-- Migrasi 000001: tabel parameter yang wajib berupa DATA, bukan konstanta di kode
-- (FR-13, AC-15, AGENTS.md bagian 5.1). ADM dapat mengubah isinya tanpa deploy ulang.
--
-- Nama tabel mengikuti AGENTS.md bagian 5.1: parameter_skoring, ambang_approval,
-- rentang_margin. Konvensi: snake_case, jamak/istilah domain Bahasa Indonesia.

-- Bobot & ambang tiap komponen skor kelayakan (brief §4.4).
CREATE TABLE parameter_skoring (
    kode        VARCHAR(32) PRIMARY KEY,
    nama        VARCHAR(100)  NOT NULL,
    bobot       NUMERIC(6, 2) NOT NULL CHECK (bobot >= 0),
    -- batas_1 = batas skor penuh, batas_2 = batas skor nol.
    -- Satuannya bergantung komponen: rasio untuk KAPASITAS_BAYAR, bulan untuk LAMA_USAHA.
    batas_1     NUMERIC(12, 4),
    batas_2     NUMERIC(12, 4),
    aktif       BOOLEAN       NOT NULL DEFAULT TRUE,
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_by  BIGINT
);

-- Pemetaan kolektibilitas SLIK ke skor komponen riwayat SLIK (Tabel 4.2).
-- Kol 3-5 sengaja TIDAK punya baris: pengajuannya sudah ditolak sebelum skoring,
-- dan tidak adanya baris membuat skoring BERHENTI, bukan memakai nilai default.
CREATE TABLE parameter_riwayat_slik (
    kolektibilitas SMALLINT PRIMARY KEY CHECK (kolektibilitas BETWEEN 1 AND 5),
    skor           NUMERIC(6, 2) NOT NULL CHECK (skor >= 0 AND skor <= 100),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_by     BIGINT
);

-- Parameter perhitungan bernama, mis. hari_kerja_per_bulan dan margin_usaha
-- untuk rumus kapasitas bayar. Disimpan sebagai data supaya rumusnya tidak
-- memuat angka 25 dan 0,30 di dalam kode.
CREATE TABLE parameter_umum (
    kunci       VARCHAR(64) PRIMARY KEY,
    nilai       NUMERIC(14, 4) NOT NULL,
    keterangan  TEXT,
    updated_at  TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_by  BIGINT
);

-- Rentang skor per grade + rentang margin/nisbah yang disetujui (Tabel 4.3).
CREATE TABLE rentang_margin (
    grade            SMALLINT PRIMARY KEY CHECK (grade BETWEEN 1 AND 5),
    skor_min         SMALLINT NOT NULL CHECK (skor_min >= 0 AND skor_min <= 100),
    skor_maks        SMALLINT NOT NULL CHECK (skor_maks >= 0 AND skor_maks <= 100),
    margin_min       NUMERIC(6, 2),
    margin_maks      NUMERIC(6, 2),
    nisbah_min       NUMERIC(6, 2),
    nisbah_maks      NUMERIC(6, 2),
    dapat_dibiayai   BOOLEAN  NOT NULL DEFAULT TRUE,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by       BIGINT,
    CONSTRAINT rentang_skor_urut CHECK (skor_min <= skor_maks)
);

-- Ambang approval per total plafon (Tabel 4.1). Level disimpan sebagai array
-- kode peran yang berurutan, mis. {KCP,KC}: urutan array = urutan approval (BR-02).
CREATE TABLE ambang_approval (
    id          BIGSERIAL PRIMARY KEY,
    plafon_min  BIGINT NOT NULL CHECK (plafon_min >= 0),
    plafon_maks BIGINT NOT NULL,
    level       VARCHAR(8)[] NOT NULL,
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_by  BIGINT,
    CONSTRAINT ambang_plafon_urut CHECK (plafon_min <= plafon_maks),
    CONSTRAINT ambang_level_tidak_kosong CHECK (array_length(level, 1) >= 1)
);

-- Satu total plafon tidak boleh cocok ke dua baris ambang sekaligus; kalau
-- rentangnya tumpang tindih, level approval jadi ambigu.
CREATE EXTENSION IF NOT EXISTS btree_gist;
ALTER TABLE ambang_approval
    ADD CONSTRAINT ambang_plafon_tidak_tumpang_tindih
    EXCLUDE USING gist (int8range(plafon_min, plafon_maks, '[]') WITH &&);
