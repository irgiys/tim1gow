'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { apiClient } from '@/lib/apiClient';
import { simpanSesi, BERANDA_PERAN } from '@/lib/auth';
import { FormField } from '@/components/FormField';
import { ErrorAlert } from '@/components/ErrorAlert';
import type { Peran } from '@/lib/types';

/**
 * Bentuk respons login, mengikuti backend apa adanya
 * (backend/internal/httpapi/auth_handler.go).
 *
 * Bentuknya FLAT — bukan { token, pengguna }. Sebelumnya halaman ini
 * mengirim field `username` dan membaca `hasil.pengguna.peran`, sehingga login
 * selalu gagal 400 "email dan password wajib diisi" dan redirect-nya
 * mengevaluasi `undefined`. Kontrak di bawah diverifikasi lewat panggilan API
 * nyata, bukan diasumsikan.
 */
interface ResponsLogin {
  token: string;
  peran: Peran;
  nama: string;
  id: number;
  berlaku_sampai: number;
}

export default function HalamanLogin() {
  const router = useRouter();
  const [email, setEmail] = useState('');
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
        email,
        password,
      });
      simpanSesi({
        token: hasil.token,
        pengguna: {
          id: String(hasil.id),
          username: email,
          namaLengkap: hasil.nama,
          peran: hasil.peran,
        },
      });
      router.push(BERANDA_PERAN[hasil.peran]);
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
        <FormField label="Email" htmlFor="email" wajib>
          <input
            id="email"
            name="email"
            type="email"
            autoComplete="username"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
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

        <button type="submit" disabled={sedangKirim || !email || !password}>
          {sedangKirim ? 'Memproses…' : 'Masuk'}
        </button>
      </form>
    </div>
  );
}
