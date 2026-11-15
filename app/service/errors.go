package service

import "errors"

var (
	// ErrNotFound dikembalikan ketika entitas yang dicari tidak ditemukan.
	ErrNotFound = errors.New("record not found")

	// ErrConflict dikembalikan saat terjadi duplikasi data unik (misal: kode hewan).
	ErrConflict = errors.New("resource conflict or duplicate entry")

	// ErrHasActiveAllocations dikembalikan saat mencoba menghapus hewan yang masih memiliki kupon alokasi aktif.
	ErrHasActiveAllocations = errors.New("hewan tidak dapat dihapus karena masih memiliki alokasi aktif")

	// ErrForeignKeyViolation dikembalikan saat penghapusan gagal karena pelanggaran integritas foreign key di database.
	ErrForeignKeyViolation = errors.New("hewan tidak dapat dihapus karena masih memiliki data riwayat distribusi terkait")

	// ErrQuotaExceeded dikembalikan saat alokasi kupon melebihi kuota paket daging hewan yang tersedia.
	ErrQuotaExceeded = errors.New("kuota paket daging untuk hewan ini sudah habis")

	// ErrInvalidCouponStatus dikembalikan saat transisi status kupon (redeem/cancel) tidak memenuhi syarat status 'allocated'.
	ErrInvalidCouponStatus = errors.New("status kupon tidak memenuhi syarat untuk operasi ini")
)

// ValidationError merepresentasikan kesalahan validasi aturan bisnis pada input pengguna.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Message
}
