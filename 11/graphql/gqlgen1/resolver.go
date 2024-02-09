package gqlgen1

// THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

import (
	"context"
)

type Resolver struct{}

// RatePhoto is the resolver for the ratePhoto field.
func (r *mutationResolver) RatePhoto(ctx context.Context, photoID string, direction string) (*Photo, error) {
	panic("not implemented")
}

// Timeline is the resolver for the timeline field.
func (r *queryResolver) Timeline(ctx context.Context) ([]*Photo, error) {
	panic("not implemented")
}

// User is the resolver for the user field.
func (r *queryResolver) User(ctx context.Context, userID string) (*User, error) {
	panic("not implemented")
}

// Photos is the resolver for the photos field.
func (r *queryResolver) Photos(ctx context.Context, userID string) ([]*Photo, error) {
	panic("not implemented")
}

// Mutation returns MutationResolver implementation.
func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
