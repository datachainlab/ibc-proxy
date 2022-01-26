package keeper

import (
	"encoding/binary"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

func (k Keeper) GetProxyClientState(
	ctx sdk.Context,
	upstreamPrefix exported.Prefix,
	counterpartyClientIdentifier string, // clientID corresponding to downstream on upstream
	upstreamClientID string, // client id corresponding to upstream on proxy
) (exported.ClientState, bool) {
	store := k.ProxyClientStore(ctx, upstreamPrefix, upstreamClientID, counterpartyClientIdentifier)
	bz := store.Get(host.ClientStateKey())
	if len(bz) == 0 {
		return nil, false
	}
	return clienttypes.MustUnmarshalClientState(k.cdc, bz), true
}

func (k Keeper) GetProxyClientConsensusState(
	ctx sdk.Context,
	upstreamPrefix exported.Prefix,
	counterpartyClientIdentifier string,
	upstreamClientID string,
	consensusHeight exported.Height,
) (exported.ConsensusState, bool) {
	store := k.ProxyClientStore(ctx, upstreamPrefix, upstreamClientID, counterpartyClientIdentifier)
	bz := store.Get(host.ConsensusStateKey(consensusHeight))
	if len(bz) == 0 {
		return nil, false
	}
	return clienttypes.MustUnmarshalConsensusState(k.cdc, bz), true
}

func (k Keeper) GetProxyConnection(
	ctx sdk.Context,
	upstreamPrefix exported.Prefix,
	upstreamClientID string,
	connectionID string,
) (connectiontypes.ConnectionEnd, bool) {
	var connection connectiontypes.ConnectionEnd
	store := k.ProxyStore(ctx, upstreamPrefix, upstreamClientID)
	bz := store.Get(host.ConnectionKey(connectionID))
	if len(bz) == 0 {
		return connection, false
	}
	k.cdc.MustUnmarshal(bz, &connection)
	return connection, true
}

func (k Keeper) GetProxyChannel(
	ctx sdk.Context,
	upstreamPrefix exported.Prefix,
	upstreamClientID string,
	portID,
	channelID string,
) (channeltypes.Channel, bool) {
	var channel channeltypes.Channel
	store := k.ProxyStore(ctx, upstreamPrefix, upstreamClientID)
	bz := store.Get(host.ChannelKey(portID, channelID))
	if len(bz) == 0 {
		return channel, false
	}
	k.cdc.MustUnmarshal(bz, &channel)
	return channel, true
}

func (k Keeper) SetProxyUpstreamBlockTime(
	ctx sdk.Context,
	upstreamPrefix exported.Prefix, // upstream's prefix
	upstreamClientID string, // client id corresponding to upstream on proxy
	height exported.Height,
) error {
	consensusState, found := k.clientKeeper.GetClientConsensusState(ctx, upstreamClientID, height)
	if !found {
		return sdkerrors.Wrapf(
			clienttypes.ErrConsensusStateNotFound,
			"clientID (%s), height (%s)", upstreamClientID, height,
		)
	}
	var bz [8]byte
	binary.BigEndian.PutUint64(bz[:], consensusState.GetTimestamp())
	store := k.ProxyStore(ctx, upstreamPrefix, upstreamClientID)
	store.Set([]byte(fmt.Sprintf("block/%s", height.String())), bz[:])
	return nil
}

func (k Keeper) SetProxyClientState(
	ctx sdk.Context,
	upstreamPrefix exported.Prefix, // upstream's prefix
	counterpartyClientIdentifier string, // clientID corresponding to downstream on upstream
	upstreamClientID string, // client id corresponding to upstream on proxy
	clientState exported.ClientState,
) error {
	store := k.ProxyClientStore(ctx, upstreamPrefix, upstreamClientID, counterpartyClientIdentifier)
	bz := clienttypes.MustMarshalClientState(k.cdc, clientState)
	store.Set(host.ClientStateKey(), bz)
	return nil
}

func (k Keeper) SetProxyClientConsensusState(
	ctx sdk.Context,
	upstreamPrefix exported.Prefix, // upstream's prefix
	counterpartyClientIdentifier string,
	upstreamClientID string,
	consensusHeight exported.Height,
	consensusState exported.ConsensusState,
) error {
	store := k.ProxyClientStore(ctx, upstreamPrefix, upstreamClientID, counterpartyClientIdentifier)
	bz := clienttypes.MustMarshalConsensusState(k.cdc, consensusState)
	store.Set(host.ConsensusStateKey(consensusHeight), bz)
	return nil
}

func (k Keeper) SetProxyConnection(
	ctx sdk.Context,
	upstreamPrefix exported.Prefix, // upstream's prefix
	upstreamClientID string,
	connectionID string,
	connectionEnd connectiontypes.ConnectionEnd,
) error {
	store := k.ProxyStore(ctx, upstreamPrefix, upstreamClientID)
	bz := k.cdc.MustMarshal(&connectionEnd)
	store.Set(host.ConnectionKey(connectionID), bz)
	return nil
}

func (k Keeper) SetProxyChannel(
	ctx sdk.Context,
	upstreamPrefix exported.Prefix, // upstream's prefix
	upstreamClientID string,
	portID,
	channelID string,
	channelEnd exported.ChannelI,
) error {
	store := k.ProxyStore(ctx, upstreamPrefix, upstreamClientID)
	channel := channelEnd.(channeltypes.Channel)
	bz := k.cdc.MustMarshal(&channel)
	store.Set(host.ChannelKey(portID, channelID), bz)
	return nil
}

func (k Keeper) SetProxyPacketCommitment(
	ctx sdk.Context,
	upstreamPrefix exported.Prefix, // upstream's prefix
	upstreamClientID string,
	portID,
	channelID string,
	sequence uint64,
	commitmentBytes []byte,
) error {
	store := k.ProxyStore(ctx, upstreamPrefix, upstreamClientID)
	store.Set(host.PacketCommitmentKey(portID, channelID, sequence), commitmentBytes)
	return nil
}

func (k Keeper) SetProxyPacketAcknowledgement(
	ctx sdk.Context,
	upstreamPrefix exported.Prefix, // upstream's prefix
	upstreamClientID string,
	portID,
	channelID string,
	sequence uint64,
	acknowledgement []byte,
) error {
	store := k.ProxyStore(ctx, upstreamPrefix, upstreamClientID)
	store.Set(host.PacketAcknowledgementKey(portID, channelID, sequence), channeltypes.CommitAcknowledgement(acknowledgement))
	return nil
}

func (k Keeper) SetProxyPacketReceiptAbsence(
	ctx sdk.Context,
	upstreamPrefix exported.Prefix, // upstream's prefix
	upstreamClientID string,
	portID,
	channelID string,
	sequence uint64,
) error {
	store := k.ProxyStore(ctx, upstreamPrefix, upstreamClientID)
	store.Set(host.PacketReceiptKey(portID, channelID, sequence), []byte{byte(1)})
	return nil
}

func (k Keeper) SetProxyNextSequenceRecv(
	ctx sdk.Context,
	upstreamPrefix exported.Prefix, // upstream's prefix
	upstreamClientID string,
	portID,
	channelID string,
	nextSequenceRecv uint64,
) error {
	store := k.ProxyStore(ctx, upstreamPrefix, upstreamClientID)
	bz := sdk.Uint64ToBigEndian(nextSequenceRecv)
	store.Set(host.NextSequenceRecvKey(portID, channelID), bz)
	return nil
}
