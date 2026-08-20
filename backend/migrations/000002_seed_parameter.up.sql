-- Seed nilai awal tabel parameter, dari brief §4.1, Tabel 4.2, Tabel 4.3, dan §4.4.
--
-- IDEMPOTEN (brief §7.2 butir 5): ON CONFLICT DO NOTHING, sehingga menjalankan
-- ulang tidak menimpa perubahan yang sudah dibuat ADM lewat FR-13. Kalau seed
-- memakai DO UPDATE, demo yang mengubah bobot akan ter-reset diam-diam setelah
-- restart, dan AC-15 jadi tidak bisa dibuktikan.

INSERT INTO parameter_skoring (kode, nama, bobot, batas_1, batas_2) VALUES
    ('KAPASITAS_BAYAR', 'Kapasitas bayar',        35, 0.30, 0.60),
    ('RIWAYAT_SLIK',    'Riwayat SLIK',           25, NULL, NULL),
    ('LAMA_USAHA',      'Lama usaha',             20,   36,    6),
    ('SURVEI_LAPANGAN', 'Hasil survei lapangan',  20, NULL, NULL)
ON CONFLICT (kode) DO NOTHING;

-- Tabel 4.2: Kol-1 -> 100, Kol-2 -> 40. Kol 3-5 TIDAK di-seed: pengajuannya
-- ditolak otomatis sebelum tahap skoring (status REJECTED_SLIK).
INSERT INTO parameter_riwayat_slik (kolektibilitas, skor) VALUES
    (1, 100),
    (2,  40)
ON CONFLICT (kolektibilitas) DO NOTHING;

-- §4.4: omzet harian x 25 hari x margin usaha 30 %.
INSERT INTO parameter_umum (kunci, nilai, keterangan) VALUES
    ('hari_kerja_per_bulan', 25,   'Jumlah hari usaha berjalan per bulan untuk rumus kapasitas bayar'),
    ('margin_usaha',          0.30, 'Asumsi margin usaha nasabah mikro untuk rumus kapasitas bayar')
ON CONFLICT (kunci) DO NOTHING;

-- Tabel 4.3. Grade 5 tidak dibiayai: margin/nisbah NULL, dapat_dibiayai FALSE.
INSERT INTO rentang_margin
    (grade, skor_min, skor_maks, margin_min, margin_maks, nisbah_min, nisbah_maks, dapat_dibiayai) VALUES
    (1,  85, 100, 11.0, 13.0, 20.0, 25.0, TRUE),
    (2,  70,  84, 13.0, 15.5, 25.0, 30.0, TRUE),
    (3,  55,  69, 15.5, 18.0, 30.0, 35.0, TRUE),
    (4,  40,  54, 18.0, 21.0, 35.0, 40.0, TRUE),
    (5,   0,  39, NULL, NULL, NULL, NULL, FALSE)
ON CONFLICT (grade) DO NOTHING;

-- Tabel 4.1. Batas atas memakai rentang tertutup: 50.000.000 masuk baris pertama,
-- 50.000.001 masuk baris kedua, sesuai "> Rp 50.000.000" pada brief.
INSERT INTO ambang_approval (plafon_min, plafon_maks, level) VALUES
    (  5000000,  50000000, ARRAY['KCP']),
    ( 50000001, 200000000, ARRAY['KCP','KC']),
    (200000001, 500000000, ARRAY['KCP','KC','KOM'])
ON CONFLICT DO NOTHING;
