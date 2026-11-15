# KurbanTrack Backend

RESTful API backend untuk sistem manajemen hewan qurban, estimasi paket daging, dan alokasi kupon distribusi qurban. Proyek ini dibangun menggunakan bahasa pemrograman **Go** dengan web framework **Fiber** dan database **PostgreSQL**, menerapkan prinsip **Clean Architecture** sesuai standar Modul 4 Praktikum Pemrograman Backend.

---

## 1. Project Overview

KurbanTrack dirancang untuk mempermudah panitia qurban dan koordinator wilayah dalam mengelola siklus hidup hewan qurban—mulai dari pendaftaran hewan, pencatatan bobot, estimasi kuota paket daging, hingga alokasi dan penebusan kupon distribusi kepada penerima yang berhak (mustahik).

Pada tahap rilis saat ini:
- **Animals CRUD**: Telah diimplementasikan penuh (*fully implemented end-to-end*) dari routing, validasi aturan bisnis, repository, hingga integrasi database PostgreSQL.
- **Distribution API**: Masih berupa kerangka kerja bersih (*scaffold only*) yang siap dikompilasi, tanpa logika alokasi atau transaksi penuh.

---

## 2. Tech Stack

- **Bahasa Pemrograman**: Go (v1.22+)
- **HTTP Web Framework**: [Fiber v2](https://github.com/gofiber/fiber) (v2.52.16)
- **Database**: PostgreSQL (v14+ / v18)
- **Database Driver & Pool**: [pgx/v5](https://github.com/jackc/pgx) (`pgxpool`)
- **Environment Loader**: [godotenv](https://github.com/joho/godotenv)
- **Arsitektur**: Clean Architecture (Modul 4)

---

## 3. Project Structure

Struktur direktori mematuhi batasan arsitektur Modul 4 tanpa menggunakan package `controller`, `handler`, `domain`, `usecase`, atau `delivery`:

```text
kurbanTrack/
├── app/
│   ├── model/                  # Definisi struct entitas, request/response DTO, dan konstanta
│   │   ├── animal.go
│   │   ├── distribution.go
│   │   ├── distribution_log.go
│   │   └── user.go
│   ├── repository/             # Akses database PostgreSQL, SQL terparameterisasi & row scanning
│   │   ├── animal_repository.go
│   │   ├── distribution_log_repository.go
│   │   ├── distribution_repository.go
│   │   ├── errors.go
│   │   └── user_repository.go
│   └── service/                # Business logic murni & parsing request HTTP Fiber
│       ├── animal_rules.go
│       ├── animal_service.go
│       ├── distribution_service.go
│       ├── errors.go
│       └── health_service.go
├── config/                     # Konfigurasi aplikasi Fiber, env loader, dan logger
│   ├── app.go
│   ├── env.go
│   └── logger.go
├── database/                   # Manajemen connection pool PostgreSQL (pgxpool)
│   └── postgres.go
├── helper/                     # Helper respon JSON terstandarisasi
│   └── response.go
├── logs/                       # Direktori log sistem (diabaikan Git)
├── middleware/                 # Middleware kustom (jika diperlukan)
├── migrations/                 # Skrip migrasi DDL PostgreSQL
│   └── 001_init_schema.sql
├── route/                      # Pemetaan rute URL ke method di layer service
│   └── route.go
├── .env.example                # Template variabel lingkungan
├── .gitignore
├── go.mod
├── go.sum
├── main.go                     # Titik masuk utama, dependency wiring & graceful shutdown
└── README.md
```

---

## 4. Prerequisites

Sebelum menjalankan aplikasi, pastikan perangkat Anda telah terpasang:
1. **Go**: Versi 1.22 atau yang lebih baru.
2. **PostgreSQL**: Versi 14+ (direkomendasikan versi 16 atau 18).
3. **Git**: Untuk manajemen repositori kode sumber.

---

## 5. Installation

1. Clone repositori ke direktori lokal:
   ```bash
   git clone <URL_REPOSITORY>
   cd kurbanTrack
   ```

2. Unduh seluruh dependensi Go:
   ```bash
   go mod download
   ```

---

## 6. Environment Configuration

Salin file `.env.example` menjadi `.env`:
```bash
cp .env.example .env
```

Sesuaikan konfigurasi pada `.env` dengan kredensial database PostgreSQL lokal Anda:
```env
APP_PORT=3000
APP_ENV=development

DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_postgres_password
DB_NAME=kurbantrack_db
DB_SSLMODE=disable
```

> [!NOTE]
> File `.env` bersifat rahasia dan telah masuk dalam daftar `.gitignore` sehingga tidak akan terlacak oleh Git.

---

## 7. Database Migration

Pastikan database `kurbantrack_db` telah dibuat di PostgreSQL:
```sql
CREATE DATABASE kurbantrack_db;
```

Jalankan file migrasi skema SQL melalui terminal menggunakan PostgreSQL CLI (`psql`):
```bash
psql -h localhost -p 5432 -U postgres -d kurbantrack_db -v ON_ERROR_STOP=1 -f migrations/001_init_schema.sql
```

Skrip migrasi ini akan membuat 4 tabel:
- `users`: Data pengguna (admin, koordinator wilayah, panitia distribusi).
- `animals`: Data hewan qurban, bobot, dan estimasi paket.
- `distributions`: Kupon alokasi penerima paket daging.
- `distribution_logs`: Jejak audit perubahan status kupon.

---

## 8. Running the Application

Jalankan server backend:
```bash
go run main.go
```

Aplikasi akan berjalan pada port yang telah ditentukan (misal `http://localhost:3000`).

---

## 9. Running Tests

Proyek ini dilengkapi dengan unit test dan HTTP integration test menggunakan mock repository in-memory sehingga **dapat dijalankan secara mandiri tanpa memerlukan PostgreSQL aktif**:

```bash
# Menjalankan seluruh pengujian unit & integration test
go test -v ./...

# Menjalankan verifikasi kompilasi
go build ./...

# Memeriksa format kode
gofmt -d .
```

Jika ingin menjalankan pengujian live koneksi database secara fisik, setel variabel `TEST_DATABASE_URL`:
```bash
TEST_DATABASE_URL="postgres://postgres:password@localhost:5432/kurbantrack_db?sslmode=disable" go test -v ./database
```

---

## 10. API Endpoint Documentation

### Base URL: `/api/v1`

Format respon JSON konsisten:
```json
{
  "success": true,
  "message": "Pesan deskriptif",
  "data": null,
  "error": null
}
```

---

### A. Health Check
- **`GET /health`**
  - Memeriksa kesiapan server dan status koneksi live PostgreSQL.
  - **Status**: `200 OK` (sehat) / `503 Service Unavailable` (jika DB terputus).

---

### B. Animals CRUD (Fully Implemented)

#### 1. Pendaftaran Hewan Qurban Baru
- **Method & Path**: `POST /api/v1/animals`
- **Status Sukses**: `201 Created`
- **Request Body**:
  ```json
  {
    "code": "SP-001",
    "species": "sapi",
    "sohibul_name": "Haji Ahmad",
    "weight_kg": 350.50,
    "estimated_packages": 100
  }
  ```
- **Aturan Bisnis**:
  - `code`: Wajib diisi dan unik (`409 Conflict` jika duplikat).
  - `species`: Hanya menerima `sapi`, `kambing`, atau `domba`.
  - `weight_kg`: Harus `> 0`, `<= 9999.99`, dan maksimal 2 angka desimal.
  - `estimated_packages`: Harus integer `> 0`.
  - `status`: Otomatis diisi `registered`.
  - `allocated_packages`: Otomatis diisi `0`.

#### 2. Mengambil Semua Hewan Qurban
- **Method & Path**: `GET /api/v1/animals`
- **Status Sukses**: `200 OK`

#### 3. Mengambil Detail Hewan Qurban
- **Method & Path**: `GET /api/v1/animals/:id`
- **Status Sukses**: `200 OK` (`404 Not Found` jika ID tidak ada).

#### 4. Memperbarui Data Hewan Qurban
- **Method & Path**: `PUT /api/v1/animals/:id`
- **Status Sukses**: `200 OK`
- **Request Body**:
  ```json
  {
    "code": "SP-001",
    "species": "sapi",
    "sohibul_name": "Haji Ahmad Dahlan",
    "weight_kg": 355.00,
    "estimated_packages": 105,
    "status": "slaughtered"
  }
  ```
- **Aturan Bisnis**:
  - `id`, `created_at`, dan `allocated_packages` tidak dapat diubah dari request.
  - `estimated_packages` tidak boleh lebih kecil dari alokasi aktif yang sudah ada.

#### 5. Menghapus Data Hewan Qurban
- **Method & Path**: `DELETE /api/v1/animals/:id`
- **Status Sukses**: `200 OK`
- **Aturan Bisnis**:
  - Ditolak (`409 Conflict`) jika hewan masih memiliki alokasi kupon aktif (`status = 'allocated'`).
  - Ditolak (`409 Conflict`) jika melanggar foreign key integrity PostgreSQL.

---

### C. Distribution API (Scaffold Only)

Endpoint distribusi berikut saat ini telah didaftarkan sebagai kerangka kerja (*scaffold*) yang mengembalikan respons transparan:

| Method | Endpoint | Respon Status | Keterangan |
| :--- | :--- | :---: | :--- |
| `POST` | `/api/v1/distributions` | `501 Not Implemented` | Alokasi paket daging belum diimplementasikan |
| `GET` | `/api/v1/distributions` | `200 OK` | Daftar alokasi distribusi (status: `scaffold_only`) |
| `PATCH` | `/api/v1/distributions/:code/redeem` | `501 Not Implemented` | Alur penebusan kupon belum diimplementasikan |
| `PATCH` | `/api/v1/distributions/:code/cancel` | `501 Not Implemented` | Alur pembatalan kupon belum diimplementasikan |

---

## 11. Current Implementation Status

| Komponen / Fitur | Status Implementasi | Keterangan |
| :--- | :---: | :--- |
| **Health Check (`GET /health`)** | **Selesai (100%)** | Pengecekan server Fiber & ping koneksi database PostgreSQL. |
| **Animals CRUD** | **Selesai (100%)** | Validasi aturan bisnis lengkap, repository, dan HTTP endpoint. |
| **Database Migration** | **Selesai (100%)** | 4 tabel (`users`, `animals`, `distributions`, `distribution_logs`). |
| **Unit & Integration Tests** | **Selesai (100%)** | 41 pengujian otomatis lolos (model, rules, service, HTTP route). |
| **Distribution API** | **Scaffold Only** | Route dan service stub siap; business logic alokasi belum dibuka. |
| **Authentication & Authorization** | **Belum** | Fitur registrasi/login JWT berada di luar cakupan rilis ini. |

---

## 12. License

Proyek ini dikembangkan untuk keperluan akademik Praktikum Pemrograman Backend (Semester 5).
