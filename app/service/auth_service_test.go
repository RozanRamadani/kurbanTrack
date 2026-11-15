package service_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"kurbantrack-backend/app/model"
	"kurbantrack-backend/app/repository"
	"kurbantrack-backend/app/service"
	"kurbantrack-backend/helper"
)

type mockUserRepoForServiceTest struct {
	users          map[int]*model.User
	nextID         int
	createErr      error
	findByEmailErr error
}

func newMockUserRepoForServiceTest() *mockUserRepoForServiceTest {
	return &mockUserRepoForServiceTest{
		users:  make(map[int]*model.User),
		nextID: 1,
	}
}

func (m *mockUserRepoForServiceTest) Create(ctx context.Context, user *model.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	for _, u := range m.users {
		if u.Email == user.Email {
			return repository.ErrConflict
		}
	}
	user.ID = m.nextID
	m.nextID++
	user.CreatedAt = time.Now()
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepoForServiceTest) FindByID(ctx context.Context, id int) (*model.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepoForServiceTest) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	if m.findByEmailErr != nil {
		return nil, m.findByEmailErr
	}
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (m *mockUserRepoForServiceTest) FindAll(ctx context.Context) ([]model.User, error) {
	list := make([]model.User, 0, len(m.users))
	for _, u := range m.users {
		list = append(list, *u)
	}
	return list, nil
}

func TestAuthService_RegisterUser_Success(t *testing.T) {
	repo := newMockUserRepoForServiceTest()
	svc := service.NewAuthService(repo)

	req := &model.UserCreateRequest{
		Name:     "Budi Panitia",
		Email:    "budi@kurbantrack.local",
		Password: "password123",
		Role:     model.RolePanitiaDistribusi,
	}

	res, err := svc.RegisterUser(context.Background(), req)
	if err != nil {
		t.Fatalf("RegisterUser gagal: %v", err)
	}

	if res.ID <= 0 || res.Name != "Budi Panitia" || res.Email != "budi@kurbantrack.local" {
		t.Errorf("UserResponse tidak sesuai: %+v", res)
	}
	if res.Role != model.RolePanitiaDistribusi {
		t.Errorf("Role tidak sesuai: %s", res.Role)
	}
}

func TestAuthService_RegisterUser_NilRequest(t *testing.T) {
	repo := newMockUserRepoForServiceTest()
	svc := service.NewAuthService(repo)

	_, err := svc.RegisterUser(context.Background(), nil)
	if err == nil {
		t.Fatal("RegisterUser seharusnya menghasilkan error untuk request nil")
	}

	var valErr *service.ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("Ekspektasi ValidationError, didapat: %T", err)
	}
}

func TestAuthService_RegisterUser_DuplicateEmail(t *testing.T) {
	repo := newMockUserRepoForServiceTest()
	svc := service.NewAuthService(repo)

	req := &model.UserCreateRequest{
		Name:     "Budi Panitia",
		Email:    "budi@kurbantrack.local",
		Password: "password123",
		Role:     model.RolePanitiaDistribusi,
	}

	_, err := svc.RegisterUser(context.Background(), req)
	if err != nil {
		t.Fatalf("Registrasi pertama gagal: %v", err)
	}

	// Registrasi kedua dengan email yang sama
	_, err = svc.RegisterUser(context.Background(), req)
	if !errors.Is(err, service.ErrConflict) {
		t.Errorf("Ekspektasi ErrConflict saat email duplikat, didapat: %v", err)
	}
}

func TestAuthService_RegisterUser_PasswordLessThan6Chars(t *testing.T) {
	repo := newMockUserRepoForServiceTest()
	svc := service.NewAuthService(repo)

	req := &model.UserCreateRequest{
		Name:     "Budi Panitia",
		Email:    "budi@kurbantrack.local",
		Password: "12345", // 5 karakter
		Role:     model.RolePanitiaDistribusi,
	}

	_, err := svc.RegisterUser(context.Background(), req)
	if err == nil {
		t.Fatal("RegisterUser seharusnya gagal saat password < 6 karakter")
	}

	var valErr *service.ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("Ekspektasi ValidationError, didapat: %T", err)
	}
}

func TestAuthService_RegisterUser_EmptyNameOrEmail(t *testing.T) {
	repo := newMockUserRepoForServiceTest()
	svc := service.NewAuthService(repo)

	// Nama kosong
	req1 := &model.UserCreateRequest{
		Name:     "",
		Email:    "test@test.local",
		Password: "password123",
	}
	if _, err := svc.RegisterUser(context.Background(), req1); err == nil {
		t.Error("Harus error jika nama kosong")
	}

	// Email kosong
	req2 := &model.UserCreateRequest{
		Name:     "Tester",
		Email:    "",
		Password: "password123",
	}
	if _, err := svc.RegisterUser(context.Background(), req2); err == nil {
		t.Error("Harus error jika email kosong")
	}
}

func TestAuthService_RegisterUser_InvalidRole(t *testing.T) {
	repo := newMockUserRepoForServiceTest()
	svc := service.NewAuthService(repo)

	req := &model.UserCreateRequest{
		Name:     "Attacker",
		Email:    "attacker@evil.local",
		Password: "password123",
		Role:     "superadmin_invalid",
	}

	_, err := svc.RegisterUser(context.Background(), req)
	if err == nil {
		t.Fatal("RegisterUser seharusnya gagal saat role tidak valid")
	}

	var valErr *service.ValidationError
	if !errors.As(err, &valErr) || valErr.Field != "role" {
		t.Errorf("Ekspektasi ValidationError pada field role, didapat: %v", err)
	}
}

func TestAuthService_RegisterUser_DefaultRole(t *testing.T) {
	repo := newMockUserRepoForServiceTest()
	svc := service.NewAuthService(repo)

	// Request dengan role kosong
	req := &model.UserCreateRequest{
		Name:     "Budi",
		Email:    "budi@test.local",
		Password: "password123",
		Role:     "",
	}

	res, err := svc.RegisterUser(context.Background(), req)
	if err != nil {
		t.Fatalf("RegisterUser gagal: %v", err)
	}

	if res.Role != model.RolePanitiaDistribusi {
		t.Errorf("Role default harus panitia_distribusi, didapat: %s", res.Role)
	}
}

func TestAuthService_RegisterUser_RepositoryGenericError(t *testing.T) {
	repo := newMockUserRepoForServiceTest()
	repo.createErr = errors.New("database connection failed")
	svc := service.NewAuthService(repo)

	req := &model.UserCreateRequest{
		Name:     "Budi",
		Email:    "budi@test.local",
		Password: "password123",
		Role:     model.RolePanitiaDistribusi,
	}

	_, err := svc.RegisterUser(context.Background(), req)
	if err == nil || err.Error() != "database connection failed" {
		t.Errorf("Ekspektasi error database connection failed, didapat: %v", err)
	}
}

func TestAuthService_LoginUser_Success(t *testing.T) {
	_ = os.Setenv("JWT_SECRET", "test-secret-key-for-auth-tests")
	repo := newMockUserRepoForServiceTest()
	svc := service.NewAuthService(repo)

	// Register user terlebih dahulu
	hashed, _ := helper.HashPassword("secretPassword123")
	user := &model.User{
		Name:         "Admin Test",
		Email:        "admin@test.local",
		PasswordHash: hashed,
		Role:         model.RoleAdmin,
	}
	_ = repo.Create(context.Background(), user)

	loginReq := &model.UserLoginRequest{
		Email:    "admin@test.local",
		Password: "secretPassword123",
	}

	res, err := svc.LoginUser(context.Background(), loginReq)
	if err != nil {
		t.Fatalf("LoginUser gagal: %v", err)
	}

	if res.Token == "" {
		t.Error("Token JWT tidak boleh kosong")
	}
	if res.User.Email != "admin@test.local" || res.User.Role != model.RoleAdmin {
		t.Errorf("User data pada AuthResponse tidak sesuai: %+v", res.User)
	}
}

func TestAuthService_LoginUser_NilRequest(t *testing.T) {
	repo := newMockUserRepoForServiceTest()
	svc := service.NewAuthService(repo)

	_, err := svc.LoginUser(context.Background(), nil)
	if err == nil {
		t.Fatal("LoginUser seharusnya gagal saat request nil")
	}

	var valErr *service.ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("Ekspektasi ValidationError, didapat: %T", err)
	}
}

func TestAuthService_LoginUser_EmailNotFound(t *testing.T) {
	repo := newMockUserRepoForServiceTest()
	svc := service.NewAuthService(repo)

	loginReq := &model.UserLoginRequest{
		Email:    "notfound@test.local",
		Password: "somePassword123",
	}

	_, err := svc.LoginUser(context.Background(), loginReq)
	if !errors.Is(err, service.ErrUnauthorized) {
		t.Errorf("Ekspektasi ErrUnauthorized saat email tidak ditemukan, didapat: %v", err)
	}
}

func TestAuthService_LoginUser_WrongPassword(t *testing.T) {
	repo := newMockUserRepoForServiceTest()
	svc := service.NewAuthService(repo)

	hashed, _ := helper.HashPassword("correctPassword123")
	user := &model.User{
		Name:         "User Test",
		Email:        "user@test.local",
		PasswordHash: hashed,
		Role:         model.RolePanitiaDistribusi,
	}
	_ = repo.Create(context.Background(), user)

	loginReq := &model.UserLoginRequest{
		Email:    "user@test.local",
		Password: "wrongPassword123",
	}

	_, err := svc.LoginUser(context.Background(), loginReq)
	if !errors.Is(err, service.ErrUnauthorized) {
		t.Errorf("Ekspektasi ErrUnauthorized saat password salah, didapat: %v", err)
	}
}

func TestAuthService_LoginUser_EmptyCredentials(t *testing.T) {
	repo := newMockUserRepoForServiceTest()
	svc := service.NewAuthService(repo)

	// Email kosong
	_, err := svc.LoginUser(context.Background(), &model.UserLoginRequest{Email: "", Password: "pass"})
	if err == nil {
		t.Error("Harus error jika email kosong")
	}

	// Password kosong
	_, err = svc.LoginUser(context.Background(), &model.UserLoginRequest{Email: "test@test.com", Password: ""})
	if err == nil {
		t.Error("Harus error jika password kosong")
	}
}

func TestAuthService_LoginUser_RepositoryGenericError(t *testing.T) {
	repo := newMockUserRepoForServiceTest()
	repo.findByEmailErr = errors.New("db query timeout")
	svc := service.NewAuthService(repo)

	loginReq := &model.UserLoginRequest{
		Email:    "user@test.local",
		Password: "password123",
	}

	_, err := svc.LoginUser(context.Background(), loginReq)
	if err == nil || err.Error() != "db query timeout" {
		t.Errorf("Ekspektasi db query timeout, didapat: %v", err)
	}
}
