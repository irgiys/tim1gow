import type { Metadata } from 'next';
import type { ReactNode } from 'react';
import './globals.css';
import { TopBar } from '@/components/TopBar';

export const metadata: Metadata = {
  title: 'iMitra — Originasi Pembiayaan Mikro Syariah',
  description: 'Sistem originasi pembiayaan mikro syariah Bank Syariah Nasional',
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="id">
      <body>
        <TopBar />
        <main>{children}</main>
      </body>
    </html>
  );
}
