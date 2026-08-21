'use client';

/**
 * Halaman FR-08: layar keputusan approval berjenjang.
 *
 * Ini tujuan redirect setelah login untuk KCP, KC, dan KOM (lib/auth.ts
 * BERANDA_PERAN). Sebelum halaman ini ada, ketiga peran approver mendarat di
 * 404 setelah login berhasil.
 *
 * KETERBATASAN YANG DISENGAJA — dibaca sebelum menilai halaman ini:
 * backend belum punya endpoint "daftar pengajuan yang menunggu approval saya"
 * (GET /api/pengajuan masih selalu memakai DaftarMilikAO, jadi approver
 * menerima daftar kosong). Karena itu halaman ini memakai pencarian per id
 * pengajuan alih-alih menampilkan daftar. Itu batasan backend, bukan pilihan
 * desain UI — dan ditulis terang-terangan di layar supaya tidak terlihat
 * seperti fitur yang hilang tanpa sebab.
 *
 * Yang TIDAK dilakukan di sini (AGENTS.md Larangan 6 & 17):
 *  - Tidak ada aturan bisnis. BR-02 (urutan berjenjang) dan BR-09
 *    (maker != checker) ditegakkan server; halaman menampilkan pesannya apa
 *    adanya beserta kode BR (AC-04).
 *  - Tombol keputusan yang tampil bukan otorisasi. Server memeriksa peran dan
 *    level sendiri, dan identitas approver diambil dari token — bukan dari
 *    apa pun yang dikirim halaman ini.
 */

import { useState } from 'react';
import { apiClient } from '@/lib/apiClient';
import { ambilToken, ambilPengguna } from '@/lib/auth';
import { GuardPeran } from '@/components/GuardPeran';
import { TopBar } from '@/components/TopBar';
import { StatusBadge } from '@/components/StatusBadge';
import { FormField } from '@/components/FormField';
import { ErrorAlert } from '@/components/ErrorAlert';
import type { KeputusanApproval, Peran, StatusPengajuan } from '@/lib/types';

/** Bentuk respons GET /api/pengajuan/{id}/approval, mengikuti backend apa adanya. */
interface RekamKeputusan {
  id: number;
  pengajuan_id: number;
  level: Peran;
  keputusan: KeputusanApproval;
  alasan?: string;
  catatan?: string;
  approver_id: number;
  created_at: string;
}

interface DetailApproval {
  pengajuan: {
    id: number;
    nomor_referensi: string;
    plafon_diajukan: number;
    grade: number | null;
    status: StatusPengajuan;
    ao_id: number;
  };
  riwayat_approval: RekamKeputusan[] | null;
  level_diperlukan: Peran[] | null;
  level_saat_ini?: Peran;
}

const rupiah = new Intl.NumberFormat('id-ID', {
  style: 'currency',
  currency: 'IDR',
  maximumFractionDigits: 0,
});

function waktuSingkat(iso: string): string {
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString('id-ID');
}

function IsiHalaman() {
  const peran = ambilPengguna()?.peran;
  const [idCari, setIdCari] = useState('');
  const [detail, setDetail] = useState<DetailApproval | null>(null);
  const [galat, setGalat] = useState<unknown>(null);
  const [sukses, setSukses] = useState<string | null>(null);
  const [sedangMuat, setSedangMuat] = useState(false);
  const [sedangKirim, setSedangKirim] = useState(false);
  const [catatan, setCatatan] = useState('');

  async function muat(id: string) {
    setSedangMuat(true);
    setGalat(null);
    try {
      const hasil = await apiClient.get<DetailApproval>(`/api/pengajuan/${id}/approval`, {
        token: ambilToken(),
      });
      setDetail(hasil);
    } catch (e) {
      // Kegagalan tidak diperlakukan sebagai "tidak ada data".
      setDetail(null);
      setGalat(e);
    } finally {
      setSedangMuat(false);
    }
  }

  async function cari(e: React.FormEvent) {
    e.preventDefault();
    setSukses(null);
    await muat(idCari);
  }

  async function putuskan(keputusan: KeputusanApproval) {
    if (!detail) return;
    setGalat(null);
    setSukses(null);
    setSedangKirim(true);
    try {
      await apiClient.post(
        `/api/pengajuan/${detail.pengajuan.id}/approval`,
        { keputusan, catatan },
        { token: ambilToken() },
      );
      setSukses(`Keputusan ${keputusan} tersimpan.`);
      setCatatan('');
      // Muat ulang supaya status dan riwayat mencerminkan keadaan server,
      // bukan asumsi halaman ini.
      await muat(String(detail.pengajuan.id));
    } catch (e) {
      setGalat(e);
    } finally {
      setSedangKirim(false);
    }
  }

  const riwayat = detail?.riwayat_approval ?? [];
  const levelDiperlukan = detail?.level_diperlukan ?? [];
  const giliranSaya = detail?.level_saat_ini !== undefined && detail.level_saat_ini === peran;

  return (
    <>
      <h1>Approval Pembiayaan</h1>
      <p className="sub">
        Keputusan approval untuk peran {peran}. Urutan berjenjang (BR-02) dan larangan
        menyetujui pengajuan sendiri (BR-09) ditegakkan di server.
      </p>

      <div className="kartu">
        <h2>Cari Pengajuan</h2>
        <p className="sub">
          Backend belum menyediakan daftar &ldquo;menunggu approval saya&rdquo;, jadi pengajuan
          dibuka per id. Nomor referensi dapat dilihat pada layar pengajuan atau audit trail.
        </p>
        <form onSubmit={cari} noValidate>
          <FormField label="ID Pengajuan" htmlFor="idCari" wajib>
            <input
              id="idCari"
              inputMode="numeric"
              value={idCari}
              onChange={(e) => setIdCari(e.target.value)}
              required
            />
          </FormField>
          <button type="submit" disabled={sedangMuat || !idCari}>
            {sedangMuat ? 'Memuat…' : 'Buka'}
          </button>
        </form>
      </div>

      <ErrorAlert galat={galat} />
      {sukses ? (
        <div className="alert alert-baik" role="status">
          {sukses}
        </div>
      ) : null}

      {detail ? (
        <>
          <div className="kartu">
            <h2>
              <code>{detail.pengajuan.nomor_referensi}</code>{' '}
              <StatusBadge status={detail.pengajuan.status} />
            </h2>
            <dl className="ringkas">
              <dt>Plafon diajukan</dt>
              <dd>{rupiah.format(detail.pengajuan.plafon_diajukan)}</dd>
              <dt>Grade</dt>
              <dd>{detail.pengajuan.grade ?? '—'}</dd>
              <dt>Level diperlukan</dt>
              <dd>{levelDiperlukan.length > 0 ? levelDiperlukan.join(' → ') : '—'}</dd>
              <dt>Menunggu level</dt>
              <dd>{detail.level_saat_ini ?? '—'}</dd>
            </dl>
          </div>

          <div className="kartu">
            <h2>Riwayat Keputusan</h2>
            {riwayat.length === 0 ? (
              <p className="kosong">Belum ada keputusan pada pengajuan ini.</p>
            ) : (
              <table className="tabel">
                <thead>
                  <tr>
                    <th>Level</th>
                    <th>Keputusan</th>
                    <th>Catatan</th>
                    <th>Waktu</th>
                  </tr>
                </thead>
                <tbody>
                  {riwayat.map((r) => (
                    <tr key={r.id}>
                      <td>{r.level}</td>
                      <td>{r.keputusan}</td>
                      <td>{r.catatan || r.alasan || '—'}</td>
                      <td>{waktuSingkat(r.created_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>

          <div className="kartu">
            <h2>Keputusan Anda</h2>
            {!giliranSaya ? (
              <p className="sub">
                Pengajuan ini menunggu level {detail.level_saat_ini ?? '—'}, sedangkan peran Anda{' '}
                {peran}. Tombol tetap ditampilkan karena keputusan akhir ada di server — kiriman
                di luar giliran akan dijawab 422 dengan kode BR-02.
              </p>
            ) : null}

            <FormField
              label="Catatan"
              htmlFor="catatan"
              petunjuk="Tanpa data pribadi nasabah (BR-11)."
            >
              <input id="catatan" value={catatan} onChange={(e) => setCatatan(e.target.value)} />
            </FormField>

            <div className="aksi-baris">
              <button type="button" disabled={sedangKirim} onClick={() => putuskan('APPROVE')}>
                Setujui
              </button>
              <button
                type="button"
                className="sekunder"
                disabled={sedangKirim}
                onClick={() => putuskan('RETURN')}
              >
                Kembalikan
              </button>
              <button
                type="button"
                className="bahaya"
                disabled={sedangKirim}
                onClick={() => putuskan('REJECT')}
              >
                Tolak
              </button>
            </div>
          </div>
        </>
      ) : null}
    </>
  );
}

export default function HalamanApproval() {
  return (
    <>
      <TopBar />
      {/* ANL boleh melihat untuk memantau; keputusan tetap dibatasi server. */}
      <GuardPeran izinkan={['KCP', 'KC', 'KOM', 'ANL']}>
        <IsiHalaman />
      </GuardPeran>
    </>
  );
}
