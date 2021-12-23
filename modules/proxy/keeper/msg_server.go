package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	"github.com/datachainlab/ibc-proxy/modules/proxy/types"
)

var _ types.MsgServer = (*Keeper)(nil)

// ProxyClientState implements types.MsgServer
func (k *Keeper) ProxyClientState(goCtx context.Context, msg *types.MsgProxyClientState) (*types.MsgProxyClientStateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	clientState, err := clienttypes.UnpackClientState(msg.ClientState)
	if err != nil {
		return nil, err
	}
	consensusState, err := clienttypes.UnpackConsensusState(msg.ConsensusState)
	if err != nil {
		return nil, err
	}

	err = k.ClientState(ctx, msg.UpstreamClientId, &msg.UpstreamPrefix, msg.CounterpartyClientId, clientState, consensusState, msg.ProofClient, msg.ProofConsensus, msg.ProofHeight, msg.ConsensusHeight)
	if err != nil {
		return nil, err
	}
	return &types.MsgProxyClientStateResponse{}, nil
}

// ProxyConnectionOpenTry implements types.MsgServer
func (k *Keeper) ProxyConnectionOpenTry(goCtx context.Context, msg *types.MsgProxyConnectionOpenTry) (*types.MsgProxyConnectionOpenTryResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	downstreamClientState, err := clienttypes.UnpackClientState(msg.DownstreamClientState)
	if err != nil {
		return nil, err
	}
	downstreamConsensusState, err := clienttypes.UnpackConsensusState(msg.DownstreamConsensusState)
	if err != nil {
		return nil, err
	}
	proxyClientState, err := clienttypes.UnpackClientState(msg.ProxyClientState)
	if err != nil {
		return nil, err
	}

	err = k.ConnOpenTry(ctx, msg.ConnectionId, &msg.UpstreamPrefix, msg.Connection, downstreamClientState, downstreamConsensusState, proxyClientState, msg.ProofInit, msg.ProofClient, msg.ProofConsensus, msg.ProofHeight, msg.ConsensusHeight, msg.ProofProxyClient, msg.ProofProxyConsensus, msg.ProofProxyHeight, msg.ProxyConsensusHeight)
	if err != nil {
		return nil, err
	}
	return &types.MsgProxyConnectionOpenTryResponse{}, nil
}

// ProxyConnectionOpenAck implements types.MsgServer
func (k *Keeper) ProxyConnectionOpenAck(goCtx context.Context, msg *types.MsgProxyConnectionOpenAck) (*types.MsgProxyConnectionOpenAckResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	downstreamClientState, err := clienttypes.UnpackClientState(msg.DownstreamClientState)
	if err != nil {
		return nil, err
	}
	downstreamConsensusState, err := clienttypes.UnpackConsensusState(msg.DownstreamConsensusState)
	if err != nil {
		return nil, err
	}
	proxyClientState, err := clienttypes.UnpackClientState(msg.ProxyClientState)
	if err != nil {
		return nil, err
	}

	err = k.ConnOpenAck(ctx, msg.ConnectionId, &msg.UpstreamPrefix, msg.Connection, downstreamClientState, downstreamConsensusState, proxyClientState, msg.ProofTry, msg.ProofClient, msg.ProofConsensus, msg.ProofHeight, msg.ConsensusHeight, msg.ProofProxyClient, msg.ProofProxyConsensus, msg.ProofProxyHeight, msg.ProxyConsensusHeight)
	if err != nil {
		return nil, err
	}
	return &types.MsgProxyConnectionOpenAckResponse{}, nil
}

// ProxyConnectionOpenConfirm implements types.MsgServer
func (k *Keeper) ProxyConnectionOpenConfirm(goCtx context.Context, msg *types.MsgProxyConnectionOpenConfirm) (*types.MsgProxyConnectionOpenConfirmResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.ConnOpenConfirm(ctx, msg.ConnectionId, msg.UpstreamClientId, &msg.UpstreamPrefix, msg.CounterpartyConnectionId, msg.ProofAck, msg.ProofHeight)
	if err != nil {
		return nil, err
	}
	return &types.MsgProxyConnectionOpenConfirmResponse{}, nil
}

// ProxyConnectionOpenTry implements types.MsgServer
func (k *Keeper) ProxyChannelOpenTry(goCtx context.Context, msg *types.MsgProxyChannelOpenTry) (*types.MsgProxyChannelOpenTryResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.ChanOpenTry(ctx, msg.UpstreamClientId, &msg.UpstreamPrefix, msg.Order, msg.ConnectionHops, msg.PortId, msg.ChannelId, msg.DownstreamPortId, msg.Version, msg.ProofInit, msg.ProofHeight)
	if err != nil {
		return nil, err
	}
	return &types.MsgProxyChannelOpenTryResponse{}, nil
}

// ProxyChannelOpenAck implements types.MsgServer
func (k *Keeper) ProxyChannelOpenAck(goCtx context.Context, msg *types.MsgProxyChannelOpenAck) (*types.MsgProxyChannelOpenAckResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.ChanOpenAck(ctx, msg.UpstreamClientId, &msg.UpstreamPrefix, msg.Order, msg.ConnectionHops, msg.PortId, msg.ChannelId, msg.DownstreamPortId, msg.DownstreamChannelId, msg.Version, msg.ProofTry, msg.ProofHeight)
	if err != nil {
		return nil, err
	}
	return &types.MsgProxyChannelOpenAckResponse{}, nil
}

// ProxyChannelOpenConfirm implements types.MsgServer
func (k *Keeper) ProxyChannelOpenConfirm(goCtx context.Context, msg *types.MsgProxyChannelOpenConfirm) (*types.MsgProxyChannelOpenConfirmResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.ChanOpenConfirm(ctx, msg.UpstreamClientId, &msg.UpstreamPrefix, msg.PortId, msg.ChannelId, msg.DownstreamChannelId, msg.ProofAck, msg.ProofHeight)
	if err != nil {
		return nil, err
	}
	return &types.MsgProxyChannelOpenConfirmResponse{}, nil
}

// ProxyRecvPacket implements types.MsgServer
func (k *Keeper) ProxyRecvPacket(goCtx context.Context, msg *types.MsgProxyRecvPacket) (*types.MsgProxyRecvPacketResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.RecvPacket(ctx, msg.UpstreamClientId, &msg.UpstreamPrefix, msg.Packet, msg.Proof, msg.ProofHeight)
	if err != nil {
		return nil, err
	}
	return &types.MsgProxyRecvPacketResponse{}, nil
}

// ProxyAcknowledgePacket implements types.MsgServer
func (k *Keeper) ProxyAcknowledgePacket(goCtx context.Context, msg *types.MsgProxyAcknowledgePacket) (*types.MsgProxyAcknowledgePacketResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.AcknowledgePacket(ctx, msg.UpstreamClientId, &msg.UpstreamPrefix, msg.Packet, msg.Acknowledgement, msg.Proof, msg.ProofHeight)
	if err != nil {
		return nil, err
	}
	return &types.MsgProxyAcknowledgePacketResponse{}, nil
}
