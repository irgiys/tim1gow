'use client';

/**
 * Halaman FR-05 (sisi ANL): jalankan SLIK check pada satu pengajuan.
 *
 * Kontrak backend yang dipakai (httpapi — endpoint sedang dikembangkan):
 *   POST /api/pengajuan/{id}/slik  -> HasilSlik (sukses) atau ApiError (gagal)
 *
 * Catatan penting yang membentuk halaman ini:
 *
 *  - BR-04: masa berlaku SLIK (berlakuSampai) ditampilkan apa adanya dari
 *    server. Frontend TIDAK menghitung apakah sudah kedaluwarsa (Larangan 17).
 *  - BR-11: URL hanya memakai id pengajuan internal, bukan NIK. Respons SLIK
 *    tidak memuat NIK dan halaman ini tidak menampilkannya.
 *  - AC-04: pesan galat dari server ditampilkan apa adanya lewat ErrorAlert
 *    supaya kode BR tidak hilang.
 *  - 502 (SLIK tidak tersedia): ditampilkan sebagai kegagalan yang jelas, bukan
 *    seolah-olah hasilnya bersih. Tidak ada nilai default diam-diam.
 *  - Larangan 17: halaman ini TIDAK menghitung grade, kelayakan, atau keputusan
 *    apa pun dari kolektibilitas. Semua itu dihitung server.
 *  - Menyembunyikan tombol bukan otorisasi (Larangan 6): GuardPeran hanya
 *    mencegah tampilan yang membingungkan, server yang menolak peran lain.
 */

import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { apiClient } from '@/lib/apiClient';
import { ambilToken } from '@/lib/auth';
import { GuardPeran } from '@/components/GuardPeran';
import { TopBar } from '@/components/TopBar';
import { StatusBadge } from '@/components/StatusBadge';
import { ErrorAlert } from '@/components/ErrorAlert';

/**
 * Bentuk respons sukses POST /api/pengajuan/{id}/slik dari backend.
 *
 * Nilai-nilai ini ditampilkan apa adanya. Frontend TIDAK menafsirkan
 * kolektibilitas menjadi grade/keputusan — itu urusan service layer (BR-04,
 * BR-05, Larangan 17).
 */
interface HasilSlik {
  kolektibilitas: number;
  jumlahFasilitasAktif: number;
  totalBakiDebet: number;
  tanggalData: string;
  berlakuSampai: string;
  status: string;
}

export default function HalamanSlikCheck({ params }: { params: { id: string } }) {
  const pengajuanId = params.id;

  const [hasil, setHasil] = useState<HasilSlik | null>(null);
  const [galat, setGalat] = useState<unknown>(null);
  const [mengirim, setMengirim] = useState(false);

  /**
   * Jalankan SLIK check. Kegagalan (termasuk 502 SLIK tidak tersedia) dilempar
   * oleh apiClient dan ditampilkan lewat ErrorAlert — tidak pernah dianggap
   * sukses atau diisi nilai default.
   */
  const jalankan = useCallback(async () => {
    setGalat(null);
    setHasil(null);
    setMengirim(true);
    try {
      const res = await apiClient.post<HasilSlik>(
        `/api/pengajuan/${pengajuanId}/slik`,
        undefined,
        { token: ambilToken() },
      );
      setHasil(res);
    } catch (e) {
      // Kegagalan tidak pernah dianggap sukses: hasil tidak diisi diam-diam.
      setGalat(e);
    } finally {
      setMengirim(false);
    }
  }, [pengajuanId]);

  return (
    <GuardPeran izinkan={['ANL']}>
      <TopBar />
      <main>
        <div style={{ marginBottom: '1rem' }}>
          <Link href="/pengajuan" style={{ color: 'var(--utama, #00875A)', textDecoration: 'none', fontWeight: 500 }}>
            ← Kembali ke Daftar Pengajuan
          </Link>
        </div>
        <h1>SLIK Check (FR-05)</h1>
        <p className="sub">
          Pengajuan #{pengajuanId} — jalankan pengecekan SLIK untuk nasabah pada pengajuan ini.
          Hasil akan menentukan kelanjutan proses di sisi server.
        </p>

        <button type="button" onClick={() => void jalankan()} disabled={mengirim}>
          {mengirim ? 'Memproses SLIK…' : 'Jalankan SLIK Check'}
        </button>

        {/* Galat ditampilkan apa adanya dari server (AC-04), termasuk 502. */}
        {galat ? <ErrorAlert galat={galat} /> : null}

        {hasil ? <KartuHasilSlik hasil={hasil} /> : null}
      </main>
    </GuardPeran>
  );
}

/**
 * Menampilkan hasil SLIK check dalam kartu terstruktur.
 *
 * Semua nilai ditampilkan apa adanya dari server. Tidak ada interpretasi
 * kolektibilitas → grade/keputusan di sini (Larangan 17).
 */
function KartuHasilSlik({ hasil }: { hasil: HasilSlik }) {
  return (
    <section className="kartu" style={{ marginTop: '1rem' }}>
      <header
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: '0.75rem',
          marginBottom: '0.75rem',
        }}
      >
        <h2>Hasil SLIK</h2>
        <StatusBadge status={hasil.status as never} />
      </header>

      <dl style={{ display: 'grid', gridTemplateColumns: 'auto 1fr', gap: '0.5rem 1rem' }}>
        <dt>Kolektibilitas</dt>
        <dd>{hasil.kolektibilitas}</dd>

        <dt>Jumlah Fasilitas Aktif</dt>
        <dd>{hasil.jumlahFasilitasAktif}</dd>

        <dt>Total Baki Debet</dt>
        <dd>{formatRupiah(hasil.totalBakiDebet)}</dd>

        <dt>Tanggal Data</dt>
        <dd>{hasil.tanggalData}</dd>

        <dt>Berlaku Sampai</dt>
        <dd>{hasil.berlakuSampai}</dd>
      </dl>
    </section>
  );
}

/** Format angka ke mata uang Rupiah. Murni tampilan, bukan aturan bisnis. */
function formatRupiah(nilai: number): string {
  return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR' }).format(nilai);
}
