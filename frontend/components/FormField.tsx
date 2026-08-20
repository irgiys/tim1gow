import type { ReactNode } from 'react';

interface FormFieldProps {
  label: string;
  htmlFor: string;
  /** Pesan galat per field dari respons 400 API. */
  galat?: string | null;
  petunjuk?: string;
  wajib?: boolean;
  children: ReactNode;
}

/** Pembungkus satu field form: label, input, petunjuk, pesan galat. */
export function FormField({ label, htmlFor, galat, petunjuk, wajib, children }: FormFieldProps) {
  return (
    <div className="field">
      <label htmlFor={htmlFor}>
        {label}
        {wajib ? <span className="wajib"> *</span> : null}
      </label>
      {children}
      {petunjuk ? <small className="petunjuk">{petunjuk}</small> : null}
      {galat ? (
        <small className="galat-field" role="alert">
          {galat}
        </small>
      ) : null}
    </div>
  );
}
