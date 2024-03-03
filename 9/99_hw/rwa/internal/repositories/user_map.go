package repositories

import (
	"context"
	"hw9/internal/services"
	"hw9/pkg"
	"sync"
	"time"
)

type profileID = string

type UserMap struct {
	mu   *sync.Mutex
	data map[profileID]services.Profile

	idLen int
}

func (u *UserMap) createUser(args services.CreateUserArgs) services.Profile {
	return services.Profile{
		ID:        pkg.RandStringRunes(u.idLen),
		Email:     args.Email,
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
		Username:  args.Username,
	}
}

// doWithContext is a function which wraps call of basic functions to be used with context.
func doWithContext[In, Out any](ctx context.Context, do func(in In) Out, in In) func() (Out, error) {
	return func() (Out, error) {
		var zero Out

		resultCh := make(chan Out)
		go func() {
			resultCh <- do(in)
		}()

		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case result := <-resultCh:
			return result, nil
		}
	}
}

func (u *UserMap) CreateUser(ctx context.Context, args services.CreateUserArgs) (services.Profile, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	return doWithContext(ctx, u.createUser, args)()
}

func NewUserMap(idLen int) *UserMap {
	return &UserMap{
		mu:    &sync.Mutex{},
		data:  make(map[profileID]services.Profile),
		idLen: idLen,
	}
}
