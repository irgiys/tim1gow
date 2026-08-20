package domain

import "time"

// AuditTrailEntry mewakili satu baris log audit append-only (FR-09, AC-12, AC-13).
//
// Catatan: Catatan TIDAK BOLEH memuat data pribadi (NIK, foto dokumen) sesuai BR-11.
type AuditTrailEntry struct {
	ID            int64           `json:"id" gorm:"primaryKey;autoIncrement"`
	PengajuanID   *int64          `json:"pengajuan_id,omitempty" gorm:"column:pengajuan_id"`
	Aksi          AksiAudit       `json:"aksi" gorm:"column:aksi;not null"`
	StatusSebelum StatusPengajuan `json:"status_sebelum,omitempty" gorm:"column:status_sebelum"`
	StatusSesudah StatusPengajuan `json:"status_sesudah,omitempty" gorm:"column:status_sesudah"`
	Catatan       string          `json:"catatan,omitempty" gorm:"column:catatan"`
	ActorID       int64           `json:"actor_id" gorm:"column:actor_id;not null"`
	ActorRole     Peran           `json:"actor_role" gorm:"column:actor_role;not null"`
	CreatedAt     time.Time       `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

func (AuditTrailEntry) TableName() string {
	return "audit_trail"
}

// CatatAuditInput adalah data masukan saat service merekam jejak audit baru.
type CatatAuditInput struct {
	PengajuanID   *int64
	Aksi          AksiAudit
	StatusSebelum StatusPengajuan
	StatusSesudah StatusPengajuan
	Catatan       string
	ActorID       int64
	ActorRole     Peran
}
