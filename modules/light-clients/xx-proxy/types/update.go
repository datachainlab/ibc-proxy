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
	// XXX hack the store
	ps := NewProxyStore(cdc, clientStore)
	clientState, consensusState, err := cs.GetProxyClientState().CheckHeaderAndUpdateState(ctx, cdc, ps, header)
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

func (cs *ClientState) CheckMisbehaviourAndUpdateState(_ sdk.Context, _ codec.BinaryCodec, _ sdk.KVStore, _ exported.Misbehaviour) (exported.ClientState, error) {
	panic("not implemented") // TODO: Implement
}

func (cs *ClientState) CheckSubstituteAndUpdateState(ctx sdk.Context, cdc codec.BinaryCodec, subjectClientStore sdk.KVStore, substituteClientStore sdk.KVStore, substituteClient exported.ClientState, height exported.Height) (exported.ClientState, error) {
	panic("not implemented") // TODO: Implement
}

type ProxyStore struct {
	sdk.KVStore
	cdc codec.BinaryCodec
}

var _ sdk.KVStore = ProxyStore{}

func NewProxyStore(cdc codec.BinaryCodec, store sdk.KVStore) ProxyStore {
	return ProxyStore{cdc: cdc, KVStore: store}
}

func (s ProxyStore) Get(key []byte) []byte {
	if !strings.HasPrefix(string(key), host.KeyConsensusStatePrefix+"/") {
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
