package types

import (
	"errors"
	"fmt"

	ics23 "github.com/confio/ics23/go"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store/dbadapter"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/exported"
	proxytypes "github.com/datachainlab/ibc-proxy/modules/light-clients/xx-proxy/types"
	dbm "github.com/tendermint/tm-db"
)

var _ exported.ClientState = (*ClientState)(nil)
var _ codectypes.UnpackInterfacesMessage = (*ClientState)(nil)

func NewClientState(clientState exported.ClientState, depth uint32) *ClientState {
	anyClientState, err := clienttypes.PackClientState(clientState)
	if err != nil {
		panic(err)
	}
	return &ClientState{
		UnderlyingClientState: anyClientState,
		Depth:                 depth,
	}
}

func (cs *ClientState) ClientType() string {
	return cs.GetUnderlyingClientState().ClientType()
}

func (cs *ClientState) GetUnderlyingClientState() exported.ClientState {
	state, err := clienttypes.UnpackClientState(cs.UnderlyingClientState)
	if err != nil {
		panic(err)
	}
	return state
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (cs *ClientState) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	return unpacker.UnpackAny(cs.UnderlyingClientState, new(exported.ClientState))
}

func (cs *ClientState) GetLatestHeight() exported.Height {
	return cs.GetUnderlyingClientState().GetLatestHeight()
}

func (cs *ClientState) Status(
	ctx sdk.Context,
	clientStore sdk.KVStore,
	cdc codec.BinaryCodec,
) exported.Status {
	return cs.GetUnderlyingClientState().Status(ctx, clientStore, cdc)
}

func (cs *ClientState) Validate() error {
	if cs.UnderlyingClientState == nil {
		return errors.New("Base cannot be nil")
	}
	return cs.GetUnderlyingClientState().Validate()
}

func (cs *ClientState) GetProofSpecs() []*ics23.ProofSpec {
	return cs.GetUnderlyingClientState().GetProofSpecs()
}

// Initialization function
// Clients must validate the initial consensus state, and may store any client-specific metadata
// necessary for correct light client operation
func (cs *ClientState) Initialize(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, consState exported.ConsensusState) error {
	if cs.UnderlyingClientState == nil {
		return sdkerrors.Wrap(errors.New("invalid clientState"), "the base of a clientState must not be empty")
	} else if consState == nil {
		return sdkerrors.Wrap(errors.New("invalid consensusState"), "consensusState must not be empty")
	}
	return nil
}

// Genesis function
func (cs *ClientState) ExportMetadata(store sdk.KVStore) []exported.GenesisMetadata {
	return cs.GetUnderlyingClientState().ExportMetadata(store)
}

// Upgrade functions
// NOTE: proof heights are not included as upgrade to a new revision is expected to pass only on the last
// height committed by the current revision. Clients are responsible for ensuring that the planned last
// height of the current revision is somehow encoded in the proof verification process.
// This is to ensure that no premature upgrades occur, since upgrade plans committed to by the counterparty
// may be cancelled or modified before the last planned height.
func (cs *ClientState) VerifyUpgradeAndUpdateState(ctx sdk.Context, cdc codec.BinaryCodec, store sdk.KVStore, newClient exported.ClientState, newConsState exported.ConsensusState, proofUpgradeClient []byte, proofUpgradeConsState []byte) (exported.ClientState, exported.ConsensusState, error) {
	return cs.GetUnderlyingClientState().VerifyUpgradeAndUpdateState(ctx, cdc, store, newClient, newConsState, proofUpgradeClient, proofUpgradeConsState)
}

// Utility function that zeroes out any client customizable fields in client state
// Ledger enforced fields are maintained while all custom fields are zero values
// Used to verify upgrades
func (cs ClientState) ZeroCustomFields() exported.ClientState {
	any, err := clienttypes.PackClientState(cs.GetUnderlyingClientState().ZeroCustomFields())
	if err != nil {
		panic(err)
	}
	cs.UnderlyingClientState = any
	return &cs
}

// State verification functions

func (cs *ClientState) VerifyClientState(store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, prefix exported.Prefix, counterpartyClientIdentifier string, proofBytes []byte, clientState exported.ClientState) error {
	proof, err := UnmarshalProof(cdc, proofBytes)
	if err != nil {
		return err
	}
	if err := validateProof(cs, proof); err != nil {
		return err
	}
	head, branches, leaf := proof.Head, proof.Branches, proof.Leaf
	if !head.ProofHeight.EQ(height) {
		return fmt.Errorf("first proof's height must be %v, but got %v", height, proof.Head.ProofHeight)
	}

	/// Verification process ///

	// step1-1. verify proxy client state on c0

	// client for p on c0
	proxyClientState, err := unpackProxyClientState(cdc, head.ClientState)
	if err != nil {
		return err
	}
	if err := cs.GetUnderlyingClientState().VerifyClientState(
		store, cdc, height, prefix, counterpartyClientIdentifier, head.ClientProof, proxyClientState,
	); err != nil {
		return err
	}

	// step1-2. verify proxy consensus state on c0

	proxyConsensusState, err := unpackProxyConsensusState(cdc, head.ConsensusState)
	if err != nil {
		return err
	}
	if err := cs.GetUnderlyingClientState().VerifyClientConsensusState(
		store, cdc, height, counterpartyClientIdentifier, head.ConsensusHeight, prefix, head.ConsensusProof, proxyConsensusState,
	); err != nil {
		return err
	}

	// step2. verify state with nodes

	for _, branch := range branches {
		targetClientState, err := unpackProxyClientState(cdc, branch.ClientState)
		if err != nil {
			return err
		}
		targetConsensusState, err := unpackProxyConsensusState(cdc, branch.ConsensusState)
		if err != nil {
			return err
		}
		store := makeMemStore(cdc, proxyConsensusState, branch.ProofHeight)

		if err := proxyClientState.IBCVerifyClientState(
			store, cdc, branch.ProofHeight, proxyClientState.IbcPrefix, proxyClientState.UpstreamClientId, branch.ClientProof, targetClientState,
		); err != nil {
			return err
		}
		if err := proxyClientState.IBCVerifyClientConsensusState(
			store, cdc, branch.ProofHeight, proxyClientState.UpstreamClientId, branch.ConsensusHeight, proxyClientState.IbcPrefix, branch.ConsensusProof, targetConsensusState,
		); err != nil {
			return err
		}

		proxyClientState = targetClientState
		proxyConsensusState = targetConsensusState
	}

	// step3. verify existence of target client state on proxy
	store = makeMemStore(cdc, proxyConsensusState, leaf.ProofHeight)
	if err := proxyClientState.IBCVerifyClientState(
		store, cdc, leaf.ProofHeight, proxyClientState.IbcPrefix, proxyClientState.UpstreamClientId, leaf.Proof, clientState,
	); err != nil {
		return err
	}

	return nil
}

// the type of proof must be Any
func (cs *ClientState) VerifyClientConsensusState(store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, counterpartyClientIdentifier string, consensusHeight exported.Height, prefix exported.Prefix, proofBytes []byte, consensusState exported.ConsensusState) error {
	proof, err := UnmarshalProof(cdc, proofBytes)
	if err != nil {
		return err
	}
	if err := validateProof(cs, proof); err != nil {
		return err
	}
	head, branches, leaf := proof.Head, proof.Branches, proof.Leaf
	if !head.ProofHeight.EQ(height) {
		return fmt.Errorf("first proof's height must be %v, but got %v", height, head.ProofHeight)
	}

	/// Verification process ///

	// step1-1. verify proxy client state on c0

	// client for p on c0
	proxyClientState, err := unpackProxyClientState(cdc, head.ClientState)
	if err != nil {
		return err
	}
	if err := cs.GetUnderlyingClientState().VerifyClientState(
		store, cdc, height, prefix, counterpartyClientIdentifier, head.ClientProof, proxyClientState,
	); err != nil {
		return err
	}

	// step1-2. verify proxy consensus state on c0

	proxyConsensusState, err := unpackProxyConsensusState(cdc, head.ConsensusState)
	if err != nil {
		return err
	}
	if err := cs.GetUnderlyingClientState().VerifyClientConsensusState(
		store, cdc, height, counterpartyClientIdentifier, head.ConsensusHeight, prefix, head.ConsensusProof, proxyConsensusState,
	); err != nil {
		return err
	}

	// step2. verify state with nodes

	for _, branch := range branches {
		targetClientState, err := unpackProxyClientState(cdc, branch.ClientState)
		if err != nil {
			return err
		}
		targetConsensusState, err := unpackProxyConsensusState(cdc, branch.ConsensusState)
		if err != nil {
			return err
		}

		store := makeMemStore(cdc, proxyConsensusState, branch.ProofHeight)

		if err := proxyClientState.IBCVerifyClientState(
			store, cdc, branch.ProofHeight, proxyClientState.IbcPrefix, proxyClientState.UpstreamClientId, branch.ClientProof, targetClientState,
		); err != nil {
			return err
		}
		if err := proxyClientState.IBCVerifyClientConsensusState(
			store, cdc, branch.ProofHeight, proxyClientState.UpstreamClientId, branch.ConsensusHeight, proxyClientState.IbcPrefix, branch.ConsensusProof, targetConsensusState,
		); err != nil {
			return err
		}

		proxyClientState = targetClientState
		proxyConsensusState = targetConsensusState
	}

	// step3. verify existence of target client state on proxy
	store = makeMemStore(cdc, proxyConsensusState, leaf.ProofHeight)
	if err := proxyClientState.IBCVerifyClientConsensusState(
		store, cdc, leaf.ProofHeight, proxyClientState.UpstreamClientId, consensusHeight, proxyClientState.IbcPrefix, leaf.Proof, consensusState,
	); err != nil {
		return err
	}

	return nil
}

func (cs *ClientState) VerifyConnectionState(store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, prefix exported.Prefix, proof []byte, connectionID string, connectionEnd exported.ConnectionI) error {
	return cs.GetUnderlyingClientState().VerifyConnectionState(store, cdc, height, prefix, proof, connectionID, connectionEnd)
}

func (cs *ClientState) VerifyChannelState(store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, prefix exported.Prefix, proof []byte, portID string, channelID string, channel exported.ChannelI) error {
	return cs.GetUnderlyingClientState().VerifyChannelState(store, cdc, height, prefix, proof, portID, channelID, channel)
}

func (cs *ClientState) VerifyPacketCommitment(ctx sdk.Context, store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, currentTimestamp uint64, delayPeriod uint64, prefix exported.Prefix, proof []byte, portID string, channelID string, sequence uint64, commitmentBytes []byte) error {
	return cs.GetUnderlyingClientState().VerifyPacketCommitment(ctx, store, cdc, height, currentTimestamp, delayPeriod, prefix, proof, portID, channelID, sequence, commitmentBytes)
}

func (cs *ClientState) VerifyPacketAcknowledgement(ctx sdk.Context, store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, currentTimestamp uint64, delayPeriod uint64, prefix exported.Prefix, proof []byte, portID string, channelID string, sequence uint64, acknowledgement []byte) error {
	return cs.GetUnderlyingClientState().VerifyPacketAcknowledgement(ctx, store, cdc, height, currentTimestamp, delayPeriod, prefix, proof, portID, channelID, sequence, acknowledgement)
}

func (cs *ClientState) VerifyPacketReceiptAbsence(ctx sdk.Context, store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, currentTimestamp uint64, delayPeriod uint64, prefix exported.Prefix, proof []byte, portID string, channelID string, sequence uint64) error {
	return cs.GetUnderlyingClientState().VerifyPacketReceiptAbsence(ctx, store, cdc, height, currentTimestamp, delayPeriod, prefix, proof, portID, channelID, sequence)
}

func (cs *ClientState) VerifyNextSequenceRecv(ctx sdk.Context, store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, currentTimestamp uint64, delayPeriod uint64, prefix exported.Prefix, proof []byte, portID string, channelID string, nextSequenceRecv uint64) error {
	return cs.GetUnderlyingClientState().VerifyNextSequenceRecv(ctx, store, cdc, height, currentTimestamp, delayPeriod, prefix, proof, portID, channelID, nextSequenceRecv)
}

func validateProof(cs *ClientState, proof *MultiProof) error {
	if l := len(proof.Branches); l != int(cs.Depth) {
		return fmt.Errorf("invalid branches length: expected=%v got=%v", cs.Depth, len(proof.Branches))
	}
	return nil
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

func unpackProxyClientState(cdc codec.BinaryCodec, anyClientState *types.Any) (*proxytypes.ClientState, error) {
	var clientState exported.ClientState
	if err := cdc.UnpackAny(anyClientState, &clientState); err != nil {
		return nil, err
	}
	s, ok := clientState.(*proxytypes.ClientState)
	if !ok {
		return nil, fmt.Errorf("the type of proxyClientState must be %T", &proxytypes.ClientState{})
	}
	return s, nil
}

func unpackProxyConsensusState(cdc codec.BinaryCodec, anyConsensusState *types.Any) (*proxytypes.ConsensusState, error) {
	var consensusState exported.ConsensusState
	if err := cdc.UnpackAny(anyConsensusState, &consensusState); err != nil {
		return nil, err
	}
	s, ok := consensusState.(*proxytypes.ConsensusState)
	if !ok {
		return nil, fmt.Errorf("the type of proxyConsensusState must be %T", &proxytypes.ConsensusState{})
	}
	return s, nil
}
