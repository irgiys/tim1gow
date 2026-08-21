-- Rollback Migrasi 000004.
--
-- Urutan kebalikan dari .up.sql: tabel anak lebih dulu, lalu constraint, lalu
-- rename kolom dikembalikan, terakhir `pengguna`. Tanpa urutan ini rollback
-- gagal karena FK masih menunjuk tabel yang hendak dihapus.

DROP TABLE IF EXISTS nomor_referensi_counter;
DROP TABLE IF EXISTS notifikasi;
DROP TABLE IF EXISTS rincian_komponen_skor;
DROP TABLE IF EXISTS hasil_skoring;
DROP TABLE IF EXISTS hasil_slik;
DROP TABLE IF EXISTS survei;
DROP TABLE IF EXISTS dokumen;
DROP TABLE IF EXISTS pengajuan_anggota;

-- FK aktor pada tabel milik 000003 dilepas supaya `pengguna` dapat dihapus.
ALTER TABLE audit_trail          DROP CONSTRAINT IF EXISTS fk_audit_actor;
ALTER TABLE keputusan_approval   DROP CONSTRAINT IF EXISTS fk_keputusan_approver;
ALTER TABLE pengajuan            DROP CONSTRAINT IF EXISTS fk_pengajuan_ao;

-- Kolom tambahan pada `pengajuan` dilepas.
ALTER TABLE pengajuan
    DROP COLUMN IF EXISTS lama_usaha_bulan,
    DROP COLUMN IF EXISTS omzet_harian,
    DROP COLUMN IF EXISTS margin_atau_nisbah,
    DROP COLUMN IF EXISTS tenor_bulan,
    DROP COLUMN IF EXISTS plafon_disetujui,
    DROP COLUMN IF EXISTS jenis_akad,
    DROP COLUMN IF EXISTS jenis_usaha,
    DROP COLUMN IF EXISTS alamat_usaha,
    DROP COLUMN IF EXISTS nik,
    DROP COLUMN IF EXISTS nama_nasabah,
    DROP COLUMN IF EXISTS tipe;

-- Nama kolom dikembalikan ke bentuk 000003.
ALTER INDEX IF EXISTS idx_pengajuan_ao_id RENAME TO idx_pengajuan_created_by;

ALTER TABLE pengajuan RENAME COLUMN diperbarui_pada TO updated_at;
ALTER TABLE pengajuan RENAME COLUMN dibuat_pada     TO created_at;
ALTER TABLE pengajuan RENAME COLUMN plafon_diajukan TO total_plafon;
ALTER TABLE pengajuan RENAME COLUMN ao_id           TO created_by;

DROP TABLE IF EXISTS pengguna;
