// Akses data pengajuan, dokumen, dan survei (FR-02, FR-03, FR-04).
//
// Lapisan ini hanya membaca/menulis baris — tidak ada aturan bisnis di sini
// (AGENTS.md Larangan 17). Nama kolom mengikuti docs/SDD-iMitra.md BAB 4.1.
package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/irgiys/tim1gow/backend/internal/service"
)

// ---------------------------------------------------------------------------
// Model GORM. Sengaja terpisah dari struct service.* supaya tag kolom tidak
// bocor ke lapisan aturan bisnis.
// ---------------------------------------------------------------------------

type rowPengajuan struct {
	ID               int64  `gorm:"column:id;primaryKey"`
	NomorReferensi   string `gorm:"column:nomor_referensi"`
	Tipe             string `gorm:"column:tipe"`
	AOID             int64  `gorm:"column:ao_id"`
	NamaNasabah      string `gorm:"column:nama_nasabah"`
	NIK              string `gorm:"column:nik"`
	AlamatUsaha      string `gorm:"column:alamat_usaha"`
	JenisUsaha       string `gorm:"column:jenis_usaha"`
	JenisAkad        string `gorm:"column:jenis_akad"`
	PlafonDiajukan   int64  `gorm:"column:plafon_diajukan"`
	PlafonDisetujui  *int64 `gorm:"column:plafon_disetujui"`
	TenorBulan       int    `gorm:"column:tenor_bulan"`
	MarginAtauNisbah *float64
	Status           string    `gorm:"column:status"`
	DibuatPada       time.Time `gorm:"column:dibuat_pada"`
	DiperbaruiPada   time.Time `gorm:"column:diperbarui_pada"`
}

func (rowPengajuan) TableName() string { return "pengajuan" }

type rowAnggota struct {
	ID            int64  `gorm:"column:id;primaryKey"`
	PengajuanID   int64  `gorm:"column:pengajuan_id"`
	NamaAnggota   string `gorm:"column:nama_anggota"`
	NIKAnggota    string `gorm:"column:nik_anggota"`
	PlafonAnggota int64  `gorm:"column:plafon_anggota"`
	StatusAnggota string `gorm:"column:status_anggota"`
}

func (rowAnggota) TableName() string { return "pengajuan_anggota" }

type rowDokumen struct {
	ID               int64      `gorm:"column:id;primaryKey"`
	PengajuanID      int64      `gorm:"column:pengajuan_id"`
	JenisDokumen     string     `gorm:"column:jenis_dokumen"`
	URLBerkas        string     `gorm:"column:url_berkas"`
	Status           string     `gorm:"column:status"`
	AlasanPenolakan  *string    `gorm:"column:alasan_penolakan"`
	DiverifikasiOleh *int64     `gorm:"column:diverifikasi_oleh"`
	DiverifikasiPada *time.Time `gorm:"column:diverifikasi_pada"`
	DibuatPada       time.Time  `gorm:"column:dibuat_pada"`
}

func (rowDokumen) TableName() string { return "dokumen" }

type rowSurvei struct {
	ID           int64     `gorm:"column:id;primaryKey"`
	PengajuanID  int64     `gorm:"column:pengajuan_id"`
	AOID         int64     `gorm:"column:ao_id"`
	Latitude     float64   `gorm:"column:latitude"`
	Longitude    float64   `gorm:"column:longitude"`
	FotoURL      string    `gorm:"column:foto_url"`
	Catatan      string    `gorm:"column:catatan"`
	NilaiKondisi *int16    `gorm:"column:nilai_kondisi"`
	Status       string    `gorm:"column:status"`
	DibuatPada   time.Time `gorm:"column:dibuat_pada"`
}

func (rowSurvei) TableName() string { return "survei" }

// ---------------------------------------------------------------------------
// Pemetaan row <-> entitas service
// ---------------------------------------------------------------------------

func kePengajuan(r rowPengajuan) service.Pengajuan {
	return service.Pengajuan{
		ID:               r.ID,
		NomorReferensi:   r.NomorReferensi,
		Tipe:             service.TipePengajuan(r.Tipe),
		AOID:             r.AOID,
		NamaNasabah:      r.NamaNasabah,
		NIK:              r.NIK,
		AlamatUsaha:      r.AlamatUsaha,
		JenisUsaha:       r.JenisUsaha,
		JenisAkad:        r.JenisAkad,
		PlafonDiajukan:   r.PlafonDiajukan,
		PlafonDisetujui:  r.PlafonDisetujui,
		TenorBulan:       r.TenorBulan,
		MarginAtauNisbah: r.MarginAtauNisbah,
		Status:           r.Status,
		DibuatPada:       r.DibuatPada,
		DiperbaruiPada:   r.DiperbaruiPada,
	}
}

func keDokumen(r rowDokumen) service.Dokumen {
	return service.Dokumen{
		ID:               r.ID,
		PengajuanID:      r.PengajuanID,
		JenisDokumen:     r.JenisDokumen,
		URLBerkas:        r.URLBerkas,
		Status:           service.StatusDokumen(r.Status),
		AlasanPenolakan:  r.AlasanPenolakan,
		DiverifikasiOleh: r.DiverifikasiOleh,
		DiverifikasiPada: r.DiverifikasiPada,
		DibuatPada:       r.DibuatPada,
	}
}

// ---------------------------------------------------------------------------
// PengajuanRepository
// ---------------------------------------------------------------------------

type gormPengajuanRepo struct{ db *gorm.DB }

// NewPengajuanRepository membuat implementasi GORM untuk PengajuanRepository.
func NewPengajuanRepository(db *gorm.DB) service.PengajuanRepository {
	return &gormPengajuanRepo{db: db}
}

func (r *gormPengajuanRepo) Simpan(ctx context.Context, p *service.Pengajuan) error {
	row := rowPengajuan{
		NomorReferensi: p.NomorReferensi,
		Tipe:           string(p.Tipe),
		AOID:           p.AOID,
		NamaNasabah:    p.NamaNasabah,
		NIK:            p.NIK,
		AlamatUsaha:    p.AlamatUsaha,
		JenisUsaha:     p.JenisUsaha,
		JenisAkad:      p.JenisAkad,
		PlafonDiajukan: p.PlafonDiajukan,
		TenorBulan:     p.TenorBulan,
		Status:         p.Status,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	p.ID = row.ID
	p.DibuatPada = row.DibuatPada
	p.DiperbaruiPada = row.DiperbaruiPada
	return nil
}

func (r *gormPengajuanRepo) Perbarui(ctx context.Context, p *service.Pengajuan) error {
	return r.db.WithContext(ctx).Model(&rowPengajuan{}).
		Where("id = ?", p.ID).
		Updates(map[string]any{
			"status":             p.Status,
			"plafon_disetujui":   p.PlafonDisetujui,
			"margin_atau_nisbah": p.MarginAtauNisbah,
			"diperbarui_pada":    time.Now(),
		}).Error
}

func (r *gormPengajuanRepo) CariID(ctx context.Context, id int64) (service.Pengajuan, error) {
	var row rowPengajuan
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return service.Pengajuan{}, service.ErrTidakDitemukan
	}
	if err != nil {
		return service.Pengajuan{}, err
	}
	return kePengajuan(row), nil
}

func (r *gormPengajuanRepo) DaftarMilikAO(ctx context.Context, aoID int64) ([]service.Pengajuan, error) {
	var rows []rowPengajuan
	err := r.db.WithContext(ctx).
		Where("ao_id = ?", aoID).
		Order("dibuat_pada DESC, id DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]service.Pengajuan, 0, len(rows))
	for _, row := range rows {
		out = append(out, kePengajuan(row))
	}
	return out, nil
}

// AmbilNomorUrutHarian menaikkan counter harian dengan penguncian baris.
//
// UPDATE ... RETURNING pada satu pernyataan bersifat atomik; INSERT lebih dulu
// (ON CONFLICT DO NOTHING) memastikan barisnya ada. Dua permintaan bersamaan
// karenanya tidak pernah memperoleh angka yang sama (BR-12).
func (r *gormPengajuanRepo) AmbilNomorUrutHarian(ctx context.Context, tanggal time.Time) (int, error) {
	hari := tanggal.Format("2006-01-02")
	db := r.db.WithContext(ctx)

	if err := db.Exec(`
		INSERT INTO nomor_referensi_counter (tanggal, urutan_akhir)
		VALUES (?, 0) ON CONFLICT (tanggal) DO NOTHING`, hari).Error; err != nil {
		return 0, err
	}

	var urutan int
	err := db.Raw(`
		UPDATE nomor_referensi_counter
		SET urutan_akhir = urutan_akhir + 1
		WHERE tanggal = ?
		RETURNING urutan_akhir`, hari).Scan(&urutan).Error
	if err != nil {
		return 0, err
	}
	return urutan, nil
}

func (r *gormPengajuanRepo) SimpanAnggota(ctx context.Context, pengajuanID int64, anggota []service.PengajuanAnggota) error {
	if len(anggota) == 0 {
		return nil
	}
	rows := make([]rowAnggota, 0, len(anggota))
	for _, a := range anggota {
		status := a.StatusAnggota
		if status == "" {
			status = "AKTIF"
		}
		rows = append(rows, rowAnggota{
			PengajuanID:   pengajuanID,
			NamaAnggota:   a.NamaAnggota,
			NIKAnggota:    a.NIKAnggota,
			PlafonAnggota: a.PlafonAnggota,
			StatusAnggota: status,
		})
	}
	return r.db.WithContext(ctx).Create(&rows).Error
}

func (r *gormPengajuanRepo) DaftarAnggota(ctx context.Context, pengajuanID int64) ([]service.PengajuanAnggota, error) {
	var rows []rowAnggota
	err := r.db.WithContext(ctx).
		Where("pengajuan_id = ?", pengajuanID).Order("id ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]service.PengajuanAnggota, 0, len(rows))
	for _, row := range rows {
		out = append(out, service.PengajuanAnggota{
			ID:            row.ID,
			PengajuanID:   row.PengajuanID,
			NamaAnggota:   row.NamaAnggota,
			NIKAnggota:    row.NIKAnggota,
			PlafonAnggota: row.PlafonAnggota,
			StatusAnggota: row.StatusAnggota,
		})
	}
	return out, nil
}

func (r *gormPengajuanRepo) DalamTransaksi(ctx context.Context, fn func(tx service.PengajuanRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&gormPengajuanRepo{db: tx})
	})
}

// ---------------------------------------------------------------------------
// DokumenRepository
// ---------------------------------------------------------------------------

type gormDokumenRepo struct{ db *gorm.DB }

// NewDokumenRepository membuat implementasi GORM untuk DokumenRepository.
func NewDokumenRepository(db *gorm.DB) service.DokumenRepository {
	return &gormDokumenRepo{db: db}
}

// Simpan menyisipkan dokumen baru, atau MENGGANTI baris berjenis sama pada
// pengajuan yang sama saat AO mengunggah ulang berkas yang ditolak (AC-03).
// Indeks unik (pengajuan_id, jenis_dokumen) yang membuat perilaku ini aman.
func (r *gormDokumenRepo) Simpan(ctx context.Context, d *service.Dokumen) error {
	var row rowDokumen
	err := r.db.WithContext(ctx).
		Where("pengajuan_id = ? AND jenis_dokumen = ?", d.PengajuanID, d.JenisDokumen).
		First(&row).Error

	switch {
	case err == nil:
		// Re-upload: status kembali UPLOADED, jejak verifikasi lama dibersihkan
		// supaya dokumen tidak terlihat masih ditolak setelah diganti.
		perubahan := map[string]any{
			"url_berkas":        d.URLBerkas,
			"status":            string(d.Status),
			"alasan_penolakan":  nil,
			"diverifikasi_oleh": nil,
			"diverifikasi_pada": nil,
		}
		if err := r.db.WithContext(ctx).Model(&rowDokumen{}).
			Where("id = ?", row.ID).Updates(perubahan).Error; err != nil {
			return err
		}
		d.ID = row.ID
		d.DibuatPada = row.DibuatPada
		return nil

	case errors.Is(err, gorm.ErrRecordNotFound):
		baru := rowDokumen{
			PengajuanID:  d.PengajuanID,
			JenisDokumen: d.JenisDokumen,
			URLBerkas:    d.URLBerkas,
			Status:       string(d.Status),
		}
		if err := r.db.WithContext(ctx).Create(&baru).Error; err != nil {
			return err
		}
		d.ID = baru.ID
		d.DibuatPada = baru.DibuatPada
		return nil

	default:
		return err
	}
}

func (r *gormDokumenRepo) Perbarui(ctx context.Context, d *service.Dokumen) error {
	return r.db.WithContext(ctx).Model(&rowDokumen{}).
		Where("id = ?", d.ID).
		Updates(map[string]any{
			"status":            string(d.Status),
			"alasan_penolakan":  d.AlasanPenolakan,
			"diverifikasi_oleh": d.DiverifikasiOleh,
			"diverifikasi_pada": d.DiverifikasiPada,
		}).Error
}

func (r *gormDokumenRepo) CariID(ctx context.Context, id int64) (service.Dokumen, error) {
	var row rowDokumen
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return service.Dokumen{}, service.ErrTidakDitemukan
	}
	if err != nil {
		return service.Dokumen{}, err
	}
	return keDokumen(row), nil
}

func (r *gormDokumenRepo) DaftarPerPengajuan(ctx context.Context, pengajuanID int64) ([]service.Dokumen, error) {
	var rows []rowDokumen
	err := r.db.WithContext(ctx).
		Where("pengajuan_id = ?", pengajuanID).Order("id ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]service.Dokumen, 0, len(rows))
	for _, row := range rows {
		out = append(out, keDokumen(row))
	}
	return out, nil
}

func (r *gormDokumenRepo) CariAktif(ctx context.Context, pengajuanID int64, jenis string) (service.Dokumen, error) {
	var row rowDokumen
	err := r.db.WithContext(ctx).
		Where("pengajuan_id = ? AND jenis_dokumen = ?", pengajuanID, jenis).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return service.Dokumen{}, service.ErrTidakDitemukan
	}
	if err != nil {
		return service.Dokumen{}, err
	}
	return keDokumen(row), nil
}

// ---------------------------------------------------------------------------
// SurveiRepository
// ---------------------------------------------------------------------------

type gormSurveiRepo struct{ db *gorm.DB }

// NewSurveiRepository membuat implementasi GORM untuk SurveiRepository.
func NewSurveiRepository(db *gorm.DB) service.SurveiRepository {
	return &gormSurveiRepo{db: db}
}

// Simpan menyisipkan satu hasil survei.
//
// Kolom `omzet_harian` dan `lama_usaha_bulan` hidup di tabel `pengajuan`
// (SDD BAB 4.1), bukan di `survei`, sehingga keduanya ikut diperbarui di sini
// agar menjadi masukan skoring. NilaiKondisi dipetakan dari LamaUsahaBulan?
// TIDAK — nilai kondisi adalah penilaian ANL skala 1-5 dan dikirim terpisah.
func (r *gormSurveiRepo) Simpan(ctx context.Context, s *service.Survei) error {
	row := rowSurvei{
		PengajuanID: s.PengajuanID,
		AOID:        s.AOID,
		Latitude:    s.Latitude,
		Longitude:   s.Longitude,
		FotoURL:     s.FotoURL,
		Catatan:     s.CatatanKondisi,
		Status:      string(s.Status),
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		s.ID = row.ID
		s.DibuatPada = row.DibuatPada

		// Omzet & lama usaha menjadi masukan komponen skoring; keduanya
		// disimpan pada pengajuan supaya SkoringService membacanya dari satu
		// tempat.
		return tx.Model(&rowPengajuan{}).
			Where("id = ?", s.PengajuanID).
			Updates(map[string]any{
				"omzet_harian":     s.OmzetHarian,
				"lama_usaha_bulan": s.LamaUsahaBulan,
				"diperbarui_pada":  time.Now(),
			}).Error
	})
}

func (r *gormSurveiRepo) DaftarPerPengajuan(ctx context.Context, pengajuanID int64) ([]service.Survei, error) {
	var rows []rowSurvei
	err := r.db.WithContext(ctx).
		Where("pengajuan_id = ?", pengajuanID).Order("id ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]service.Survei, 0, len(rows))
	for _, row := range rows {
		out = append(out, service.Survei{
			ID:             row.ID,
			PengajuanID:    row.PengajuanID,
			AOID:           row.AOID,
			Latitude:       row.Latitude,
			Longitude:      row.Longitude,
			FotoURL:        row.FotoURL,
			CatatanKondisi: row.Catatan,
			Status:         service.StatusSurvei(row.Status),
			DibuatPada:     row.DibuatPada,
		})
	}
	return out, nil
}

func (r *gormSurveiRepo) AdaSurveiValid(ctx context.Context, pengajuanID int64) (bool, error) {
	var jumlah int64
	err := r.db.WithContext(ctx).Model(&rowSurvei{}).
		Where("pengajuan_id = ? AND status = ?", pengajuanID, string(service.StatusSurveiValid)).
		Count(&jumlah).Error
	return jumlah > 0, err
}

// ---------------------------------------------------------------------------
// Repository parameter pendukung
// ---------------------------------------------------------------------------

type gormBatasPlafonRepo struct{ db *gorm.DB }

// NewBatasPlafonRepository membaca batas plafon dari tabel parameter_umum.
func NewBatasPlafonRepository(db *gorm.DB) service.BatasPlafonRepository {
	return &gormBatasPlafonRepo{db: db}
}

// BatasPlafon mengembalikan ditemukan=false bila salah satu batas belum
// diatur. Nilai nol TIDAK dipakai sebagai default: batas yang hilang membuat
// BR-01 tidak dapat ditegakkan, dan service memilih berhenti daripada
// meloloskan plafon sembarang (AGENTS.md Larangan 3).
func (r *gormBatasPlafonRepo) BatasPlafon(ctx context.Context) (int64, int64, bool, error) {
	var rows []struct {
		Kunci string
		Nilai float64
	}
	err := r.db.WithContext(ctx).
		Table("parameter_umum").
		Select("kunci, nilai").
		Where("kunci IN ?", []string{"batas_plafon_min", "batas_plafon_maks"}).
		Scan(&rows).Error
	if err != nil {
		return 0, 0, false, err
	}

	var minimum, maksimum int64
	var adaMin, adaMaks bool
	for _, row := range rows {
		switch row.Kunci {
		case "batas_plafon_min":
			minimum, adaMin = int64(row.Nilai), true
		case "batas_plafon_maks":
			maksimum, adaMaks = int64(row.Nilai), true
		}
	}
	if !adaMin || !adaMaks {
		return 0, 0, false, nil
	}
	return minimum, maksimum, true, nil
}

type gormDokumenWajibRepo struct{ db *gorm.DB }

// NewDokumenWajibRepository membaca daftar jenis dokumen wajib dari tabel
// dokumen_wajib (migrasi 000005).
func NewDokumenWajibRepository(db *gorm.DB) service.DokumenWajibRepository {
	return &gormDokumenWajibRepo{db: db}
}

func (r *gormDokumenWajibRepo) JenisDokumenWajib(ctx context.Context) ([]string, error) {
	var jenis []string
	err := r.db.WithContext(ctx).
		Table("dokumen_wajib").
		Where("aktif = TRUE").
		Order("jenis_dokumen ASC").
		Pluck("jenis_dokumen", &jenis).Error
	return jenis, err
}
