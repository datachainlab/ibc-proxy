package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	"github.com/datachainlab/ibc-proxy/modules/proxy/types"
)

var _ types.MsgServer = (*Keeper)(nil)

// ProxyConnectionOpenTry implements types.MsgServer
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

// ProxyConnectionOpenAck implements types.MsgServer
func (k *Keeper) ProxyConnectionOpenAck(goCtx context.Context, msg *types.MsgProxyConnectionOpenAck) (*types.MsgProxyConnectionOpenAckResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	clientState, err := clienttypes.UnpackClientState(msg.ClientState)
	if err != nil {
		return nil, err
	}
	consensusState, err := clienttypes.UnpackConsensusState(msg.ConsensusState)
	if err != nil {
		return nil, err
	}

	err = k.ConnOpenAck(ctx, msg.ConnectionId, msg.UpstreamClientId, msg.UpstreamPrefix, msg.Connection, clientState, msg.Version, msg.ProofTry, msg.ProofClient, msg.ProofConsensus, msg.ProofHeight, msg.ConsensusHeight, consensusState)
	if err != nil {
		return nil, err
	}
	return &types.MsgProxyConnectionOpenAckResponse{}, nil
}

// ProxyConnectionOpenConfirm implements types.MsgServer
func (k *Keeper) ProxyConnectionOpenConfirm(goCtx context.Context, msg *types.MsgProxyConnectionOpenConfirm) (*types.MsgProxyConnectionOpenConfirmResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.ConnOpenConfirm(ctx, msg.ConnectionId, msg.UpstreamClientId, msg.UpstreamPrefix, msg.Connection, msg.ProofAck, msg.ProofHeight)
	if err != nil {
		return nil, err
	}
	return &types.MsgProxyConnectionOpenConfirmResponse{}, nil
}
