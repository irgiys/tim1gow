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
      <div style={{ display: 'flex', alignItems: 'center', gap: '1.5rem' }}>
        <a href="/pengajuan" className="merek" style={{ textDecoration: 'none' }}>
          iMitra
        </a>
        {pengguna ? (
          <nav style={{ display: 'flex', gap: '1rem', fontSize: '0.9rem' }}>
            <a href="/pengajuan" style={{ color: 'var(--fg)', textDecoration: 'none', fontWeight: 600 }}>
              📋 Pengajuan
            </a>
            {['KCP', 'KC', 'KOM'].includes(pengguna.peran) ? (
              <a href="/approval" style={{ color: 'var(--fg-lemah)', textDecoration: 'none' }}>
                ✍️ Approval
              </a>
            ) : null}
          </nav>
        ) : null}
      </div>
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
