package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	connectiontypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

// upstream: chainA, downstream: chainB
func (k Keeper) ChanOpenTry(
	ctx sdk.Context,

	upstreamClientID string, // the client ID corresponding to light client for chainA on chainB
	upstreamPrefix exported.Prefix, // store prefix on chainA

	order channeltypes.Order, // the channel order on chainA
	connectionHops []string, // hops from upstream
	upstreamPortID string, // the portID on chainA
	upstreamChannelID string, // the channelID on chainA
	downstreamPortID string, // the portID on chainB
	version string, // the version of chainA's channel

	proofInit []byte, // proof that chainA stored channel in state
	proofHeight exported.Height, // height at which relayer constructs proof of chainA storing channel in state
) error {
	if l := len(connectionHops); l != 1 {
		return fmt.Errorf("hops length must be 1, but got %v", l)
	}

	connectionEnd, found := k.GetProxyConnection(ctx, upstreamPrefix, upstreamClientID, connectionHops[0])
	if !found {
		return sdkerrors.Wrapf(
			connectiontypes.ErrConnectionNotFound,
			"connection '%#v:%v:%v' not found", upstreamPrefix, upstreamClientID, connectionHops[0],
		)
	}
	if err := k.validateChannelOrder(order, connectionEnd); err != nil {
		return err
	}

	// expectedCounterpaty is the counterparty of the counterparty's channel end
	// (i.e self)
	expectedCounterparty := channeltypes.NewCounterparty(downstreamPortID, "")
	expectedChannel := channeltypes.NewChannel(
		channeltypes.INIT, order, expectedCounterparty,
		connectionHops, version,
	)

	if err := k.VerifyAndProxyChannelState(
		ctx, upstreamClientID, upstreamPrefix,
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

	upstreamClientID string, // the client ID corresponding to light client for chainA on chainB
	upstreamPrefix exported.Prefix, // store prefix on chainA

	order channeltypes.Order, // the channel order on chainA
	connectionHops []string, // hops from upstream

	upstreamPortID string, // the portID on chainA
	upstreamChannelID string, // the channelID on chainA
	downstreamPortID string, // the portID on chainB
	downstreamChannelID string, // the channelID on chainB

	version string, // the version of chainA's channel
	proofTry []byte, // proof that chainA stored channel in state
	proofHeight exported.Height, // height at which relayer constructs proof of chainA storing channel in state
) error {
	if l := len(connectionHops); l != 1 {
		return fmt.Errorf("hops length must be 1, but got %v", l)
	}

	connectionEnd, found := k.GetProxyConnection(ctx, upstreamPrefix, upstreamClientID, connectionHops[0])
	if !found {
		return sdkerrors.Wrapf(
			connectiontypes.ErrConnectionNotFound,
			"connection '%#v:%v:%v' not found", upstreamPrefix, upstreamClientID, connectionHops[0],
		)
	}
	if err := k.validateChannelOrder(order, connectionEnd); err != nil {
		return err
	}

	expectedCounterparty := channeltypes.NewCounterparty(downstreamPortID, downstreamChannelID)
	expectedChannel := channeltypes.NewChannel(
		channeltypes.TRYOPEN, order, expectedCounterparty,
		connectionHops, version,
	)

	if err := k.VerifyAndProxyChannelState(
		ctx, upstreamClientID, upstreamPrefix,
		proofHeight, proofTry,
		upstreamPortID, upstreamChannelID, expectedChannel,
	); err != nil {
		return err
	}

	return nil
}

// source: downstream, counterparty: upstream
func (k Keeper) ChanOpenConfirm(
	ctx sdk.Context,

	upstreamClientID string, // the client ID corresponding to light client for chainA on chainB
	upstreamPrefix exported.Prefix, // store prefix on chainA

	upstreamPortID string, // the portID on chainA
	upstreamChannelID string, // the channelID on chainA
	downstreamChannelID string, // the channelID on chainB

	proofAck []byte, // proof that chainA stored channel in state
	proofHeight exported.Height, // height at which relayer constructs proof of chainA storing channel in state
) error {

	channel, found := k.GetProxyChannel(ctx, upstreamPrefix, upstreamClientID, upstreamPortID, upstreamChannelID)
	if !found {
		return fmt.Errorf("channel '%#v:%v:%v:%v' not found", upstreamPrefix, upstreamClientID, upstreamPortID, upstreamChannelID)
	} else if channel.Counterparty.ChannelId != "" {
		return fmt.Errorf("counterparty.ChannelID must be empty, but got '%v'", channel.Counterparty.ChannelId)
	} else if channel.State != channeltypes.INIT {
		return fmt.Errorf("channel state must be %s", channeltypes.INIT)
	}

	channel.Counterparty.ChannelId = downstreamChannelID
	channel.State = channeltypes.OPEN

	if err := k.VerifyAndProxyChannelState(
		ctx, upstreamClientID, upstreamPrefix,
		proofHeight, proofAck,
		upstreamPortID, upstreamChannelID, channel,
	); err != nil {
		return err
	}

	return nil
}

func (k Keeper) validateChannelOrder(order channeltypes.Order, connectionEnd connectiontypes.ConnectionEnd) error {
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
	return nil
}
