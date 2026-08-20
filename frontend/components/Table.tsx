import type { ReactNode } from 'react';

export interface KolomTabel<T> {
  judul: string;
  render: (baris: T) => ReactNode;
  /** Rata kanan untuk kolom angka. */
  angka?: boolean;
}

interface TableProps<T> {
  kolom: KolomTabel<T>[];
  data: T[];
  kunci: (baris: T) => string;
  pesanKosong?: string;
}

/** Tabel sederhana untuk daftar pengajuan, dokumen, audit trail. */
export function Table<T>({ kolom, data, kunci, pesanKosong = 'Belum ada data.' }: TableProps<T>) {
  if (data.length === 0) {
    return <p className="kosong">{pesanKosong}</p>;
  }

  return (
    <table className="tabel">
      <thead>
        <tr>
          {kolom.map((k) => (
            <th key={k.judul} className={k.angka ? 'angka' : undefined}>
              {k.judul}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {data.map((baris) => (
          <tr key={kunci(baris)}>
            {kolom.map((k) => (
              <td key={k.judul} className={k.angka ? 'angka' : undefined}>
                {k.render(baris)}
              </td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  );
}
