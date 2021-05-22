package keeper

import (
	"bytes"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

func (k Keeper) GetClientStateCommitment(
	ctx sdk.Context,
	counterpartyClientIdentifier string, // clientID corresponding to downstream on upstream
	upstreamClientID string, // client id corresponding to upstream on proxy
) (exported.ClientState, bool) {
	store := k.ProxyCommitmentClientStore(ctx, upstreamClientID, counterpartyClientIdentifier)
	bz := store.Get(host.ClientStateKey())
	if len(bz) == 0 {
		return nil, false
	}
	return clienttypes.MustUnmarshalClientState(k.cdc, bz), true
}

func (k Keeper) GetClientConsensusState(
	ctx sdk.Context,
	counterpartyClientIdentifier string,
	upstreamClientID string,
	consensusHeight exported.Height,
) (exported.ConsensusState, bool) {
	store := k.ProxyCommitmentClientStore(ctx, upstreamClientID, counterpartyClientIdentifier)
	bz := store.Get(host.ConsensusStateKey(consensusHeight))
	if len(bz) == 0 {
		return nil, false
	}
	return clienttypes.MustUnmarshalConsensusState(k.cdc, bz), true
}

func (k Keeper) GetConnectionState(
	ctx sdk.Context,
	counterpartyClientIdentifier string,
	upstreamClientID string,
	connectionID string,
) (*connectiontypes.ConnectionEnd, bool) {
	var connection connectiontypes.ConnectionEnd
	store := k.ProxyCommitmentClientStore(ctx, upstreamClientID, counterpartyClientIdentifier)
	bz := store.Get(host.ConnectionKey(connectionID))
	if len(bz) == 0 {
		return nil, false
	}
	k.cdc.MustUnmarshal(bz, &connection)
	return &connection, true
}

// Commitment provides downstream to verifiable commitment
// CONTRACT: the storeKey for commitments must be equal upstream's prefix

func (k Keeper) setClientStateCommitment(
	ctx sdk.Context,
	counterpartyPrefix exported.Prefix, // upstream's prefix
	counterpartyClientIdentifier string, // clientID corresponding to downstream on upstream
	upstreamClientID string, // client id corresponding to upstream on proxy
	clientState exported.ClientState,
) error {
	if err := k.validateParams(counterpartyPrefix); err != nil {
		return err
	}
	store := k.ProxyCommitmentClientStore(ctx, upstreamClientID, counterpartyClientIdentifier)
	bz := clienttypes.MustMarshalClientState(k.cdc, clientState)
	store.Set(host.ClientStateKey(), bz)
	return nil
}

func (k Keeper) setClientConsensusStateCommitment(
	ctx sdk.Context,
	counterpartyPrefix exported.Prefix,
	counterpartyClientIdentifier string,
	upstreamClientID string,
	consensusHeight exported.Height,
	consensusState exported.ConsensusState,
) error {
	if err := k.validateParams(counterpartyPrefix); err != nil {
		return err
	}
	store := k.ProxyCommitmentClientStore(ctx, upstreamClientID, counterpartyClientIdentifier)
	bz := clienttypes.MustMarshalConsensusState(k.cdc, consensusState)
	store.Set(host.ConsensusStateKey(consensusHeight), bz)
	return nil
}

func (k Keeper) setConnectionStateCommitment(
	ctx sdk.Context,
	counterpartyPrefix exported.Prefix,
	counterpartyClientIdentifier string,
	upstreamClientID string,
	connectionID string,
	connectionEnd connectiontypes.ConnectionEnd,
) error {
	if err := k.validateParams(counterpartyPrefix); err != nil {
		return err
	}
	store := k.ProxyCommitmentStore(ctx, upstreamClientID)
	bz := k.cdc.MustMarshal(&connectionEnd)
	store.Set(host.ConnectionKey(connectionID), bz)
	return nil
}

func (k Keeper) setChannelStateCommitment(
	ctx sdk.Context,
	counterpartyPrefix exported.Prefix,
	counterpartyClientIdentifier string,
	upstreamClientID string,
	portID,
	channelID string,
	channelEnd exported.ChannelI,
) error {
	if err := k.validateParams(counterpartyPrefix); err != nil {
		return err
	}
	store := k.ProxyCommitmentStore(ctx, upstreamClientID)
	channel := channelEnd.(channeltypes.Channel)
	bz := k.cdc.MustMarshal(&channel)
	store.Set(host.ChannelKey(portID, channelID), bz)
	return nil
}

func (k Keeper) setPacketCommitment(
	ctx sdk.Context,
	counterpartyPrefix exported.Prefix,
	counterpartyClientIdentifier string,
	upstreamClientID string,
	portID,
	channelID string,
	sequence uint64,
	commitmentBytes []byte,
) error {
	if err := k.validateParams(counterpartyPrefix); err != nil {
		return err
	}
	store := k.ProxyCommitmentStore(ctx, upstreamClientID)
	store.Set(host.PacketCommitmentKey(portID, channelID, sequence), commitmentBytes)
	return nil
}

func (k Keeper) setPacketAcknowledgement(
	ctx sdk.Context,
	counterpartyPrefix exported.Prefix,
	counterpartyClientIdentifier string,
	upstreamClientID string,
	portID,
	channelID string,
	sequence uint64,
	acknowledgement []byte,
) error {
	if err := k.validateParams(counterpartyPrefix); err != nil {
		return err
	}
	store := k.ProxyCommitmentStore(ctx, upstreamClientID)
	store.Set(host.PacketAcknowledgementKey(portID, channelID, sequence), acknowledgement)
	return nil
}

func (k Keeper) setPacketReceiptAbsence(
	ctx sdk.Context,
	counterpartyPrefix exported.Prefix,
	counterpartyClientIdentifier string,
	upstreamClientID string,
	portID,
	channelID string,
	sequence uint64,
) error {
	if err := k.validateParams(counterpartyPrefix); err != nil {
		return err
	}
	store := k.ProxyCommitmentStore(ctx, upstreamClientID)
	store.Set(host.PacketReceiptKey(portID, channelID, sequence), []byte{byte(1)})
	return nil
}

func (k Keeper) setNextSequenceRecv(
	ctx sdk.Context,
	counterpartyPrefix exported.Prefix,
	counterpartyClientIdentifier string,
	upstreamClientID string,
	portID,
	channelID string,
	nextSequenceRecv uint64,
) error {
	if err := k.validateParams(counterpartyPrefix); err != nil {
		return err
	}
	store := k.ProxyCommitmentStore(ctx, upstreamClientID)
	bz := sdk.Uint64ToBigEndian(nextSequenceRecv)
	store.Set(host.NextSequenceRecvKey(portID, channelID), bz)
	return nil
}

func (k Keeper) validateParams(prefix exported.Prefix) error {
	mp := prefix.(*commitmenttypes.MerklePrefix)
	// NOTE: currently proxy's prefix must equal upstream's prefix
	if !bytes.Equal([]byte(k.ibcStoreKey.Name()), mp.KeyPrefix) {
		return fmt.Errorf("unsupported prefix: %v != %v", "ibc", string(mp.KeyPrefix))
	}
	return nil
}
