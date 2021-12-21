package keeper

import (
	"math"

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
	counterpartyClientID string,
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
		upstreamPrefix, counterpartyClientID, proof, clientState); err != nil {
		return sdkerrors.Wrapf(err, "failed client state verification for target client: %s", upstreamClientID)
	}
	return nil
}

func (k Keeper) VerifyAndProxyClientState(
	ctx sdk.Context,
	upstreamClientID string,
	upstreamPrefix exported.Prefix,
	counterpartyClientID string,
	height exported.Height,
	proof []byte,
	clientState exported.ClientState, // the state of downstream that upstream has
) error {
	if err := k.VerifyClientState(ctx, upstreamClientID, upstreamPrefix, counterpartyClientID, height, proof, clientState); err != nil {
		return err
	}
	return k.SetProxyClientState(
		ctx,
		upstreamPrefix,
		counterpartyClientID,
		upstreamClientID,
		clientState,
	)
}

func (k Keeper) VerifyClientConsensusState(
	ctx sdk.Context,
	upstreamClientID string,
	upstreamPrefix exported.Prefix,
	counterpartyClientID string,
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
		counterpartyClientID, consensusHeight, upstreamPrefix, proof, consensusState,
	); err != nil {
		return sdkerrors.Wrapf(err, "failed consensus state verification for client (%s)", upstreamClientID)
	}
	return nil
}

func (k Keeper) VerifyAndProxyClientConsensusState(
	ctx sdk.Context,
	upstreamClientID string,
	upstreamPrefix exported.Prefix,
	counterpartyClientID string,
	height exported.Height,
	consensusHeight exported.Height,
	proof []byte,
	consensusState exported.ConsensusState, // the state of downstream that upstream has
) error {
	if err := k.VerifyClientConsensusState(ctx, upstreamClientID, upstreamPrefix, counterpartyClientID, height, consensusHeight, proof, consensusState); err != nil {
		return err
	}
	return k.SetProxyClientConsensusState(
		ctx,
		upstreamPrefix,
		counterpartyClientID,
		upstreamClientID,
		consensusHeight,
		consensusState,
	)
}

func (k Keeper) VerifyConnectionState(
	ctx sdk.Context,
	upstreamClientID string,
	upstreamPrefix exported.Prefix,
	connection connectiontypes.ConnectionEnd,
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
		upstreamPrefix, proof, connectionID, connection,
	); err != nil {
		return sdkerrors.Wrapf(err, "failed connection state verification for client (%s)", upstreamClientID)
	}

	return nil
}

func (k Keeper) VerifyAndProxyConnectionState(
	ctx sdk.Context,
	upstreamClientID string,
	upstreamPrefix exported.Prefix,
	connection connectiontypes.ConnectionEnd,
	height exported.Height,
	proof []byte,
	connectionID string, // ID of the connection that upstream has
) error {
	if err := k.VerifyConnectionState(ctx, upstreamClientID, upstreamPrefix, connection, height, proof, connectionID); err != nil {
		return err
	}
	return k.SetProxyConnection(
		ctx,
		upstreamPrefix,
		upstreamClientID,
		connectionID,
		connection,
	)
}

func (k Keeper) VerifyChannelState(
	ctx sdk.Context,
	upstreamClientID string,
	upstreamPrefix exported.Prefix,
	connection exported.ConnectionI,
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

	return nil
}

func (k Keeper) VerifyAndProxyChannelState(
	ctx sdk.Context,
	upstreamClientID string,
	upstreamPrefix exported.Prefix,
	connection exported.ConnectionI,
	height exported.Height,
	proof []byte,
	portID,
	channelID string,
	channel exported.ChannelI, // the channel of downstream that upstream has
) error {
	if err := k.VerifyChannelState(ctx, upstreamClientID, upstreamPrefix, connection, height, proof, portID, channelID, channel); err != nil {
		return err
	}
	return k.SetProxyChannel(
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
	connection exported.ConnectionI,
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
		ctx, k.clientKeeper.ClientStore(ctx, upstreamClientID), k.cdc, height,
		connection.GetDelayPeriod(), k.getBlockDelay(ctx, connection),
		upstreamPrefix, proof, portID, channelID,
		sequence, commitmentBytes,
	); err != nil {
		return sdkerrors.Wrapf(err, "failed packet commitment verification for client (%s)", connection.GetClientID())
	}

	return nil
}

func (k Keeper) VerifyAndProxyPacketCommitment(
	ctx sdk.Context,
	upstreamClientID string,
	upstreamPrefix exported.Prefix,
	connection exported.ConnectionI,
	height exported.Height,
	proof []byte,
	portID,
	channelID string,
	sequence uint64,
	commitmentBytes []byte,
) error {
	if err := k.VerifyPacketCommitment(ctx, upstreamClientID, upstreamPrefix, connection, height, proof, portID, channelID, sequence, commitmentBytes); err != nil {
		return err
	}

	return k.SetProxyPacketCommitment(
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
	connection exported.ConnectionI,
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
		ctx, k.clientKeeper.ClientStore(ctx, upstreamClientID), k.cdc, height,
		connection.GetDelayPeriod(), k.getBlockDelay(ctx, connection),
		upstreamPrefix, proof, portID, channelID,
		sequence, acknowledgement,
	); err != nil {
		return sdkerrors.Wrapf(err, "failed packet acknowledgement verification for client (%s)", connection.GetClientID())
	}

	return nil
}

func (k Keeper) VerifyAndProxyPacketAcknowledgement(
	ctx sdk.Context,
	upstreamClientID string,
	upstreamPrefix exported.Prefix,
	connection exported.ConnectionI,
	height exported.Height,
	proof []byte,
	portID,
	channelID string,
	sequence uint64,
	acknowledgement []byte,
) error {
	if err := k.VerifyPacketAcknowledgement(ctx, upstreamClientID, upstreamPrefix, connection, height, proof, portID, channelID, sequence, acknowledgement); err != nil {
		return err
	}

	return k.SetProxyPacketAcknowledgement(
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
	connection exported.ConnectionI,
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
		ctx, k.clientKeeper.ClientStore(ctx, upstreamClientID), k.cdc, height,
		connection.GetDelayPeriod(), k.getBlockDelay(ctx, connection),
		upstreamPrefix, proof, portID, channelID,
		sequence,
	); err != nil {
		return sdkerrors.Wrapf(err, "failed packet receipt absence verification for client (%s)", connection.GetClientID())
	}

	return nil
}

func (k Keeper) VerifyAndProxyPacketReceiptAbsence(
	ctx sdk.Context,
	upstreamClientID string,
	upstreamPrefix exported.Prefix,
	connection exported.ConnectionI,
	height exported.Height,
	proof []byte,
	portID,
	channelID string,
	sequence uint64,
) error {
	if err := k.VerifyPacketReceiptAbsence(ctx, upstreamClientID, upstreamPrefix, connection, height, proof, portID, channelID, sequence); err != nil {
		return err
	}

	return k.SetProxyPacketReceiptAbsence(
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
	connection exported.ConnectionI,
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
		ctx, k.clientKeeper.ClientStore(ctx, upstreamClientID), k.cdc, height,
		connection.GetDelayPeriod(), k.getBlockDelay(ctx, connection),
		upstreamPrefix, proof, portID, channelID,
		nextSequenceRecv,
	); err != nil {
		return sdkerrors.Wrapf(err, "failed next sequence receive verification for client (%s)", connection.GetClientID())
	}

	return nil
}

func (k Keeper) VerifyAndProxyNextSequenceRecv(
	ctx sdk.Context,
	upstreamClientID string,
	upstreamPrefix exported.Prefix,
	connection exported.ConnectionI,
	height exported.Height,
	proof []byte,
	portID,
	channelID string,
	nextSequenceRecv uint64,
) error {
	if err := k.VerifyNextSequenceRecv(ctx, upstreamClientID, upstreamPrefix, connection, height, proof, portID, channelID, nextSequenceRecv); err != nil {
		return err
	}

	return k.SetProxyNextSequenceRecv(
		ctx,
		upstreamPrefix,
		upstreamClientID,
		portID,
		channelID,
		nextSequenceRecv,
	)
}

// getBlockDelay calculates the block delay period from the time delay of the connection
// and the maximum expected time per block.
func (k Keeper) getBlockDelay(ctx sdk.Context, connection exported.ConnectionI) uint64 {
	// expectedTimePerBlock should never be zero, however if it is then return a 0 blcok delay for safety
	// as the expectedTimePerBlock parameter was not set.
	expectedTimePerBlock := k.connectionKeeper.GetMaxExpectedTimePerBlock(ctx)
	if expectedTimePerBlock == 0 {
		return 0
	}
	// calculate minimum block delay by dividing time delay period
	// by the expected time per block. Round up the block delay.
	timeDelay := connection.GetDelayPeriod()
	return uint64(math.Ceil(float64(timeDelay) / float64(expectedTimePerBlock)))
}
