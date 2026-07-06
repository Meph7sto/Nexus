package repository

import (
	"context"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/adminpermission"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type adminPermissionRepository struct {
	client *dbent.Client
}

func NewAdminPermissionRepository(client *dbent.Client) service.AdminPermissionRepository {
	return &adminPermissionRepository{client: client}
}

func (r *adminPermissionRepository) ListByUserID(ctx context.Context, userID int64) ([]service.AdminPermission, error) {
	rows, err := clientFromContext(ctx, r.client).AdminPermission.Query().
		Where(adminpermission.UserIDEQ(userID)).
		Order(dbent.Asc(adminpermission.FieldResource)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]service.AdminPermission, 0, len(rows))
	for _, row := range rows {
		actions := make([]service.AdminPermissionAction, 0, len(row.Actions))
		for _, action := range row.Actions {
			actions = append(actions, service.AdminPermissionAction(action))
		}
		out = append(out, service.AdminPermission{
			Resource: service.AdminPermissionResource(row.Resource),
			Actions:  actions,
		})
	}
	return out, nil
}

func (r *adminPermissionRepository) ReplaceForUser(ctx context.Context, userID int64, permissions []service.AdminPermission) error {
	if err := service.ValidateAdminPermissions(permissions); err != nil {
		return err
	}
	client := clientFromContext(ctx, r.client)
	if _, err := client.AdminPermission.Delete().Where(adminpermission.UserIDEQ(userID)).Exec(ctx); err != nil {
		return err
	}
	for _, perm := range permissions {
		if len(perm.Actions) == 0 {
			continue
		}
		actions := make([]string, 0, len(perm.Actions))
		for _, action := range perm.Actions {
			actions = append(actions, string(action))
		}
		if _, err := client.AdminPermission.Create().
			SetUserID(userID).
			SetResource(string(perm.Resource)).
			SetActions(actions).
			Save(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (r *adminPermissionRepository) DeleteForUser(ctx context.Context, userID int64) error {
	_, err := clientFromContext(ctx, r.client).AdminPermission.Delete().
		Where(adminpermission.UserIDEQ(userID)).
		Exec(ctx)
	return err
}

func (r *adminPermissionRepository) HasPermission(ctx context.Context, userID int64, resource service.AdminPermissionResource, action service.AdminPermissionAction) (bool, error) {
	perms, err := r.ListByUserID(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, perm := range perms {
		if perm.Resource != resource {
			continue
		}
		for _, existing := range perm.Actions {
			if existing == action {
				return true, nil
			}
		}
	}
	return false, nil
}
