package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

// Update and Misbehaviour functions
func (cs ClientState) CheckHeaderAndUpdateState(ctx sdk.Context, cdc codec.BinaryCodec, clientStore sdk.KVStore, header exported.Header) (exported.ClientState, exported.ConsensusState, error) {
	clientState, consensusState, err := cs.GetUnderlyingClientState().CheckHeaderAndUpdateState(ctx, cdc, clientStore, header)
	if err != nil {
		return nil, nil, err
	}
	anyClientState, err := clienttypes.PackClientState(clientState)
	if err != nil {
		return nil, nil, err
	}
	cs.UnderlyingClientState = anyClientState
	return &cs, consensusState, nil
}

func (cs *ClientState) CheckMisbehaviourAndUpdateState(ctx sdk.Context, cdc codec.BinaryCodec, store sdk.KVStore, misbehaviour exported.Misbehaviour) (exported.ClientState, error) {
	clientState, err := cs.GetUnderlyingClientState().CheckMisbehaviourAndUpdateState(ctx, cdc, store, misbehaviour)
	if err != nil {
		return nil, err
	}
	anyClientState, err := clienttypes.PackClientState(clientState)
	if err != nil {
		return nil, err
	}
	cs.UnderlyingClientState = anyClientState
	return cs, nil
}

func (cs *ClientState) CheckSubstituteAndUpdateState(ctx sdk.Context, cdc codec.BinaryCodec, subjectClientStore sdk.KVStore, substituteClientStore sdk.KVStore, substituteClient exported.ClientState) (exported.ClientState, error) {
	clientState, err := cs.GetUnderlyingClientState().CheckSubstituteAndUpdateState(ctx, cdc, subjectClientStore, substituteClientStore, substituteClient)
	anyClientState, err := clienttypes.PackClientState(clientState)
	if err != nil {
		return nil, err
	}
	cs.UnderlyingClientState = anyClientState
	return cs, nil
}
