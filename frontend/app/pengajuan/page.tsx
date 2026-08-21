'use client';

/**
 * Halaman FR-02: daftar pengajuan + form pembuatan pengajuan baru.
 *
 * Ini tujuan redirect setelah login untuk AO dan ANL (lib/auth.ts
 * BERANDA_PERAN). Sebelum halaman ini ada, login yang berhasil justru berakhir
 * di 404 — jadi tidak ada satu pun alur UI yang dapat diselesaikan.
 *
 * Yang TIDAK dilakukan di sini (AGENTS.md Larangan 6 & 17):
 *  - Tidak ada aturan bisnis. Batas plafon (BR-01) ditegakkan server; halaman
 *    ini hanya menampilkan pesannya apa adanya, termasuk kode BR-nya (AC-04).
 *  - Nomor referensi TIDAK dibangkitkan di sini (Larangan 4) — ia datang dari
 *    respons server.
 *  - Tombol yang disembunyikan bukan otorisasi; form buat hanya ditampilkan
 *    untuk AO, tetapi server tetap menolak peran lain dengan 403.
 */

import { useCallback, useEffect, useState } from 'react';
import { apiClient } from '@/lib/apiClient';
import { ambilToken, ambilPengguna } from '@/lib/auth';
import { GuardPeran } from '@/components/GuardPeran';
import { TopBar } from '@/components/TopBar';
import { Table, type KolomTabel } from '@/components/Table';
import { StatusBadge } from '@/components/StatusBadge';
import { FormField } from '@/components/FormField';
import { ErrorAlert } from '@/components/ErrorAlert';
import type { JenisAkad, StatusPengajuan } from '@/lib/types';

/**
 * Bentuk baris pengajuan mengikuti respons backend apa adanya
 * (httpapi/pengajuan_handler.go: pengajuanResponse).
 *
 * Perhatikan: TIDAK ada field nik. Backend sengaja tidak mengirimkannya
 * (BR-11), jadi tipe ini pun tidak menyediakan tempatnya.
 */
interface BarisPengajuan {
  id: number;
  nomorReferensi: string;
  tipe: string;
  namaNasabah: string;
  alamatUsaha: string;
  jenisUsaha: string;
  jenisAkad: JenisAkad;
  plafonDiajukan: number;
  tenorBulan: number;
  status: StatusPengajuan;
}

interface ResponsDaftar {
  data: BarisPengajuan[];
}

const rupiah = new Intl.NumberFormat('id-ID', {
  style: 'currency',
  currency: 'IDR',
  maximumFractionDigits: 0,
});

const KOLOM: KolomTabel<BarisPengajuan>[] = [
  { judul: 'Nomor Referensi', render: (b) => <code>{b.nomorReferensi}</code> },
  { judul: 'Nasabah', render: (b) => b.namaNasabah },
  { judul: 'Akad', render: (b) => b.jenisAkad },
  { judul: 'Plafon', angka: true, render: (b) => rupiah.format(b.plafonDiajukan) },
  { judul: 'Tenor', angka: true, render: (b) => `${b.tenorBulan} bln` },
  { judul: 'Status', render: (b) => <StatusBadge status={b.status} /> },
];

function IsiHalaman() {
  const [daftar, setDaftar] = useState<BarisPengajuan[]>([]);
  const [galatMuat, setGalatMuat] = useState<unknown>(null);
  const [sedangMuat, setSedangMuat] = useState(true);
  const peran = ambilPengguna()?.peran;

  const muat = useCallback(async () => {
    setSedangMuat(true);
    setGalatMuat(null);
    try {
      const hasil = await apiClient.get<ResponsDaftar>('/api/pengajuan', {
        token: ambilToken(),
      });
      setDaftar(hasil.data ?? []);
    } catch (e) {
      // Kegagalan memuat tidak diperlakukan sebagai "daftar kosong".
      setGalatMuat(e);
    } finally {
      setSedangMuat(false);
    }
  }, []);

  useEffect(() => {
    void muat();
  }, [muat]);

  return (
    <>
      <h1>Pengajuan Pembiayaan</h1>
      <p className="sub">
        {peran === 'AO'
          ? 'Daftar pengajuan yang Anda buat.'
          : 'Daftar pengajuan sesuai lingkup peran Anda.'}
      </p>

      {peran === 'AO' ? <FormBuat onBerhasil={muat} /> : null}

      <div className="kartu">
        <h2>Daftar Pengajuan</h2>
        <ErrorAlert galat={galatMuat} />
        {sedangMuat ? (
          <p className="sub">Memuat…</p>
        ) : (
          <Table
            kolom={KOLOM}
            data={daftar}
            kunci={(b) => String(b.id)}
            pesanKosong="Belum ada pengajuan."
          />
        )}
      </div>
    </>
  );
}

const KOSONG = {
  namaNasabah: '',
  nik: '',
  alamatUsaha: '',
  jenisUsaha: '',
  jenisAkad: 'MURABAHAH' as JenisAkad,
  plafonDiajukan: '',
  tenorBulan: '12',
};

function FormBuat({ onBerhasil }: { onBerhasil: () => Promise<void> }) {
  const [nilai, setNilai] = useState(KOSONG);
  const [galat, setGalat] = useState<unknown>(null);
  const [sukses, setSukses] = useState<string | null>(null);
  const [sedangKirim, setSedangKirim] = useState(false);

  function ubah<K extends keyof typeof KOSONG>(k: K, v: (typeof KOSONG)[K]) {
    setNilai((s) => ({ ...s, [k]: v }));
  }

  async function kirim(e: React.FormEvent) {
    e.preventDefault();
    setGalat(null);
    setSukses(null);
    setSedangKirim(true);
    try {
      const hasil = await apiClient.post<BarisPengajuan>(
        '/api/pengajuan',
        {
          tipe: 'INDIVIDU',
          namaNasabah: nilai.namaNasabah,
          nik: nilai.nik,
          alamatUsaha: nilai.alamatUsaha,
          jenisUsaha: nilai.jenisUsaha,
          jenisAkad: nilai.jenisAkad,
          // Plafon dikirim sebagai angka; validasi batasnya milik server (BR-01).
          plafonDiajukan: Number(nilai.plafonDiajukan),
          tenorBulan: Number(nilai.tenorBulan),
        },
        { token: ambilToken() },
      );
      // Nomor referensi berasal dari server (BR-12, Larangan 4).
      setSukses(`Pengajuan ${hasil.nomorReferensi} berhasil dibuat.`);
      setNilai(KOSONG);
      await onBerhasil();
    } catch (e) {
      setGalat(e);
    } finally {
      setSedangKirim(false);
    }
  }

  const belumLengkap =
    !nilai.namaNasabah || !nilai.nik || !nilai.alamatUsaha || !nilai.jenisUsaha || !nilai.plafonDiajukan;

  return (
    <div className="kartu">
      <h2>Buat Pengajuan Baru</h2>

      <ErrorAlert galat={galat} />
      {sukses ? (
        <div className="alert alert-baik" role="status">
          {sukses}
        </div>
      ) : null}

      <form onSubmit={kirim} noValidate>
        <FormField label="Nama Nasabah" htmlFor="namaNasabah" wajib>
          <input
            id="namaNasabah"
            value={nilai.namaNasabah}
            onChange={(e) => ubah('namaNasabah', e.target.value)}
            required
          />
        </FormField>

        <FormField label="NIK" htmlFor="nik" wajib petunjuk="16 digit. Tidak ditampilkan kembali di daftar (BR-11).">
          <input
            id="nik"
            inputMode="numeric"
            value={nilai.nik}
            onChange={(e) => ubah('nik', e.target.value)}
            required
          />
        </FormField>

        <FormField label="Alamat Usaha" htmlFor="alamatUsaha" wajib>
          <input
            id="alamatUsaha"
            value={nilai.alamatUsaha}
            onChange={(e) => ubah('alamatUsaha', e.target.value)}
            required
          />
        </FormField>

        <FormField label="Jenis Usaha" htmlFor="jenisUsaha" wajib>
          <input
            id="jenisUsaha"
            value={nilai.jenisUsaha}
            onChange={(e) => ubah('jenisUsaha', e.target.value)}
            required
          />
        </FormField>

        <FormField label="Jenis Akad" htmlFor="jenisAkad" wajib>
          <select
            id="jenisAkad"
            value={nilai.jenisAkad}
            onChange={(e) => ubah('jenisAkad', e.target.value as JenisAkad)}
          >
            <option value="MURABAHAH">MURABAHAH</option>
            <option value="MUSYARAKAH">MUSYARAKAH</option>
          </select>
        </FormField>

        <FormField
          label="Plafon Diajukan (Rp)"
          htmlFor="plafonDiajukan"
          wajib
          petunjuk="Batas berlaku dibaca server dari tabel parameter, bukan dari halaman ini."
        >
          <input
            id="plafonDiajukan"
            inputMode="numeric"
            value={nilai.plafonDiajukan}
            onChange={(e) => ubah('plafonDiajukan', e.target.value)}
            required
          />
        </FormField>

        <FormField label="Tenor (bulan)" htmlFor="tenorBulan" wajib>
          <input
            id="tenorBulan"
            inputMode="numeric"
            value={nilai.tenorBulan}
            onChange={(e) => ubah('tenorBulan', e.target.value)}
            required
          />
        </FormField>

        <button type="submit" disabled={sedangKirim || belumLengkap}>
          {sedangKirim ? 'Mengirim…' : 'Buat Pengajuan'}
        </button>
      </form>
    </div>
  );
}

export default function HalamanPengajuan() {
  return (
    <>
      <TopBar />
      {/* Approver ikut diizinkan MELIHAT; server tetap yang membatasi aksi. */}
      <GuardPeran izinkan={['AO', 'ANL', 'KCP', 'KC', 'KOM']}>
        <IsiHalaman />
      </GuardPeran>
    </>
  );
}
