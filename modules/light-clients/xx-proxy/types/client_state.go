package types

import (
	ics23 "github.com/confio/ics23/go"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

const ProxyClientType string = "proxyclient"

var _ exported.ClientState = (*ClientState)(nil)
var _ codectypes.UnpackInterfacesMessage = (*ClientState)(nil)

func NewClientState(upstreamClientID string) *ClientState {
	return &ClientState{UpstreamClientId: upstreamClientID}
}

func (cs *ClientState) ClientType() string {
	return ProxyClientType
}

func (cs *ClientState) GetProxyClientState() exported.ClientState {
	if cs.ProxyClientState == nil {
		return nil
	}
	clientState, err := clienttypes.UnpackClientState(cs.ProxyClientState)
	if err != nil {
		panic(err)
	}
	return clientState
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (cs *ClientState) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	if err := unpacker.UnpackAny(cs.ProxyClientState, new(exported.ClientState)); err != nil {
		return err
	}
	return nil
}

// GetLatestHeight returns the latest height of the upstream instead of the proxy
func (cs *ClientState) GetLatestHeight() exported.Height {
	return cs.UpstreamHeight
}

func (cs *ClientState) Status(
	ctx sdk.Context,
	clientStore sdk.KVStore,
	cdc codec.BinaryCodec,
) exported.Status {
	return cs.GetProxyClientState().Status(ctx, NewProxyExtractorStore(cdc, clientStore), cdc)
}

func (cs *ClientState) Validate() error {
	if cs.ProxyClientState == nil {
		return sdkerrors.Wrap(clienttypes.ErrInvalidClient, "ProxyClientState must be non-empty")
	}
	if cs.UpstreamClientId == "" {
		return sdkerrors.Wrap(clienttypes.ErrInvalidClient, "UpstreamClientId must be non-empty")
	}
	if cs.ProxyPrefix == nil {
		return sdkerrors.Wrap(clienttypes.ErrInvalidClient, "ProxyPrefix must be non-empty")
	}
	if cs.IbcPrefix == nil {
		return sdkerrors.Wrap(clienttypes.ErrInvalidClient, "IbcPrefix must be non-empty")
	}
	if cs.UpstreamHeight.IsZero() {
		return sdkerrors.Wrap(clienttypes.ErrInvalidClient, "UpstreamHeight must be non-empty")
	}
	if cs.UpstreamTimestamp == 0 {
		return sdkerrors.Wrap(clienttypes.ErrInvalidClient, "UpstreamTimestamp must be non-zero")
	}
	return cs.GetProxyClientState().Validate()
}

func (cs *ClientState) GetProofSpecs() []*ics23.ProofSpec {
	return cs.GetProxyClientState().GetProofSpecs()
}

// Initialization function
// Clients must validate the initial consensus state, and may store any client-specific metadata
// necessary for correct light client operation
func (cs *ClientState) Initialize(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, consState exported.ConsensusState) error {
	if cs.ProxyClientState == nil || cs.IbcPrefix == nil || cs.ProxyPrefix == nil {
		return sdkerrors.Wrap(clienttypes.ErrInvalidClient, "each fields of the clientState must be non-empty")
	} else if consState == nil {
		return sdkerrors.Wrap(clienttypes.ErrInvalidConsensus, "each fields of the consensusState must be non-empty")
	}
	if _, err := clienttypes.UnpackClientState(cs.ProxyClientState); err != nil {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidClient, "failed to unpack client state: %v", err)
	}
	cons, ok := consState.(*ConsensusState)
	if !ok {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidConsensus, "invalid initial consensus state. expected type: %T, got: %T",
			&ConsensusState{}, consState)
	}
	if _, err := clienttypes.UnpackConsensusState(cons.ProxyConsensusState); err != nil {
		return sdkerrors.Wrapf(clienttypes.ErrInvalidConsensus, "failed to unpack client state: %v", err)
	}
	SetUpstreamBlockTime(clientStore, cs.UpstreamHeight, cs.UpstreamTimestamp)
	return nil
}

// Genesis function
func (cs *ClientState) ExportMetadata(store sdk.KVStore) []exported.GenesisMetadata {
	return cs.GetProxyClientState().ExportMetadata(store)
}

// Upgrade functions
// NOTE: proof heights are not included as upgrade to a new revision is expected to pass only on the last
// height committed by the current revision. Clients are responsible for ensuring that the planned last
// height of the current revision is somehow encoded in the proof verification process.
// This is to ensure that no premature upgrades occur, since upgrade plans committed to by the counterparty
// may be cancelled or modified before the last planned height.
func (cs *ClientState) VerifyUpgradeAndUpdateState(ctx sdk.Context, cdc codec.BinaryCodec, store sdk.KVStore, newClient exported.ClientState, newConsState exported.ConsensusState, proofUpgradeClient []byte, proofUpgradeConsState []byte) (exported.ClientState, exported.ConsensusState, error) {
	return cs.GetProxyClientState().VerifyUpgradeAndUpdateState(ctx, cdc, store, newClient, newConsState, proofUpgradeClient, proofUpgradeConsState)
}

// Utility function that zeroes out any client customizable fields in client state
// Ledger enforced fields are maintained while all custom fields are zero values
// Used to verify upgrades
func (cs *ClientState) ZeroCustomFields() exported.ClientState {
	return cs.GetProxyClientState().ZeroCustomFields()
}

// IBC verification function
func (cs *ClientState) IBCVerifyClientState(store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, prefix exported.Prefix, counterpartyClientIdentifier string, proof []byte, clientState exported.ClientState) error {
	return cs.GetProxyClientState().VerifyClientState(NewProxyExtractorStore(cdc, store), cdc, height, prefix, counterpartyClientIdentifier, proof, clientState)
}

// IBC verification function
func (cs *ClientState) IBCVerifyClientConsensusState(store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, counterpartyClientIdentifier string, consensusHeight exported.Height, prefix exported.Prefix, proof []byte, consensusState exported.ConsensusState) error {
	return cs.GetProxyClientState().VerifyClientConsensusState(NewProxyExtractorStore(cdc, store), cdc, height, counterpartyClientIdentifier, consensusHeight, prefix, proof, consensusState)
}

// State verification functions
func (cs *ClientState) VerifyClientState(store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, prefix exported.Prefix, counterpartyClientIdentifier string, proof []byte, clientState exported.ClientState) error {
	return cs.GetProxyClientState().VerifyClientState(NewProxyExtractorStore(cdc, store), cdc, height, newPrefix(cs.ProxyPrefix, prefix, cs.UpstreamClientId), counterpartyClientIdentifier, proof, clientState)
}

func (cs *ClientState) VerifyClientConsensusState(store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, counterpartyClientIdentifier string, consensusHeight exported.Height, prefix exported.Prefix, proof []byte, consensusState exported.ConsensusState) error {
	return cs.GetProxyClientState().VerifyClientConsensusState(NewProxyExtractorStore(cdc, store), cdc, height, counterpartyClientIdentifier, consensusHeight, newPrefix(cs.ProxyPrefix, prefix, cs.UpstreamClientId), proof, consensusState)
}

func (cs *ClientState) VerifyConnectionState(store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, prefix exported.Prefix, proof []byte, connectionID string, connectionEnd exported.ConnectionI) error {
	return cs.GetProxyClientState().VerifyConnectionState(NewProxyExtractorStore(cdc, store), cdc, height, newPrefix(cs.ProxyPrefix, prefix, cs.UpstreamClientId), proof, connectionID, connectionEnd)
}

func (cs *ClientState) VerifyChannelState(store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, prefix exported.Prefix, proof []byte, portID string, channelID string, channel exported.ChannelI) error {
	return cs.GetProxyClientState().VerifyChannelState(NewProxyExtractorStore(cdc, store), cdc, height, newPrefix(cs.ProxyPrefix, prefix, cs.UpstreamClientId), proof, portID, channelID, channel)
}

func (cs *ClientState) VerifyPacketCommitment(ctx sdk.Context, store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, currentTimestamp uint64, delayPeriod uint64, prefix exported.Prefix, proof []byte, portID string, channelID string, sequence uint64, commitmentBytes []byte) error {
	return cs.GetProxyClientState().VerifyPacketCommitment(ctx, NewProxyExtractorStore(cdc, store), cdc, height, currentTimestamp, delayPeriod, newPrefix(cs.ProxyPrefix, prefix, cs.UpstreamClientId), proof, portID, channelID, sequence, commitmentBytes)
}

func (cs *ClientState) VerifyPacketAcknowledgement(ctx sdk.Context, store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, currentTimestamp uint64, delayPeriod uint64, prefix exported.Prefix, proof []byte, portID string, channelID string, sequence uint64, acknowledgement []byte) error {
	return cs.GetProxyClientState().VerifyPacketAcknowledgement(ctx, NewProxyExtractorStore(cdc, store), cdc, height, currentTimestamp, delayPeriod, newPrefix(cs.ProxyPrefix, prefix, cs.UpstreamClientId), proof, portID, channelID, sequence, acknowledgement)
}

func (cs *ClientState) VerifyPacketReceiptAbsence(ctx sdk.Context, store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, currentTimestamp uint64, delayPeriod uint64, prefix exported.Prefix, proof []byte, portID string, channelID string, sequence uint64) error {
	return cs.GetProxyClientState().VerifyPacketReceiptAbsence(ctx, NewProxyExtractorStore(cdc, store), cdc, height, currentTimestamp, delayPeriod, newPrefix(cs.ProxyPrefix, prefix, cs.UpstreamClientId), proof, portID, channelID, sequence)
}

func (cs *ClientState) VerifyNextSequenceRecv(ctx sdk.Context, store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height, currentTimestamp uint64, delayPeriod uint64, prefix exported.Prefix, proof []byte, portID string, channelID string, nextSequenceRecv uint64) error {
	return cs.GetProxyClientState().VerifyNextSequenceRecv(ctx, NewProxyExtractorStore(cdc, store), cdc, height, currentTimestamp, delayPeriod, newPrefix(cs.ProxyPrefix, prefix, cs.UpstreamClientId), proof, portID, channelID, nextSequenceRecv)
}

func newPrefix(proxyPrefix, upstreamPrefix exported.Prefix, upstreamClientID string) exported.Prefix {
	return commitmenttypes.MultiPrefix{
		Prefix:     proxyPrefix,
		PathPrefix: append([]byte(upstreamClientID+"/"), upstreamPrefix.Bytes()...),
	}
}
