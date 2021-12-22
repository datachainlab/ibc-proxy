package keeper

import (
	"bytes"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/dbadapter"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/exported"
	multivtypes "github.com/datachainlab/ibc-proxy/modules/light-clients/xx-multiv/types"
	proxytypes "github.com/datachainlab/ibc-proxy/modules/light-clients/xx-proxy/types"
	dbm "github.com/tendermint/tm-db"
)

// CONTRACT: upstream is A, downstream is B, we are proxy(P)
func (k Keeper) ConnOpenTry(
	ctx sdk.Context,

	connectionID string, // the connection ID corresponding to B on A
	upstreamPrefix exported.Prefix, // store prefix on upstream chain
	connection connectiontypes.ConnectionEnd, // the connection corresponding to B on A (its state must be INIT)

	downstreamClientState exported.ClientState, // clientState for chainB
	downstreamConsensusState exported.ConsensusState, // consensusState for chainB
	proxyClientState exported.ClientState, // clientState for proxy
	proxyConsensusState exported.ConsensusState, // consensusState for proxy

	proofInit []byte, // proof that chainA stored connectionEnd in state (on ConnOpenInit)
	proofClient []byte, // proof that chainA stored a light client of chainB
	proofConsensus []byte, // proof that chainA stored chainB's consensus state at consensus height
	proofHeight exported.Height, // height at which relayer constructs proof of A storing connectionEnd in state
	consensusHeight exported.Height, // latest height of chain B which chain A has stored in its chain B client
	proofProxyClient []byte,
	proofProxyConsensus []byte,
	proofProxyHeight exported.Height,
	proxyConsensusHeight exported.Height,
) error {

	proxyClientState2, ok := proxyClientState.(*proxytypes.ClientState)
	if !ok {
		return fmt.Errorf("clientType mismatch %v", proxyClientState.ClientType())
	}
	if err := k.ValidateSelfClient(ctx, proxyClientState2); err != nil {
		return err
	}
	upstreamClientID := proxyClientState2.UpstreamClientId

	_, found := k.GetProxyConnection(ctx, upstreamPrefix, upstreamClientID, connectionID)
	if found {
		return fmt.Errorf("connection '%v:%v' already exists", upstreamClientID, connectionID)
	}

	if connection.State != connectiontypes.INIT {
		return fmt.Errorf("connection state must be %s", connectiontypes.INIT)
	}

	// Check that ChainA stored the clientState provided in the msg
	if err := k.VerifyAndProxyClientState(ctx, upstreamClientID, upstreamPrefix, connection.GetClientID(), proofHeight, proofClient, downstreamClientState); err != nil {
		return err
	}

	// Check that ChainA stored the correct ConsensusState of chainB or proxy at the given consensusHeight
	if err := k.VerifyAndProxyClientConsensusState(
		ctx, upstreamClientID, upstreamPrefix, connection.GetClientID(), proofHeight, consensusHeight, proofConsensus, downstreamConsensusState,
	); err != nil {
		return err
	}

	if dcs, ok := downstreamClientState.(*multivtypes.ClientState); ok {
		downstreamClientState = dcs.GetUnderlyingClientState()
	}

	store := makeMemStore(k.cdc, downstreamConsensusState, proofProxyHeight)

	if err := downstreamClientState.VerifyClientState(
		store, k.cdc, proofProxyHeight, connection.Counterparty.GetPrefix(), connection.Counterparty.ClientId, proofProxyClient, proxyClientState,
	); err != nil {
		return err
	}

	if err := downstreamClientState.VerifyClientConsensusState(
		store, k.cdc, proofProxyHeight, connection.Counterparty.ClientId, proxyConsensusHeight, connection.Counterparty.GetPrefix(), proofProxyConsensus, proxyConsensusState,
	); err != nil {
		return err
	}

	// Ensure that ChainA stored expected connectionEnd in its state during ConnOpenTry
	if err := k.VerifyAndProxyConnectionState(
		ctx, upstreamClientID, upstreamPrefix, connection, proofHeight, proofInit, connectionID,
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
	upstreamPrefix exported.Prefix,
	connection connectiontypes.ConnectionEnd, // the connection corresponding to A on B (its state must be TRYOPEN)
	downstreamClientState exported.ClientState, // clientState for chainA
	downstreamConsensusState exported.ConsensusState, // consensusState for chainA
	proxyClientState exported.ClientState, // clientState for proxy
	proxyConsensusState exported.ConsensusState, // consensusState for proxy
	proofTry []byte, // proof that connectionEnd was added to ChainB state in ConnOpenTry
	proofClient []byte, // proof of client state on chainB for chainA
	proofConsensus []byte, // proof that chainB has stored ConsensusState of chainA on its client
	proofHeight exported.Height, // height that relayer constructed proofTry
	consensusHeight exported.Height, // latest height of chainA that chainB has stored on its chainA client
	proofProxyClient []byte,
	proofProxyConsensus []byte,
	proofProxyHeight exported.Height,
	proxyConsensusHeight exported.Height,
) error {
	proxyClientState2, ok := proxyClientState.(*proxytypes.ClientState)
	if !ok {
		return fmt.Errorf("clientType mismatch %v", proxyClientState.ClientType())
	}
	if err := k.ValidateSelfClient(ctx, proxyClientState2); err != nil {
		return err
	}
	upstreamClientID := proxyClientState2.UpstreamClientId

	_, found := k.GetProxyConnection(ctx, upstreamPrefix, upstreamClientID, connectionID)
	if found {
		return fmt.Errorf("connection '%v:%v' already exists", upstreamClientID, connectionID)
	}

	if connection.State != connectiontypes.TRYOPEN {
		return fmt.Errorf("connection state must be %s", connectiontypes.TRYOPEN)
	}

	// Check that ChainB stored the clientState provided in the msg
	if err := k.VerifyAndProxyClientState(ctx, upstreamClientID, upstreamPrefix, connection.GetClientID(), proofHeight, proofClient, downstreamClientState); err != nil {
		return err
	}

	// Ensure that ChainB has stored the correct ConsensusState for chainA at the consensusHeight
	if err := k.VerifyAndProxyClientConsensusState(
		ctx, upstreamClientID, upstreamPrefix, connection.GetClientID(), proofHeight, consensusHeight, proofConsensus, downstreamConsensusState,
	); err != nil {
		return err
	}

	if dcs, ok := downstreamClientState.(*multivtypes.ClientState); ok {
		downstreamClientState = dcs.GetUnderlyingClientState()
	}

	store := makeMemStore(k.cdc, downstreamConsensusState, proofProxyHeight)

	if err := downstreamClientState.VerifyClientState(
		store, k.cdc, proofProxyHeight, connection.Counterparty.GetPrefix(), connection.Counterparty.ClientId, proofProxyClient, proxyClientState,
	); err != nil {
		return err
	}

	if err := downstreamClientState.VerifyClientConsensusState(
		store, k.cdc, proofProxyHeight, connection.Counterparty.ClientId, proxyConsensusHeight, connection.Counterparty.GetPrefix(), proofProxyConsensus, proxyConsensusState,
	); err != nil {
		return err
	}

	// Ensure that ChainB stored expected connectionEnd in its state during ConnOpenTry
	if err := k.VerifyAndProxyConnectionState(
		ctx, upstreamClientID, upstreamPrefix, connection, proofHeight, proofTry, connectionID,
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

	connection, found := k.GetProxyConnection(ctx, upstreamPrefix, upstreamClientID, connectionID)
	if !found {
		return fmt.Errorf("connection '%v:%v' not found", upstreamClientID, connectionID)
	}

	if connection.State != connectiontypes.INIT {
		return fmt.Errorf("connection state must be %s", connectiontypes.INIT)
	}

	connection.State = connectiontypes.OPEN
	connection.Counterparty.ConnectionId = counterpartyConnectionID

	// Ensure that ChainA stored expected connectionEnd in its state during ConnOpenTry
	if err := k.VerifyAndProxyConnectionState(
		ctx, upstreamClientID, upstreamPrefix, connection, proofHeight, proofAck, connectionID,
	); err != nil {
		return err
	}

	return nil
}

func (k Keeper) ValidateSelfClient(ctx sdk.Context, clientState *proxytypes.ClientState) error {
	if !bytes.Equal(k.GetIBCCommitmentPrefix().(*commitmenttypes.MerklePrefix).Bytes(), clientState.IbcPrefix.Bytes()) {
		return fmt.Errorf("IBC commitment prefix mismatch: %X != %X", k.GetIBCCommitmentPrefix().(*commitmenttypes.MerklePrefix).Bytes(), clientState.IbcPrefix.Bytes())
	}
	if !bytes.Equal(k.GetProxyCommitmentPrefix().(*commitmenttypes.MerklePrefix).Bytes(), clientState.ProxyPrefix.Bytes()) {
		return fmt.Errorf("Proxy commitment prefix mismatch: %X != %X", k.GetProxyCommitmentPrefix().(*commitmenttypes.MerklePrefix).Bytes(), clientState.ProxyPrefix.Bytes())
	}
	return k.clientKeeper.ValidateSelfClient(ctx, clientState.GetProxyClientState())
}

func makeMemStore(cdc codec.BinaryCodec, consensusState exported.ConsensusState, proofHeight exported.Height) dbadapter.Store {
	consensusStateBytes, err := clienttypes.MarshalConsensusState(cdc, consensusState)
	if err != nil {
		panic(err)
	}
	store := dbadapter.Store{DB: dbm.NewMemDB()}
	store.Set(host.ConsensusStateKey(proofHeight), consensusStateBytes)
	return store
}
