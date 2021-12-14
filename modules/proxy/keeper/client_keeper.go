package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	connectiontypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
	multivtypes "github.com/datachainlab/ibc-proxy/modules/light-clients/xx-multiv/types"
)

// ClientKeeper override `ValidateSelfClient` in the keeper of ibc-client
// Original method doesn't yet support a consensus state for a general client
type ClientKeeper struct {
	connectiontypes.ClientKeeper
}

func NewClientKeeper(k connectiontypes.ClientKeeper) ClientKeeper {
	return ClientKeeper{ClientKeeper: k}
}

func (k ClientKeeper) ValidateSelfClient(ctx sdk.Context, clientState exported.ClientState) error {
	cs, ok := clientState.(*multivtypes.ClientState)
	if !ok {
		return k.ClientKeeper.ValidateSelfClient(ctx, clientState)
	}
	return k.ClientKeeper.ValidateSelfClient(ctx, cs.GetUnderlyingClientState())
}
