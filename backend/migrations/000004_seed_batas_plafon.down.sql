-- Membalik 000004: hapus dua baris batas plafon BR-01.
DELETE FROM parameter_umum WHERE kunci IN ('plafon_minimum', 'plafon_maksimum');
