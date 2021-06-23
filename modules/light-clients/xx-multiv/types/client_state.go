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

func NewClientState(base *codectypes.Any) *ClientState {
	return &ClientState{
		Base: base,
	}
}

func (cs *ClientState) ClientType() string {
	return cs.GetBaseClientState().ClientType()
}

func (cs *ClientState) GetBaseClientState() exported.ClientState {
	state, err := clienttypes.UnpackClientState(cs.Base)
	if err != nil {
		panic(err)
	}
	return state
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (cs *ClientState) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	return unpacker.UnpackAny(cs.Base, new(exported.ClientState))
}

func (cs *ClientState) GetLatestHeight() exported.Height {
	return cs.GetBaseClientState().GetLatestHeight()
}

func (cs *ClientState) Status(
	ctx sdk.Context,
	clientStore sdk.KVStore,
	cdc codec.BinaryCodec,
) exported.Status {
	return cs.GetBaseClientState().Status(ctx, clientStore, cdc)
}

func (cs *ClientState) Validate() error {
	if cs.Base == nil {
		return errors.New("Base cannot be nil")
	}
	return cs.GetBaseClientState().Validate()
}

func (cs *ClientState) GetProofSpecs() []*ics23.ProofSpec {
	return cs.GetBaseClientState().GetProofSpecs()
}

// Initialization function
// Clients must validate the initial consensus state, and may store any client-specific metadata
// necessary for correct light client operation
func (cs *ClientState) Initialize(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, consState exported.ConsensusState) error {
	if cs.Base == nil {
		return sdkerrors.Wrap(errors.New("invalid clientState"), "the base of a clientState must not be empty")
	} else if consState == nil {
		return sdkerrors.Wrap(errors.New("invalid consensusState"), "consensusState must not be empty")
	}
	return nil
}

// Genesis function
func (cs *ClientState) ExportMetadata(store sdk.KVStore) []exported.GenesisMetadata {
	return cs.GetBaseClientState().ExportMetadata(store)
}

// Upgrade functions
// NOTE: proof heights are not included as upgrade to a new revision is expected to pass only on the last
// height committed by the current revision. Clients are responsible for ensuring that the planned last
// height of the current revision is somehow encoded in the proof verification process.
// This is to ensure that no premature upgrades occur, since upgrade plans committed to by the counterparty
// may be cancelled or modified before the last planned height.
func (cs *ClientState) VerifyUpgradeAndUpdateState(ctx sdk.Context, cdc codec.BinaryCodec, store sdk.KVStore, newClient exported.ClientState, newConsState exported.ConsensusState, proofUpgradeClient []byte, proofUpgradeConsState []byte) (exported.ClientState, exported.ConsensusState, error) {
	return cs.GetBaseClientState().VerifyUpgradeAndUpdateState(ctx, cdc, store, newClient, newConsState, proofUpgradeClient, proofUpgradeConsState)
}

// Utility function that zeroes out any client customizable fields in client state
// Ledger enforced fields are maintained while all custom fields are zero values
// Used to verify upgrades
func (cs *ClientState) ZeroCustomFields() exported.ClientState {
	any, err := clienttypes.PackClientState(cs.GetBaseClientState().ZeroCustomFields())
	if err != nil {
		panic(err)
	}
	return &ClientState{
		Base: any,
	}
}

// State verification functions

func (cs *ClientState) VerifyClientState(store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, prefix exported.Prefix, counterpartyClientIdentifier string, proofBytes []byte, clientState exported.ClientState) error {
	proof, err := UnmarshalProof(cdc, proofBytes)
	if err != nil {
		// XXX if got unexpected proof format, fallback to the base client
		return cs.GetBaseClientState().VerifyClientState(store, cdc, height, prefix, counterpartyClientIdentifier, proofBytes, clientState)
	}

	switch proof.(type) {
	case *MultiProof:
	default:
		return cs.GetBaseClientState().VerifyClientState(store, cdc, height, prefix, counterpartyClientIdentifier, proofBytes, clientState)
	}
	mproof := proof.(*MultiProof)

	if l := len(mproof.Proofs); l < 2 {
		return fmt.Errorf("unexpected proofs length: %v", l)
	}

	h, ok := mproof.Proofs[0].Proof.(*Proof_Branch)
	if !ok {
		return fmt.Errorf("first element must be %v, but got %v", &Proof_Branch{}, mproof.Proofs[0].Proof)
	}
	head := h.Branch
	if !head.ProofHeight.EQ(height) {
		return fmt.Errorf("first proof's height must be %v, but got %v", height, head.ProofHeight)
	}

	l, ok := mproof.Proofs[len(mproof.Proofs)-1].Proof.(*Proof_LeafClient)
	if !ok {
		return fmt.Errorf("last element must be %v, but got %v", &Proof_LeafClient{}, mproof.Proofs[len(mproof.Proofs)-1].Proof)
	}
	leaf := l.LeafClient

	/// Verification process ///

	// step1-1. verify proxy client state on c0

	// client for p on c0
	proxyClientState, err := unpackProxyClientState(cdc, head.ClientState)
	if err != nil {
		return err
	}
	if err := cs.GetBaseClientState().VerifyClientState(
		store, cdc, height, prefix, counterpartyClientIdentifier, head.ClientProof, proxyClientState,
	); err != nil {
		return err
	}

	// step1-2. verify proxy consensus state on c0

	proxyConsensusState, err := unpackProxyConsensusState(cdc, head.ConsensusState)
	if err != nil {
		return err
	}
	if err := cs.GetBaseClientState().VerifyClientConsensusState(
		store, cdc, height, counterpartyClientIdentifier, head.ConsensusHeight, prefix, head.ConsensusProof, proxyConsensusState,
	); err != nil {
		return err
	}

	// step2. verify state with nodes

	for _, p := range mproof.Proofs[1 : len(mproof.Proofs)-1] {
		b, ok := p.Proof.(*Proof_Branch)
		if !ok {
			return fmt.Errorf("unexpected proof type: %T", p.Proof)
		}
		branch := b.Branch

		consensusStateBytes, err := clienttypes.MarshalConsensusState(cdc, proxyConsensusState)
		if err != nil {
			return err
		}
		targetClientState, err := unpackProxyClientState(cdc, branch.ClientState)
		if err != nil {
			return err
		}
		targetConsensusState, err := unpackProxyConsensusState(cdc, branch.ConsensusState)
		if err != nil {
			return err
		}

		// setup store
		mem := dbadapter.Store{DB: dbm.NewMemDB()}
		mem.Set(host.ConsensusStateKey(branch.ProofHeight), consensusStateBytes)

		if err := proxyClientState.IBCVerifyClientState(
			mem, cdc, branch.ProofHeight, proxyClientState.IbcPrefix, proxyClientState.UpstreamClientId, branch.ClientProof, targetClientState,
		); err != nil {
			return err
		}
		if err := proxyClientState.IBCVerifyClientConsensusState(
			mem, cdc, branch.ProofHeight, proxyClientState.UpstreamClientId, branch.ConsensusHeight, proxyClientState.IbcPrefix, branch.ConsensusProof, targetConsensusState,
		); err != nil {
			return err
		}

		proxyClientState = targetClientState
		proxyConsensusState = targetConsensusState
	}

	// step3. verify existence of target client state on proxy

	consensusStateBytes, err := clienttypes.MarshalConsensusState(cdc, proxyConsensusState)
	if err != nil {
		return err
	}
	// setup store
	mem := dbadapter.Store{DB: dbm.NewMemDB()}
	mem.Set(host.ConsensusStateKey(leaf.ProofHeight), consensusStateBytes)

	if err := proxyClientState.IBCVerifyClientState(
		mem, cdc, leaf.ProofHeight, proxyClientState.IbcPrefix, proxyClientState.UpstreamClientId, leaf.Proof, clientState,
	); err != nil {
		return err
	}

	return nil
}

// the type of proof must be Any
func (cs *ClientState) VerifyClientConsensusState(store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, counterpartyClientIdentifier string, consensusHeight exported.Height, prefix exported.Prefix, proofBytes []byte, consensusState exported.ConsensusState) error {
	proof, err := UnmarshalProof(cdc, proofBytes)
	if err != nil {
		// XXX if got unexpected proof format, fallback to the base client
		return cs.GetBaseClientState().VerifyClientConsensusState(store, cdc, height, counterpartyClientIdentifier, consensusHeight, prefix, proofBytes, consensusState)
	}
	switch proof.(type) {
	case *MultiProof:
	default:
		return cs.GetBaseClientState().VerifyClientConsensusState(store, cdc, height, counterpartyClientIdentifier, consensusHeight, prefix, proofBytes, consensusState)
	}
	mproof := proof.(*MultiProof)

	if l := len(mproof.Proofs); l < 2 {
		return fmt.Errorf("unexpected proofs length: %v", l)
	}

	h, ok := mproof.Proofs[0].Proof.(*Proof_Branch)
	if !ok {
		return fmt.Errorf("first element must be %v, but got %v", &Proof_Branch{}, mproof.Proofs[0].Proof)
	}
	head := h.Branch
	if !head.ProofHeight.EQ(height) {
		return fmt.Errorf("first proof's height must be %v, but got %v", height, head.ProofHeight)
	}
	l, ok := mproof.Proofs[len(mproof.Proofs)-1].Proof.(*Proof_LeafConsensus)
	if !ok {
		return fmt.Errorf("last element must be %v, but got %v", &Proof_LeafConsensus{}, mproof.Proofs[len(mproof.Proofs)-1].Proof)
	}
	leaf := l.LeafConsensus

	/// Verification process ///

	// step1-1. verify proxy client state on c0

	// client for p on c0
	proxyClientState, err := unpackProxyClientState(cdc, head.ClientState)
	if err != nil {
		return err
	}
	if err := cs.GetBaseClientState().VerifyClientState(
		store, cdc, height, prefix, counterpartyClientIdentifier, head.ClientProof, proxyClientState,
	); err != nil {
		return err
	}

	// step1-2. verify proxy consensus state on c0

	proxyConsensusState, err := unpackProxyConsensusState(cdc, head.ConsensusState)
	if err != nil {
		return err
	}
	if err := cs.GetBaseClientState().VerifyClientConsensusState(
		store, cdc, height, counterpartyClientIdentifier, head.ConsensusHeight, prefix, head.ConsensusProof, proxyConsensusState,
	); err != nil {
		return err
	}

	// step2. verify state with nodes

	for _, p := range mproof.Proofs[1 : len(mproof.Proofs)-1] {
		b, ok := p.Proof.(*Proof_Branch)
		if !ok {
			return fmt.Errorf("unexpected proof type: %T", p.Proof)
		}
		branch := b.Branch

		consensusStateBytes, err := clienttypes.MarshalConsensusState(cdc, proxyConsensusState)
		if err != nil {
			return err
		}
		targetClientState, err := unpackProxyClientState(cdc, branch.ClientState)
		if err != nil {
			return err
		}
		targetConsensusState, err := unpackProxyConsensusState(cdc, branch.ConsensusState)
		if err != nil {
			return err
		}

		// setup store
		mem := dbadapter.Store{DB: dbm.NewMemDB()}
		mem.Set(host.ConsensusStateKey(branch.ProofHeight), consensusStateBytes)

		if err := proxyClientState.IBCVerifyClientState(
			mem, cdc, branch.ProofHeight, proxyClientState.IbcPrefix, proxyClientState.UpstreamClientId, branch.ClientProof, targetClientState,
		); err != nil {
			return err
		}
		if err := proxyClientState.IBCVerifyClientConsensusState(
			mem, cdc, branch.ProofHeight, proxyClientState.UpstreamClientId, branch.ConsensusHeight, proxyClientState.IbcPrefix, branch.ConsensusProof, targetConsensusState,
		); err != nil {
			return err
		}

		proxyClientState = targetClientState
		proxyConsensusState = targetConsensusState
	}

	// step3. verify existence of target client state on proxy

	consensusStateBytes, err := clienttypes.MarshalConsensusState(cdc, proxyConsensusState)
	if err != nil {
		return err
	}
	// setup store
	mem := dbadapter.Store{DB: dbm.NewMemDB()}
	mem.Set(host.ConsensusStateKey(leaf.ProofHeight), consensusStateBytes)

	if err := proxyClientState.IBCVerifyClientConsensusState(
		mem, cdc, leaf.ProofHeight, proxyClientState.UpstreamClientId, leaf.ConsensusHeight, proxyClientState.IbcPrefix, leaf.Proof, consensusState,
	); err != nil {
		return err
	}

	return nil
}

func (cs *ClientState) VerifyConnectionState(store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, prefix exported.Prefix, proof []byte, connectionID string, connectionEnd exported.ConnectionI) error {
	return cs.GetBaseClientState().VerifyConnectionState(store, cdc, height, prefix, proof, connectionID, connectionEnd)
}

func (cs *ClientState) VerifyChannelState(store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, prefix exported.Prefix, proof []byte, portID string, channelID string, channel exported.ChannelI) error {
	return cs.GetBaseClientState().VerifyChannelState(store, cdc, height, prefix, proof, portID, channelID, channel)
}

func (cs *ClientState) VerifyPacketCommitment(ctx sdk.Context, store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, currentTimestamp uint64, delayPeriod uint64, prefix exported.Prefix, proof []byte, portID string, channelID string, sequence uint64, commitmentBytes []byte) error {
	return cs.GetBaseClientState().VerifyPacketCommitment(ctx, store, cdc, height, currentTimestamp, delayPeriod, prefix, proof, portID, channelID, sequence, commitmentBytes)
}

func (cs *ClientState) VerifyPacketAcknowledgement(ctx sdk.Context, store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, currentTimestamp uint64, delayPeriod uint64, prefix exported.Prefix, proof []byte, portID string, channelID string, sequence uint64, acknowledgement []byte) error {
	return cs.GetBaseClientState().VerifyPacketAcknowledgement(ctx, store, cdc, height, currentTimestamp, delayPeriod, prefix, proof, portID, channelID, sequence, acknowledgement)
}

func (cs *ClientState) VerifyPacketReceiptAbsence(ctx sdk.Context, store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, currentTimestamp uint64, delayPeriod uint64, prefix exported.Prefix, proof []byte, portID string, channelID string, sequence uint64) error {
	return cs.GetBaseClientState().VerifyPacketReceiptAbsence(ctx, store, cdc, height, currentTimestamp, delayPeriod, prefix, proof, portID, channelID, sequence)
}

func (cs *ClientState) VerifyNextSequenceRecv(ctx sdk.Context, store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, currentTimestamp uint64, delayPeriod uint64, prefix exported.Prefix, proof []byte, portID string, channelID string, nextSequenceRecv uint64) error {
	return cs.GetBaseClientState().VerifyNextSequenceRecv(ctx, store, cdc, height, currentTimestamp, delayPeriod, prefix, proof, portID, channelID, nextSequenceRecv)
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
