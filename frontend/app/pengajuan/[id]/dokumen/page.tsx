'use client';

/**
 * Halaman FR-03 (sisi AO): unggah & unggah-ulang dokumen wajib satu pengajuan.
 *
 * Kontrak backend yang dipakai (httpapi/pengajuan_handler.go):
 *   GET  /api/pengajuan/{id}/dokumen        -> { data: dokumenResponse[] }
 *   POST /api/pengajuan/{id}/dokumen        body { jenisDokumen, urlBerkas }
 *
 * Catatan penting yang membentuk halaman ini:
 *
 *  - AC-03: dokumen yang DITOLAK dapat diunggah ulang **hanya dokumen itu**,
 *    tanpa mengisi ulang seluruh pengajuan. Karena itu satuan interaksi di sini
 *    adalah PER JENIS DOKUMEN, bukan satu form besar untuk semuanya.
 *  - BR-03 ditegakkan server: dokumen berstatus VERIFIED tidak dapat diunggah
 *    ulang. Halaman ini menonaktifkan tombolnya sebagai kenyamanan tampilan,
 *    tetapi penolakan yang sah tetap datang dari server (AGENTS.md Larangan 6).
 *  - BR-11: respons daftar sengaja TIDAK memuat urlBerkas maupun NIK. Halaman
 *    ini tidak menampilkan dan tidak menyimpan keduanya.
 *  - Tidak ada aturan bisnis di sini (Larangan 17): status, alasan penolakan,
 *    dan keputusan boleh/tidak semuanya berasal dari server. Pesan galat
 *    ditampilkan apa adanya lewat ErrorAlert supaya kode BR tidak hilang (AC-04).
 */

import { useCallback, useEffect, useState } from 'react';
import { apiClient } from '@/lib/apiClient';
import { ambilToken } from '@/lib/auth';
import { GuardPeran } from '@/components/GuardPeran';
import { TopBar } from '@/components/TopBar';
import { StatusBadge } from '@/components/StatusBadge';
import { FormField } from '@/components/FormField';
import { ErrorAlert } from '@/components/ErrorAlert';
import type { JenisDokumen } from '@/lib/types';

/**
 * Bentuk baris dokumen mengikuti `dokumenResponse` di backend apa adanya.
 * Tidak ada `urlBerkas` di sini — backend sengaja tidak mengirimkannya (BR-11).
 *
 * `status` dibiarkan `string`, bukan union tipe frontend: nilai yang benar-benar
 * dikirim backend adalah UPLOADED/VERIFIED/REJECTED (migrasi 000004 CHECK
 * constraint), sedangkan `lib/types.ts` masih menulis PENDING. Menyempitkan
 * tipe di sini hanya akan menyembunyikan ketidaksesuaian itu.
 */
interface BarisDokumen {
  id: number;
  pengajuanId: number;
  jenisDokumen: string;
  status: string;
  alasanPenolakan?: string;
}

interface ResponsDaftar {
  data: BarisDokumen[];
}

/**
 * Tiga jenis dokumen wajib menurut FR-03 (SRS §3.2, `lib/types.ts`).
 *
 * Daftar ini hanya menentukan BARIS APA YANG DITAMPILKAN supaya AO tahu apa
 * yang masih kurang. Kelengkapan yang menentukan pengajuan boleh lanjut tetap
 * dihitung server (DokumenService.JenisDokumenWajib dari DB).
 */
const JENIS_WAJIB: readonly { kode: JenisDokumen; label: string }[] = [
  { kode: 'KTP', label: 'KTP' },
  { kode: 'KARTU_KELUARGA', label: 'Kartu Keluarga' },
  { kode: 'SURAT_KETERANGAN_USAHA', label: 'Surat Keterangan Usaha' },
];

/** Status yang menutup unggah ulang. Server tetap penentu akhirnya (BR-03). */
function terkunci(status: string | undefined): boolean {
  return status === 'VERIFIED';
}

export default function HalamanDokumenPengajuan({ params }: { params: { id: string } }) {
  const pengajuanId = params.id;

  const [daftar, setDaftar] = useState<BarisDokumen[]>([]);
  const [galatMuat, setGalatMuat] = useState<unknown>(null);
  const [memuat, setMemuat] = useState(true);

  const muat = useCallback(async () => {
    setMemuat(true);
    setGalatMuat(null);
    try {
      const res = await apiClient.get<ResponsDaftar>(`/api/pengajuan/${pengajuanId}/dokumen`, {
        token: ambilToken(),
      });
      setDaftar(res.data ?? []);
    } catch (e) {
      // Kegagalan tidak pernah dianggap sukses: daftar tidak dikosongkan diam-diam.
      setGalatMuat(e);
    } finally {
      setMemuat(false);
    }
  }, [pengajuanId]);

  useEffect(() => {
    void muat();
  }, [muat]);

  /** Dokumen terakhir untuk satu jenis. Backend mengirim yang aktif per jenis. */
  const cariDokumen = (kode: string): BarisDokumen | undefined =>
    daftar.find((d) => d.jenisDokumen === kode);

  return (
    <GuardPeran izinkan={['AO']}>
      <TopBar />
      <main>
        <h1>Dokumen Pengajuan</h1>
        <p className="sub">
          Pengajuan #{pengajuanId} — unggah tiga dokumen wajib. Dokumen yang ditolak dapat diunggah
          ulang tanpa mengisi ulang pengajuan.
        </p>

        {galatMuat ? <ErrorAlert galat={galatMuat} /> : null}

        {memuat ? (
          <p className="sub">Memuat dokumen…</p>
        ) : (
          <div style={{ display: 'grid', gap: '1rem' }}>
            {JENIS_WAJIB.map(({ kode, label }) => (
              <KartuDokumen
                key={kode}
                pengajuanId={pengajuanId}
                kode={kode}
                label={label}
                dokumen={cariDokumen(kode)}
                onBerhasil={muat}
              />
            ))}
          </div>
        )}
      </main>
    </GuardPeran>
  );
}

/**
 * Satu kartu = satu jenis dokumen = satu satuan unggah/unggah-ulang (AC-03).
 * State form dipegang per kartu supaya kegagalan pada satu jenis tidak
 * menghapus masukan pada jenis lain.
 */
function KartuDokumen({
  pengajuanId,
  kode,
  label,
  dokumen,
  onBerhasil,
}: {
  pengajuanId: string;
  kode: JenisDokumen;
  label: string;
  dokumen?: BarisDokumen;
  onBerhasil: () => Promise<void>;
}) {
  const [urlBerkas, setUrlBerkas] = useState('');
  const [mengirim, setMengirim] = useState(false);
  const [galat, setGalat] = useState<unknown>(null);
  const [galatWajib, setGalatWajib] = useState<string | null>(null);

  const sudahTerkunci = terkunci(dokumen?.status);
  const idKolom = `url-${kode}`;

  async function kirim(e: React.FormEvent) {
    e.preventDefault();
    setGalat(null);
    setGalatWajib(null);

    // Validasi bentuk masukan (bukan aturan bisnis): hindari request kosong.
    if (urlBerkas.trim() === '') {
      setGalatWajib('Berkas dokumen wajib diisi.');
      return;
    }

    setMengirim(true);
    try {
      await apiClient.post(
        `/api/pengajuan/${pengajuanId}/dokumen`,
        { jenisDokumen: kode, urlBerkas: urlBerkas.trim() },
        { token: ambilToken() },
      );
      setUrlBerkas('');
      await onBerhasil();
    } catch (e) {
      setGalat(e);
    } finally {
      setMengirim(false);
    }
  }

  return (
    <section className="kartu">
      <header
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: '0.75rem',
          marginBottom: '0.75rem',
        }}
      >
        <h2>{label}</h2>
        {dokumen ? (
          <StatusBadge status={dokumen.status as never} />
        ) : (
          <span className="sub">Belum diunggah</span>
        )}
      </header>

      {/*
        Alasan penolakan ditampilkan apa adanya dari server (AC-03): AO harus
        tahu persis mengapa dokumennya ditolak agar unggahan ulang benar.
      */}
      {dokumen?.status === 'REJECTED' && dokumen.alasanPenolakan ? (
        <p className="alert alert-galat" role="alert">
          Ditolak — kode alasan: <strong>{dokumen.alasanPenolakan}</strong>
        </p>
      ) : null}

      {galat ? <ErrorAlert galat={galat} /> : null}

      {sudahTerkunci ? (
        <p className="sub">Dokumen sudah diverifikasi dan tidak dapat diunggah ulang (BR-03).</p>
      ) : (
        <form onSubmit={kirim} noValidate>
          <FormField
            label="Berkas dokumen"
            htmlFor={idKolom}
            wajib
            galat={galatWajib}
            petunjuk="Masukkan lokasi berkas hasil unggah (jpg, png, atau pdf)."
          >
            <input
              id={idKolom}
              name={idKolom}
              type="text"
              value={urlBerkas}
              onChange={(ev) => setUrlBerkas(ev.target.value)}
              disabled={mengirim}
              autoComplete="off"
            />
          </FormField>

          <button type="submit" disabled={mengirim}>
            {mengirim ? 'Mengunggah…' : dokumen ? 'Unggah ulang' : 'Unggah'}
          </button>
        </form>
      )}
    </section>
  );
}
