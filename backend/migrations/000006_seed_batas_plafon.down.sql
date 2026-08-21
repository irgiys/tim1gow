-- Turunkan 000006: hapus baris batas plafon.
DELETE FROM parameter_umum WHERE kunci IN ('batas_plafon_min', 'batas_plafon_maks');
