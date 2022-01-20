package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	connectiontypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

// upstream(dst): chainA, downstream(src): chainB
func (k Keeper) RecvPacket(
	ctx sdk.Context,
	upstreamClientID string, // the client ID corresponding to light client for chainA on chainB
	upstreamPrefix exported.Prefix, // store prefix on chainA
	packet exported.PacketI, // packet
	proof []byte, // proof that chanA stored packet in state
	proofHeight exported.Height, // height at which relayer constructs proof of chainA storing packet in state
) error {
	channel, found := k.GetProxyChannel(ctx, upstreamPrefix, upstreamClientID, packet.GetSourcePort(), packet.GetSourceChannel())
	if !found {
		return sdkerrors.Wrap(channeltypes.ErrChannelNotFound, packet.GetSourceChannel())
	}

	// packet must come from the channel's counterparty
	if packet.GetDestPort() != channel.Counterparty.PortId {
		return sdkerrors.Wrapf(
			channeltypes.ErrInvalidPacket,
			"packet dest port doesn't match the counterparty's port (%s ≠ %s)", packet.GetDestPort(), channel.Counterparty.PortId,
		)
	}

	if packet.GetDestChannel() != channel.Counterparty.ChannelId {
		return sdkerrors.Wrapf(
			channeltypes.ErrInvalidPacket,
			"packet dest channel doesn't match the counterparty's channel (%s ≠ %s)", packet.GetDestChannel(), channel.Counterparty.ChannelId,
		)
	}

	// Connection must be OPEN to receive a packet. It is possible for connection to not yet be open if packet was
	// sent optimistically before connection and channel handshake completed. However, to receive a packet,
	// connection and channel must both be open
	connectionEnd, found := k.GetProxyConnection(ctx, upstreamPrefix, upstreamClientID, channel.ConnectionHops[0])
	if !found {
		return sdkerrors.Wrap(connectiontypes.ErrConnectionNotFound, channel.ConnectionHops[0])
	}

	if err := k.VerifyAndProxyPacketCommitment(
		ctx,
		upstreamClientID,
		upstreamPrefix,
		connectionEnd,
		proofHeight, proof,
		packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence(),
		channeltypes.CommitPacket(k.cdc, packet),
	); err != nil {
		return err
	}

	return nil
}

// upstream: chainA(src), downstream: chainB(dst)
func (k Keeper) AcknowledgePacket(
	ctx sdk.Context,
	upstreamClientID string, // the client ID corresponding to light client for chainA on chainB
	upstreamPrefix exported.Prefix, // store prefix on chainA
	packet exported.PacketI, // packet
	acknowledgement []byte, // ack
	proof []byte, // proof that chanA stored packet in state
	proofHeight exported.Height, // height at which relayer constructs proof of chainA storing packet in state
) error {
	channel, found := k.GetProxyChannel(ctx, upstreamPrefix, upstreamClientID, packet.GetDestPort(), packet.GetDestChannel())
	if !found {
		return sdkerrors.Wrapf(
			channeltypes.ErrChannelNotFound,
			"port ID (%s) channel ID (%s)", packet.GetDestPort(), packet.GetDestChannel(),
		)
	}

	// Connection must be OPEN to receive a packet. It is possible for connection to not yet be open if packet was
	// sent optimistically before connection and channel handshake completed. However, to receive a packet,
	// connection and channel must both be open
	connectionEnd, found := k.GetProxyConnection(ctx, upstreamPrefix, upstreamClientID, channel.ConnectionHops[0])
	if !found {
		return sdkerrors.Wrap(connectiontypes.ErrConnectionNotFound, channel.ConnectionHops[0])
	}

	if err := k.VerifyAndProxyPacketAcknowledgement(
		ctx,
		upstreamClientID,
		upstreamPrefix,
		connectionEnd,
		proofHeight, proof,
		packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence(),
		acknowledgement,
	); err != nil {
		return err
	}

	return nil
}

// upstream(dst): chainA, downstream(src): chainB
func (k Keeper) TimeoutPacket(
	ctx sdk.Context,
	upstreamClientID string, // the client ID corresponding to light client for chainA on chainB
	upstreamPrefix exported.Prefix, // store prefix on chainA
	packet exported.PacketI,
	proof []byte,
	proofHeight exported.Height,
	nextSequenceRecv uint64,
) error {
	channel, found := k.GetProxyChannel(ctx, upstreamPrefix, upstreamClientID, packet.GetSourcePort(), packet.GetSourceChannel())
	if !found {
		return sdkerrors.Wrapf(
			channeltypes.ErrChannelNotFound,
			"port ID (%s) channel ID (%s)", packet.GetSourcePort(), packet.GetSourceChannel(),
		)
	}
	if channel.State != channeltypes.OPEN {
		return sdkerrors.Wrapf(
			channeltypes.ErrInvalidChannelState,
			"channel state is not OPEN (got %s)", channel.State.String(),
		)
	}

	if packet.GetDestPort() != channel.Counterparty.PortId {
		return sdkerrors.Wrapf(
			channeltypes.ErrInvalidPacket,
			"packet destination port doesn't match the counterparty's port (%s ≠ %s)", packet.GetDestPort(), channel.Counterparty.PortId,
		)
	}

	if packet.GetDestChannel() != channel.Counterparty.ChannelId {
		return sdkerrors.Wrapf(
			channeltypes.ErrInvalidPacket,
			"packet destination channel doesn't match the counterparty's channel (%s ≠ %s)", packet.GetDestChannel(), channel.Counterparty.ChannelId,
		)
	}

	connectionEnd, found := k.GetProxyConnection(ctx, upstreamPrefix, upstreamClientID, channel.ConnectionHops[0])
	if !found {
		return sdkerrors.Wrap(
			connectiontypes.ErrConnectionNotFound,
			channel.ConnectionHops[0],
		)
	}

	proofTimestamp, err := k.GetUpstreamTimestampAtHeight(ctx, upstreamClientID, proofHeight)
	if err != nil {
		return err
	}
	timeoutHeight := packet.GetTimeoutHeight()
	if (timeoutHeight.IsZero() || proofHeight.LT(timeoutHeight)) &&
		(packet.GetTimeoutTimestamp() == 0 || proofTimestamp < packet.GetTimeoutTimestamp()) {
		return sdkerrors.Wrap(channeltypes.ErrPacketTimeout, "packet timeout has not been reached for height or timestamp")
	}

	switch channel.Ordering {
	case channeltypes.ORDERED:
		// check that packet has not been received
		if nextSequenceRecv > packet.GetSequence() {
			return sdkerrors.Wrapf(
				channeltypes.ErrPacketReceived,
				"packet already received, next sequence receive > packet sequence (%d > %d)", nextSequenceRecv, packet.GetSequence(),
			)
		}
		// check that the recv sequence is as claimed
		err = k.VerifyAndProxyNextSequenceRecv(
			ctx, upstreamClientID, upstreamPrefix, connectionEnd, proofHeight, proof,
			packet.GetDestPort(), packet.GetDestChannel(), nextSequenceRecv,
		)
	case channeltypes.UNORDERED:
		err = k.VerifyAndProxyPacketReceiptAbsence(
			ctx, upstreamClientID, upstreamPrefix, connectionEnd, proofHeight, proof,
			packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence(),
		)
	default:
		panic(sdkerrors.Wrapf(channeltypes.ErrInvalidChannelOrdering, channel.Ordering.String()))
	}

	if err != nil {
		return err
	}

	return nil
}
