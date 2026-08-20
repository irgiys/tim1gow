'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { ambilPengguna, BERANDA_PERAN } from '@/lib/auth';

/** Root: arahkan ke beranda sesuai peran, atau ke login kalau belum masuk. */
export default function Halaman() {
  const router = useRouter();

  useEffect(() => {
    const pengguna = ambilPengguna();
    router.replace(pengguna ? BERANDA_PERAN[pengguna.peran] : '/login');
  }, [router]);

  return <p className="sub">Mengalihkan…</p>;
}
