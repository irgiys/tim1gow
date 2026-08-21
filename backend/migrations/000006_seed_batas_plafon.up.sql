-- =============================================================================
--  000006 — seed batas plafon (FR-02, BR-01)
-- =============================================================================
--
--  Kenapa migrasi ini ada: PengajuanService.PastikanPlafonDalamBatas membaca
--  batas minimum/maksimum lewat BatasPlafonRepository. Tanpa baris di bawah,
--  setiap submit pengajuan berhenti dengan CONFIG_ERROR "batas plafon belum
--  diatur" — BR-01 tidak dapat ditegakkan maupun didemokan.
--
--  Nilainya berupa data, bukan konstanta di kode (AGENTS.md Larangan 3):
--  ADM dapat mengubah batas tanpa deploy ulang, dan test BR-01 mengubah baris
--  ini lebih dulu untuk membuktikan angkanya benar-benar dibaca (AC-15).
--
--  Angka berasal dari brief §4 BR-01: Rp 5.000.000 s.d. Rp 500.000.000.
-- =============================================================================

INSERT INTO parameter_umum (kunci, nilai, keterangan) VALUES
    ('batas_plafon_min', 5000000,   'Plafon minimum pengajuan mikro (BR-01)'),
    ('batas_plafon_maks', 500000000, 'Plafon maksimum pengajuan mikro (BR-01)')
ON CONFLICT (kunci) DO NOTHING;
