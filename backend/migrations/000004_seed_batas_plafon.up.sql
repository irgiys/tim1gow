-- Batas plafon BR-01 sebagai DATA, bukan konstanta di kode.
--
-- BR-01 mensyaratkan plafon < Rp 5.000.000 atau > Rp 500.000.000 ditolak saat
-- submit. Angkanya tidak boleh ditulis sebagai konstanta di service (AGENTS.md
-- Larangan 3), jadi ia hidup di parameter_umum dan dibaca setiap kali submit —
-- sekaligus membuat batasnya dapat diubah ADM tanpa deploy ulang (FR-13).
--
-- Migrasi BARU, bukan perubahan 000002 yang sudah di-merge (Larangan 2).
--
-- IDEMPOTEN: ON CONFLICT DO NOTHING, supaya perubahan ADM tidak ter-reset saat
-- restart (Larangan 19).

INSERT INTO parameter_umum (kunci, nilai, keterangan) VALUES
    ('plafon_minimum',    5000000, 'BR-01: batas bawah total plafon yang dapat diajukan'),
    ('plafon_maksimum', 500000000, 'BR-01: batas atas total plafon yang dapat diajukan')
ON CONFLICT (kunci) DO NOTHING;
