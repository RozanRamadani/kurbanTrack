package model

import "time"

// Konstanta role pengguna
const (
	RoleAdmin              = "admin"
	RoleKoordinatorWilayah = "koordinator_wilayah"
	RolePanitiaDistribusi  = "panitia_distribusi"
)

// User merepresentasikan entitas pengguna dalam sistem KurbanTrack.
type User struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	RegionRT     *string   `json:"region_rt,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// UserCreateRequest adalah DTO request pembuatan user baru.
type UserCreateRequest struct {
	Name     string  `json:"name"`
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Role     string  `json:"role"`
	RegionRT *string `json:"region_rt,omitempty"`
}

// UserResponse adalah DTO respon user tanpa password hash.
type UserResponse struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	RegionRT  *string   `json:"region_rt,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// UserLoginRequest adalah DTO request untuk login pengguna.
type UserLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse adalah DTO response setelah login berhasil.
type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}
