package service

// fakeNomorRepo meniru penyimpanan nomor referensi pengajuan di database.
//
// Yang ditirukan bukan sekadar "menyimpan angka", tetapi sifat yang diminta
// BR-12: sebuah nomor yang sudah pernah diterbitkan tetap terpakai selamanya,
// termasuk ketika pengajuan pemiliknya ditolak. Karena itu fake ini mencatat
// urutan terakhir per tanggal dan TIDAK pernah menurunkannya kembali.
type fakeNomorRepo struct {
	// urutanTerakhir[tanggal] = nomor urut terakhir yang sudah diterbitkan.
	urutanTerakhir map[string]int

	// diterbitkan mencatat setiap nomor yang pernah keluar, supaya test dapat
	// membuktikan tidak ada yang dipakai ulang.
	diterbitkan map[string]bool

	// ditolak mencatat nomor milik pengajuan yang berakhir ditolak.
	ditolak map[string]bool
}

func newFakeNomorRepo() *fakeNomorRepo {
	return &fakeNomorRepo{
		urutanTerakhir: map[string]int{},
		diterbitkan:    map[string]bool{},
		ditolak:        map[string]bool{},
	}
}

// UrutanBerikutnya mengembalikan nomor urut berikutnya untuk satu tanggal.
// Di implementasi nyata ini berupa sequence/tabel dengan constraint UNIQUE,
// bukan hitungan ulang dari baris pengajuan yang masih ada — pengajuan yang
// ditolak pun tetap memakai nomornya (BR-12).
func (f *fakeNomorRepo) UrutanBerikutnya(tanggal string) (int, error) {
	f.urutanTerakhir[tanggal]++
	return f.urutanTerakhir[tanggal], nil
}

// CatatNomor menandai satu nomor sudah terpakai.
func (f *fakeNomorRepo) CatatNomor(nomor string) error {
	f.diterbitkan[nomor] = true
	return nil
}

// tandaiDitolak meniru pengajuan yang berakhir REJECTED_SLIK / REJECTED_SCORING
// / REJECTED. Nomornya tetap tercatat sebagai sudah diterbitkan.
func (f *fakeNomorRepo) tandaiDitolak(nomor string) {
	f.ditolak[nomor] = true
}
