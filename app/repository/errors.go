package repository

import "errors"

var (
	// ErrNotFound dikembalikan ketika data yang diminta tidak ditemukan di database.
	ErrNotFound = errors.New("record not found")

	// ErrConflict dikembalikan ketika terjadi pelanggaran constraint unik atau data duplikat.
	ErrConflict = errors.New("resource conflict or duplicate entry")

	// ErrForeignKeyViolation dikembalikan ketika operasi melanggar constraint foreign key.
	ErrForeignKeyViolation = errors.New("foreign key constraint violation")

	// ErrConstraintViolation dikembalikan ketika operasi melanggar check constraint database.
	ErrConstraintViolation = errors.New("database check constraint violation")

	// ErrQuotaExceeded dikembalikan ketika kuota paket daging hewan sudah habis teralokasi.
	ErrQuotaExceeded = errors.New("animal package quota exceeded")

	// ErrInvalidStatusTransition dikembalikan ketika perubahan status kupon tidak valid.
	ErrInvalidStatusTransition = errors.New("invalid distribution status transition")
)
