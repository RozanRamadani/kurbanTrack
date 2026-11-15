package repository

import "errors"

var (
	// ErrNotFound dikembalikan ketika data yang diminta tidak ditemukan di database.
	ErrNotFound = errors.New("record not found")

	// ErrConflict dikembalikan ketika terjadi pelanggaran constraint unik atau data duplikat.
	ErrConflict = errors.New("resource conflict or duplicate entry")

	// ErrForeignKeyViolation dikembalikan ketika operasi melanggar constraint foreign key.
	ErrForeignKeyViolation = errors.New("foreign key constraint violation")
)
