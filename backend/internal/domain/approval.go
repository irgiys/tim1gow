package domain

import "time"

// Pengajuan mewakili entitas pengajuan pembiayaan mikro syariah.
//
// Nama kolom mengikuti SDD BAB 4.1: `ao_id` (maker, dipakai cek BR-09),
// `plafon_diajukan`, dan `dibuat_pada`/`diperbarui_pada`. Struct ini sengaja
// memuat kolom yang dibutuhkan alur approval saja; field lengkap pengajuan
// (nasabah, akad, tenor) ada di lapisan service pengajuan.
type Pengajuan struct {
	ID             int64           `json:"id" gorm:"primaryKey;autoIncrement"`
	NomorReferensi string          `json:"nomor_referensi" gorm:"column:nomor_referensi;unique;not null"`
	PlafonDiajukan int64           `json:"plafon_diajukan" gorm:"column:plafon_diajukan;not null"`
	Grade          int             `json:"grade" gorm:"column:grade"`
	Status         StatusPengajuan `json:"status" gorm:"column:status;not null"`
	AOID           int64           `json:"ao_id" gorm:"column:ao_id;not null"`
	DibuatPada     time.Time       `json:"dibuat_pada" gorm:"column:dibuat_pada;autoCreateTime"`
	DiperbaruiPada time.Time       `json:"diperbarui_pada" gorm:"column:diperbarui_pada;autoUpdateTime"`
}

func (Pengajuan) TableName() string {
	return "pengajuan"
}

// KeputusanApprovalRecord mewakili satu rekaman keputusan approval (FR-08).
type KeputusanApprovalRecord struct {
	ID          int64             `json:"id" gorm:"primaryKey;autoIncrement"`
	PengajuanID int64             `json:"pengajuan_id" gorm:"column:pengajuan_id;not null"`
	Level       Peran             `json:"level" gorm:"column:level;not null"`
	Keputusan   KeputusanApproval `json:"keputusan" gorm:"column:keputusan;not null"`
	Alasan      string            `json:"alasan,omitempty" gorm:"column:alasan"`
	Catatan     string            `json:"catatan,omitempty" gorm:"column:catatan"`
	ApproverID  int64             `json:"approver_id" gorm:"column:approver_id;not null"`
	CreatedAt   time.Time         `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

func (KeputusanApprovalRecord) TableName() string {
	return "keputusan_approval"
}

// ApprovalDecisionRequest adalah payload dari klien saat approver memutuskan.
type ApprovalDecisionRequest struct {
	Keputusan KeputusanApproval `json:"keputusan"`
	Alasan    string            `json:"alasan"`
	Catatan   string            `json:"catatan"`
}

// PengajuanApprovalDetail menggabungkan data pengajuan dan histori keputusan untuk layar approval.
type PengajuanApprovalDetail struct {
	Pengajuan       Pengajuan                 `json:"pengajuan"`
	RiwayatApproval []KeputusanApprovalRecord `json:"riwayat_approval"`
	LevelDiperlukan []Peran                   `json:"level_diperlukan"`
	LevelSaatIni    Peran                     `json:"level_saat_ini,omitempty"`
}
