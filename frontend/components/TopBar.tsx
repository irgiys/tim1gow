'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { ambilPengguna, hapusSesi, LABEL_PERAN } from '@/lib/auth';
import type { Pengguna } from '@/lib/types';

/** Bilah atas: merek, identitas pengguna aktif, tombol keluar. */
export function TopBar() {
  const router = useRouter();
  const [pengguna, setPengguna] = useState<Pengguna | null>(null);

  useEffect(() => {
    setPengguna(ambilPengguna());
  }, []);

  function keluar() {
    hapusSesi();
    setPengguna(null);
    router.push('/login');
  }

  return (
    <header className="topbar">
      <span className="merek">iMitra</span>
      {pengguna ? (
        <span className="peran">
          {pengguna.namaLengkap} — {LABEL_PERAN[pengguna.peran]} ({pengguna.peran}){' '}
          <button type="button" className="sekunder" onClick={keluar}>
            Keluar
          </button>
        </span>
      ) : null}
    </header>
  );
}
