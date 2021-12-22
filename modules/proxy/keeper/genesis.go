package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/datachainlab/ibc-proxy/modules/proxy/types"
)

func (k Keeper) InitGenesis(ctx sdk.Context, state types.GenesisState) {}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{}
}
