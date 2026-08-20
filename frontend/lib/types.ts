/**
 * Tipe bersama frontend iMitra.
 *
 * Sumber kebenaran enum ada di `AGENTS.md` bagian 4.1 dan `docs/SDD-iMitra.md`.
 * Menambah nilai enum baru DILARANG tanpa memperbarui kedua berkas itu
 * (`AGENTS.md` Larangan 5).
 *
 * Catatan BR-11: tipe di berkas ini sengaja TIDAK memuat NIK atau path foto pada
 * bentuk ringkas yang dipakai daftar/URL. NIK hanya muncul pada detail yang
 * di-fetch per pengajuan, dan tidak pernah dijadikan bagian path URL.
 */

/** Kode peran — dipakai persis seperti ini di kode, database, dan UI. */
export type Peran = 'AO' | 'ANL' | 'KCP' | 'KC' | 'KOM' | 'ADM';

export const SEMUA_PERAN: readonly Peran[] = ['AO', 'ANL', 'KCP', 'KC', 'KOM', 'ADM'] as const;

/** Peran yang bertindak sebagai approver berjenjang (FR-08). */
export const PERAN_APPROVER: readonly Peran[] = ['KCP', 'KC', 'KOM'] as const;

/** Jenis akad pembiayaan. */
export type JenisAkad = 'MURABAHAH' | 'MUSYARAKAH';

/**
 * Status pengajuan. Nilai DRAFT, REJECTED_SLIK, REJECTED_SCORING, APPROVED
 * berasal dari brief dan tidak boleh diganti namanya.
 */
export type StatusPengajuan =
  | 'DRAFT'
  | 'SUBMITTED'
  | 'VERIFYING'
  | 'SLIK_CHECKED'
  | 'SCORED'
  | 'WAITING_APPROVAL_L1'
  | 'WAITING_APPROVAL_L2'
  | 'WAITING_APPROVAL_L3'
  | 'RETURNED'
  | 'REJECTED_SLIK'
  | 'REJECTED_SCORING'
  | 'REJECTED'
  | 'APPROVED';

/** Status dokumen (FR-03). */
export type StatusDokumen = 'PENDING' | 'VERIFIED' | 'REJECTED';

/** Status survei lapangan (FR-04). */
export type StatusSurvei = 'DRAFT' | 'VALID' | 'INVALID';

/** Keputusan approval (FR-08). */
export type KeputusanApproval = 'APPROVE' | 'REJECT' | 'RETURN';

/** Jenis dokumen wajib (FR-03). */
export type JenisDokumen = 'KTP' | 'KARTU_KELUARGA' | 'SURAT_KETERANGAN_USAHA';

export interface Pengguna {
  id: string;
  username: string;
  namaLengkap: string;
  peran: Peran;
}

/** Bentuk ringkas untuk daftar/dashboard — tanpa data pribadi (BR-11). */
export interface PengajuanRingkas {
  id: string;
  nomorReferensi: string;
  namaNasabah: string;
  jenisAkad: JenisAkad;
  plafonDiajukan: number;
  tenorBulan: number;
  status: StatusPengajuan;
  diperbaruiPada: string;
}

export interface Dokumen {
  id: string;
  jenis: JenisDokumen;
  status: StatusDokumen;
  kodeAlasanTolak: string | null;
  diunggahPada: string;
}

export interface KomponenSkor {
  nama: string;
  bobot: number;
  skorKomponen: number;
  kontribusi: number;
}

/** Hasil skoring — rincian komponen wajib ditampilkan dan disimpan (BR-08). */
export interface HasilSkoring {
  skorAkhir: number;
  grade: number;
  komponen: KomponenSkor[];
  gradeOverride: number | null;
  alasanOverride: string | null;
}

export interface BarisAudit {
  id: string;
  waktu: string;
  aktor: string;
  peranAktor: Peran;
  aksi: string;
  statusSebelum: string | null;
  statusSesudah: string | null;
  keterangan: string | null;
}

/**
 * Bentuk respons error seragam dari API (`AGENTS.md` bagian 4.3).
 * Field `rule` memuat kode BR — jangan disusun ulang di UI, karena AC-04
 * menuntut pesan yang menyebut kode BR-nya.
 */
export interface ApiError {
  error: string;
  message: string;
  rule?: string;
}
