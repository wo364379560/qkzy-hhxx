package controller_admin

import (
	"context"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
)

func ChangeRegistrationIP(ctx context.Context, actor uint, password, ip, note string, remove bool, audit model.AdminAccessAudit) error {
	if err := verifySuperAdminPassword(actor, password); err != nil {
		return err
	}
	return repository.ChangeRegistrationIP(ctx, datasource.Datasource.Mysql.GetClient(), actor, ip, note, remove, audit)
}
