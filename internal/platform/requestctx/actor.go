package requestctx

import (
	"context"

	"github.com/google/uuid"
)

type actorKey struct{}

type Actor struct {
	UserID       uuid.UUID
	SessionID    uuid.UUID
	Username     string
	DisplayName  string
	ClientType   string
	Roles        []string
	Permissions  []string
	IsSuperAdmin bool
	ViaCookie    bool
}

func WithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorKey{}, actor)
}

func ActorFrom(ctx context.Context) (Actor, bool) {
	actor, ok := ctx.Value(actorKey{}).(Actor)
	return actor, ok
}

func (a Actor) HasPermission(permission string) bool {
	if a.IsSuperAdmin {
		return true
	}
	for _, item := range a.Permissions {
		if item == permission {
			return true
		}
	}
	return false
}

func (a Actor) HasRole(role string) bool {
	for _, item := range a.Roles {
		if item == role {
			return true
		}
	}
	return false
}
