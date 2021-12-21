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
	upstreamPrefix exported.Prefix,
	order channeltypes.Order,
	connectionHops []string, // from upstream
	upstreamPortID string,
	upstreamChannelID string,
	downstreamPortID string,
	version string,
	proofInit []byte,
	proofHeight exported.Height,
) error {

	connectionEnd, found := k.GetProxyConnection(ctx, upstreamPrefix, upstreamClientID, connectionHops[0])
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
	expectedCounterparty := channeltypes.NewCounterparty(downstreamPortID, "")
	expectedChannel := channeltypes.NewChannel(
		channeltypes.INIT, order, expectedCounterparty,
		connectionHops, version,
	)

	if err := k.VerifyAndProxyChannelState(
		ctx, upstreamClientID, upstreamPrefix, connectionEnd,
		proofHeight, proofInit,
		upstreamPortID, upstreamChannelID, expectedChannel,
	); err != nil {
		return err
	}

	return nil
}

// source: downstream, counterparty: upstream
func (k Keeper) ChanOpenAck(
	ctx sdk.Context,

	upstreamClientID string,
	upstreamPrefix exported.Prefix,
	order channeltypes.Order,
	connectionHops []string, // from upstream

	portID,
	channelID string,
	downstreamPortID,
	downstreamChannelID string,
	version string,
	proofTry []byte,
	proofHeight exported.Height,
) error {

	connectionEnd, found := k.GetProxyConnection(ctx, upstreamPrefix, upstreamClientID, connectionHops[0])
	if !found {
		return fmt.Errorf("connection '%v:%v' not found", upstreamClientID, connectionHops[0])
	}

	expectedCounterparty := channeltypes.NewCounterparty(downstreamPortID, downstreamChannelID)
	expectedChannel := channeltypes.NewChannel(
		channeltypes.TRYOPEN, order, expectedCounterparty,
		connectionHops, version,
	)

	if err := k.VerifyAndProxyChannelState(
		ctx, upstreamClientID, upstreamPrefix, connectionEnd,
		proofHeight, proofTry,
		portID, channelID, expectedChannel,
	); err != nil {
		return err
	}

	return nil
}

// source: downstream, counterparty: upstream
func (k Keeper) ChanOpenConfirm(
	ctx sdk.Context,

	upstreamClientID string,
	upstreamPrefix exported.Prefix,

	portID string,
	channelID string,
	downstreamChannelID string,
	proofAck []byte,
	proofHeight exported.Height,
) error {

	channel, found := k.GetProxyChannel(ctx, upstreamPrefix, upstreamClientID, portID, channelID)
	if !found {
		return fmt.Errorf("channel '%v:%v:%v' not found", upstreamClientID, portID, channelID)
	} else if channel.Counterparty.ChannelId != "" {
		return fmt.Errorf("fatal error")
	} else if channel.State != channeltypes.INIT {
		return fmt.Errorf("channel state must be %s", channeltypes.INIT)
	}
	connectionEnd, found := k.GetProxyConnection(ctx, upstreamPrefix, upstreamClientID, channel.ConnectionHops[0])
	if !found {
		return fmt.Errorf("connection '%v:%v' not found", upstreamClientID, channel.ConnectionHops[0])
	}

	channel.Counterparty.ChannelId = downstreamChannelID
	channel.State = channeltypes.OPEN

	if err := k.VerifyAndProxyChannelState(
		ctx, upstreamClientID, upstreamPrefix, connectionEnd,
		proofHeight, proofAck,
		portID, channelID, channel,
	); err != nil {
		return err
	}

	return nil
}
