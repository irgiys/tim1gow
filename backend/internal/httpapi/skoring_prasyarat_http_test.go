package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/irgiys/tim1gow/backend/internal/config"
	"github.com/irgiys/tim1gow/backend/internal/service"
)

// Test di berkas ini menjaga satu properti keamanan aturan bisnis:
//
//	Prasyarat BR-03 dan kolektibilitas ditentukan oleh KEADAAN TERSIMPAN,
//	bukan oleh apa yang diklaim pemanggil di badan request.
//
// Sebelumnya endpoint skoring menerima semuaDokumenVerified, adaSurveiValid,
// slikSudahDijalankan, dan kolektibilitas sebagai field JSON. Test lama hijau
// karena test itu sendiri yang mengirim `true` — guard-nya tidak pernah benar
// benar diuji terhadap penyerang. Test di bawah menutup celah itu.

// TestHTTP_BR03_KlaimKlienTidakDapatMelewatiGuard adalah regression test utama:
// klien mengirim seluruh penanda prasyarat bernilai true, tetapi keadaan
// tersimpan mengatakan dokumen belum diverifikasi. Permintaan WAJIB ditolak.
func TestHTTP_BR03_KlaimKlienTidakDapatMelewatiGuard(t *testing.T) {
	h, pra := routerSkoringDenganPrasyarat(newFakeParamRepoSkoring())

	// Keadaan sebenarnya: dokumen wajib belum diverifikasi.
	pra.keadaan.SemuaDokumenVerified = false

	// Klien berusaha mengaku prasyaratnya lengkap, memakai persis nama field
	// yang dulu diterima handler.
	body := dataSkoring()
	body["semuaDokumenVerified"] = true
	body["adaSurveiValid"] = true
	body["slikSudahDijalankan"] = true

	rec := postJSON(t, h, "/api/pengajuan/7/skoring", body)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, mau 422 — klaim klien seharusnya diabaikan (body=%s)",
			rec.Code, rec.Body.String())
	}
	errResp := decodeError(t, rec)
	if errResp.Rule != "BR-03" {
		t.Errorf(`rule = %q, mau "BR-03"`, errResp.Rule)
	}

	// Kasus pembanding (Larangan 18): begitu keadaan TERSIMPAN benar-benar
	// terpenuhi, permintaan yang sama diterima — tanpa field klaim apa pun.
	pra.keadaan.SemuaDokumenVerified = true
	if rec2 := postJSON(t, h, "/api/pengajuan/7/skoring", dataSkoring()); rec2.Code != http.StatusOK {
		t.Fatalf("keadaan lengkap: status = %d, mau 200 (body=%s)", rec2.Code, rec2.Body.String())
	}
}

// TestHTTP_BR03_KlaimKolektibilitasKlienDiabaikan memastikan klien tidak dapat
// memperbaiki grade-nya sendiri. Keadaan tersimpan kol-2 memaksa grade minimal
// 3 walau klien mengirim kolektibilitas 1.
func TestHTTP_BR03_KlaimKolektibilitasKlienDiabaikan(t *testing.T) {
	h, pra := routerSkoringDenganPrasyarat(newFakeParamRepoSkoring())

	pra.keadaan.Kolektibilitas = 2

	body := dataSkoring()
	body["kolektibilitas"] = 1 // klaim klien: riwayat bersih

	rec := postJSON(t, h, "/api/pengajuan/7/skoring", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (body=%s)", rec.Code, rec.Body.String())
	}

	var resp skoringResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode respons: %v", err)
	}
	if resp.Grade < 3 {
		t.Errorf("grade = %d, mau >= 3 — kolektibilitas klien seharusnya diabaikan", resp.Grade)
	}
	if !resp.GradeMinimalDipaksa {
		t.Error("gradeMinimalDipaksa = false; kol-2 tersimpan wajib memaksa grade minimal 3")
	}
}

// TestHTTP_BR03_SlikBelumDijalankanDitolak menjaga AGENTS.md Larangan 15:
// ketiadaan hasil SLIK bukan SLIK bersih. Skoring berhenti.
func TestHTTP_BR03_SlikBelumDijalankanDitolak(t *testing.T) {
	h, pra := routerSkoringDenganPrasyarat(newFakeParamRepoSkoring())

	pra.keadaan.SlikSudahDijalankan = false
	pra.keadaan.Kolektibilitas = 0

	rec := postJSON(t, h, "/api/pengajuan/7/skoring", dataSkoring())
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, mau 422 (body=%s)", rec.Code, rec.Body.String())
	}
	errResp := decodeError(t, rec)
	if errResp.Rule != "BR-03" {
		t.Errorf(`rule = %q, mau "BR-03"`, errResp.Rule)
	}
	if !strings.Contains(strings.ToLower(errResp.Message), "slik") {
		t.Errorf("message = %q; mau menyebut SLIK sebagai sebabnya", errResp.Message)
	}

	// Kasus pembanding: SLIK sudah dijalankan -> diterima.
	pra.keadaan.SlikSudahDijalankan = true
	pra.keadaan.Kolektibilitas = 1
	if rec2 := postJSON(t, h, "/api/pengajuan/7/skoring", dataSkoring()); rec2.Code != http.StatusOK {
		t.Fatalf("SLIK sudah jalan: status = %d, mau 200 (body=%s)", rec2.Code, rec2.Body.String())
	}
}

// TestHTTP_BR03_SumberPrasyaratHilangMenolak menjaga sifat fail-closed: guard
// yang kehilangan sumber datanya WAJIB berhenti, bukan meloloskan permintaan.
// Tanpa test ini, wiring yang lupa memanggil DenganPrasyarat akan melewati
// BR-03 tanpa suara.
func TestHTTP_BR03_SumberPrasyaratHilangMenolak(t *testing.T) {
	// Sengaja TIDAK memanggil DenganPrasyarat.
	h := NewSkoringHandler(
		service.NewSkoringService(newFakeParamRepoSkoring()),
		service.NewMarginService(newFakeParamRepoSkoring()),
	)
	r := NewRouterWithAllHandlers(config.Config{AppEnv: "test"}, nil, nil, nil, h)

	rec := postJSON(t, r, "/api/pengajuan/7/skoring", dataSkoring())
	if rec.Code == http.StatusOK {
		t.Fatalf("status = 200; guard tanpa sumber data seharusnya MENOLAK, bukan meloloskan (body=%s)",
			rec.Body.String())
	}
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, mau 500 CONFIG_ERROR", rec.Code)
	}

	// Status 500 saja TIDAK cukup sebagai bukti. Ketika guard di-bypass,
	// perhitungan tetap gagal karena kolektibilitas 0 tidak dikenal tabel
	// parameter — juga 500, juga CONFIG_ERROR. Test yang berhenti di kode
	// status akan hijau untuk alasan yang salah (terbukti saat mutation
	// testing). Karena itu pesannya harus menunjuk PRASYARAT, bukan skoring.
	errResp := decodeError(t, rec)
	if errResp.Error != "CONFIG_ERROR" {
		t.Errorf("error = %q, mau CONFIG_ERROR", errResp.Error)
	}
	if !strings.Contains(strings.ToLower(errResp.Message), "prasyarat") {
		t.Errorf("message = %q; mau menyebut prasyarat — penolakan ini harus datang dari guard BR-03, "+
			"bukan dari kegagalan perhitungan di hilir", errResp.Message)
	}
}

// TestHTTP_BR03_GagalBacaKeadaanTidakDianggapLolos memastikan kegagalan
// membaca keadaan tidak ditelan menjadi "prasyarat terpenuhi".
func TestHTTP_BR03_GagalBacaKeadaanTidakDianggapLolos(t *testing.T) {
	h, pra := routerSkoringDenganPrasyarat(newFakeParamRepoSkoring())
	pra.err = errors.New("koneksi database terputus")

	rec := postJSON(t, h, "/api/pengajuan/7/skoring", dataSkoring())
	if rec.Code == http.StatusOK {
		t.Fatalf("status = 200; kegagalan baca keadaan tidak boleh dianggap lolos (body=%s)",
			rec.Body.String())
	}

	// Pesan error tidak boleh membocorkan detail internal ke klien.
	if strings.Contains(rec.Body.String(), "koneksi database terputus") {
		t.Error("detail error internal bocor ke respons klien")
	}
}

// TestHTTP_BR03_PengajuanTidakDitemukanBukan422 memastikan id yang tidak ada
// menghasilkan 404, bukan 422 BR-03 yang menyesatkan.
func TestHTTP_BR03_PengajuanTidakDitemukanBukan422(t *testing.T) {
	h, pra := routerSkoringDenganPrasyarat(newFakeParamRepoSkoring())
	pra.err = service.ErrTidakDitemukan

	rec := postJSON(t, h, "/api/pengajuan/404404/skoring", dataSkoring())
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, mau 404 untuk pengajuan yang tidak ada (body=%s)",
			rec.Code, rec.Body.String())
	}
}

// pastikan fakePrasyaratRepo memenuhi kontrak produksi.
var _ service.PrasyaratSkoringRepository = (*fakePrasyaratRepo)(nil)

var _ = context.Background
