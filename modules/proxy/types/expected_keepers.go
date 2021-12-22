package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

type ClientKeeper interface {
	ClientStore(ctx sdk.Context, clientID string) sdk.KVStore
	SetClientState(ctx sdk.Context, clientID string, clientState exported.ClientState)
	GetClientState(ctx sdk.Context, clientID string) (exported.ClientState, bool)
	SetClientConsensusState(ctx sdk.Context, clientID string, height exported.Height, consensusState exported.ConsensusState)
	GetClientConsensusState(ctx sdk.Context, clientID string, height exported.Height) (exported.ConsensusState, bool)
	CreateClient(ctx sdk.Context, clientState exported.ClientState, consensusState exported.ConsensusState) (string, error)
}
