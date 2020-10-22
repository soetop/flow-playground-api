package migrate_test

import (
	"fmt"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dapperlabs/flow-playground-api/compute"
	"github.com/dapperlabs/flow-playground-api/controller"
	"github.com/dapperlabs/flow-playground-api/migrate"
	"github.com/dapperlabs/flow-playground-api/model"
	"github.com/dapperlabs/flow-playground-api/storage"
	"github.com/dapperlabs/flow-playground-api/storage/memory"
)

const (
	numAccounts = 4
	cacheSize   = 256
)

func TestMigrateNilToV0(t *testing.T) {
	migrateTest(migrate.V0, func(t *testing.T, c migrateTestCase) {
		projID := uuid.New()

		migrated, err := c.migrator.MigrateProject(projID, nil, migrate.V0)
		require.NoError(t, err)
		assert.False(t, migrated)
	})(t)
}

func TestMigrateV0ToV0(t *testing.T) {
	migrateTest(migrate.V0, func(t *testing.T, c migrateTestCase) {
		projID := uuid.New()

		migrated, err := c.migrator.MigrateProject(projID, migrate.V0, migrate.V0)
		require.NoError(t, err)
		assert.False(t, migrated)
	})(t)
}

func TestMigrateV0ToV0_1_0(t *testing.T) {
	migrateTest(migrate.V0, func(t *testing.T, c migrateTestCase) {
		proj, err := c.projects.Create(c.user, model.NewProject{})
		require.NoError(t, err)

		assert.Equal(t, migrate.V0, proj.Version)

		assertAllAccountsExist(t, c.scripts, proj)

		migrated, err := c.migrator.MigrateProject(proj.ID, proj.Version, migrate.V0_1_0)
		require.NoError(t, err)
		assert.True(t, migrated)

		err = c.projects.Get(proj.ID, proj)
		require.NoError(t, err)

		assert.Equal(t, migrate.V0_1_0, proj.Version)

		assertAllAccountsExist(t, c.scripts, proj)
	})(t)
}

func TestMigrateV0_1_0ToV0_2_0(t *testing.T) {
	migrateTest(migrate.V0_1_0, func(t *testing.T, c migrateTestCase) {
		v0_2_0 := semver.MustParse("v0.2.0")

		proj, err := c.projects.Create(c.user, model.NewProject{})
		require.NoError(t, err)

		assert.Equal(t, migrate.V0_1_0, proj.Version)

		migrated, err := c.migrator.MigrateProject(proj.ID, proj.Version, v0_2_0)
		require.NoError(t, err)
		assert.True(t, migrated)

		err = c.projects.Get(proj.ID, proj)
		require.NoError(t, err)

		assert.Equal(t, v0_2_0, proj.Version)
	})(t)
}

type migrateTestCase struct {
	store    storage.Store
	computer *compute.Computer
	scripts  *controller.Scripts
	projects *controller.Projects
	migrator *migrate.Migrator
	user     *model.User
}

func migrateTest(startVersion *semver.Version, f func(t *testing.T, c migrateTestCase)) func(t *testing.T) {
	return func(t *testing.T) {
		store := memory.NewStore()
		computer, err := compute.NewComputer(cacheSize)
		require.NoError(t, err)

		scripts := controller.NewScripts(store, computer)
		projects := controller.NewProjects(startVersion, store, computer, numAccounts)

		migrator := migrate.NewMigrator(projects)

		user := model.User{
			ID: uuid.New(),
		}

		err = store.InsertUser(&user)
		require.NoError(t, err)

		f(t, migrateTestCase{
			store:    store,
			computer: computer,
			scripts:  scripts,
			projects: projects,
			migrator: migrator,
			user:     &user,
		})
	}
}

func assertAllAccountsExist(t *testing.T, scripts *controller.Scripts, proj *model.InternalProject) {
	for i := 1; i <= numAccounts; i++ {
		script := fmt.Sprintf(`pub fun main() { getAccount(0x%x) }`, i)

		result, err := scripts.CreateExecution(proj, script, nil)
		require.NoError(t, err)

		assert.Empty(t, result.Errors)
	}
}
