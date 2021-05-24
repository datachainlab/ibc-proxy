package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	connectiontypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

// source: downstream, counterparty: upstream
func (k Keeper) ChanOpenTry(
	ctx sdk.Context,
	upstreamClientID string,
	order channeltypes.Order,
	connectionHops []string, // from upstream
	portID,
	previousChannelID string,
	counterparty channeltypes.Counterparty,
	version,
	counterpartyVersion string,
	proofInit []byte,
	proofHeight exported.Height,
) error {

	connectionEnd, found := k.GetConnection(ctx, upstreamClientID, connectionHops[0])
	if !found {
		return fmt.Errorf("connection '%v:%v' not found", upstreamClientID, connectionHops[0])
	}

	getVersions := connectionEnd.GetVersions()
	if len(getVersions) != 1 {
		return sdkerrors.Wrapf(
			connectiontypes.ErrInvalidVersion,
			"single version must be negotiated on connection before opening channel, got: %v",
			getVersions,
		)
	}

	if !connectiontypes.VerifySupportedFeature(getVersions[0], order.String()) {
		return sdkerrors.Wrapf(
			connectiontypes.ErrInvalidVersion,
			"connection version %s does not support channel ordering: %s",
			getVersions[0], order.String(),
		)
	}

	// expectedCounterpaty is the counterparty of the counterparty's channel end
	// (i.e self)
	expectedCounterparty := channeltypes.NewCounterparty(portID, "")
	expectedChannel := channeltypes.NewChannel(
		channeltypes.INIT, order, expectedCounterparty,
		connectionHops, counterpartyVersion,
	)

	if err := k.VerifyChannelState(
		ctx, upstreamClientID, connectionEnd,
		proofHeight, proofInit,
		counterparty.PortId, counterparty.ChannelId, expectedChannel,
	); err != nil {
		return err
	}

	k.SetChannel(ctx, upstreamClientID, counterparty.PortId, counterparty.ChannelId, expectedChannel)
	return nil
}

// source: downstream, counterparty: upstream
func (k Keeper) ChanOpenAck(
	ctx sdk.Context,

	upstreamClientID string,
	order channeltypes.Order,
	connectionHops []string, // from upstream

	portID,
	channelID string,
	counterparty channeltypes.Counterparty,
	version,
	counterpartyVersion string,
	proofTry []byte,
	proofHeight exported.Height,
) error {

	connectionEnd, found := k.GetConnection(ctx, upstreamClientID, connectionHops[0])
	if !found {
		return fmt.Errorf("connection '%v:%v' not found", upstreamClientID, connectionHops[0])
	}

	expectedCounterparty := channeltypes.NewCounterparty(portID, channelID)
	expectedChannel := channeltypes.NewChannel(
		channeltypes.TRYOPEN, order, expectedCounterparty,
		connectionHops, counterpartyVersion,
	)

	if err := k.VerifyChannelState(
		ctx, upstreamClientID, connectionEnd,
		proofHeight, proofTry,
		counterparty.PortId, counterparty.ChannelId, expectedChannel,
	); err != nil {
		return err
	}

	k.SetChannel(ctx, upstreamClientID, counterparty.PortId, counterparty.ChannelId, expectedChannel)
	return nil
}

// source: downstream, counterparty: upstream
func (k Keeper) ChanOpenConfirm(
	ctx sdk.Context,

	upstreamClientID string,

	sourceChannelID string,
	counterpartyPortID,
	counterpartyChannelID string,
	proofAck []byte,
	proofHeight exported.Height,
) error {

	channel, found := k.GetChannel(ctx, upstreamClientID, counterpartyPortID, counterpartyChannelID)
	if !found {
		return fmt.Errorf("channel '%v:%v:%v' not found", upstreamClientID, counterpartyPortID, counterpartyChannelID)
	} else if channel.Counterparty.ChannelId != "" {
		return fmt.Errorf("fatal error")
	} else if channel.State != channeltypes.INIT {
		return fmt.Errorf("channel state must be %s", channeltypes.INIT)
	}
	connectionEnd, found := k.GetConnection(ctx, upstreamClientID, channel.ConnectionHops[0])
	if !found {
		return fmt.Errorf("connection '%v:%v' not found", upstreamClientID, channel.ConnectionHops[0])
	}

	channel.Counterparty.ChannelId = sourceChannelID
	channel.State = channeltypes.OPEN

	if err := k.VerifyChannelState(
		ctx, upstreamClientID, connectionEnd,
		proofHeight, proofAck,
		counterpartyPortID, counterpartyChannelID, channel,
	); err != nil {
		return err
	}

	k.SetChannel(ctx, upstreamClientID, counterpartyPortID, counterpartyChannelID, channel)
	return nil
}
