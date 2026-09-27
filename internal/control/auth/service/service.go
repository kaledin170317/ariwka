package service

import (
	"context"
	"errors"
	"strconv"

	"github.com/google/uuid"

	"golang.org/x/crypto/bcrypt"

	"vpn/internal/control/user/model"
	"vpn/pkg/auth"
)

type Repository interface {
	CreatePassword(context.Context, model.User, string) error
	Password(context.Context, string) (string, string, error)
	TelegramUser(context.Context, string, string) (string, error)
}

type Users interface {
	Create(context.Context, string) (model.User, error)
}

type Service struct {
	Repo  Repository
	JWT   auth.JWT
	Users Users
}

func (s Service) Register(ctx context.Context, username, password string) (model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, err
	}

	u, err := s.Users.Create(ctx, username)
	if err != nil {
		return model.User{}, err
	}

	err = s.Repo.CreatePassword(ctx, u, string(hash))
	if err != nil {
		return model.User{}, err
	}

	return u, nil
}

func (s Service) Login(ctx context.Context, username, password string) (string, error) {
	id, hash, err := s.Repo.Password(ctx, username)
	if err != nil {
		return "", err
	}
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return "", errors.New("invalid credentials")
	}
	return s.JWT.Sign(id)
}

func (s Service) LoginTelegramUser(ctx context.Context, id int64) (string, error) {
	userID, err := s.Repo.TelegramUser(ctx, strconv.FormatInt(id, 10), uuid.NewString())
	if err != nil {
		return "", err
	}
	return s.JWT.Sign(userID)
}
