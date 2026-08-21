'use client';

import { useEffect, useState, type ReactNode } from 'react';
import { useRouter } from 'next/navigation';
import { ambilPengguna } from '@/lib/auth';
import type { Peran } from '@/lib/types';

interface GuardPeranProps {
  /** Peran yang boleh MELIHAT halaman ini. */
  izinkan: readonly Peran[];
  children: ReactNode;
}

/**
 * Membungkus halaman agar hanya ditampilkan untuk peran tertentu.
 *
 * INI BUKAN OTORISASI. Guard ini hanya mencegah tampilan yang membingungkan.
 * Penolakan yang dinilai (AC-02) adalah 403 dari server; setiap endpoint
 * memeriksa peran sendiri di backend (AGENTS.md Larangan 6).
 *
 * Cara pakai di route milik anggota lain:
 *   <GuardPeran izinkan={['ANL']}> ...isi halaman... </GuardPeran>
 */
export function GuardPeran({ izinkan, children }: GuardPeranProps) {
  const router = useRouter();
  const [keadaan, setKeadaan] = useState<'memuat' | 'boleh' | 'tolak'>('memuat');

  const izinkanKunci = izinkan.join(',');

  useEffect(() => {
    const pengguna = ambilPengguna();
    if (!pengguna) {
      router.replace('/login');
      return;
    }
    setKeadaan(izinkan.includes(pengguna.peran) ? 'boleh' : 'tolak');
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [izinkanKunci, router]);

  if (keadaan === 'memuat') {
    return <p className="sub">Memuat…</p>;
  }

  if (keadaan === 'tolak') {
    return (
      <div className="alert alert-galat" role="alert">
        Halaman ini tidak tersedia untuk peran Anda.
      </div>
    );
  }

  return <>{children}</>;
}
