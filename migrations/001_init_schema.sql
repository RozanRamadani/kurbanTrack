-- Migration: 001_init_schema.sql
-- Deskripsi: Inisialisasi skema database KurbanTrack (tabel users, animals, distributions, distribution_logs)
-- Database: PostgreSQL

-- ============================================================================
-- 1. TABEL USERS
-- Menyimpan data pengguna aplikasi (admin, koordinator_wilayah, panitia_distribusi)
-- ============================================================================
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role VARCHAR(30) NOT NULL CHECK (role IN ('admin', 'koordinator_wilayah', 'panitia_distribusi')),
    region_rt VARCHAR(10),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- 2. TABEL ANIMALS
-- Menyimpan data hewan qurban, estimasi paket daging, dan kuota alokasi
-- ============================================================================
CREATE TABLE IF NOT EXISTS animals (
    id SERIAL PRIMARY KEY,
    code VARCHAR(20) UNIQUE NOT NULL,
    species VARCHAR(20) NOT NULL CHECK (species IN ('sapi', 'kambing', 'domba')),
    sohibul_name VARCHAR(150) NOT NULL,
    weight_kg NUMERIC(6,2) NOT NULL CHECK (weight_kg > 0),
    estimated_packages INT NOT NULL CHECK (estimated_packages > 0),
    allocated_packages INT NOT NULL DEFAULT 0 CHECK (allocated_packages >= 0 AND allocated_packages <= estimated_packages),
    status VARCHAR(20) NOT NULL DEFAULT 'registered' CHECK (status IN ('registered', 'slaughtered', 'distributed')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- 3. TABEL DISTRIBUTIONS
-- Menyimpan alokasi paket daging hewan qurban kepada penerima kupon
-- ============================================================================
CREATE TABLE IF NOT EXISTS distributions (
    id SERIAL PRIMARY KEY,
    coupon_code VARCHAR(30) UNIQUE NOT NULL,
    animal_id INT NOT NULL REFERENCES animals(id) ON DELETE RESTRICT,
    coordinator_id INT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    recipient_nik VARCHAR(16) UNIQUE NOT NULL CHECK (recipient_nik ~ '^[0-9]{16}$'),
    recipient_name VARCHAR(100) NOT NULL,
    recipient_rt VARCHAR(10) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'allocated' CHECK (status IN ('allocated', 'redeemed', 'cancelled')),
    redeemed_at TIMESTAMP WITH TIME ZONE NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index non-unique untuk optimasi join foreign key dan query filter status
CREATE INDEX IF NOT EXISTS idx_distributions_animal_id ON distributions(animal_id);
CREATE INDEX IF NOT EXISTS idx_distributions_coordinator_id ON distributions(coordinator_id);
CREATE INDEX IF NOT EXISTS idx_distributions_status ON distributions(status);
CREATE INDEX IF NOT EXISTS idx_distributions_recipient_rt ON distributions(recipient_rt);

-- ============================================================================
-- 4. TABEL DISTRIBUTION_LOGS
-- Mencatat jejak audit (audit trail) setiap perubahan status kupon distribusi
-- ============================================================================
CREATE TABLE IF NOT EXISTS distribution_logs (
    id SERIAL PRIMARY KEY,
    distribution_id INT NOT NULL REFERENCES distributions(id) ON DELETE CASCADE,
    actor_id INT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    from_status VARCHAR(20),
    to_status VARCHAR(20) NOT NULL,
    note TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index non-unique untuk optimasi pencarian riwayat log distribusi
CREATE INDEX IF NOT EXISTS idx_distribution_logs_distribution_id ON distribution_logs(distribution_id);
CREATE INDEX IF NOT EXISTS idx_distribution_logs_actor_id ON distribution_logs(actor_id);
