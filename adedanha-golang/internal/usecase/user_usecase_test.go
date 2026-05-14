package usecase

import (
	"testing"

	"adedanha-golang/internal/domain"
)

// Mock repository for testing
type mockUserRepo struct {
	users    map[string]domain.User
	byEmail  map[string]domain.User
	createFn func(domain.User) error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:   make(map[string]domain.User),
		byEmail: make(map[string]domain.User),
	}
}

func (m *mockUserRepo) Create(user domain.User) error {
	if m.createFn != nil {
		return m.createFn(user)
	}
	if _, exists := m.byEmail[user.Email]; exists {
		return domain.ErrEmailExists
	}
	m.users[user.ID] = user
	m.byEmail[user.Email] = user
	return nil
}

func (m *mockUserRepo) GetByID(id string) (domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

func (m *mockUserRepo) GetByEmail(email string) (domain.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

func (m *mockUserRepo) Update(id, name, email string) (domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	u.Name = name
	u.Email = email
	m.users[id] = u
	return u, nil
}

func (m *mockUserRepo) UpdateAvatar(id, avatar string) error {
	if _, ok := m.users[id]; !ok {
		return domain.ErrUserNotFound
	}
	return nil
}

func (m *mockUserRepo) Exists(id string) (bool, error) {
	_, ok := m.users[id]
	return ok, nil
}

func (m *mockUserRepo) GetOnlineUsersByIDs(ids []string) ([]domain.OnlineUser, error) {
	return []domain.OnlineUser{}, nil
}

func (m *mockUserRepo) GetAvailablePlayersByIDs(ids []string) ([]domain.OnlineUser, error) {
	return []domain.OnlineUser{}, nil
}

// Tests

func TestRegister_Success(t *testing.T) {
	repo := newMockUserRepo()
	uc := NewUserUseCase(repo)

	user, err := uc.Register("John", "john@example.com")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if user.Name != "John" {
		t.Errorf("Expected name 'John', got '%s'", user.Name)
	}
	if user.Email != "john@example.com" {
		t.Errorf("Expected email 'john@example.com', got '%s'", user.Email)
	}
	if user.ID == "" {
		t.Error("Expected non-empty ID")
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	repo := newMockUserRepo()
	uc := NewUserUseCase(repo)

	_, err := uc.Register("John", "invalid")
	if err != domain.ErrInvalidEmail {
		t.Errorf("Expected ErrInvalidEmail, got %v", err)
	}
}

func TestRegister_EmptyName(t *testing.T) {
	repo := newMockUserRepo()
	uc := NewUserUseCase(repo)

	_, err := uc.Register("", "john@example.com")
	if err != domain.ErrNameRequired {
		t.Errorf("Expected ErrNameRequired, got %v", err)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := newMockUserRepo()
	uc := NewUserUseCase(repo)

	_, _ = uc.Register("John", "john@example.com")
	_, err := uc.Register("Jane", "john@example.com")
	if err != domain.ErrEmailExists {
		t.Errorf("Expected ErrEmailExists, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	repo := newMockUserRepo()
	uc := NewUserUseCase(repo)

	registered, _ := uc.Register("John", "john@example.com")

	user, err := uc.Login("john@example.com")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if user.ID != registered.ID {
		t.Errorf("Expected ID '%s', got '%s'", registered.ID, user.ID)
	}
}

func TestLogin_NotFound(t *testing.T) {
	repo := newMockUserRepo()
	uc := NewUserUseCase(repo)

	_, err := uc.Login("nobody@example.com")
	if err != domain.ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestLogin_EmptyEmail(t *testing.T) {
	repo := newMockUserRepo()
	uc := NewUserUseCase(repo)

	_, err := uc.Login("")
	if err != domain.ErrEmailRequired {
		t.Errorf("Expected ErrEmailRequired, got %v", err)
	}
}

func TestUpdateAvatar_TooLarge(t *testing.T) {
	repo := newMockUserRepo()
	uc := NewUserUseCase(repo)

	largeAvatar := make([]byte, 600000)
	err := uc.UpdateAvatar("user-1", string(largeAvatar))
	if err != domain.ErrAvatarTooLarge {
		t.Errorf("Expected ErrAvatarTooLarge, got %v", err)
	}
}

func TestUpdate_Success(t *testing.T) {
	repo := newMockUserRepo()
	uc := NewUserUseCase(repo)

	registered, _ := uc.Register("John", "john@example.com")

	updated, err := uc.Update(registered.ID, "Johnny", "johnny@example.com")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if updated.Name != "Johnny" {
		t.Errorf("Expected name 'Johnny', got '%s'", updated.Name)
	}
}
