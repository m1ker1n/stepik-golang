package services

import (
	"context"
	"hw9/internal/handlers"
)

type User struct {
	creator UserCreator
}

func NewUser(creator UserCreator) *User {
	return &User{
		creator: creator,
	}
}

type CreateUserArgs handlers.RegistrationArgs
type Profile handlers.Profile

type UserCreator interface {
	CreateUser(ctx context.Context, args CreateUserArgs) (Profile, error)
}

func (u *User) RegisterUser(ctx context.Context, args handlers.RegistrationArgs) (handlers.Profile, error) {
	user, err := u.creator.CreateUser(ctx, CreateUserArgs(args))
	if err != nil {
		return handlers.Profile{}, err
	}
	return handlers.Profile(user), nil
}
