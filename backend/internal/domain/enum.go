package domain

// Nilai enum di berkas ini WAJIB sama dengan AGENTS.md bagian 4.1 dan
// docs/SDD-iMitra.md. Menambah nilai baru tanpa memperbarui keduanya dilarang
// (AGENTS.md Larangan 5).

// StatusPengajuan adalah status alur pengajuan pembiayaan.
type StatusPengajuan string

const (
	StatusDraft             StatusPengajuan = "DRAFT"
	StatusSubmitted         StatusPengajuan = "SUBMITTED"
	StatusVerifying         StatusPengajuan = "VERIFYING"
	StatusSlikChecked       StatusPengajuan = "SLIK_CHECKED"
	StatusScored            StatusPengajuan = "SCORED"
	StatusWaitingApprovalL1 StatusPengajuan = "WAITING_APPROVAL_L1"
	StatusWaitingApprovalL2 StatusPengajuan = "WAITING_APPROVAL_L2"
	StatusWaitingApprovalL3 StatusPengajuan = "WAITING_APPROVAL_L3"
	StatusApproved          StatusPengajuan = "APPROVED"

	StatusRejectedSlik    StatusPengajuan = "REJECTED_SLIK"
	StatusRejectedScoring StatusPengajuan = "REJECTED_SCORING"
	StatusReturned        StatusPengajuan = "RETURNED"
	StatusRejected        StatusPengajuan = "REJECTED"
)

// Akad adalah jenis akad pembiayaan.
type Akad string

const (
	AkadMurabahah  Akad = "MURABAHAH"
	AkadMusyarakah Akad = "MUSYARAKAH"
)

// Peran adalah kode peran pengguna. Dipakai persis seperti ini di kode, DB, dan UI.
type Peran string

const (
	PeranAO  Peran = "AO"
	PeranANL Peran = "ANL"
	PeranKCP Peran = "KCP"
	PeranKC  Peran = "KC"
	PeranKOM Peran = "KOM"
	PeranADM Peran = "ADM"
)

// KeputusanApproval adalah pilihan keputusan approver (KCP / KC / KOM).
type KeputusanApproval string

const (
	KeputusanApprove KeputusanApproval = "APPROVE"
	KeputusanReject  KeputusanApproval = "REJECT"
	KeputusanReturn  KeputusanApproval = "RETURN"
)

// AksiAudit adalah jenis aksi yang dicatat di audit_trail.
type AksiAudit string

const (
	AksiBuatPengajuan     AksiAudit = "BUAT_PENGAJUAN"
	AksiSubmitPengajuan   AksiAudit = "SUBMIT_PENGAJUAN"
	AksiUploadDokumen     AksiAudit = "UPLOAD_DOKUMEN"
	AksiVerifikasiDokumen AksiAudit = "VERIFIKASI_DOKUMEN"
	AksiRekamSurvei       AksiAudit = "REKAM_SURVEI"
	AksiSlikCheck         AksiAudit = "SLIK_CHECK"
	AksiSkoring           AksiAudit = "SKORING"
	AksiOverrideSkor      AksiAudit = "OVERRIDE_SKOR"
	AksiAjukanApproval    AksiAudit = "AJUKAN_APPROVAL"
	AksiKeputusanApproval AksiAudit = "KEPUTUSAN_APPROVAL"
	AksiLogin             AksiAudit = "LOGIN"
)

// Kode komponen skor kelayakan. Baris parameternya hidup di tabel
// parameter_skoring; konstanta di bawah hanya KUNCI baris, bukan nilai bobotnya.
const (
	KomponenKapasitasBayar = "KAPASITAS_BAYAR"
	KomponenRiwayatSlik    = "RIWAYAT_SLIK"
	KomponenLamaUsaha      = "LAMA_USAHA"
	KomponenSurveiLapangan = "SURVEI_LAPANGAN"
)
