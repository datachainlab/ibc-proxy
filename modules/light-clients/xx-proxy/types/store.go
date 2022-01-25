package types

import (
	"encoding/binary"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

// GetConsensusState retrieves the consensus state from the client prefixed
// store. An error is returned if the consensus state does not exist.
func GetConsensusState(store sdk.KVStore, cdc codec.BinaryCodec, height exported.Height) (*ConsensusState, error) {
	bz := store.Get(host.ConsensusStateKey(height))
	if bz == nil {
		return nil, sdkerrors.Wrapf(
			clienttypes.ErrConsensusStateNotFound,
			"consensus state does not exist for height %s", height,
		)
	}

	consensusStateI, err := clienttypes.UnmarshalConsensusState(cdc, bz)
	if err != nil {
		return nil, sdkerrors.Wrapf(clienttypes.ErrInvalidConsensus, "unmarshal error: %v", err)
	}

	consensusState, ok := consensusStateI.(*ConsensusState)
	if !ok {
		return nil, sdkerrors.Wrapf(
			clienttypes.ErrInvalidConsensus,
			"invalid consensus type %T, expected %T", consensusState, &ConsensusState{},
		)
	}

	return consensusState, nil
}

func SetUpstreamBlockTime(clientStore sdk.KVStore, height exported.Height, timestamp uint64) {
	var bz [8]byte
	binary.BigEndian.PutUint64(bz[:], timestamp)
	clientStore.Set([]byte(fmt.Sprintf("/upstreamBlockTimes/%s", height.String())), bz[:])
}

func GetUpstreamBlockTime(clientStore sdk.KVStore, height exported.Height) (uint64, bool) {
	bz := clientStore.Get([]byte(fmt.Sprintf("/upstreamBlockTimes/%s", height.String())))
	if l := len(bz); l == 0 {
		return 0, false
	} else if l != 8 {
		panic("the state is corrupted")
	}
	return binary.BigEndian.Uint64(bz), true
}
