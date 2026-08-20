'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { apiClient } from '@/lib/apiClient';
import { simpanSesi, BERANDA_PERAN } from '@/lib/auth';
import { FormField } from '@/components/FormField';
import { ErrorAlert } from '@/components/ErrorAlert';
import type { Pengguna } from '@/lib/types';

interface ResponsLogin {
  token: string;
  pengguna: Pengguna;
}

export default function HalamanLogin() {
  const router = useRouter();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [galat, setGalat] = useState<unknown>(null);
  const [sedangKirim, setSedangKirim] = useState(false);

  async function kirim(e: React.FormEvent) {
    e.preventDefault();
    setGalat(null);
    setSedangKirim(true);
    try {
      // Kontrak endpoint: docs/SDD-iMitra.md BAB 5 (FR-01).
      const hasil = await apiClient.post<ResponsLogin>('/api/auth/login', {
        username,
        password,
      });
      simpanSesi({ token: hasil.token, pengguna: hasil.pengguna });
      router.push(BERANDA_PERAN[hasil.pengguna.peran]);
    } catch (e) {
      // Kegagalan tidak pernah dianggap sukses: tidak ada redirect di sini.
      setGalat(e);
    } finally {
      setSedangKirim(false);
    }
  }

  return (
    <div className="kartu kartu-sempit">
      <h1>Masuk</h1>
      <p className="sub">iMitra — Originasi Pembiayaan Mikro Syariah</p>

      <ErrorAlert galat={galat} />

      <form onSubmit={kirim} noValidate>
        <FormField label="Username" htmlFor="username" wajib>
          <input
            id="username"
            name="username"
            autoComplete="username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            required
          />
        </FormField>

        <FormField label="Password" htmlFor="password" wajib>
          <input
            id="password"
            name="password"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </FormField>

        <button type="submit" disabled={sedangKirim || !username || !password}>
          {sedangKirim ? 'Memproses…' : 'Masuk'}
        </button>
      </form>
    </div>
  );
}
