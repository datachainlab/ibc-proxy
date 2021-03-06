package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

type ClientKeeper interface {
	ClientStore(ctx sdk.Context, clientID string) sdk.KVStore
	GetClientState(ctx sdk.Context, clientID string) (exported.ClientState, bool)
	GetSelfConsensusState(ctx sdk.Context, height exported.Height) (exported.ConsensusState, bool)
	ValidateSelfClient(ctx sdk.Context, clientState exported.ClientState) error
}
