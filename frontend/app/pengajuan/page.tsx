'use client';

/**
 * Halaman FR-02, FR-03, FR-04, FR-06, FR-07, FR-08:
 * - Daftar pengajuan pembiayaan (AO, ANL, Approvers)
 * - Pembuatan pengajuan baru (AO - FR-02)
 * - Upload dokumen persyaratan & re-upload dokumen (AO - FR-03)
 * - Rekam survei lapangan OTS (AO - FR-04)
 * - Verifikasi dokumen & alasan penolakan (ANL - FR-03)
 * - Skoring kelayakan & penetapan margin (ANL - FR-06, FR-07)
 * - Ajukan ke approval berjenjang (ANL - FR-08)
 */

import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { apiClient } from '@/lib/apiClient';
import { ambilToken, ambilPengguna } from '@/lib/auth';
import { GuardPeran } from '@/components/GuardPeran';
import { TopBar } from '@/components/TopBar';
import { Table, type KolomTabel } from '@/components/Table';
import { StatusBadge } from '@/components/StatusBadge';
import { FormField } from '@/components/FormField';
import { ErrorAlert } from '@/components/ErrorAlert';
import type { JenisAkad, Peran, StatusPengajuan } from '@/lib/types';

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

interface DokumenItem {
  id: number;
  pengajuanId: number;
  jenisDokumen: string;
  status: 'PENDING' | 'VERIFIED' | 'REJECTED';
  alasanPenolakan?: string;
}

interface ResponsDokumen {
  data: DokumenItem[];
}

interface RincianKomponen {
  kode: string;
  nama: string;
  skorMentah: number;
  bobot: number;
  kontribusi: number;
}

interface HasilSkoringResponse {
  pengajuanId: number;
  skorAkhir: number;
  grade: number;
  totalBobot: number;
  gradeMinimalDipaksa: boolean;
  rincian: RincianKomponen[];
}

const rupiah = new Intl.NumberFormat('id-ID', {
  style: 'currency',
  currency: 'IDR',
  maximumFractionDigits: 0,
});

function IsiHalaman() {
  const [daftar, setDaftar] = useState<BarisPengajuan[]>([]);
  const [galatMuat, setGalatMuat] = useState<unknown>(null);
  const [sedangMuat, setSedangMuat] = useState(true);
  const [pengajuanDipilih, setPengajuanDipilih] = useState<BarisPengajuan | null>(null);
  const peran = ambilPengguna()?.peran;

  const kolom: KolomTabel<BarisPengajuan>[] = [
    { judul: 'Nomor Referensi', render: (b) => <code>{b.nomorReferensi}</code> },
    { judul: 'Nasabah', render: (b) => b.namaNasabah },
    { judul: 'Akad', render: (b) => b.jenisAkad },
    { judul: 'Plafon', angka: true, render: (b) => rupiah.format(b.plafonDiajukan) },
    { judul: 'Tenor', angka: true, render: (b) => `${b.tenorBulan} bln` },
    { judul: 'Status', render: (b) => <StatusBadge status={b.status} /> },
    {
      judul: 'Aksi / Menu',
      render: (b) => (
        <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
          {peran === 'ANL' ? (
            <>
              <Link
                href={`/pengajuan/${b.id}/slik`}
                style={{
                  padding: '0.35rem 0.65rem',
                  fontSize: '0.82rem',
                  fontWeight: 600,
                  background: 'var(--utama, #00875A)',
                  color: '#ffffff',
                  borderRadius: '4px',
                  textDecoration: 'none',
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: '0.25rem',
                }}
              >
                🔍 SLIK Check
              </Link>
              <Link
                href={`/pengajuan/${b.id}/verifikasi`}
                style={{
                  padding: '0.35rem 0.65rem',
                  fontSize: '0.82rem',
                  fontWeight: 500,
                  background: '#f1f5f9',
                  color: '#1e293b',
                  border: '1px solid #cbd5e1',
                  borderRadius: '4px',
                  textDecoration: 'none',
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: '0.25rem',
                }}
              >
                📄 Verifikasi Dokumen
              </Link>
            </>
          ) : peran === 'AO' ? (
            <Link
              href={`/pengajuan/${b.id}/dokumen`}
              style={{
                padding: '0.35rem 0.65rem',
                fontSize: '0.82rem',
                background: '#f1f5f9',
                color: '#1e293b',
                border: '1px solid #cbd5e1',
                borderRadius: '4px',
                textDecoration: 'none',
                display: 'inline-block',
              }}
            >
              📁 Dokumen
            </Link>
          ) : null}
        </div>
      ),
    },
  ];

  const muat = useCallback(async () => {
    setSedangMuat(true);
    setGalatMuat(null);
    try {
      const hasil = await apiClient.get<ResponsDaftar>('/api/pengajuan', {
        token: ambilToken(),
      });
      setDaftar(hasil.data ?? []);
      setPengajuanDipilih((prev) => {
        if (!prev) return null;
        return (hasil.data ?? []).find((p) => p.id === prev.id) ?? prev;
      });
    } catch (e) {
      setGalatMuat(e);
    } finally {
      setSedangMuat(false);
    }
  }, []);

  useEffect(() => {
    void muat();
  }, [muat]);

  const KOLOM: KolomTabel<BarisPengajuan>[] = [
    { judul: 'Nomor Referensi', render: (b) => <code>{b.nomorReferensi}</code> },
    { judul: 'Nasabah', render: (b) => b.namaNasabah },
    { judul: 'Akad', render: (b) => b.jenisAkad },
    { judul: 'Plafon', angka: true, render: (b) => rupiah.format(b.plafonDiajukan) },
    { judul: 'Tenor', angka: true, render: (b) => `${b.tenorBulan} bln` },
    { judul: 'Status', render: (b) => <StatusBadge status={b.status} /> },
    {
      judul: 'Aksi',
      render: (b) => (
        <button
          type="button"
          className="sekunder"
          style={{ padding: '0.25rem 0.6rem', fontSize: '0.82rem' }}
          onClick={() => setPengajuanDipilih(b)}
        >
          {pengajuanDipilih?.id === b.id ? 'Sedang Dipilih' : 'Kelola / Detail'}
        </button>
      ),
    },
  ];

  return (
    <>
      <h1>Pengajuan Pembiayaan</h1>
      <p className="sub">
        {peran === 'AO'
          ? 'Daftar pengajuan yang Anda buat.'
          : 'Daftar pengajuan sesuai lingkup peran Anda.'}
      </p>

      {peran === 'AO' ? <FormBuat onBerhasil={muat} /> : null}

      <div className="kartu" style={{ marginBottom: '1.5rem' }}>
        <h2>Daftar Pengajuan</h2>
        <ErrorAlert galat={galatMuat} />
        {sedangMuat ? (
          <p className="sub">Memuat…</p>
        ) : (
          <Table
            kolom={kolom}
            data={daftar}
            kunci={(b) => String(b.id)}
            pesanKosong="Belum ada pengajuan."
          />
        )}
      </div>

      {pengajuanDipilih ? (
        <PanelDetailPengajuan
          pengajuan={pengajuanDipilih}
          onTutup={() => setPengajuanDipilih(null)}
          onUpdate={muat}
        />
      ) : null}
    </>
  );
}

// ---------------------------------------------------------------------------
// Panel Detail & Aksi (Dokumen, Survei, Skoring)
// ---------------------------------------------------------------------------

function PanelDetailPengajuan({
  pengajuan,
  onTutup,
  onUpdate,
}: {
  pengajuan: BarisPengajuan;
  onTutup: () => void;
  onUpdate: () => Promise<void>;
}) {
  const peran = ambilPengguna()?.peran;
  const [tabAktif, setTabAktif] = useState<'dokumen' | 'survei' | 'skoring'>(
    peran === 'ANL' ? 'skoring' : 'dokumen',
  );

  return (
    <div className="kartu" style={{ border: '2px solid var(--utama)', scrollMarginTop: '2rem' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
        <div>
          <h2 style={{ margin: 0 }}>Kelola Pengajuan: {pengajuan.nomorReferensi}</h2>
          <p className="sub" style={{ margin: 0, marginTop: '0.25rem' }}>
            Nasabah: <strong>{pengajuan.namaNasabah}</strong> | Plafon: {rupiah.format(pengajuan.plafonDiajukan)} ({pengajuan.jenisAkad}) | Status: <StatusBadge status={pengajuan.status} />
          </p>
        </div>
        <button type="button" className="sekunder" onClick={onTutup}>
          Tutup
        </button>
      </div>

      <div className="tab-wadah">
        <button
          type="button"
          className={`tab-tombol ${tabAktif === 'dokumen' ? 'aktif' : ''}`}
          onClick={() => setTabAktif('dokumen')}
        >
          Dokumen Persyaratan (FR-03)
        </button>
        <button
          type="button"
          className={`tab-tombol ${tabAktif === 'survei' ? 'aktif' : ''}`}
          onClick={() => setTabAktif('survei')}
        >
          Survei Lapangan OTS (FR-04)
        </button>
        <button
          type="button"
          className={`tab-tombol ${tabAktif === 'skoring' ? 'aktif' : ''}`}
          onClick={() => setTabAktif('skoring')}
        >
          Skoring & Approval (FR-06/07/08)
        </button>
      </div>

      {tabAktif === 'dokumen' ? (
        <BagianDokumen pengajuanId={pengajuan.id} peran={peran} onUpdate={onUpdate} />
      ) : tabAktif === 'survei' ? (
        <BagianSurvei pengajuanId={pengajuan.id} peran={peran} />
      ) : (
        <BagianSkoring pengajuan={pengajuan} peran={peran} onUpdate={onUpdate} />
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// FR-03: Dokumen & Verifikasi
// ---------------------------------------------------------------------------

function BagianDokumen({
  pengajuanId,
  peran,
  onUpdate,
}: {
  pengajuanId: number;
  peran?: string;
  onUpdate: () => Promise<void>;
}) {
  const [dokumenList, setDokumenList] = useState<DokumenItem[]>([]);
  const [sedangMuat, setSedangMuat] = useState(true);
  const [galat, setGalat] = useState<unknown>(null);
  const [sukses, setSukses] = useState<string | null>(null);

  // Form Upload State (AO)
  const [jenisDokumen, setJenisDokumen] = useState('KTP');
  const [urlBerkas, setUrlBerkas] = useState('');
  const [sedangUpload, setSedangUpload] = useState(false);

  // Verifikasi State (ANL)
  const [sedangVerifikasi, setSedangVerifikasi] = useState<number | null>(null);
  const [alasanTolakMap, setAlasanTolakMap] = useState<Record<number, string>>({});

  const muatDokumen = useCallback(async () => {
    setSedangMuat(true);
    setGalat(null);
    try {
      const res = await apiClient.get<ResponsDokumen>(`/api/pengajuan/${pengajuanId}/dokumen`, {
        token: ambilToken(),
      });
      setDokumenList(res.data ?? []);
    } catch (e) {
      setGalat(e);
    } finally {
      setSedangMuat(false);
    }
  }, [pengajuanId]);

  useEffect(() => {
    void muatDokumen();
  }, [muatDokumen]);

  async function handleUpload(e: React.FormEvent) {
    e.preventDefault();
    if (!urlBerkas.trim()) return;
    setSedangUpload(true);
    setGalat(null);
    setSukses(null);
    try {
      await apiClient.post(
        `/api/pengajuan/${pengajuanId}/dokumen`,
        {
          jenisDokumen,
          urlBerkas: urlBerkas.trim(),
        },
        { token: ambilToken() },
      );
      setSukses(`Dokumen ${jenisDokumen} berhasil diunggah.`);
      setUrlBerkas('');
      await muatDokumen();
      await onUpdate();
    } catch (err) {
      setGalat(err);
    } finally {
      setSedangUpload(false);
    }
  }

  async function handleVerifikasi(dokId: number, setujui: boolean) {
    setSedangVerifikasi(dokId);
    setGalat(null);
    setSukses(null);
    const kodeAlasan = setujui ? '' : (alasanTolakMap[dokId] || 'DOK_TIDAK_JELAS');

    try {
      await apiClient.patch(
        `/api/pengajuan/${pengajuanId}/dokumen/${dokId}/verifikasi`,
        { setujui, kodeAlasan },
        { token: ambilToken() },
      );
      setSukses(`Dokumen ID ${dokId} berhasil di-${setujui ? 'verifikasi (Disetujui)' : 'tolak'}.`);
      await muatDokumen();
      await onUpdate();
    } catch (err) {
      setGalat(err);
    } finally {
      setSedangVerifikasi(null);
    }
  }

  return (
    <div>
      <ErrorAlert galat={galat} />
      {sukses ? (
        <div className="alert alert-baik" role="status">
          {sukses}
        </div>
      ) : null}

      <div style={{ marginBottom: '1.5rem' }}>
        <h3 style={{ fontSize: '1.05rem', margin: '0 0 0.75rem' }}>Daftar Dokumen Tersimpan</h3>
        {sedangMuat ? (
          <p className="sub">Memuat dokumen…</p>
        ) : dokumenList.length === 0 ? (
          <p className="kosong">Belum ada dokumen yang diunggah untuk pengajuan ini.</p>
        ) : (
          <div className="kotak-dokumen">
            {dokumenList.map((dok) => (
              <div key={dok.id} className="baris-dokumen">
                <div className="baris-dokumen-info">
                  <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                    <strong>{dok.jenisDokumen}</strong>
                    <span
                      className={`badge ${
                        dok.status === 'VERIFIED'
                          ? 'badge-baik'
                          : dok.status === 'REJECTED'
                          ? 'badge-buruk'
                          : 'badge-proses'
                      }`}
                    >
                      {dok.status}
                    </span>
                  </div>
                  {dok.alasanPenolakan ? (
                    <small style={{ color: 'var(--buruk)' }}>
                      Alasan Tolak: <code>{dok.alasanPenolakan}</code>
                    </small>
                  ) : null}
                </div>

                {peran === 'ANL' ? (
                  <div className="baris-dokumen-aksi">
                    <button
                      type="button"
                      disabled={sedangVerifikasi === dok.id}
                      onClick={() => handleVerifikasi(dok.id, true)}
                      style={{ padding: '0.35rem 0.75rem', fontSize: '0.85rem' }}
                    >
                      {sedangVerifikasi === dok.id ? 'Memproses…' : '✓ Setujui'}
                    </button>
                    <select
                      value={alasanTolakMap[dok.id] || 'DOK_TIDAK_JELAS'}
                      onChange={(e) =>
                        setAlasanTolakMap((prev) => ({ ...prev, [dok.id]: e.target.value }))
                      }
                      style={{ padding: '0.35rem', fontSize: '0.85rem' }}
                    >
                      <option value="DOK_TIDAK_JELAS">Buram / Tidak Terbaca</option>
                      <option value="DOK_KADALUARSA">Kadaluarsa</option>
                      <option value="DATA_TIDAK_COCOK">Data Tidak Cocok</option>
                      <option value="DOK_TIDAK_LENGKAP">Tidak Lengkap</option>
                    </select>
                    <button
                      type="button"
                      className="bahaya"
                      disabled={sedangVerifikasi === dok.id}
                      onClick={() => handleVerifikasi(dok.id, false)}
                      style={{ padding: '0.35rem 0.75rem', fontSize: '0.85rem' }}
                    >
                      ✕ Tolak
                    </button>
                  </div>
                ) : null}
              </div>
            ))}
          </div>
        )}
      </div>

      {peran === 'AO' ? (
        <div style={{ background: '#f8fafc', padding: '1rem', borderRadius: '6px', border: '1px solid var(--garis)' }}>
          <h3 style={{ fontSize: '1.05rem', margin: '0 0 0.75rem' }}>Unggah / Re-upload Dokumen (AO)</h3>
          <form onSubmit={handleUpload} noValidate>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 2fr auto', gap: '0.75rem', alignItems: 'flex-end' }}>
              <FormField label="Jenis Dokumen" htmlFor="jenisDokumen" wajib>
                <select
                  id="jenisDokumen"
                  value={jenisDokumen}
                  onChange={(e) => setJenisDokumen(e.target.value)}
                >
                  <option value="KTP">KTP (Wajib)</option>
                  <option value="KK">Kartu Keluarga (Wajib)</option>
                  <option value="SURAT_KETERANGAN_USAHA">Surat Keterangan Usaha (Wajib)</option>
                </select>
              </FormField>

              <FormField
                label="URL / Link Berkas"
                htmlFor="urlBerkas"
                wajib
                petunjuk="Contoh: https://storage.bsn.co.id/dok/ktp.jpg"
              >
                <input
                  id="urlBerkas"
                  value={urlBerkas}
                  onChange={(e) => setUrlBerkas(e.target.value)}
                  placeholder="https://..."
                  required
                />
              </FormField>

              <div style={{ marginBottom: '1rem' }}>
                <button type="submit" disabled={sedangUpload || !urlBerkas.trim()}>
                  {sedangUpload ? 'Mengunggah…' : 'Unggah Dokumen'}
                </button>
              </div>
            </div>
          </form>
        </div>
      ) : null}
    </div>
  );
}

// ---------------------------------------------------------------------------
// FR-04: Survei Lapangan (OTS)
// ---------------------------------------------------------------------------

function BagianSurvei({
  pengajuanId,
  peran,
}: {
  pengajuanId: number;
  peran?: string;
}) {
  const [latitude, setLatitude] = useState('-6.2088');
  const [longitude, setLongitude] = useState('106.8456');
  const [fotoUrl, setFotoUrl] = useState('https://storage.bsn.co.id/survei/toko-001.jpg');
  const [omzetHarian, setOmzetHarian] = useState('1500000');
  const [lamaUsahaBulan, setLamaUsahaBulan] = useState('24');
  const [catatanKondisi, setCatatanKondisi] = useState('Usaha aktif, stok memadai dan pelanggan ramai.');
  const [status, setStatus] = useState('VALID');

  const [galat, setGalat] = useState<unknown>(null);
  const [sukses, setSukses] = useState<string | null>(null);
  const [sedangKirim, setSedangKirim] = useState(false);

  async function handleSimpanSurvei(e: React.FormEvent) {
    e.preventDefault();
    setGalat(null);
    setSukses(null);
    setSedangKirim(true);

    try {
      await apiClient.post(
        `/api/pengajuan/${pengajuanId}/survei`,
        {
          latitude: Number(latitude),
          longitude: Number(longitude),
          fotoUrl: fotoUrl.trim(),
          omzetHarian: Number(omzetHarian),
          lamaUsahaBulan: Number(lamaUsahaBulan),
          catatanKondisi: catatanKondisi.trim(),
          status,
        },
        { token: ambilToken() },
      );
      setSukses('Hasil survei lapangan OTS berhasil direkam dengan status ' + status + '.');
    } catch (err) {
      setGalat(err);
    } finally {
      setSedangKirim(false);
    }
  }

  if (peran !== 'AO') {
    return (
      <div>
        <p className="sub">
          Perekaman survei lapangan On-The-Spot (OTS) merupakan wewenang Account Officer (AO).
        </p>
      </div>
    );
  }

  return (
    <div>
      <ErrorAlert galat={galat} />
      {sukses ? (
        <div className="alert alert-baik" role="status">
          {sukses}
        </div>
      ) : null}

      <form onSubmit={handleSimpanSurvei} noValidate>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem' }}>
          <FormField label="Latitude" htmlFor="latitude" wajib>
            <input
              id="latitude"
              type="number"
              step="any"
              value={latitude}
              onChange={(e) => setLatitude(e.target.value)}
              required
            />
          </FormField>

          <FormField label="Longitude" htmlFor="longitude" wajib>
            <input
              id="longitude"
              type="number"
              step="any"
              value={longitude}
              onChange={(e) => setLongitude(e.target.value)}
              required
            />
          </FormField>
        </div>

        <FormField
          label="URL Foto Bukti Usaha / Lokasi"
          htmlFor="fotoUrl"
          wajib
          petunjuk="Path foto adalah data pribadi dan tidak masuk log/URL publik (BR-11)."
        >
          <input
            id="fotoUrl"
            value={fotoUrl}
            onChange={(e) => setFotoUrl(e.target.value)}
            required
          />
        </FormField>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '1rem' }}>
          <FormField label="Estimasi Omzet Harian (Rp)" htmlFor="omzetHarian" wajib>
            <input
              id="omzetHarian"
              inputMode="numeric"
              value={omzetHarian}
              onChange={(e) => setOmzetHarian(e.target.value)}
              required
            />
          </FormField>

          <FormField label="Lama Usaha (Bulan)" htmlFor="lamaUsahaBulan" wajib>
            <input
              id="lamaUsahaBulan"
              inputMode="numeric"
              value={lamaUsahaBulan}
              onChange={(e) => setLamaUsahaBulan(e.target.value)}
              required
            />
          </FormField>

          <FormField label="Status Hasil Survei" htmlFor="statusSurvei" wajib>
            <select
              id="statusSurvei"
              value={status}
              onChange={(e) => setStatus(e.target.value)}
            >
              <option value="VALID">VALID (Memenuhi Syarat)</option>
              <option value="INVALID">INVALID (Tidak Memenuhi)</option>
            </select>
          </FormField>
        </div>

        <FormField label="Catatan Kondisi Usaha" htmlFor="catatanKondisi" wajib>
          <textarea
            id="catatanKondisi"
            rows={3}
            value={catatanKondisi}
            onChange={(e) => setCatatanKondisi(e.target.value)}
            required
          />
        </FormField>

        <button type="submit" disabled={sedangKirim}>
          {sedangKirim ? 'Menyimpan Survei…' : 'Simpan Hasil Survei (OTS)'}
        </button>
      </form>
    </div>
  );
}

// ---------------------------------------------------------------------------
// FR-06, FR-07, FR-08: Skoring, Margin, & Ajukan Approval (ANL)
// ---------------------------------------------------------------------------

function BagianSkoring({
  pengajuan,
  peran,
  onUpdate,
}: {
  pengajuan: BarisPengajuan;
  peran?: string;
  onUpdate: () => Promise<void>;
}) {
  // State SLIK
  const [hasilSlik, setHasilSlik] = useState<{
    kolektibilitas: number;
    jumlahFasilitasAktif: number;
    totalBakiDebet: number;
    tanggalData: string;
    referenceId: string;
    statusPanggilan: string;
  } | null>(null);
  const [sedangSlik, setSedangSlik] = useState(false);
  const [galatSlik, setGalatSlik] = useState<unknown>(null);
  const [suksesSlik, setSuksesSlik] = useState<string | null>(null);

  // Input Skoring
  const [angsuranBulanan, setAngsuranBulanan] = useState('1800000');
  const [omzetHarian, setOmzetHarian] = useState('1500000');
  const [lamaUsahaBulan, setLamaUsahaBulan] = useState('24');
  const [nilaiSurvei, setNilaiSurvei] = useState('85');

  // State Skoring
  const [hasilSkoring, setHasilSkoring] = useState<HasilSkoringResponse | null>(null);
  const [sedangSkoring, setSedangSkoring] = useState(false);
  const [galatSkoring, setGalatSkoring] = useState<unknown>(null);

  // State Margin
  const [nilaiMargin, setNilaiMargin] = useState('12.0');
  const [sedangMargin, setSedangMargin] = useState(false);
  const [galatMargin, setGalatMargin] = useState<unknown>(null);
  const [suksesMargin, setSuksesMargin] = useState<string | null>(null);

  // State Approval
  const [sedangAjukan, setSedangAjukan] = useState(false);
  const [galatAjukan, setGalatAjukan] = useState<unknown>(null);
  const [suksesAjukan, setSuksesAjukan] = useState<string | null>(null);

  async function handleSlikCheck() {
    setGalatSlik(null);
    setSuksesSlik(null);
    setSedangSlik(true);

    try {
      const res = await apiClient.post<{
        kolektibilitas: number;
        jumlahFasilitasAktif: number;
        totalBakiDebet: number;
        tanggalData: string;
        referenceId: string;
        statusPanggilan: string;
      }>(
        `/api/pengajuan/${pengajuan.id}/slik`,
        {},
        { token: ambilToken() },
      );
      setHasilSlik(res);
      setSuksesSlik(
        `SLIK check berhasil diproses. Kolektibilitas: ${res.kolektibilitas} (${
          res.kolektibilitas === 1
            ? 'Lancar'
            : res.kolektibilitas === 2
            ? 'Dalam Perhatian Khusus'
            : 'Macet / Ditolak'
        }).`,
      );
      await onUpdate();
    } catch (err) {
      setGalatSlik(err);
    } finally {
      setSedangSlik(false);
    }
  }

  async function handleHitungSkoring(e: React.FormEvent) {
    e.preventDefault();
    setGalatSkoring(null);
    setSedangSkoring(true);

    try {
      const res = await apiClient.post<HasilSkoringResponse>(
        `/api/pengajuan/${pengajuan.id}/skoring`,
        {
          angsuranBulanan: Number(angsuranBulanan),
          omzetHarian: Number(omzetHarian),
          lamaUsahaBulan: Number(lamaUsahaBulan),
          nilaiSurvei: Number(nilaiSurvei),
        },
        { token: ambilToken() },
      );
      setHasilSkoring(res);
      // Set default margin suggestion based on grade
      if (res.grade === 1) setNilaiMargin('12.0');
      else if (res.grade === 2) setNilaiMargin('14.0');
      else if (res.grade === 3) setNilaiMargin('16.5');
      else if (res.grade === 4) setNilaiMargin('19.0');
      await onUpdate();
    } catch (err) {
      setGalatSkoring(err);
    } finally {
      setSedangSkoring(false);
    }
  }

  async function handleSimpanMargin(e: React.FormEvent) {
    e.preventDefault();
    if (!hasilSkoring) return;
    setGalatMargin(null);
    setSuksesMargin(null);
    setSedangMargin(true);

    try {
      await apiClient.post(
        `/api/pengajuan/${pengajuan.id}/margin`,
        {
          akad: pengajuan.jenisAkad,
          grade: hasilSkoring.grade,
          nilai: Number(nilaiMargin),
        },
        { token: ambilToken() },
      );
      setSuksesMargin(`Margin ${nilaiMargin}% berhasil divalidasi dan disimpan.`);
      await onUpdate();
    } catch (err) {
      setGalatMargin(err);
    } finally {
      setSedangMargin(false);
    }
  }

  async function handleAjukanApproval() {
    setGalatAjukan(null);
    setSuksesAjukan(null);
    setSedangAjukan(true);

    try {
      await apiClient.post(
        `/api/pengajuan/${pengajuan.id}/ajukan-approval`,
        {},
        { token: ambilToken() },
      );
      setSuksesAjukan('Pengajuan berhasil dikirim ke jalur komite approval berjenjang.');
      await onUpdate();
    } catch (err) {
      setGalatAjukan(err);
    } finally {
      setSedangAjukan(false);
    }
  }

  if (peran !== 'ANL') {
    return (
      <div>
        <p className="sub">
          Tahap skoring kelayakan, penetapan margin, dan pengajuan ke approval merupakan wewenang Analis Mikro (ANL).
        </p>
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '1.5rem' }}>
      {/* 1. SLIK Check */}
      <div style={{ background: '#f8fafc', padding: '1.25rem', borderRadius: '8px', border: '1px solid var(--garis)' }}>
        <h3 style={{ fontSize: '1.1rem', margin: '0 0 0.5rem' }}>1. Pengecekan Riwayat Kredit (SLIK Check — FR-05)</h3>
        <p className="sub" style={{ margin: '0 0 1rem' }}>
          Sistem akan memeriksa riwayat kredit nasabah ke server mock SLIK secara otomatis berdasarkan NIK pemohon.
        </p>

        <ErrorAlert galat={galatSlik} />
        {suksesSlik ? (
          <div className="alert alert-baik" role="status">
            {suksesSlik}
          </div>
        ) : null}

        <button
          type="button"
          disabled={sedangSlik}
          onClick={handleSlikCheck}
          style={{ background: 'var(--utama)', padding: '0.6rem 1.25rem' }}
        >
          {sedangSlik ? 'Memeriksa SLIK…' : '🔍 Jalankan SLIK Check (FR-05)'}
        </button>

        {hasilSlik ? (
          <div style={{ marginTop: '1rem', padding: '0.85rem', background: '#fff', border: '1px solid var(--garis)', borderRadius: '6px' }}>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: '0.75rem', fontSize: '0.9rem' }}>
              <div>
                <span style={{ color: 'var(--fg-lemah)' }}>Kolektibilitas:</span>
                <div>
                  <strong className={`badge ${hasilSlik.kolektibilitas <= 2 ? 'badge-baik' : 'badge-buruk'}`}>
                    Kolektibilitas {hasilSlik.kolektibilitas}
                  </strong>
                </div>
              </div>
              <div>
                <span style={{ color: 'var(--fg-lemah)' }}>Fasilitas Aktif:</span>
                <div><strong>{hasilSlik.jumlahFasilitasAktif} fasilitas</strong></div>
              </div>
              <div>
                <span style={{ color: 'var(--fg-lemah)' }}>Total Baki Debet:</span>
                <div><strong>{rupiah.format(hasilSlik.totalBakiDebet)}</strong></div>
              </div>
              <div>
                <span style={{ color: 'var(--fg-lemah)' }}>Tanggal Data:</span>
                <div><strong>{hasilSlik.tanggalData || '-'}</strong></div>
              </div>
            </div>
          </div>
        ) : null}
      </div>

      {/* 2. Form Skoring */}
      <div style={{ background: '#f8fafc', padding: '1.25rem', borderRadius: '8px', border: '1px solid var(--garis)' }}>
        <h3 style={{ fontSize: '1.1rem', margin: '0 0 0.5rem' }}>2. Hitung Skoring Kelayakan (FR-06)</h3>
        <p className="sub" style={{ margin: '0 0 1rem' }}>
          Prasyarat BR-03: Seluruh dokumen wajib harus <code>VERIFIED</code>, minimal 1 survei <code>VALID</code>, dan SLIK check sudah dijalankan.
        </p>

        <ErrorAlert galat={galatSkoring} />

        <form onSubmit={handleHitungSkoring} noValidate>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr 1fr', gap: '0.75rem' }}>
            <FormField label="Estimasi Angsuran (Rp)" htmlFor="angsuranBulanan" wajib>
              <input
                id="angsuranBulanan"
                inputMode="numeric"
                value={angsuranBulanan}
                onChange={(e) => setAngsuranBulanan(e.target.value)}
                required
              />
            </FormField>

            <FormField label="Omzet Harian (Rp)" htmlFor="omzetHarianSkor" wajib>
              <input
                id="omzetHarianSkor"
                inputMode="numeric"
                value={omzetHarian}
                onChange={(e) => setOmzetHarian(e.target.value)}
                required
              />
            </FormField>

            <FormField label="Lama Usaha (Bulan)" htmlFor="lamaUsahaBulanSkor" wajib>
              <input
                id="lamaUsahaBulanSkor"
                inputMode="numeric"
                value={lamaUsahaBulan}
                onChange={(e) => setLamaUsahaBulan(e.target.value)}
                required
              />
            </FormField>

            <FormField label="Nilai Survei (0-100)" htmlFor="nilaiSurvei" wajib>
              <input
                id="nilaiSurvei"
                inputMode="numeric"
                value={nilaiSurvei}
                onChange={(e) => setNilaiSurvei(e.target.value)}
                required
              />
            </FormField>
          </div>

          <button type="submit" disabled={sedangSkoring}>
            {sedangSkoring ? 'Menghitung Skoring…' : 'Jalankan Skoring Kelayakan'}
          </button>
        </form>

        {hasilSkoring ? (
          <div style={{ marginTop: '1.25rem', padding: '1rem', background: '#fff', border: '1px solid var(--garis)', borderRadius: '6px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.75rem' }}>
              <div>
                <span style={{ fontSize: '1.2rem', fontWeight: 'bold' }}>
                  Skor Akhir: {hasilSkoring.skorAkhir} / 100
                </span>
                <span style={{ marginLeft: '1rem', fontSize: '1rem', fontWeight: 600 }}>
                  Grade Risiko: <span className="badge badge-baik">Grade {hasilSkoring.grade}</span>
                </span>
              </div>
            </div>

            <h4 style={{ margin: '0.5rem 0', fontSize: '0.95rem' }}>Rincian Komponen Skor (BR-08):</h4>
            <table className="tabel" style={{ width: '100%', fontSize: '0.85rem' }}>
              <thead>
                <tr>
                  <th>Komponen</th>
                  <th>Skor Mentah</th>
                  <th>Bobot</th>
                  <th>Kontribusi</th>
                </tr>
              </thead>
              <tbody>
                {hasilSkoring.rincian.map((r) => (
                  <tr key={r.kode}>
                    <td>{r.nama} ({r.kode})</td>
                    <td className="angka">{r.skorMentah.toFixed(1)}</td>
                    <td className="angka">{r.bobot}</td>
                    <td className="angka">{r.kontribusi.toFixed(2)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}
      </div>

      {/* 2. Form Margin & Approval */}
      {hasilSkoring ? (
        <div style={{ background: '#f8fafc', padding: '1.25rem', borderRadius: '8px', border: '1px solid var(--garis)' }}>
          <h3 style={{ fontSize: '1.1rem', margin: '0 0 0.5rem' }}>2. Penetapan Margin & Ajukan Approval (FR-07, FR-08)</h3>
          <p className="sub" style={{ margin: '0 0 1rem' }}>
            Tetapkan margin/nisbah sesuai rentang grade {hasilSkoring.grade}. Margin di luar rentang akan diblokir oleh sistem (BR-06).
          </p>

          <ErrorAlert galat={galatMargin} />
          {suksesMargin ? (
            <div className="alert alert-baik" role="status">
              {suksesMargin}
            </div>
          ) : null}

          <form onSubmit={handleSimpanMargin} noValidate style={{ marginBottom: '1.5rem' }}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr auto', gap: '0.75rem', alignItems: 'flex-end', maxWidth: '380px' }}>
              <FormField label={`Nilai Margin / Nisbah (${pengajuan.jenisAkad}) %`} htmlFor="nilaiMargin" wajib>
                <input
                  id="nilaiMargin"
                  type="number"
                  step="0.1"
                  value={nilaiMargin}
                  onChange={(e) => setNilaiMargin(e.target.value)}
                  required
                />
              </FormField>
              <div style={{ marginBottom: '1rem' }}>
                <button type="submit" disabled={sedangMargin}>
                  {sedangMargin ? 'Menyimpan…' : 'Validasi Margin'}
                </button>
              </div>
            </div>
          </form>

          <hr style={{ border: 'none', borderTop: '1px solid var(--garis)', margin: '1rem 0' }} />

          <div>
            <h4 style={{ margin: '0 0 0.5rem', fontSize: '1rem' }}>3. Kirim ke Komite Approval</h4>
            <p className="sub" style={{ margin: '0 0 1rem' }}>
              Setelah skoring dan margin divalidasi, ajukan pengajuan ini ke komite approval berjenjang (KCP / KC / KOM).
            </p>

            <ErrorAlert galat={galatAjukan} />
            {suksesAjukan ? (
              <div className="alert alert-baik" role="status">
                {suksesAjukan}
              </div>
            ) : null}

            <button
              type="button"
              disabled={sedangAjukan}
              onClick={handleAjukanApproval}
              style={{ background: 'var(--baik)', padding: '0.65rem 1.25rem', fontSize: '1rem' }}
            >
              {sedangAjukan ? 'Mengirim ke Approval…' : '🚀 Ajukan ke Komite Approval'}
            </button>
          </div>
        </div>
      ) : null}
    </div>
  );
}

// ---------------------------------------------------------------------------
// FR-02: Form Pembuatan Pengajuan Baru (AO)
// ---------------------------------------------------------------------------

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
          plafonDiajukan: Number(nilai.plafonDiajukan),
          tenorBulan: Number(nilai.tenorBulan),
        },
        { token: ambilToken() },
      );
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
    <div className="kartu" style={{ marginBottom: '1.5rem' }}>
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

const IZINKAN_PERAN: Peran[] = ['AO', 'ANL', 'KCP', 'KC', 'KOM'];

export default function HalamanPengajuan() {
  return (
    <>
      <TopBar />
      <GuardPeran izinkan={IZINKAN_PERAN}>
        <IsiHalaman />
      </GuardPeran>
    </>
  );
}
