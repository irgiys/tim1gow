package service

import (
	"context"
	"testing"
	"time"

	"github.com/irgiys/tim1gow/backend/internal/domain"
)

// Test AC-08: ANL override grade dari 2 ke 3; tercatat di audit trail dengan identitas ANL.
func TestAudit_AC08_OverrideGradeTercatat(t *testing.T) {
	ctx := context.Background()
	auditRepo := newFakeAuditRepo()
	auditSvc := NewAuditService(auditRepo)

	pengajuanID := int64(101)
	anlID := int64(22)
	anlRole := domain.PeranANL

	err := auditSvc.Catat(ctx, domain.CatatAuditInput{
		PengajuanID:   &pengajuanID,
		Aksi:          domain.AksiOverrideSkor,
		StatusSebelum: domain.StatusScored,
		StatusSesudah: domain.StatusScored,
		Catatan:       "override grade dari 2 ke 3: sektor usaha mengalami fluktuasi musiman",
		ActorID:       anlID,
		ActorRole:     anlRole,
	})
	if err != nil {
		t.Fatalf("Catat audit override gagal: %v", err)
	}

	riwayat, err := auditSvc.AmbilRiwayatPengajuan(ctx, pengajuanID)
	if err != nil {
		t.Fatalf("AmbilRiwayatPengajuan gagal: %v", err)
	}
	if len(riwayat) != 1 {
		t.Fatalf("riwayat audit diharapkan 1 baris, dapat %d", len(riwayat))
	}

	entry := riwayat[0]
	if entry.ActorID != anlID {
		t.Fatalf("actor_id audit diharapkan %d, dapat %d", anlID, entry.ActorID)
	}
	if entry.ActorRole != anlRole {
		t.Fatalf("actor_role audit diharapkan %s, dapat %s", anlRole, entry.ActorRole)
	}
	if entry.Aksi != domain.AksiOverrideSkor {
		t.Fatalf("aksi audit diharapkan %s, dapat %s", domain.AksiOverrideSkor, entry.Aksi)
	}
}

// AC-08 bagian kedua: "sistem menolak jika alasan kosong". Bagian ini sebelumnya
// tidak diuji oleh test mana pun DAN tidak ditegakkan di kode — override grade
// dengan catatan kosong lolos begitu saja.
//
// Alasan wajib khusus untuk OVERRIDE_SKOR: override adalah keputusan manual yang
// menimpa hasil perhitungan sistem, jadi tanpa alasan tersimpan, audit trail
// tidak menjawab "mengapa" (BR-10: tidak ada perubahan tanpa jejak sebab).
func TestAudit_AC08_OverrideTanpaAlasanDitolak(t *testing.T) {
	ctx := context.Background()
	pengajuanID := int64(103)

	buatInput := func(catatan string) domain.CatatAuditInput {
		return domain.CatatAuditInput{
			PengajuanID:   &pengajuanID,
			Aksi:          domain.AksiOverrideSkor,
			StatusSebelum: domain.StatusScored,
			StatusSesudah: domain.StatusScored,
			Catatan:       catatan,
			ActorID:       22,
			ActorRole:     domain.PeranANL,
		}
	}

	kasus := []struct {
		nama    string
		catatan string
	}{
		{"kosong", ""},
		{"hanya spasi", "   "},
		{"hanya tab dan newline", "	\n"},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			svc := NewAuditService(newFakeAuditRepo())

			err := svc.Catat(ctx, buatInput(k.catatan))
			if err == nil {
				t.Fatalf("override dengan alasan %q seharusnya ditolak (AC-08), dapat nil", k.catatan)
			}

			// Tidak boleh ada baris audit yang tertulis saat ditolak.
			riwayat, _ := svc.AmbilRiwayatPengajuan(ctx, pengajuanID)
			if len(riwayat) != 0 {
				t.Errorf("override ditolak tetapi %d baris audit tetap tercatat", len(riwayat))
			}
		})
	}

	// Kasus pembanding (AGENTS.md Larangan 18): dengan alasan terisi harus
	// LOLOS, supaya test di atas tidak lolos hanya karena Catat selalu menolak.
	t.Run("dengan alasan terisi diterima", func(t *testing.T) {
		svc := NewAuditService(newFakeAuditRepo())

		err := svc.Catat(ctx, buatInput("grade diturunkan: omzet menurun tiga bulan terakhir"))
		if err != nil {
			t.Fatalf("override dengan alasan terisi seharusnya diterima: %v", err)
		}

		riwayat, err := svc.AmbilRiwayatPengajuan(ctx, pengajuanID)
		if err != nil {
			t.Fatalf("AmbilRiwayatPengajuan gagal: %v", err)
		}
		if len(riwayat) != 1 {
			t.Fatalf("riwayat diharapkan 1 baris, dapat %d", len(riwayat))
		}
	})

	// Aksi lain TIDAK boleh ikut terkena aturan ini — mis. pencatatan skoring
	// otomatis tanpa catatan tetap sah. Kalau ikut ditolak, alur normal rusak.
	t.Run("aksi non-override tanpa catatan tetap diterima", func(t *testing.T) {
		svc := NewAuditService(newFakeAuditRepo())

		in := buatInput("")
		in.Aksi = domain.AksiSkoring

		if err := svc.Catat(ctx, in); err != nil {
			t.Fatalf("aksi %s tanpa catatan seharusnya tetap diterima: %v", domain.AksiSkoring, err)
		}
	})
}

// Test AC-12: Audit trail menampilkan riwayat lengkap satu pengajuan dari DRAFT sampai APPROVED, urut waktu, dengan aktor di setiap baris.
func TestAudit_AC12_RiwayatLengkapUrutWaktu(t *testing.T) {
	ctx := context.Background()
	auditRepo := newFakeAuditRepo()
	auditSvc := NewAuditService(auditRepo)

	pengajuanID := int64(102)

	// Simulasi perjalanan alur pengajuan lengkap:
	// 1. AO membuat pengajuan (DRAFT)
	_ = auditSvc.Catat(ctx, domain.CatatAuditInput{
		PengajuanID:   &pengajuanID,
		Aksi:          domain.AksiBuatPengajuan,
		StatusSebelum: "",
		StatusSesudah: domain.StatusDraft,
		Catatan:       "pengajuan dibuat oleh AO",
		ActorID:       10,
		ActorRole:     domain.PeranAO,
	})
	time.Sleep(1 * time.Millisecond)

	// 2. AO submit pengajuan (SUBMITTED)
	_ = auditSvc.Catat(ctx, domain.CatatAuditInput{
		PengajuanID:   &pengajuanID,
		Aksi:          domain.AksiSubmitPengajuan,
		StatusSebelum: domain.StatusDraft,
		StatusSesudah: domain.StatusSubmitted,
		Catatan:       "pengajuan disubmit oleh AO",
		ActorID:       10,
		ActorRole:     domain.PeranAO,
	})
	time.Sleep(1 * time.Millisecond)

	// 3. ANL verifikasi dokumen (VERIFYING -> SLIK_CHECKED)
	_ = auditSvc.Catat(ctx, domain.CatatAuditInput{
		PengajuanID:   &pengajuanID,
		Aksi:          domain.AksiVerifikasiDokumen,
		StatusSebelum: domain.StatusSubmitted,
		StatusSesudah: domain.StatusVerifying,
		Catatan:       "semua dokumen terverifikasi",
		ActorID:       20,
		ActorRole:     domain.PeranANL,
	})
	time.Sleep(1 * time.Millisecond)

	// 4. ANL lakukan skoring (SCORED)
	_ = auditSvc.Catat(ctx, domain.CatatAuditInput{
		PengajuanID:   &pengajuanID,
		Aksi:          domain.AksiSkoring,
		StatusSebelum: domain.StatusSlikChecked,
		StatusSesudah: domain.StatusScored,
		Catatan:       "skoring kelayakan selesai: skor 86, grade 1",
		ActorID:       20,
		ActorRole:     domain.PeranANL,
	})
	time.Sleep(1 * time.Millisecond)

	// 5. ANL ajukan ke approval (WAITING_APPROVAL_L1)
	_ = auditSvc.Catat(ctx, domain.CatatAuditInput{
		PengajuanID:   &pengajuanID,
		Aksi:          domain.AksiAjukanApproval,
		StatusSebelum: domain.StatusScored,
		StatusSesudah: domain.StatusWaitingApprovalL1,
		Catatan:       "diajukan ke approval KCP",
		ActorID:       20,
		ActorRole:     domain.PeranANL,
	})
	time.Sleep(1 * time.Millisecond)

	// 6. KCP approve (APPROVED)
	_ = auditSvc.Catat(ctx, domain.CatatAuditInput{
		PengajuanID:   &pengajuanID,
		Aksi:          domain.AksiKeputusanApproval,
		StatusSebelum: domain.StatusWaitingApprovalL1,
		StatusSesudah: domain.StatusApproved,
		Catatan:       "keputusan level KCP: APPROVE",
		ActorID:       30,
		ActorRole:     domain.PeranKCP,
	})

	riwayat, err := auditSvc.AmbilRiwayatPengajuan(ctx, pengajuanID)
	if err != nil {
		t.Fatalf("AmbilRiwayatPengajuan gagal: %v", err)
	}

	if len(riwayat) != 6 {
		t.Fatalf("riwayat audit diharapkan memuat 6 tahap, dapat %d", len(riwayat))
	}

	// Verifikasi urut waktu dan setiap baris memiliki aktor
	for i, entry := range riwayat {
		if entry.ActorID <= 0 {
			t.Errorf("baris ke-%d tidak memiliki actor_id yang valid (BR-10)", i)
		}
		if entry.ActorRole == "" {
			t.Errorf("baris ke-%d tidak memiliki actor_role yang valid (BR-10)", i)
		}
		if entry.CreatedAt.IsZero() {
			t.Errorf("baris ke-%d tidak memiliki timestamp (BR-10)", i)
		}
		if i > 0 {
			prev := riwayat[i-1]
			if entry.CreatedAt.Before(prev.CreatedAt) {
				t.Errorf("baris ke-%d tidak berurutan secara waktu terhadap baris sebelumnya", i)
			}
		}
	}

	// Baris pertama DRAFT, baris terakhir APPROVED
	if riwayat[0].StatusSesudah != domain.StatusDraft {
		t.Fatalf("tahap awal diharapkan DRAFT, dapat %s", riwayat[0].StatusSesudah)
	}
	if riwayat[len(riwayat)-1].StatusSesudah != domain.StatusApproved {
		t.Fatalf("tahap akhir diharapkan APPROVED, dapat %s", riwayat[len(riwayat)-1].StatusSesudah)
	}
}
