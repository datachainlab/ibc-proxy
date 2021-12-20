package keeper

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	connectiontypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

// source: downstream, counterparty: upstream
func (k Keeper) RecvPacket(
	ctx sdk.Context,
	upstreamClientID string,
	upstreamPrefix exported.Prefix,
	packet exported.PacketI,
	proof []byte,
	proofHeight exported.Height,
) error {
	channel, found := k.GetChannel(ctx, upstreamClientID, packet.GetSourcePort(), packet.GetSourceChannel())
	if !found {
		return sdkerrors.Wrap(channeltypes.ErrChannelNotFound, packet.GetDestChannel())
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
	connectionEnd, found := k.GetConnection(ctx, upstreamClientID, channel.ConnectionHops[0])
	if !found {
		return sdkerrors.Wrap(connectiontypes.ErrConnectionNotFound, channel.ConnectionHops[0])
	}

	// check if packet timeouted by comparing it with the latest timestamp of the chain
	if packet.GetTimeoutTimestamp() != 0 && uint64(ctx.BlockTime().UnixNano()) >= packet.GetTimeoutTimestamp() {
		return sdkerrors.Wrapf(
			channeltypes.ErrPacketTimeout,
			"block timestamp >= packet timeout timestamp (%s >= %s)", ctx.BlockTime(), time.Unix(0, int64(packet.GetTimeoutTimestamp())),
		)
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

	// TODO persists a packet

	return nil
}

func (k Keeper) AcknowledgePacket(
	ctx sdk.Context,
	upstreamClientID string,
	upstreamPrefix exported.Prefix,
	packet exported.PacketI,
	acknowledgement []byte,
	proof []byte,
	proofHeight exported.Height,
) error {
	channel, found := k.GetChannel(ctx, upstreamClientID, packet.GetDestPort(), packet.GetDestChannel())
	if !found {
		return sdkerrors.Wrapf(
			channeltypes.ErrChannelNotFound,
			"port ID (%s) channel ID (%s)", packet.GetDestPort(), packet.GetDestChannel(),
		)
	}

	// Connection must be OPEN to receive a packet. It is possible for connection to not yet be open if packet was
	// sent optimistically before connection and channel handshake completed. However, to receive a packet,
	// connection and channel must both be open
	connectionEnd, found := k.GetConnection(ctx, upstreamClientID, channel.ConnectionHops[0])
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

	// TODO persists an acknowledgement

	return nil
}
