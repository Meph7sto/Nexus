package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/service"
	_ "modernc.org/sqlite"
)

func newAdminPermissionTestClient(t *testing.T) *dbent.Client {
	t.Helper()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", t.Name()))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestAdminPermissionRepositoryReplaceListAndHasPermission(t *testing.T) {
	ctx := context.Background()
	client := newAdminPermissionTestClient(t)

	user := client.User.Create().
		SetEmail("limited-admin@example.com").
		SetUsername("limited-admin").
		SetPasswordHash("hash").
		SetRole(service.RoleAdmin).
		SetBalance(0).
		SetConcurrency(1).
		SetStatus(service.StatusActive).
		SaveX(ctx)

	repo := NewAdminPermissionRepository(client)
	perms := []service.AdminPermission{
		{Resource: service.AdminResourceUsers, Actions: []service.AdminPermissionAction{service.AdminActionView, service.AdminActionUpdate}},
	}

	if err := repo.ReplaceForUser(ctx, user.ID, perms); err != nil {
		t.Fatalf("ReplaceForUser failed: %v", err)
	}

	got, err := repo.ListByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListByUserID failed: %v", err)
	}
	if len(got) != 1 || got[0].Resource != service.AdminResourceUsers {
		t.Fatalf("unexpected permissions: %#v", got)
	}

	ok, err := repo.HasPermission(ctx, user.ID, service.AdminResourceUsers, service.AdminActionUpdate)
	if err != nil || !ok {
		t.Fatalf("HasPermission update = %v, %v; want true, nil", ok, err)
	}
	ok, err = repo.HasPermission(ctx, user.ID, service.AdminResourceUsers, service.AdminActionDelete)
	if err != nil || ok {
		t.Fatalf("HasPermission delete = %v, %v; want false, nil", ok, err)
	}
}
