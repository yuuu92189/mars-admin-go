package requestctx

import "testing"

func TestSuperAdminHasEveryPermission(t *testing.T) {
	actor := Actor{IsSuperAdmin: true}
	if !actor.HasPermission("anything:at:all") {
		t.Fatal("super administrator should bypass permission list")
	}
}

func TestNormalActorRequiresExactPermission(t *testing.T) {
	actor := Actor{Permissions: []string{"user:read"}}
	if actor.HasPermission("user:delete") {
		t.Fatal("unexpected permission grant")
	}
	if !actor.HasPermission("user:read") {
		t.Fatal("expected exact permission")
	}
}
