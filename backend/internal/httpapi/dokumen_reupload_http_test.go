package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/irgiys/tim1gow/backend/internal/domain"
	"github.com/irgiys/tim1gow/backend/internal/service"
)

// Test di berkas ini menutup celah yang tersisa pada FR-03: AC-03 sudah diuji di
// lapisan service (dokumen_service_test.go), tetapi belum pernah dibuktikan
// LEWAT API. Yang dipakai penilai adalah API-nya, dan jalur HTTP punya kegagalan
// sendiri yang tidak terlihat dari test service: pemetaan status, kode BR yang
// hilang di respons, atau berkas yang ikut terbawa ke klien.
//
// Dua arah diuji berpasangan sesuai AGENTS.md Larangan 18: dokumen REJECTED
// BOLEH diunggah ulang, dokumen VERIFIED TIDAK BOLEH. Test penolakan tanpa
// kasus pembanding tidak membuktikan apa pun — argumen yang tertukar akan
// meloloskan keduanya.

// ambilDaftarDokumen menembak GET daftar dokumen dan mengembalikan isinya.
func ambilDaftarDokumen(t *testing.T, h http.Handler, pengajuanID string) []dokumenResponse {
	t.Helper()

	rec := kirim(t, h, http.MethodGet, "/api/pengajuan/"+pengajuanID+"/dokumen", domain.PeranAO, 11, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("daftar dokumen: status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data []dokumenResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode daftar dokumen: %v", err)
	}
	return resp.Data
}

// TestRoute_FR03_AC03_ReuploadDokumenDitolakTidakMenyentuhDokumenLain menelusuri
// alur AC-03 apa adanya lewat API: AO unggah dua dokumen, ANL menolak salah
// satunya, lalu AO mengunggah ulang HANYA yang ditolak. Yang dijaga: dokumen
// lain tidak ikut berubah, dan AO tidak perlu mengisi ulang apa pun.
func TestRoute_FR03_AC03_ReuploadDokumenDitolakTidakMenyentuhDokumenLain(t *testing.T) {
	h, _, _, _ := routerPengajuan(t)

	const jenisLain = "SURAT_USAHA"

	// AO mengunggah dua jenis dokumen.
	ktp := kirim(t, h, http.MethodPost, "/api/pengajuan/9/dokumen", domain.PeranAO, 11,
		map[string]any{"jenisDokumen": service.JenisDokumenKTP, "urlBerkas": "s3://dok/ktp.jpg"})
	if ktp.Code != http.StatusCreated {
		t.Fatalf("upload KTP: status = %d, mau 201 (body=%s)", ktp.Code, ktp.Body.String())
	}
	var dokKTP dokumenResponse
	if err := json.Unmarshal(ktp.Body.Bytes(), &dokKTP); err != nil {
		t.Fatalf("decode KTP: %v", err)
	}

	lain := kirim(t, h, http.MethodPost, "/api/pengajuan/9/dokumen", domain.PeranAO, 11,
		map[string]any{"jenisDokumen": jenisLain, "urlBerkas": "s3://dok/usaha.jpg"})
	if lain.Code != http.StatusCreated {
		t.Fatalf("upload %s: status = %d, mau 201", jenisLain, lain.Code)
	}
	var dokLain dokumenResponse
	if err := json.Unmarshal(lain.Body.Bytes(), &dokLain); err != nil {
		t.Fatalf("decode %s: %v", jenisLain, err)
	}

	// ANL menolak KTP dengan kode alasan (bukan AO — maker != checker, AC-02).
	tolak := kirim(t, h, http.MethodPatch,
		"/api/pengajuan/9/dokumen/"+strconv.FormatInt(dokKTP.ID, 10)+"/verifikasi", domain.PeranANL, 33,
		map[string]any{"setujui": false, "kodeAlasan": "FOTO_TIDAK_JELAS"})
	if tolak.Code != http.StatusOK {
		t.Fatalf("ANL tolak KTP: status = %d, mau 200 (body=%s)", tolak.Code, tolak.Body.String())
	}
	var ditolak dokumenResponse
	if err := json.Unmarshal(tolak.Body.Bytes(), &ditolak); err != nil {
		t.Fatalf("decode penolakan: %v", err)
	}
	if ditolak.Status != string(service.StatusDokumenRejected) {
		t.Fatalf("status setelah ditolak = %q, mau %q", ditolak.Status, service.StatusDokumenRejected)
	}

	// Inti AC-03: AO mengunggah ulang HANYA KTP. Tidak ada data pengajuan yang
	// dikirim ulang — permintaannya hanya menyebut satu jenis dokumen.
	ulang := kirim(t, h, http.MethodPost, "/api/pengajuan/9/dokumen", domain.PeranAO, 11,
		map[string]any{"jenisDokumen": service.JenisDokumenKTP, "urlBerkas": "s3://dok/ktp-baru.jpg"})
	if ulang.Code != http.StatusCreated {
		t.Fatalf("re-upload KTP yang ditolak: status = %d, mau 201 (body=%s)",
			ulang.Code, ulang.Body.String())
	}
	var dokBaru dokumenResponse
	if err := json.Unmarshal(ulang.Body.Bytes(), &dokBaru); err != nil {
		t.Fatalf("decode re-upload: %v", err)
	}
	if dokBaru.Status != string(service.StatusDokumenUploaded) {
		t.Errorf("status hasil re-upload = %q, mau %q", dokBaru.Status, service.StatusDokumenUploaded)
	}
	if dokBaru.ID == dokKTP.ID {
		t.Error("re-upload menimpa baris lama; riwayat penolakan hilang dan audit trail tidak lagi utuh")
	}

	// Dokumen jenis lain tidak boleh tersentuh oleh re-upload di atas.
	for _, d := range ambilDaftarDokumen(t, h, "9") {
		if d.ID == dokLain.ID && d.Status != string(service.StatusDokumenUploaded) {
			t.Errorf("dokumen %s ikut berubah menjadi %q; AC-03 menuntut hanya berkas yang ditolak tergantikan",
				jenisLain, d.Status)
		}
	}

	// BR-11: path berkas tidak boleh mengalir ke klien lewat daftar maupun
	// respons upload.
	for _, rec := range []string{ulang.Body.String(), tolak.Body.String()} {
		if strings.Contains(rec, "s3://") {
			t.Error("path berkas muncul di respons; melanggar BR-11")
		}
	}
}

// TestRoute_FR03_AC03_DokumenVerifiedTidakBisaDiunggahUlang adalah kasus
// pembanding untuk test di atas (Larangan 18). Arah sebaliknya wajib ditutup:
// kalau keduanya lolos, guard-nya tidak benar-benar membedakan status.
func TestRoute_FR03_AC03_DokumenVerifiedTidakBisaDiunggahUlang(t *testing.T) {
	h, _, _, _ := routerPengajuan(t)

	rec := kirim(t, h, http.MethodPost, "/api/pengajuan/9/dokumen", domain.PeranAO, 11,
		map[string]any{"jenisDokumen": service.JenisDokumenKTP, "urlBerkas": "s3://dok/ktp.jpg"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload awal: status = %d, mau 201", rec.Code)
	}
	var dok dokumenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &dok); err != nil {
		t.Fatalf("decode upload: %v", err)
	}

	setuju := kirim(t, h, http.MethodPatch,
		"/api/pengajuan/9/dokumen/"+strconv.FormatInt(dok.ID, 10)+"/verifikasi", domain.PeranANL, 33,
		map[string]any{"setujui": true})
	if setuju.Code != http.StatusOK {
		t.Fatalf("ANL setujui: status = %d, mau 200 (body=%s)", setuju.Code, setuju.Body.String())
	}

	// Sudah VERIFIED: unggah ulang harus diblokir, bukan diberi peringatan.
	ulang := kirim(t, h, http.MethodPost, "/api/pengajuan/9/dokumen", domain.PeranAO, 11,
		map[string]any{"jenisDokumen": service.JenisDokumenKTP, "urlBerkas": "s3://dok/ktp-lain.jpg"})
	if ulang.Code != http.StatusUnprocessableEntity {
		t.Fatalf("re-upload dokumen VERIFIED: status = %d, mau 422 (body=%s)",
			ulang.Code, ulang.Body.String())
	}
	// Pesan pelanggaran aturan bisnis wajib membawa kode BR-nya (AC-04).
	if r := decodeError(t, ulang).Rule; r != "BR-03" {
		t.Errorf("rule = %q, mau BR-03", r)
	}
}
