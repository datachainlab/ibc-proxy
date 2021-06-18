package types

import (
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

// Update and Misbehaviour functions
func (cs ClientState) CheckHeaderAndUpdateState(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, header exported.Header) (exported.ClientState, exported.ConsensusState, error) {
	clientState, consensusState, err := cs.GetProxyClientState().CheckHeaderAndUpdateState(ctx, cdc, NewProxyStore(cdc, clientStore), header)
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

func (cs *ClientState) CheckSubstituteAndUpdateState(ctx sdk.Context, cdc codec.BinaryCodec, subjectClientStore sdk.KVStore, substituteClientStore sdk.KVStore, substituteClient exported.ClientState, height exported.Height) (exported.ClientState, error) {
	clientState, err := cs.GetProxyClientState().CheckSubstituteAndUpdateState(ctx, cdc, subjectClientStore, substituteClientStore, substituteClient, height)
	anyClientState, err := clienttypes.PackClientState(clientState)
	if err != nil {
		return nil, err
	}
	cs.ProxyClientState = anyClientState
	return cs, nil
}

// ProxyStore provides the proxy for the underlying store
// if the key of consensus state is given, get the client state from proxy client state
type ProxyStore struct {
	sdk.KVStore
	cdc codec.BinaryCodec
}

var _ sdk.KVStore = (*ProxyStore)(nil)

func NewProxyStore(cdc codec.BinaryCodec, store sdk.KVStore) ProxyStore {
	return ProxyStore{cdc: cdc, KVStore: store}
}

func (s ProxyStore) Get(key []byte) []byte {
	k := string(key)
	if !strings.HasPrefix(k, host.KeyConsensusStatePrefix+"/") || strings.HasSuffix(k, "/processedTime") {
		return s.KVStore.Get(key)
	}
	v := s.KVStore.Get(key)
	if len(v) == 0 {
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
