package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	clientkeeper "github.com/cosmos/ibc-go/modules/core/02-client/keeper"
	"github.com/cosmos/ibc-go/modules/core/exported"
	multivtypes "github.com/datachainlab/ibc-proxy/modules/light-clients/xx-multiv/types"
)

// ClientKeeper override `GetSelfConsensusState` and `ValidateSelfClient` in the keeper of ibc-client
// Original method doesn't yet support a consensus state for a general client
type ClientKeeper struct {
	clientkeeper.Keeper
}

func NewClientKeeper(k clientkeeper.Keeper) ClientKeeper {
	return ClientKeeper{Keeper: k}
}

func (k ClientKeeper) ValidateSelfClient(ctx sdk.Context, clientState exported.ClientState) error {
	cs, ok := clientState.(*multivtypes.ClientState)
	if !ok {
		return k.Keeper.ValidateSelfClient(ctx, clientState)
	}
	return k.Keeper.ValidateSelfClient(ctx, cs.GetBaseClientState())
}
