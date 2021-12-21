package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	connectiontypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

// caller: B
// CONTRACT: upstream is A, downstream is B, we are proxy(P)
func (k Keeper) ConnOpenTry(
	ctx sdk.Context,

	connectionID string, // the connection ID corresponding to B on A
	upstreamClientID string, // the client ID corresponding to A on P
	upstreamPrefix exported.Prefix,
	connection connectiontypes.ConnectionEnd, // the connection corresponding to B on A (its state must be INIT)

	clientState exported.ClientState, // clientState for chainB
	proofInit []byte, // proof that chainA stored connectionEnd in state (on ConnOpenInit)
	proofClient []byte, // proof that chainA stored a light client of chainB
	proofConsensus []byte, // proof that chainA stored chainB's consensus state at consensus height
	proofHeight exported.Height, // height at which relayer constructs proof of A storing connectionEnd in state
	consensusHeight exported.Height, // latest height of chain B which chain A has stored in its chain B client
	expectedConsensusState exported.ConsensusState,
) error {
	if !k.IsProxyEnabled(ctx, upstreamClientID) {
		return fmt.Errorf("clientID '%v' doesn't have proxy enabled", upstreamClientID)
	}

	_, found := k.GetProxyConnection(ctx, upstreamPrefix, upstreamClientID, connectionID)
	if found {
		return fmt.Errorf("connection '%v:%v' already exists", upstreamClientID, connectionID)
	}

	if connection.State != connectiontypes.INIT {
		return fmt.Errorf("connection state must be %s", connectiontypes.INIT)
	}

	// Check that ChainA stored the clientState provided in the msg
	if err := k.VerifyAndProxyClientState(ctx, upstreamClientID, upstreamPrefix, connection.GetClientID(), proofHeight, proofClient, clientState); err != nil {
		return err
	}

	// Ensure that ChainB stored expected connectionEnd in its state during ConnOpenTry
	if err := k.VerifyAndProxyConnectionState(
		ctx, upstreamClientID, upstreamPrefix, connection, proofHeight, proofInit, connectionID,
	); err != nil {
		return err
	}

	// Check that ChainA stored the correct ConsensusState of chainB or proxy at the given consensusHeight
	if err := k.VerifyAndProxyClientConsensusState(
		ctx, upstreamClientID, upstreamPrefix, connection.GetClientID(), proofHeight, consensusHeight, proofConsensus, expectedConsensusState,
	); err != nil {
		return err
	}

	return nil
}

// caller: A
// CONTRACT: upstream is B, downstream is A, we are proxy(P)
func (k Keeper) ConnOpenAck(
	ctx sdk.Context,
	connectionID string, // connectionID corresponding to B on A
	upstreamClientID string, // clientID corresponding to B on P
	upstreamPrefix exported.Prefix,
	connection connectiontypes.ConnectionEnd, // the connection corresponding to A on B (its state must be TRYOPEN)
	clientState exported.ClientState, // client state for chainA on chainB
	version *connectiontypes.Version, // version that ChainB chose in ConnOpenTry
	proofTry []byte, // proof that connectionEnd was added to ChainB state in ConnOpenTry
	proofClient []byte, // proof of client state on chainB for chainA
	proofConsensus []byte, // proof that chainB has stored ConsensusState of chainA on its client
	proofHeight exported.Height, // height that relayer constructed proofTry
	consensusHeight exported.Height, // latest height of chainA that chainB has stored on its chainA client
	expectedConsensusState exported.ConsensusState,
) error {
	if !k.IsProxyEnabled(ctx, upstreamClientID) {
		return fmt.Errorf("clientID '%v' doesn't have proxy enabled", upstreamClientID)
	}

	_, found := k.GetProxyConnection(ctx, upstreamPrefix, upstreamClientID, connectionID)
	if found {
		return fmt.Errorf("connection '%v:%v' already exists", upstreamClientID, connectionID)
	}

	if connection.State != connectiontypes.TRYOPEN {
		return fmt.Errorf("connection state must be %s", connectiontypes.TRYOPEN)
	}

	// Check that ChainB stored the clientState provided in the msg
	if err := k.VerifyAndProxyClientState(ctx, upstreamClientID, upstreamPrefix, connection.GetClientID(), proofHeight, proofClient, clientState); err != nil {
		return err
	}

	// Ensure that ChainB stored expected connectionEnd in its state during ConnOpenTry
	if err := k.VerifyAndProxyConnectionState(
		ctx, upstreamClientID, upstreamPrefix, connection, proofHeight, proofTry, connectionID,
	); err != nil {
		return err
	}

	// Ensure that ChainB has stored the correct ConsensusState for chainA at the consensusHeight
	if err := k.VerifyAndProxyClientConsensusState(
		ctx, upstreamClientID, upstreamPrefix, connection.GetClientID(), proofHeight, consensusHeight, proofConsensus, expectedConsensusState,
	); err != nil {
		return err
	}

	return nil
}

// caller: B
// CONTRACT: upstream is A, downstream is B, we are proxy(P)
func (k Keeper) ConnOpenConfirm(
	ctx sdk.Context,
	connectionID string, // the connection ID corresponding to A on B
	upstreamClientID string, // the client ID corresponding to A
	upstreamPrefix exported.Prefix,
	counterpartyConnectionID string,
	proofAck []byte, // proof that connection opened on ChainA during ConnOpenAck
	proofHeight exported.Height, // height that relayer constructed proofAck
) error {
	if !k.IsProxyEnabled(ctx, upstreamClientID) {
		return fmt.Errorf("clientID '%v' doesn't have proxy enabled", upstreamClientID)
	}

	connection, found := k.GetProxyConnection(ctx, upstreamPrefix, upstreamClientID, connectionID)
	if !found {
		return fmt.Errorf("connection '%v:%v' not found", upstreamClientID, connectionID)
	}

	if connection.State != connectiontypes.INIT {
		return fmt.Errorf("connection state must be %s", connectiontypes.INIT)
	}

	connection.State = connectiontypes.OPEN
	connection.Counterparty.ConnectionId = counterpartyConnectionID

	// Ensure that ChainB stored expected connectionEnd in its state during ConnOpenTry
	if err := k.VerifyAndProxyConnectionState(
		ctx, upstreamClientID, upstreamPrefix, connection, proofHeight, proofAck, connectionID,
	); err != nil {
		return err
	}

	return nil
}
