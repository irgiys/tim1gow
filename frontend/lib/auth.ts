/**
 * Helper sesi & guard peran sisi klien.
 *
 * PERINGATAN YANG WAJIB DIBACA SEBELUM MEMAKAI BERKAS INI:
 *
 * Guard di berkas ini adalah KENYAMANAN TAMPILAN, BUKAN OTORISASI.
 * Menyembunyikan tombol bukan otorisasi (AGENTS.md Larangan 6). Penolakan yang
 * dinilai pada AC-02 adalah **403 dari server**. Jangan pernah mengandalkan
 * `bolehAkses()` sebagai satu-satunya penghalang: setiap endpoint memeriksa
 * peran sendiri di backend.
 */

'use client';

import type { Peran, Pengguna } from './types';

const KUNCI_TOKEN = 'imitra.token';
const KUNCI_PENGGUNA = 'imitra.pengguna';

export interface Sesi {
  token: string;
  pengguna: Pengguna;
}

/** Simpan sesi hasil login. Tidak memuat NIK atau data pribadi (BR-11). */
export function simpanSesi(sesi: Sesi): void {
  if (typeof window === 'undefined') return;
  window.localStorage.setItem(KUNCI_TOKEN, sesi.token);
  window.localStorage.setItem(KUNCI_PENGGUNA, JSON.stringify(sesi.pengguna));
}

export function ambilToken(): string | null {
  if (typeof window === 'undefined') return null;
  return window.localStorage.getItem(KUNCI_TOKEN);
}

export function ambilPengguna(): Pengguna | null {
  if (typeof window === 'undefined') return null;
  const mentah = window.localStorage.getItem(KUNCI_PENGGUNA);
  if (!mentah) return null;
  try {
    return JSON.parse(mentah) as Pengguna;
  } catch {
    // Data sesi rusak: bersihkan, jangan dipakai dengan nilai default.
    hapusSesi();
    return null;
  }
}

export function hapusSesi(): void {
  if (typeof window === 'undefined') return;
  window.localStorage.removeItem(KUNCI_TOKEN);
  window.localStorage.removeItem(KUNCI_PENGGUNA);
}

export function sudahLogin(): boolean {
  return ambilToken() !== null && ambilPengguna() !== null;
}

/**
 * Cek apakah peran pengguna termasuk dalam daftar yang diizinkan.
 * Hanya untuk memutuskan apa yang DITAMPILKAN. Server tetap yang memutuskan
 * apa yang DIIZINKAN.
 */
export function bolehAkses(peranDiizinkan: readonly Peran[]): boolean {
  const pengguna = ambilPengguna();
  if (!pengguna) return false;
  return peranDiizinkan.includes(pengguna.peran);
}

/** Label peran untuk ditampilkan di UI. */
export const LABEL_PERAN: Record<Peran, string> = {
  AO: 'Account Officer Mikro',
  ANL: 'Analis Mikro',
  KCP: 'Kepala Cabang Pembantu',
  KC: 'Kepala Cabang',
  KOM: 'Komite Pembiayaan',
  ADM: 'Admin',
};

/** Route beranda per peran, dipakai setelah login berhasil. */
export const BERANDA_PERAN: Record<Peran, string> = {
  AO: '/pengajuan',
  ANL: '/pengajuan',
  KCP: '/approval',
  KC: '/approval',
  KOM: '/approval',
  ADM: '/parameter',
};
