package service

import (
	"context"

	"github.com/google/uuid"

	"vpn/internal/control/user/model"
)

type Repository interface {
	Create(context.Context, model.User) (model.User, error)
	List(context.Context) ([]model.User, error)
	Get(context.Context, string) (model.User, error)
}

type Service struct{ Repo Repository }

func (s Service) Create(ctx context.Context, username string) (model.User, error) {
	u := model.User{ID: uuid.NewString(), Username: username, Role: "user"}
	return s.Repo.Create(ctx, u)
}

func (s Service) List(ctx context.Context) ([]model.User, error) {
	return s.Repo.List(ctx)
}

func (s Service) Get(ctx context.Context, id string) (model.User, error) {
	return s.Repo.Get(ctx, id)
}
