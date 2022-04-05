package keeper

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	v043 "github.com/cosmos/cosmos-sdk/x/gov/migrations/v043"
	v046 "github.com/cosmos/cosmos-sdk/x/gov/migrations/v046"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate1to2 migrates from version 1 to 2.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	return errors.New("Migration from x/gov v1 (Cosmos SDK v0.42) to v2 is not supported any more")
}

// Migrate2to3 migrates from version 2 to 3.
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	return v046.MigrateStore(ctx, m.keeper.storeKey, m.keeper.cdc)
}
