package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	"github.com/datachainlab/ibc-proxy/modules/proxy/types"
)

var _ types.MsgServer = (*Keeper)(nil)

func (k *Keeper) ProxyConnectionOpenTry(goCtx context.Context, msg *types.MsgProxyConnectionOpenTry) (*types.MsgProxyConnectionOpenTryResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	clientState, err := clienttypes.UnpackClientState(msg.ClientState)
	if err != nil {
		return nil, err
	}
	consensusState, err := clienttypes.UnpackConsensusState(msg.ConsensusState)
	if err != nil {
		return nil, err
	}

	err = k.ConnOpenTry(ctx, msg.ConnectionId, msg.UpstreamClientId, msg.UpstreamPrefix, msg.Connection, clientState, msg.ProofInit, msg.ProofClient, msg.ProofConsensus, msg.ProofHeight, msg.ConsensusHeight, consensusState)
	if err != nil {
		return nil, err
	}
	return &types.MsgProxyConnectionOpenTryResponse{}, nil
}
