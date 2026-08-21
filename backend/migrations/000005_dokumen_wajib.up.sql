-- =============================================================================
--  000005 — dokumen_wajib (FR-03, BR-03)
-- =============================================================================
--
--  Kenapa tabel ini ada: guard BR-03 ("semua dokumen wajib VERIFIED") tidak
--  dapat dievaluasi dari database selama daftar jenis dokumen wajib hanya
--  hidup sebagai konstanta di kode Go. Sebelum migrasi ini,
--  service.DokumenWajibRepository sama sekali tidak punya implementasi
--  produksi — hanya fake di test — sehingga BR-03 di jalur HTTP terpaksa
--  mempercayai klaim klien.
--
--  Daftarnya berupa data supaya ADM dapat mengubahnya tanpa deploy ulang
--  (AGENTS.md Larangan 3, FR-13).
-- =============================================================================

CREATE TABLE IF NOT EXISTS dokumen_wajib (
    jenis_dokumen VARCHAR(30) PRIMARY KEY,
    keterangan    TEXT,
    aktif         BOOLEAN     NOT NULL DEFAULT TRUE,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by    BIGINT
);

-- Kode jenis berkas, bukan ambang/bobot aturan bisnis. Nilainya sama dengan
-- konstanta JenisDokumen* di internal/service/pengajuan_repository.go.
--
-- DO NOTHING, bukan DO UPDATE (AGENTS.md Larangan 19): kalau ADM menonaktifkan
-- salah satu jenis saat demo, restart tidak boleh diam-diam mengaktifkannya
-- kembali — AC-15 justru menjadi tidak dapat dibuktikan.
INSERT INTO dokumen_wajib (jenis_dokumen, keterangan) VALUES
    ('KTP',                    'Kartu Tanda Penduduk pemohon'),
    ('KK',                     'Kartu Keluarga pemohon'),
    ('SURAT_KETERANGAN_USAHA', 'Surat keterangan usaha dari kelurahan/desa')
ON CONFLICT (jenis_dokumen) DO NOTHING;
