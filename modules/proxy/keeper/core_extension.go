package keeper

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
	multivtypes "github.com/datachainlab/ibc-proxy/modules/light-clients/xx-multiv/types"
	proxyclienttypes "github.com/datachainlab/ibc-proxy/modules/light-clients/xx-proxy/types"
)

// ClientKeeper override `ValidateSelfClient` in the keeper of ibc-client
// Original method doesn't yet support a consensus state for a general client
type ClientKeeper struct {
	connectiontypes.ClientKeeper
}

func NewClientKeeper(k connectiontypes.ClientKeeper) ClientKeeper {
	return ClientKeeper{ClientKeeper: k}
}

func (k ClientKeeper) ValidateSelfClient(ctx sdk.Context, clientState exported.ClientState) error {
	cs, ok := clientState.(*multivtypes.ClientState)
	if !ok {
		return k.ClientKeeper.ValidateSelfClient(ctx, clientState)
	}
	return k.ClientKeeper.ValidateSelfClient(ctx, cs.GetUnderlyingClientState())
}

type ConnectionKeeper struct {
	clientKeeper connectiontypes.ClientKeeper
	channeltypes.ConnectionKeeper
}

func NewConnectionKeeper(clientKeeper connectiontypes.ClientKeeper, base channeltypes.ConnectionKeeper) ConnectionKeeper {
	return ConnectionKeeper{clientKeeper: clientKeeper, ConnectionKeeper: base}
}

func (k ConnectionKeeper) GetTimestampAtHeight(ctx sdk.Context, connection connectiontypes.ConnectionEnd, height exported.Height) (uint64, error) {
	clientState, found := k.clientKeeper.GetClientState(ctx, connection.GetClientID())
	if !found {
		return 0, sdkerrors.Wrapf(
			clienttypes.ErrClientNotFound,
			"clientID (%s)", connection.GetClientID(),
		)
	}
	if clientState.ClientType() == proxyclienttypes.ProxyClientType {
		blockTime, found := proxyclienttypes.GetUpstreamBlockTime(k.clientKeeper.ClientStore(ctx, connection.GetClientID()), height)
		if !found {
			return 0, sdkerrors.Wrapf(errors.New("ErrUpstreamBlockTimeNotFound"), "clientID (%s), height (%s)", connection.GetClientID(), height)
		}
		return blockTime, nil
	}
	consensusState, found := k.clientKeeper.GetClientConsensusState(ctx, connection.GetClientID(), height)
	if !found {
		return 0, sdkerrors.Wrapf(
			clienttypes.ErrConsensusStateNotFound,
			"clientID (%s), height (%s)", connection.GetClientID(), height,
		)
	}
	return consensusState.GetTimestamp(), nil
}
