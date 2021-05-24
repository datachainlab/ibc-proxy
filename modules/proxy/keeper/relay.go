package keeper

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/exported"
	proxytypes "github.com/datachainlab/ibc-proxy/modules/light-clients/xx-proxy/types"
	"github.com/datachainlab/ibc-proxy/modules/proxy/types"
)

func (k Keeper) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
) exported.Acknowledgement {
	var data types.ProxyRequestPacketData
	if err := k.cdc.Unmarshal(packet.Data, &data); err != nil {
		return channeltypes.NewErrorAcknowledgement(fmt.Sprintf("cannot unmarshal proxy packet data: %s", err.Error()))
	}
	upState, err := k.OnRecvProxyRequest(ctx, packet, data)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(fmt.Sprintf("failed to OnRecvProxyRequest: %s", err.Error()))
	}
	ackData := types.NewProxyRequestAcknowledgement(types.OK, k.GetCommitmentPrefix().(commitmenttypes.MerklePrefix), *upState)
	return channeltypes.NewResultAcknowledgement(k.cdc.MustMarshal(&ackData))
}

func (k Keeper) OnAcknowledgementPacket(ctx sdk.Context, packet channeltypes.Packet, ack channeltypes.Acknowledgement) error {
	var data types.ProxyRequestPacketData
	k.cdc.MustUnmarshal(packet.Data, &data)
	switch res := ack.Response.(type) {
	case *channeltypes.Acknowledgement_Error:
		// TODO freeze the proxy client?
		return errors.New(res.Error)
	case *channeltypes.Acknowledgement_Result:
		// if success, update the clientState with upstreamState
		var ack types.ProxyRequestAcknowledgement
		if err := k.cdc.Unmarshal(res.Result, &ack); err != nil {
			return err
		} else if ack.Status != types.OK {
			return fmt.Errorf("unexpected status: %v", ack.Status.String())
		}
		return k.OnProxyRequestAcknowledgement(ctx, packet, data, ack)
	default:
		panic("unreachable")
	}
}

// caller: downstream
// steps:
// - the channel exists
// - create a new proxy client
// - send the packet that contains the upstream and proxy client ID
func (k Keeper) SendProxyRequest(
	ctx sdk.Context,
	sourcePort,
	sourceChannel string,
	upstreamClientID string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
) (*channeltypes.Packet, string, error) {

	sourceChannelEnd, found := k.channelKeeper.GetChannel(ctx, sourcePort, sourceChannel)
	if !found {
		return nil, "", sdkerrors.Wrapf(channeltypes.ErrChannelNotFound, "port ID (%s) channel ID (%s)", sourcePort, sourceChannel)
	}

	destinationPort := sourceChannelEnd.GetCounterparty().GetPortID()
	destinationChannel := sourceChannelEnd.GetCounterparty().GetChannelID()

	// get the next sequence
	sequence, found := k.channelKeeper.GetNextSequenceSend(ctx, sourcePort, sourceChannel)
	if !found {
		return nil, "", sdkerrors.Wrapf(
			channeltypes.ErrSequenceSendNotFound,
			"source port: %s, source channel: %s", sourcePort, sourceChannel,
		)
	}

	// begin createOutgoingPacket logic
	channelCap, ok := k.scopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(sourcePort, sourceChannel))
	if !ok {
		return nil, "", sdkerrors.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
	}

	proxyClientID, err := k.clientKeeper.CreateClient(ctx, proxytypes.NewClientState(upstreamClientID, nil), nil)
	if err != nil {
		return nil, "", err
	}

	packetData := types.NewProxyRequestPacketData(upstreamClientID, proxyClientID)

	packet := channeltypes.NewPacket(
		k.cdc.MustMarshal(&packetData),
		sequence,
		sourcePort,
		sourceChannel,
		destinationPort,
		destinationChannel,
		timeoutHeight,
		timeoutTimestamp,
	)

	if err := k.channelKeeper.SendPacket(ctx, channelCap, packet); err != nil {
		return nil, proxyClientID, err
	}

	return &packet, proxyClientID, nil
}

// caller: proxy
func (k Keeper) OnRecvProxyRequest(
	ctx sdk.Context,
	packet channeltypes.Packet,
	data types.ProxyRequestPacketData,
) (*types.UpstreamState, error) {
	clientState, found := k.clientKeeper.GetClientState(ctx, data.UpstreamClientId)
	if !found {
		return nil, sdkerrors.Wrapf(clienttypes.ErrClientNotFound, "client '%v' not found", data.UpstreamClientId)
	}
	height := clientState.GetLatestHeight()
	consensusState, found := k.clientKeeper.GetClientConsensusState(ctx, data.UpstreamClientId, height)
	if !found {
		return nil, sdkerrors.Wrapf(clienttypes.ErrConsensusStateNotFound, "consensusState '%v %v' not found", data.UpstreamClientId, height.String())
	}

	anyClientState, err := clienttypes.PackClientState(clientState)
	if err != nil {
		return nil, err
	}
	anyConsensusState, err := clienttypes.PackConsensusState(consensusState)
	if err != nil {
		return nil, err
	}

	// TODO should the keeper save the downstream channel info and the proxy clientID to store?

	upState := types.NewUpstreamState(height.(clienttypes.Height), anyClientState, anyConsensusState)
	return &upState, nil
}

// caller: downstream
func (k Keeper) OnProxyRequestAcknowledgement(ctx sdk.Context, packet channeltypes.Packet, data types.ProxyRequestPacketData, ack types.ProxyRequestAcknowledgement) error {
	cs, found := k.clientKeeper.GetClientState(ctx, data.ProxyClientId)
	if !found {
		return sdkerrors.Wrapf(clienttypes.ErrClientNotFound, "client '%v' not found", data.ProxyClientId)
	}

	channel, found := k.channelKeeper.GetChannel(ctx, packet.SourcePort, packet.SourceChannel)
	if !found {
		return sdkerrors.Wrapf(channeltypes.ErrChannelNotFound, "port ID (%s) channel ID (%s)", packet.SourcePort, packet.SourceChannel)
	}
	conn, found := k.connectionKeeper.GetConnection(ctx, channel.ConnectionHops[0])
	if !found {
		return fmt.Errorf("fatal error")
	}

	// fork a {client,consensus}State and setup the proxy client

	// TODO introduce a client setup handler corresponding to each client type

	proxyClientState, found := k.clientKeeper.GetClientState(ctx, conn.ClientId)
	if !found {
		return sdkerrors.Wrapf(clienttypes.ErrClientNotFound, "client '%v' not found", conn.ClientId)
	}
	proxyConsensusState, found := k.clientKeeper.GetClientConsensusState(ctx, conn.ClientId, proxyClientState.GetLatestHeight())
	if !found {
		return sdkerrors.Wrapf(clienttypes.ErrClientNotFound, "consensusState '%v %v' not found", conn.ClientId, proxyClientState.GetLatestHeight().String())
	}

	if err := proxyClientState.Initialize(ctx, k.cdc, k.clientKeeper.ClientStore(ctx, data.ProxyClientId), proxyConsensusState); err != nil {
		return err
	}

	anyProxyClientState, err := clienttypes.PackClientState(proxyClientState)
	if err != nil {
		return err
	}
	anyProxyConsensusState, err := clienttypes.PackConsensusState(proxyConsensusState)
	if err != nil {
		return err
	}

	clientState, ok := cs.(*proxytypes.ClientState)
	if !ok {
		return errors.New("fatal error")
	}
	clientState.ProxyClientState = anyProxyClientState
	clientState.Prefix = ack.Prefix
	clientState.UpstreamClientState = ack.UpstreamState.ClientState
	clientState.UpstreamConsensusState = ack.UpstreamState.ConsensusState

	consensusState := proxytypes.NewConsensusState(anyProxyConsensusState)

	k.clientKeeper.SetClientState(ctx, data.ProxyClientId, clientState)
	k.clientKeeper.SetClientConsensusState(ctx, data.ProxyClientId, proxyClientState.GetLatestHeight(), consensusState)
	return nil
}
