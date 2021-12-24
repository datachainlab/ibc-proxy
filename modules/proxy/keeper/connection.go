package keeper

import (
	"bytes"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/dbadapter"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/exported"
	dbm "github.com/tendermint/tm-db"

	multivtypes "github.com/datachainlab/ibc-proxy/modules/light-clients/xx-multiv/types"
	proxytypes "github.com/datachainlab/ibc-proxy/modules/light-clients/xx-proxy/types"
)

// upstream: chainA, downstream: chainB
func (k Keeper) ConnOpenTry(
	ctx sdk.Context,

	connectionID string, // the connection ID corresponding to chainB on chainA
	upstreamPrefix exported.Prefix, // store prefix on chainA
	connection connectiontypes.ConnectionEnd, // the connection corresponding to chainB on chainA (its state must be INIT)

	downstreamClientState exported.ClientState, // clientState for chainB
	downstreamConsensusState exported.ConsensusState, // consensusState for chainB
	proxyClientState exported.ClientState, // clientState for proxy

	proofInit []byte, // proof that chainA stored connectionEnd in state (on ConnOpenInit)
	proofClient []byte, // proof that chainA stored chainB's clientState
	proofConsensus []byte, // proof that chainA stored chainB's consensus state at consensus height
	proofHeight exported.Height, // height at which relayer constructs proof of chainA storing connectionEnd in state
	consensusHeight exported.Height, // latest height of chain B which chain A has stored in its chain B client

	proofProxyClient []byte, // proof that chainB stored proxy's client state at proofProxyHeight
	proofProxyConsensus []byte, // proof that chainB stored proxy's consensus state at proxyConsensusHeight
	proofProxyHeight exported.Height, // height at which relayer constructs proof that chainB tracks proxy's client state
	proxyConsensusHeight exported.Height, // height at which relayer constructs proof that chainB tracks proxy's consensus state
) error {

	proxyClientState, proxyConsensusState, upstreamClientID, err := k.produceVerificationArgs(ctx, proxyClientState, proxyConsensusHeight)
	if err != nil {
		return err
	}

	_, found := k.GetProxyConnection(ctx, upstreamPrefix, upstreamClientID, connectionID)
	if found {
		return fmt.Errorf("connection '%v:%v' already exists", upstreamClientID, connectionID)
	}

	if connection.State != connectiontypes.INIT {
		return fmt.Errorf("connection state must be %s", connectiontypes.INIT)
	}

	// Ensure that chainA stored the clientState provided in the msg
	if err := k.VerifyAndProxyClientState(ctx, upstreamClientID, upstreamPrefix, connection.GetClientID(), proofHeight, proofClient, downstreamClientState); err != nil {
		return err
	}

	// Ensure that chainA stored the correct ConsensusState of chainB or proxy at the given consensusHeight
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

	// Ensure that chainA stored expected connectionEnd in its state during ConnOpenInit
	if err := k.VerifyAndProxyConnectionState(
		ctx, upstreamClientID, upstreamPrefix, connection, proofHeight, proofInit, connectionID,
	); err != nil {
		return err
	}

	return nil
}

// upstream: chainA, downstream: chainB
func (k Keeper) ConnOpenAck(
	ctx sdk.Context,

	connectionID string, // the connection ID corresponding to chainB on chainA
	upstreamPrefix exported.Prefix, // store prefix on chainA
	connectionEnd connectiontypes.ConnectionEnd, // the connection corresponding to chainB on chainA (its state must be TRYOPEN)

	downstreamClientState exported.ClientState, // clientState for chainB
	downstreamConsensusState exported.ConsensusState, // consensusState for chainB
	proxyClientState exported.ClientState, // clientState for proxy

	proofTry []byte, // proof that chainA stored connectionEnd in state (on ConnOpenTry)
	proofClient []byte, // proof that chainA stored chainB's clientState
	proofConsensus []byte, // proof that chainA stored chainB's consensus state at consensus height
	proofHeight exported.Height, // height at which relayer constructs proof of chainA storing connectionEnd in state
	consensusHeight exported.Height, // latest height of chain B which chain A has stored in its chain B client

	proofProxyClient []byte, // proof that chainB stored proxy's client state at proofProxyHeight
	proofProxyConsensus []byte, // proof that chainB stored proxy's consensus state at proxyConsensusHeight
	proofProxyHeight exported.Height, // height at which relayer constructs proof that chainB tracks proxy's client state
	proxyConsensusHeight exported.Height, // height at which relayer constructs proof that chainB tracks proxy's consensus state
) error {

	proxyClientState, proxyConsensusState, upstreamClientID, err := k.produceVerificationArgs(ctx, proxyClientState, proxyConsensusHeight)
	if err != nil {
		return err
	}

	_, found := k.GetProxyConnection(ctx, upstreamPrefix, upstreamClientID, connectionID)
	if found {
		return fmt.Errorf("connection '%v:%v' already exists", upstreamClientID, connectionID)
	}

	if connectionEnd.State != connectiontypes.TRYOPEN {
		return sdkerrors.Wrapf(
			connectiontypes.ErrInvalidConnectionState,
			"connection state is not TRYOPEN (got %s)", connectiontypes.State(connectionEnd.GetState()).String(),
		)
	}

	// Ensure that chainA stored the clientState provided in the msg
	if err := k.VerifyAndProxyClientState(ctx, upstreamClientID, upstreamPrefix, connectionEnd.GetClientID(), proofHeight, proofClient, downstreamClientState); err != nil {
		return err
	}

	// Ensure that chainA has stored the correct ConsensusState for chainA at the consensusHeight
	if err := k.VerifyAndProxyClientConsensusState(
		ctx, upstreamClientID, upstreamPrefix, connectionEnd.GetClientID(), proofHeight, consensusHeight, proofConsensus, downstreamConsensusState,
	); err != nil {
		return err
	}

	if dcs, ok := downstreamClientState.(*multivtypes.ClientState); ok {
		downstreamClientState = dcs.GetUnderlyingClientState()
	}

	store := makeMemStore(k.cdc, downstreamConsensusState, proofProxyHeight)

	if err := downstreamClientState.VerifyClientState(
		store, k.cdc, proofProxyHeight, connectionEnd.Counterparty.GetPrefix(), connectionEnd.Counterparty.ClientId, proofProxyClient, proxyClientState,
	); err != nil {
		return err
	}

	if err := downstreamClientState.VerifyClientConsensusState(
		store, k.cdc, proofProxyHeight, connectionEnd.Counterparty.ClientId, proxyConsensusHeight, connectionEnd.Counterparty.GetPrefix(), proofProxyConsensus, proxyConsensusState,
	); err != nil {
		return err
	}

	// Ensure that chainB stored expected connectionEnd in its state during ConnOpenTry
	if err := k.VerifyAndProxyConnectionState(
		ctx, upstreamClientID, upstreamPrefix, connectionEnd, proofHeight, proofTry, connectionID,
	); err != nil {
		return err
	}

	return nil
}

// upstream: chainA, downstream: chainB
func (k Keeper) ConnOpenConfirm(
	ctx sdk.Context,

	connectionID string, // the connection ID corresponding to chainB on chainA
	upstreamClientID string, // the client ID corresponding to light client for chainA on chainB
	upstreamPrefix exported.Prefix, // store prefix on chainA
	counterpartyConnectionID string, // the connection ID corresponding to chainA on chainB

	proofAck []byte, // proof that connection opened on chainA during ConnOpenAck
	proofHeight exported.Height, // height that relayer constructed proofAck
) error {

	connectionEnd, found := k.GetProxyConnection(ctx, upstreamPrefix, upstreamClientID, connectionID)
	if !found {
		return sdkerrors.Wrapf(
			connectiontypes.ErrConnectionNotFound,
			"connection '%#v:%v:%v' not found", upstreamPrefix, upstreamClientID, connectionID,
		)
	}

	if connectionEnd.State != connectiontypes.INIT {
		return sdkerrors.Wrapf(
			connectiontypes.ErrInvalidConnectionState,
			"connection state is not INIT (got %s)", connectiontypes.State(connectionEnd.GetState()).String(),
		)
	}

	connectionEnd.State = connectiontypes.OPEN
	connectionEnd.Counterparty.ConnectionId = counterpartyConnectionID

	// Ensure that chainA stored expected connectionEnd in its state during ConnOpenAck
	if err := k.VerifyAndProxyConnectionState(
		ctx, upstreamClientID, upstreamPrefix, connectionEnd, proofHeight, proofAck, connectionID,
	); err != nil {
		return err
	}

	return nil
}

// upstream: chainA, downstream: chainB
func (k Keeper) ConnOpenFinalize(
	ctx sdk.Context,

	connectionID string, // the connection ID corresponding to chainB on chainA
	upstreamClientID string, // the client ID corresponding to light client for chainA on chainB
	upstreamPrefix exported.Prefix, // store prefix on chainA

	proofConfirm []byte, // proof that connection opened on chainA during ConnOpenConfirm
	proofHeight exported.Height, // height that relayer constructed proofConfirm
) error {

	connectionEnd, found := k.GetProxyConnection(ctx, upstreamPrefix, upstreamClientID, connectionID)
	if !found {
		return sdkerrors.Wrapf(
			connectiontypes.ErrConnectionNotFound,
			"connection '%#v:%v:%v' not found", upstreamPrefix, upstreamClientID, connectionID,
		)
	}

	if connectionEnd.State != connectiontypes.TRYOPEN {
		return sdkerrors.Wrapf(
			connectiontypes.ErrInvalidConnectionState,
			"connection state is not TRYOPEN (got %s)", connectiontypes.State(connectionEnd.GetState()).String(),
		)
	}

	connectionEnd.State = connectiontypes.OPEN

	// Ensure that chainA stored expected connectionEnd in its state during ConnOpenConfirm
	if err := k.VerifyAndProxyConnectionState(
		ctx, upstreamClientID, upstreamPrefix, connectionEnd, proofHeight, proofConfirm, connectionID,
	); err != nil {
		return err
	}

	return nil
}

func (k Keeper) produceVerificationArgs(ctx sdk.Context, proxyClientState exported.ClientState, proxyConsensusHeight exported.Height) (*proxytypes.ClientState, *proxytypes.ConsensusState, string, error) {
	clientState, ok := proxyClientState.(*proxytypes.ClientState)
	if !ok {
		return nil, nil, "", fmt.Errorf("invalid client type '%v'", proxyClientState.ClientType())
	}
	if err := k.validateSelfClient(ctx, clientState); err != nil {
		return nil, nil, "", err
	}
	consensusState, err := k.getSelfConsensusState(ctx, proxyConsensusHeight)
	if err != nil {
		return nil, nil, "", err
	}
	return clientState, consensusState, clientState.UpstreamClientId, nil
}

func (k Keeper) validateSelfClient(ctx sdk.Context, clientState *proxytypes.ClientState) error {
	if !bytes.Equal(k.GetIBCCommitmentPrefix().(*commitmenttypes.MerklePrefix).Bytes(), clientState.IbcPrefix.Bytes()) {
		return fmt.Errorf("IBC commitment prefix mismatch: %X != %X", k.GetIBCCommitmentPrefix().(*commitmenttypes.MerklePrefix).Bytes(), clientState.IbcPrefix.Bytes())
	}
	if !bytes.Equal(k.GetProxyCommitmentPrefix().(*commitmenttypes.MerklePrefix).Bytes(), clientState.ProxyPrefix.Bytes()) {
		return fmt.Errorf("Proxy commitment prefix mismatch: %X != %X", k.GetProxyCommitmentPrefix().(*commitmenttypes.MerklePrefix).Bytes(), clientState.ProxyPrefix.Bytes())
	}
	return k.clientKeeper.ValidateSelfClient(ctx, clientState.GetProxyClientState())
}

func (k Keeper) getSelfConsensusState(ctx sdk.Context, consensusHeight exported.Height) (*proxytypes.ConsensusState, error) {
	selfConsensusState, ok := k.clientKeeper.GetSelfConsensusState(ctx, consensusHeight)
	if !ok {
		return nil, fmt.Errorf("self consensus state not found: height=%v", consensusHeight)
	}
	anyConsensusState, err := clienttypes.PackConsensusState(selfConsensusState)
	if err != nil {
		return nil, err
	}
	return proxytypes.NewConsensusState(anyConsensusState), nil
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
