'use client';

import { pesanGalat } from '@/lib/apiClient';

interface ErrorAlertProps {
  /** Galat apa pun dari apiClient. Pesan diambil apa adanya dari server. */
  galat: unknown;
}

/**
 * Menampilkan pesan galat dari API TANPA menyusunnya ulang.
 * Kode BR (mis. BR-03, BR-06) wajib tetap terlihat karena AC-04 secara
 * eksplisit menguji pesan yang menyebut kode BR-nya.
 */
export function ErrorAlert({ galat }: ErrorAlertProps) {
  if (!galat) return null;
  return (
    <div className="alert alert-galat" role="alert">
      {pesanGalat(galat)}
    </div>
  );
}
