package usecase

import (
	"time"

	"adedanha-golang/internal/domain"
	"adedanha-golang/internal/repository"

	"github.com/google/uuid"
)

type UserUseCase struct {
	userRepo repository.UserRepository
}

func NewUserUseCase(userRepo repository.UserRepository) *UserUseCase {
	return &UserUseCase{userRepo: userRepo}
}

func (uc *UserUseCase) Register(name, email string) (domain.User, error) {
	if err := domain.ValidateCreateUser(name, email); err != nil {
		return domain.User{}, err
	}

	user := domain.User{
		ID:        uuid.New().String(),
		Name:      name,
		Email:     email,
		CreatedAt: time.Now(),
	}

	if err := uc.userRepo.Create(user); err != nil {
		return domain.User{}, err
	}

	return user, nil
}

func (uc *UserUseCase) Login(email string) (domain.User, error) {
	if email == "" {
		return domain.User{}, domain.ErrEmailRequired
	}
	return uc.userRepo.GetByEmail(email)
}

func (uc *UserUseCase) GetByID(id string) (domain.User, error) {
	return uc.userRepo.GetByID(id)
}

func (uc *UserUseCase) Update(id, name, email string) (domain.User, error) {
	if err := domain.ValidateCreateUser(name, email); err != nil {
		return domain.User{}, err
	}
	return uc.userRepo.Update(id, name, email)
}

func (uc *UserUseCase) UpdateAvatar(id, avatar string) error {
	if avatar != "" && len(avatar) > 500000 {
		return domain.ErrAvatarTooLarge
	}
	return uc.userRepo.UpdateAvatar(id, avatar)
}

func (uc *UserUseCase) GetOnlineUsers(onlineIDs []string) ([]domain.OnlineUser, error) {
	return uc.userRepo.GetOnlineUsersByIDs(onlineIDs)
}

func (uc *UserUseCase) GetAvailablePlayers(onlineIDs []string) ([]domain.OnlineUser, error) {
	return uc.userRepo.GetAvailablePlayersByIDs(onlineIDs)
}
