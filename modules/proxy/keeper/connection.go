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
	proxyConnection connectiontypes.ConnectionEnd, // the connection corresponding to B on A

	clientState exported.ClientState, // clientState that chainA has for chainB or proxy
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

	if proxyConnection.State != connectiontypes.INIT {
		return fmt.Errorf("connection state must be %s", connectiontypes.INIT)
	}

	// Check that ChainA stored the clientState provided in the msg
	if err := k.VerifyClientState(ctx, upstreamClientID, proxyConnection, proofHeight, proofClient, clientState); err != nil {
		return err
	}

	// Ensure that ChainB stored expected connectionEnd in its state during ConnOpenTry
	if err := k.VerifyConnectionState(
		ctx, upstreamClientID, proxyConnection, proofHeight, proofInit, connectionID,
	); err != nil {
		return err
	}

	// Check that ChainA stored the correct ConsensusState of chainB or proxy at the given consensusHeight
	if err := k.VerifyClientConsensusState(
		ctx, upstreamClientID, proxyConnection, proofHeight, consensusHeight, proofConsensus, expectedConsensusState,
	); err != nil {
		return err
	}

	return nil
}

// caller: A
// CONTRACT: upstream is B, downstream is A, we are proxy(P)
func (k Keeper) ConnOpenACK(
	ctx sdk.Context,
	connectionID string, // connectionID corresponding to B on A
	upstreamClientID string, // clientID corresponding to B on P
	proxyConnection connectiontypes.ConnectionEnd, // the connection corresponding to A on B
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

	if proxyConnection.State != connectiontypes.TRYOPEN {
		return fmt.Errorf("connection state must be %s", connectiontypes.TRYOPEN)
	}

	// Check that ChainB stored the clientState provided in the msg
	if err := k.VerifyClientState(ctx, upstreamClientID, proxyConnection, proofHeight, proofClient, clientState); err != nil {
		return err
	}

	// Ensure that ChainB stored expected connectionEnd in its state during ConnOpenTry
	if err := k.VerifyConnectionState(
		ctx, upstreamClientID, proxyConnection, proofHeight, proofTry, connectionID,
	); err != nil {
		return err
	}

	// Ensure that ChainB has stored the correct ConsensusState for chainA at the consensusHeight
	if err := k.VerifyClientConsensusState(
		ctx, upstreamClientID, proxyConnection, proofHeight, consensusHeight, proofConsensus, expectedConsensusState,
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
	proxyConnection connectiontypes.ConnectionEnd, // the connection corresponding to A on B
	proofAck []byte, // proof that connection opened on ChainA during ConnOpenAck
	proofHeight exported.Height, // height that relayer constructed proofAck
) error {
	if !k.IsProxyEnabled(ctx, upstreamClientID) {
		return fmt.Errorf("clientID '%v' doesn't have proxy enabled", upstreamClientID)
	}

	if proxyConnection.State != connectiontypes.OPEN {
		return fmt.Errorf("connection state must be %s", connectiontypes.OPEN)
	}

	// Ensure that ChainB stored expected connectionEnd in its state during ConnOpenTry
	if err := k.VerifyConnectionState(
		ctx, upstreamClientID, proxyConnection, proofHeight, proofAck, connectionID,
	); err != nil {
		return err
	}

	return nil
}
