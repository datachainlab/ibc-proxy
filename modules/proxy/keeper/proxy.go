package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

// Verifier is always proxy chain
// parameters:
//	 upstreamClientID: the clientID of upstream
//   proof: the commitment proof of the client state corresponding to upstreamClientID

func (k Keeper) VerifyClientState(
	ctx sdk.Context,
	upstreamClientID string,
	upstreamPrefix exported.Prefix,
	counterpartyConnection exported.ConnectionI, // upstream
	height exported.Height,
	proof []byte,
	clientState exported.ClientState, // the state of downstream that upstream has
) error {
	targetClient, found := k.clientKeeper.GetClientState(ctx, upstreamClientID)
	if !found {
		return sdkerrors.Wrap(clienttypes.ErrClientNotFound, upstreamClientID)
	}

	if err := targetClient.VerifyClientState(
		k.clientKeeper.ClientStore(ctx, upstreamClientID), k.cdc, height,
		upstreamPrefix, counterpartyConnection.GetClientID(), proof, clientState); err != nil {
		return sdkerrors.Wrapf(err, "failed client state verification for target client: %s", upstreamClientID)
	}

	return k.setClientStateCommitment(
		ctx,
		upstreamPrefix,
		counterpartyConnection.GetClientID(),
		upstreamClientID,
		clientState,
	)
}

func (k Keeper) VerifyClientConsensusState(
	ctx sdk.Context,
	upstreamClientID string,
	upstreamPrefix exported.Prefix,
	connection exported.ConnectionI,
	height exported.Height,
	consensusHeight exported.Height,
	proof []byte,
	consensusState exported.ConsensusState, // the state of downstream that upstream has
) error {
	targetClient, found := k.clientKeeper.GetClientState(ctx, upstreamClientID)
	if !found {
		return sdkerrors.Wrap(clienttypes.ErrClientNotFound, upstreamClientID)
	}

	if err := targetClient.VerifyClientConsensusState(
		k.clientKeeper.ClientStore(ctx, upstreamClientID), k.cdc, height,
		connection.GetClientID(), consensusHeight, upstreamPrefix, proof, consensusState,
	); err != nil {
		return sdkerrors.Wrapf(err, "failed consensus state verification for client (%s)", upstreamClientID)
	}

	return k.setClientConsensusStateCommitment(
		ctx,
		upstreamPrefix,
		connection.GetClientID(),
		upstreamClientID,
		consensusHeight,
		consensusState,
	)
}

func (k Keeper) VerifyConnectionState(
	ctx sdk.Context,
	upstreamClientID string,
	upstreamPrefix exported.Prefix,
	proxyConnection connectiontypes.ConnectionEnd,
	height exported.Height,
	proof []byte,
	connectionID string, // ID of the connection that upstream has
) error {
	targetClient, found := k.clientKeeper.GetClientState(ctx, upstreamClientID)
	if !found {
		return sdkerrors.Wrap(clienttypes.ErrClientNotFound, upstreamClientID)
	}
	if err := targetClient.VerifyConnectionState(
		k.clientKeeper.ClientStore(ctx, upstreamClientID), k.cdc, height,
		upstreamPrefix, proof, connectionID, proxyConnection,
	); err != nil {
		return sdkerrors.Wrapf(err, "failed connection state verification for client (%s)", upstreamClientID)
	}

	return k.setConnectionStateCommitment(
		ctx,
		upstreamPrefix,
		upstreamClientID,
		connectionID,
		proxyConnection,
	)
}

func (k Keeper) VerifyChannelState(
	ctx sdk.Context,
	upstreamClientID string,
	upstreamPrefix exported.Prefix,
	proxyConnection exported.ConnectionI,
	height exported.Height,
	proof []byte,
	portID,
	channelID string,
	channel exported.ChannelI, // the channel of downstream that upstream has
) error {
	targetClient, found := k.clientKeeper.GetClientState(ctx, upstreamClientID)
	if !found {
		return sdkerrors.Wrap(clienttypes.ErrClientNotFound, upstreamClientID)
	}

	if err := targetClient.VerifyChannelState(
		k.clientKeeper.ClientStore(ctx, upstreamClientID), k.cdc, height,
		upstreamPrefix, proof,
		portID, channelID, channel,
	); err != nil {
		return sdkerrors.Wrapf(err, "failed channel state verification for client (%s)", upstreamClientID)
	}

	return k.setChannelStateCommitment(
		ctx,
		upstreamPrefix,
		upstreamClientID,
		portID,
		channelID,
		channel,
	)
}

// VerifyPacketCommitment verifies a proof of an outgoing packet commitment at
// the specified port, specified channel, and specified sequence.
func (k Keeper) VerifyPacketCommitment(
	ctx sdk.Context,
	upstreamClientID string,
	upstreamPrefix exported.Prefix,
	proxyConnection exported.ConnectionI,
	height exported.Height,
	proof []byte,
	portID,
	channelID string,
	sequence uint64,
	commitmentBytes []byte,
) error {
	targetClient, found := k.clientKeeper.GetClientState(ctx, upstreamClientID)
	if !found {
		return sdkerrors.Wrap(clienttypes.ErrClientNotFound, upstreamClientID)
	}

	if err := targetClient.VerifyPacketCommitment(
		k.clientKeeper.ClientStore(ctx, upstreamClientID), k.cdc, height,
		uint64(ctx.BlockTime().UnixNano()), proxyConnection.GetDelayPeriod(),
		upstreamPrefix, proof, portID, channelID,
		sequence, commitmentBytes,
	); err != nil {
		return sdkerrors.Wrapf(err, "failed packet commitment verification for client (%s)", proxyConnection.GetClientID())
	}

	return k.setPacketCommitment(
		ctx,
		upstreamPrefix,
		upstreamClientID,
		portID,
		channelID,
		sequence,
		commitmentBytes,
	)
}

// VerifyPacketAcknowledgement verifies a proof of an incoming packet
// acknowledgement at the specified port, specified channel, and specified sequence.
func (k Keeper) VerifyPacketAcknowledgement(
	ctx sdk.Context,
	upstreamClientID string,
	upstreamPrefix exported.Prefix,
	proxyConnection exported.ConnectionI,
	height exported.Height,
	proof []byte,
	portID,
	channelID string,
	sequence uint64,
	acknowledgement []byte,
) error {
	targetClient, found := k.clientKeeper.GetClientState(ctx, upstreamClientID)
	if !found {
		return sdkerrors.Wrap(clienttypes.ErrClientNotFound, upstreamClientID)
	}

	if err := targetClient.VerifyPacketAcknowledgement(
		k.clientKeeper.ClientStore(ctx, upstreamClientID), k.cdc, height,
		uint64(ctx.BlockTime().UnixNano()), proxyConnection.GetDelayPeriod(),
		upstreamPrefix, proof, portID, channelID,
		sequence, acknowledgement,
	); err != nil {
		return sdkerrors.Wrapf(err, "failed packet acknowledgement verification for client (%s)", proxyConnection.GetClientID())
	}

	return k.setPacketAcknowledgement(
		ctx,
		upstreamPrefix,
		upstreamClientID,
		portID,
		channelID,
		sequence,
		acknowledgement,
	)
}

// VerifyPacketReceiptAbsence verifies a proof of the absence of an
// incoming packet receipt at the specified port, specified channel, and
// specified sequence.
func (k Keeper) VerifyPacketReceiptAbsence(
	ctx sdk.Context,
	upstreamClientID string,
	upstreamPrefix exported.Prefix,
	proxyConnection exported.ConnectionI,
	height exported.Height,
	proof []byte,
	portID,
	channelID string,
	sequence uint64,
) error {
	targetClient, found := k.clientKeeper.GetClientState(ctx, upstreamClientID)
	if !found {
		return sdkerrors.Wrap(clienttypes.ErrClientNotFound, upstreamClientID)
	}

	if err := targetClient.VerifyPacketReceiptAbsence(
		k.clientKeeper.ClientStore(ctx, upstreamClientID), k.cdc, height,
		uint64(ctx.BlockTime().UnixNano()), proxyConnection.GetDelayPeriod(),
		upstreamPrefix, proof, portID, channelID,
		sequence,
	); err != nil {
		return sdkerrors.Wrapf(err, "failed packet receipt absence verification for client (%s)", proxyConnection.GetClientID())
	}

	return k.setPacketReceiptAbsence(
		ctx,
		upstreamPrefix,
		upstreamClientID,
		portID,
		channelID,
		sequence,
	)
}

// VerifyNextSequenceRecv verifies a proof of the next sequence number to be
// received of the specified channel at the specified port.
func (k Keeper) VerifyNextSequenceRecv(
	ctx sdk.Context,
	upstreamClientID string,
	upstreamPrefix exported.Prefix,
	proxyConnection exported.ConnectionI,
	height exported.Height,
	proof []byte,
	portID,
	channelID string,
	nextSequenceRecv uint64,
) error {
	targetClient, found := k.clientKeeper.GetClientState(ctx, upstreamClientID)
	if !found {
		return sdkerrors.Wrap(clienttypes.ErrClientNotFound, upstreamClientID)
	}

	if err := targetClient.VerifyNextSequenceRecv(
		k.clientKeeper.ClientStore(ctx, upstreamClientID), k.cdc, height,
		uint64(ctx.BlockTime().UnixNano()), proxyConnection.GetDelayPeriod(),
		upstreamPrefix, proof, portID, channelID,
		nextSequenceRecv,
	); err != nil {
		return sdkerrors.Wrapf(err, "failed next sequence receive verification for client (%s)", proxyConnection.GetClientID())
	}

	return k.setNextSequenceRecv(
		ctx,
		upstreamPrefix,
		upstreamClientID,
		portID,
		channelID,
		nextSequenceRecv,
	)
}
