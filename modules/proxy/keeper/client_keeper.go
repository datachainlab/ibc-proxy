package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	clientkeeper "github.com/cosmos/ibc-go/modules/core/02-client/keeper"
	"github.com/cosmos/ibc-go/modules/core/exported"
	proxytypes "github.com/datachainlab/ibc-proxy/modules/light-clients/xx-proxy/types"
)

// ClientKeeper override `GetSelfConsensusState` and `ValidateSelfClient` in the keeper of ibc-client
// Original method doesn't yet support a consensus state for a general client
type ClientKeeper struct {
	clientkeeper.Keeper
}

func NewClientKeeper(k clientkeeper.Keeper) ClientKeeper {
	return ClientKeeper{Keeper: k}
}

// // GetSelfConsensusState introspects the (self) past historical info at a given height
// // and returns the expected consensus state at that height.
// // For now, can only retrieve self consensus states for the current version
// func (k ClientKeeper) GetSelfConsensusState(ctx sdk.Context, height exported.Height) (exported.ConsensusState, bool) {
// 	cs, found := k.Keeper.GetSelfConsensusState(ctx, height)
// 	if !found {
// 		return cs, found
// 	}
// 	return &proxytypes.ConsensusState{
// 		UpstreamConsensusState: clienttypes.MustPackConsensusState(cs),
// 	}, true
// }

func (k ClientKeeper) ValidateSelfClient(ctx sdk.Context, clientState exported.ClientState) error {
	cs, ok := clientState.(*proxytypes.ClientState)
	if !ok {
		return k.Keeper.ValidateSelfClient(ctx, clientState)
	}
	return k.Keeper.ValidateSelfClient(ctx, cs.GetUpstreamClientState())
}
