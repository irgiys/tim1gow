import type { StatusPengajuan, StatusDokumen, StatusSurvei } from '@/lib/types';

type StatusApaPun = StatusPengajuan | StatusDokumen | StatusSurvei;

/**
 * Kelompok warna per status. Nilai enum-nya sendiri tidak boleh diganti nama
 * (AGENTS.md Larangan 5) — berkas ini hanya memetakan ke warna.
 */
const NADA: Record<string, 'netral' | 'proses' | 'baik' | 'buruk'> = {
  DRAFT: 'netral',
  PENDING: 'netral',
  SUBMITTED: 'proses',
  VERIFYING: 'proses',
  SLIK_CHECKED: 'proses',
  SCORED: 'proses',
  WAITING_APPROVAL_L1: 'proses',
  WAITING_APPROVAL_L2: 'proses',
  WAITING_APPROVAL_L3: 'proses',
  RETURNED: 'netral',
  VERIFIED: 'baik',
  VALID: 'baik',
  APPROVED: 'baik',
  INVALID: 'buruk',
  REJECTED: 'buruk',
  REJECTED_SLIK: 'buruk',
  REJECTED_SCORING: 'buruk',
};

/** Label yang lebih mudah dibaca; nilai enum tetap yang dikirim ke API. */
const LABEL: Partial<Record<string, string>> = {
  WAITING_APPROVAL_L1: 'Menunggu Approval L1',
  WAITING_APPROVAL_L2: 'Menunggu Approval L2',
  WAITING_APPROVAL_L3: 'Menunggu Approval L3',
  REJECTED_SLIK: 'Ditolak — SLIK',
  REJECTED_SCORING: 'Ditolak — Skoring',
  SLIK_CHECKED: 'SLIK Selesai',
};

export function StatusBadge({ status }: { status: StatusApaPun }) {
  const nada = NADA[status] ?? 'netral';
  return (
    <span className={`badge badge-${nada}`} title={status}>
      {LABEL[status] ?? status}
    </span>
  );
}
