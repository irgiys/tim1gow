'use client';

/**
 * Halaman FR-03 (sisi ANL): verifikasi / tolak dokumen dengan kode alasan.
 *
 * Kontrak backend yang dipakai (httpapi/pengajuan_handler.go):
 *   GET   /api/pengajuan/{id}/dokumen                          -> { data: dokumenResponse[] }
 *   PATCH /api/pengajuan/{id}/dokumen/{dokId}/verifikasi        body { setujui, kodeAlasan }
 *
 * Catatan penting yang membentuk halaman ini:
 *
 *  - AC-03: penolakan WAJIB menyertakan kode alasan. Aturan itu ditegakkan di
 *    DokumenService dan CHECK constraint migrasi. Halaman ini juga mencegah
 *    kirim-kosong, tapi itu semata mengurangi bolak-balik — 400 dari server
 *    tetap jalur yang benar dan pesannya ditampilkan apa adanya (AC-04).
 *  - BR-09 (maker != checker) pada tahap dokumen diwujudkan lewat pemisahan
 *    peran di router: AO mengunggah, ANL memverifikasi. Frontend tidak
 *    menghitung ulang aturan itu (AGENTS.md Larangan 17).
 *  - BR-11: respons tidak memuat urlBerkas maupun NIK, jadi halaman ini tidak
 *    menampilkan pratinjau berkas. Viewer dokumen butuh endpoint terautentikasi
 *    tersendiri yang belum ada di backend — lihat catatan di bawah.
 *  - Menyembunyikan tombol bukan otorisasi (Larangan 6): GuardPeran hanya
 *    mencegah tampilan yang membingungkan.
 */

import { useCallback, useEffect, useState } from 'react';
import { apiClient } from '@/lib/apiClient';
import { ambilToken } from '@/lib/auth';
import { GuardPeran } from '@/components/GuardPeran';
import { TopBar } from '@/components/TopBar';
import { StatusBadge } from '@/components/StatusBadge';
import { FormField } from '@/components/FormField';
import { ErrorAlert } from '@/components/ErrorAlert';

/** Mengikuti `dokumenResponse` backend apa adanya. Tanpa urlBerkas (BR-11). */
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
 * Kode alasan penolakan.
 *
 * SRS/SDD mewajibkan "kode alasan" tetapi TIDAK mendefinisikan daftar nilainya,
 * dan backend menerima string bebas (`verifikasiDokumenRequest.KodeAlasan`).
 * Daftar di bawah adalah usulan frontend agar kode konsisten dan bukan teks
 * bebas yang berbeda tiap ANL; nilainya masih perlu dikonfirmasi ke Tech Lead
 * dan dipindahkan ke tabel parameter kalau harus dapat dikelola ADM.
 */
const KODE_ALASAN: readonly { kode: string; label: string }[] = [
  { kode: 'TIDAK_TERBACA', label: 'Berkas tidak terbaca / buram' },
  { kode: 'TIDAK_SESUAI', label: 'Dokumen tidak sesuai jenis' },
  { kode: 'KEDALUWARSA', label: 'Dokumen kedaluwarsa' },
  { kode: 'DATA_TIDAK_COCOK', label: 'Data tidak cocok dengan pengajuan' },
  { kode: 'TIDAK_LENGKAP', label: 'Halaman tidak lengkap' },
];

export default function HalamanVerifikasiDokumen({ params }: { params: { id: string } }) {
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
      setGalatMuat(e);
    } finally {
      setMemuat(false);
    }
  }, [pengajuanId]);

  useEffect(() => {
    void muat();
  }, [muat]);

  return (
    <GuardPeran izinkan={['ANL']}>
      <TopBar />
      <main>
        <h1>Verifikasi Dokumen</h1>
        <p className="sub">
          Pengajuan #{pengajuanId} — tandai setiap dokumen sebagai terverifikasi, atau tolak dengan
          kode alasan agar AO dapat mengunggah ulang dokumen itu saja.
        </p>

        {galatMuat ? <ErrorAlert galat={galatMuat} /> : null}

        {memuat ? (
          <p className="sub">Memuat dokumen…</p>
        ) : daftar.length === 0 ? (
          <p className="sub">Belum ada dokumen yang diunggah AO untuk pengajuan ini.</p>
        ) : (
          <div style={{ display: 'grid', gap: '1rem' }}>
            {daftar.map((d) => (
              <KartuVerifikasi key={d.id} pengajuanId={pengajuanId} dokumen={d} onBerhasil={muat} />
            ))}
          </div>
        )}
      </main>
    </GuardPeran>
  );
}

/** Satu kartu = satu dokumen = satu keputusan verifikasi. */
function KartuVerifikasi({
  pengajuanId,
  dokumen,
  onBerhasil,
}: {
  pengajuanId: string;
  dokumen: BarisDokumen;
  onBerhasil: () => Promise<void>;
}) {
  const [kodeAlasan, setKodeAlasan] = useState('');
  const [mengirim, setMengirim] = useState<'setuju' | 'tolak' | null>(null);
  const [galat, setGalat] = useState<unknown>(null);
  const [galatWajib, setGalatWajib] = useState<string | null>(null);

  // Dokumen yang sudah diputuskan tidak diberi kontrol lagi; keputusan ulang
  // bukan alur yang didefinisikan FR-03 (dan BR-03 melarang unggah ulang
  // dokumen VERIFIED). Server tetap penentu akhirnya.
  const sudahDiputuskan = dokumen.status === 'VERIFIED' || dokumen.status === 'REJECTED';
  const idKolom = `alasan-${dokumen.id}`;

  async function putuskan(setujui: boolean) {
    setGalat(null);
    setGalatWajib(null);

    if (!setujui && kodeAlasan === '') {
      // Server juga menolak ini dengan 400 (AC-03); dicegah lebih awal saja.
      setGalatWajib('Penolakan wajib menyertakan kode alasan.');
      return;
    }

    setMengirim(setujui ? 'setuju' : 'tolak');
    try {
      await apiClient.patch(
        `/api/pengajuan/${pengajuanId}/dokumen/${dokumen.id}/verifikasi`,
        { setujui, kodeAlasan: setujui ? '' : kodeAlasan },
        { token: ambilToken() },
      );
      setKodeAlasan('');
      await onBerhasil();
    } catch (e) {
      setGalat(e);
    } finally {
      setMengirim(null);
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
        <h2>{dokumen.jenisDokumen}</h2>
        <StatusBadge status={dokumen.status as never} />
      </header>

      {dokumen.status === 'REJECTED' && dokumen.alasanPenolakan ? (
        <p className="sub">
          Kode alasan tersimpan: <strong>{dokumen.alasanPenolakan}</strong>
        </p>
      ) : null}

      {galat ? <ErrorAlert galat={galat} /> : null}

      {sudahDiputuskan ? (
        <p className="sub">Keputusan sudah tercatat.</p>
      ) : (
        <>
          <FormField
            label="Kode alasan penolakan"
            htmlFor={idKolom}
            galat={galatWajib}
            petunjuk="Wajib dipilih hanya bila dokumen ditolak."
          >
            <select
              id={idKolom}
              name={idKolom}
              value={kodeAlasan}
              onChange={(ev) => setKodeAlasan(ev.target.value)}
              disabled={mengirim !== null}
            >
              <option value="">— pilih kode alasan —</option>
              {KODE_ALASAN.map(({ kode, label }) => (
                <option key={kode} value={kode}>
                  {kode} — {label}
                </option>
              ))}
            </select>
          </FormField>

          <div className="aksi-baris">
            <button type="button" onClick={() => void putuskan(true)} disabled={mengirim !== null}>
              {mengirim === 'setuju' ? 'Memproses…' : 'Verifikasi'}
            </button>
            <button
              type="button"
              className="bahaya"
              onClick={() => void putuskan(false)}
              disabled={mengirim !== null}
            >
              {mengirim === 'tolak' ? 'Memproses…' : 'Tolak'}
            </button>
          </div>
        </>
      )}
    </section>
  );
}
