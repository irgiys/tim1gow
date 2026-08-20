-- Hanya menghapus baris hasil seed; tabelnya dihapus oleh migrasi 000001 down.
DELETE FROM ambang_approval WHERE plafon_min IN (5000000, 50000001, 200000001);
DELETE FROM rentang_margin WHERE grade BETWEEN 1 AND 5;
DELETE FROM parameter_umum WHERE kunci IN ('hari_kerja_per_bulan', 'margin_usaha');
DELETE FROM parameter_riwayat_slik WHERE kolektibilitas IN (1, 2);
DELETE FROM parameter_skoring
 WHERE kode IN ('KAPASITAS_BAYAR', 'RIWAYAT_SLIK', 'LAMA_USAHA', 'SURVEI_LAPANGAN');
