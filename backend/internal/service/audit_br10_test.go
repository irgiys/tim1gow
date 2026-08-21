package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/irgiys/tim1gow/backend/internal/domain"
)

// Test di berkas ini menjaga BR-10 pada FR-02/FR-03/FR-04: setiap perubahan
// keadaan wajib punya aktor dan timestamp.
//
// Sebelumnya hanya approval dan skoring yang memanggil audit. Pengajuan,
// dokumen, dan survei berubah tanpa jejak — `SELECT count(*) FROM audit_trail`
// mengembalikan 0 walau enam mutasi sudah terjadi lewat API. Endpoint /audit
// sendiri berfungsi, jadi kekosongan itu terlihat seperti "belum ada aktivitas"
// dan bukan seperti bug.

// auditPalsu merekam apa yang dicatat service, supaya test dapat memeriksa
// aktor dan aksinya — bukan hanya bahwa Catat terpanggil.
type auditPalsu struct {
	dicatat []domain.CatatAuditInput
	galat   error
}

func (a *auditPalsu) Catat(_ context.Context, in domain.CatatAuditInput) error {
	if a.galat != nil {
		return a.galat
	}
	a.dicatat = append(a.dicatat, in)
	return nil
}

func (a *auditPalsu) AmbilRiwayatPengajuan(context.Context, int64) ([]domain.AuditTrailEntry, error) {
	return nil, nil
}

func (a *auditPalsu) AmbilSemua(context.Context, int, int) ([]domain.AuditTrailEntry, error) {
	return nil, nil
}

// TestBR10_BuatPengajuanDicatatDenganAktor memastikan pembuatan pengajuan
// meninggalkan jejak beserta aktornya.
func TestBR10_BuatPengajuanDicatatDenganAktor(t *testing.T) {
	audit := &auditPalsu{}
	svc := NewPengajuanService(newFakePengajuanRepo(), newFakeBatasPlafonRepo()).
		DenganAudit(audit)

	in := inputValid()
	in.AOID = 42
	p, err := svc.Buat(context.Background(), in)
	if err != nil {
		t.Fatalf("Buat: %v", err)
	}

	if len(audit.dicatat) != 1 {
		t.Fatalf("jumlah jejak audit = %d, mau 1", len(audit.dicatat))
	}
	got := audit.dicatat[0]
	if got.Aksi != domain.AksiBuatPengajuan {
		t.Errorf("aksi = %q, mau %q", got.Aksi, domain.AksiBuatPengajuan)
	}
	if got.ActorID != 42 {
		t.Errorf("ActorID = %d, mau 42 — aktor wajib ikut tercatat (BR-10)", got.ActorID)
	}
	if got.ActorRole != domain.PeranAO {
		t.Errorf("ActorRole = %q, mau AO", got.ActorRole)
	}
	if got.PengajuanID == nil || *got.PengajuanID != p.ID {
		t.Error("PengajuanID tidak menunjuk pengajuan yang baru dibuat")
	}
	// BR-11: NIK tidak boleh masuk catatan audit.
	if strings.Contains(got.Catatan, in.NIK) {
		t.Errorf("catatan memuat NIK: %q", got.Catatan)
	}
}

// TestBR10_PengajuanGagalDicatatTidakTersimpanDiam menjaga sifat fail-closed:
// kalau audit gagal, operasinya GAGAL. Perubahan keadaan yang kehilangan
// jejaknya tidak boleh dilaporkan sukses ke pemanggil.
func TestBR10_PengajuanGagalDicatatTidakTersimpanDiam(t *testing.T) {
	audit := &auditPalsu{galat: errors.New("audit writer mati")}
	svc := NewPengajuanService(newFakePengajuanRepo(), newFakeBatasPlafonRepo()).
		DenganAudit(audit)

	if _, err := svc.Buat(context.Background(), inputValid()); err == nil {
		t.Fatal("Buat berhasil padahal audit gagal; BR-10 mewajibkan jejak, " +
			"jadi kegagalan audit tidak boleh ditelan")
	}

	// Kasus pembanding (Larangan 18): dengan audit sehat, operasi yang sama
	// harus berhasil — supaya test di atas tidak hijau hanya karena Buat
	// selalu gagal.
	sehat := &auditPalsu{}
	svcSehat := NewPengajuanService(newFakePengajuanRepo(), newFakeBatasPlafonRepo()).
		DenganAudit(sehat)
	if _, err := svcSehat.Buat(context.Background(), inputValid()); err != nil {
		t.Fatalf("audit sehat: Buat gagal: %v", err)
	}
}

// TestBR10_UploadDanVerifikasiDokumenDicatat memastikan kedua tahap dokumen
// meninggalkan jejak dengan aktor yang BERBEDA: AO mengunggah, ANL
// memverifikasi. Ini yang membuat maker/checker dapat diaudit.
func TestBR10_UploadDanVerifikasiDokumenDicatat(t *testing.T) {
	audit := &auditPalsu{}
	svc := NewDokumenService(newFakeDokumenRepo(), newFakeDokumenWajibRepo()).
		DenganAudit(audit)
	ctx := context.Background()

	d, err := svc.Upload(ctx, 1, JenisDokumenKTP, "berkas/ktp.jpg", 11)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if _, err := svc.Verifikasi(ctx, d.ID, 33, true, ""); err != nil {
		t.Fatalf("Verifikasi: %v", err)
	}

	if len(audit.dicatat) != 2 {
		t.Fatalf("jumlah jejak = %d, mau 2 (upload + verifikasi)", len(audit.dicatat))
	}

	up, ver := audit.dicatat[0], audit.dicatat[1]
	if up.Aksi != domain.AksiUploadDokumen || up.ActorID != 11 || up.ActorRole != domain.PeranAO {
		t.Errorf("jejak upload salah: aksi=%q actor=%d/%q", up.Aksi, up.ActorID, up.ActorRole)
	}
	if ver.Aksi != domain.AksiVerifikasiDokumen || ver.ActorID != 33 || ver.ActorRole != domain.PeranANL {
		t.Errorf("jejak verifikasi salah: aksi=%q actor=%d/%q", ver.Aksi, ver.ActorID, ver.ActorRole)
	}
	// BR-11: path berkas tidak boleh masuk audit.
	for _, j := range audit.dicatat {
		if strings.Contains(j.Catatan, "berkas/") {
			t.Errorf("catatan memuat path berkas: %q", j.Catatan)
		}
	}
}

// TestBR10_PenolakanDokumenMencatatAlasan memastikan keputusan penolakan bisa
// dipertanggungjawabkan: kode alasannya ikut tercatat.
func TestBR10_PenolakanDokumenMencatatAlasan(t *testing.T) {
	audit := &auditPalsu{}
	svc := NewDokumenService(newFakeDokumenRepo(), newFakeDokumenWajibRepo()).
		DenganAudit(audit)
	ctx := context.Background()

	d, err := svc.Upload(ctx, 1, JenisDokumenKTP, "berkas/ktp.jpg", 11)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if _, err := svc.Verifikasi(ctx, d.ID, 33, false, "FOTO_BURAM"); err != nil {
		t.Fatalf("Verifikasi tolak: %v", err)
	}

	terakhir := audit.dicatat[len(audit.dicatat)-1]
	if !strings.Contains(terakhir.Catatan, "FOTO_BURAM") {
		t.Errorf("catatan = %q; mau memuat kode alasan penolakan", terakhir.Catatan)
	}
}

// TestBR10_RekamSurveiDicatat memastikan survei lapangan meninggalkan jejak,
// tanpa membocorkan koordinat maupun path foto (BR-11).
func TestBR10_RekamSurveiDicatat(t *testing.T) {
	audit := &auditPalsu{}
	svc := NewSurveiService(newFakeSurveiRepo()).DenganAudit(audit)

	in := inputSurveiValid()
	in.AOID = 11
	if _, err := svc.Rekam(context.Background(), in); err != nil {
		t.Fatalf("Rekam: %v", err)
	}

	if len(audit.dicatat) != 1 {
		t.Fatalf("jumlah jejak = %d, mau 1", len(audit.dicatat))
	}
	got := audit.dicatat[0]
	if got.Aksi != domain.AksiRekamSurvei {
		t.Errorf("aksi = %q, mau %q", got.Aksi, domain.AksiRekamSurvei)
	}
	if got.ActorID != 11 || got.ActorRole != domain.PeranAO {
		t.Errorf("aktor = %d/%q, mau 11/AO", got.ActorID, got.ActorRole)
	}
	if strings.Contains(got.Catatan, in.FotoURL) {
		t.Errorf("catatan memuat path foto: %q", got.Catatan)
	}
}

// TestBR10_TanpaAuditServiceTetapJalan memastikan test unit lama yang tidak
// memasang audit tidak rusak. Ini kompromi yang disengaja dan hanya berlaku
// untuk test: jalur produksi memasang audit lewat DenganAudit di router.
func TestBR10_TanpaAuditServiceTetapJalan(t *testing.T) {
	svc := NewPengajuanService(newFakePengajuanRepo(), newFakeBatasPlafonRepo())
	if _, err := svc.Buat(context.Background(), inputValid()); err != nil {
		t.Fatalf("tanpa audit seharusnya tetap jalan untuk test unit: %v", err)
	}
}
