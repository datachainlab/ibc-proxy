package types

import (
	"fmt"
	"regexp"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

// Update and Misbehaviour functions
func (cs ClientState) CheckHeaderAndUpdateState(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, header exported.Header) (exported.ClientState, exported.ConsensusState, error) {
	h := header.(*Header)
	clientState, consensusState, err := cs.GetProxyClientState().CheckHeaderAndUpdateState(ctx, cdc, NewProxyExtractorStore(cdc, clientStore), h.GetProxyHeader())
	if err != nil {
		return nil, nil, err
	}
	anyClientState, err := clienttypes.PackClientState(clientState)
	if err != nil {
		return nil, nil, err
	}
	anyConsensusState, err := clienttypes.PackConsensusState(consensusState)
	if err != nil {
		return nil, nil, err
	}
	cs.ProxyClientState = anyClientState
	proxyConsensusState := &ConsensusState{ProxyConsensusState: anyConsensusState}

	if ubp := h.UpstreamBlockProof; ubp != nil {
		pc, err := GlobalProxyClientRegistry.MustGet(cs.GetProxyClientState().ClientType()).Build(cs.GetProxyClientState())
		if err != nil {
			return nil, nil, err
		}
		prefix := commitmenttypes.MultiPrefix{
			Prefix:     cs.ProxyPrefix,
			PathPrefix: []byte(cs.UpstreamClientId + "/"),
		}
		if err := pc.VerifyBlockTime(
			NewProxyExtractorStore(cdc, clientStore),
			cdc, ubp.ProofHeight, prefix, ubp.UpstreamHeight, ubp.Proof, ubp.UpstreamTimestamp,
		); err != nil {
			return nil, nil, err
		}
		if ubp.UpstreamHeight.GT(cs.UpstreamHeight) {
			// NOTE: should we check if proofHeight is also advanced?
			cs.UpstreamHeight = UpstreamHeight(ubp.UpstreamHeight)
			cs.UpstreamTimestamp = ubp.UpstreamTimestamp
		}
		SetUpstreamBlockTime(clientStore, ubp.UpstreamHeight, ubp.UpstreamTimestamp)
	}
	return &cs, proxyConsensusState, nil
}

func (cs *ClientState) CheckMisbehaviourAndUpdateState(ctx sdk.Context, cdc codec.BinaryCodec, store sdk.KVStore, misbehaviour exported.Misbehaviour) (exported.ClientState, error) {
	clientState, err := cs.GetProxyClientState().CheckMisbehaviourAndUpdateState(ctx, cdc, store, misbehaviour)
	if err != nil {
		return nil, err
	}
	anyClientState, err := clienttypes.PackClientState(clientState)
	if err != nil {
		return nil, err
	}
	cs.ProxyClientState = anyClientState
	return cs, nil
}

func (cs *ClientState) CheckSubstituteAndUpdateState(ctx sdk.Context, cdc codec.BinaryCodec, subjectClientStore sdk.KVStore, substituteClientStore sdk.KVStore, substituteClient exported.ClientState) (exported.ClientState, error) {
	clientState, err := cs.GetProxyClientState().CheckSubstituteAndUpdateState(ctx, cdc, subjectClientStore, substituteClientStore, substituteClient)
	anyClientState, err := clienttypes.PackClientState(clientState)
	if err != nil {
		return nil, err
	}
	cs.ProxyClientState = anyClientState
	return cs, nil
}

// proxyExtractorStore provides a store that extracts the underlying state from ProxyConsensusState
type proxyExtractorStore struct {
	sdk.KVStore
	cdc codec.BinaryCodec
}

var _ sdk.KVStore = (*proxyExtractorStore)(nil)

func NewProxyExtractorStore(cdc codec.BinaryCodec, store sdk.KVStore) proxyExtractorStore {
	return proxyExtractorStore{cdc: cdc, KVStore: store}
}

var consensusStateKeyRegexp = regexp.MustCompile(fmt.Sprintf(`^%s/\d+-\d+$`, host.KeyConsensusStatePrefix))

func (s proxyExtractorStore) Get(key []byte) []byte {
	v := s.KVStore.Get(key)
	if !consensusStateKeyRegexp.Match(key) {
		return v
	} else if len(v) == 0 {
		return v
	}
	cs, err := clienttypes.UnmarshalConsensusState(s.cdc, v)
	if err != nil {
		panic(err)
	}
	bz, err := clienttypes.MarshalConsensusState(s.cdc, cs.(*ConsensusState).GetProxyConsensusState())
	if err != nil {
		panic(err)
	}
	return bz
}
