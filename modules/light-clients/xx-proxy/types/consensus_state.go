package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

var _ exported.ConsensusState = (*ConsensusState)(nil)
var _ codectypes.UnpackInterfacesMessage = (*ConsensusState)(nil)

func NewConsensusState(consensusState *codectypes.Any) *ConsensusState {
	return &ConsensusState{ProxyConsensusState: consensusState}
}

func (cs *ConsensusState) ClientType() string {
	return ProxyClientType
}

func (cs *ConsensusState) GetProxyConsensusState() exported.ConsensusState {
	state, err := clienttypes.UnpackConsensusState(cs.ProxyConsensusState)
	if err != nil {
		panic(err)
	}
	return state
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (cs *ConsensusState) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	if err := unpacker.UnpackAny(cs.ProxyConsensusState, new(exported.ConsensusState)); err != nil {
		return err
	}
	return nil
}

// GetRoot returns the commitment root of the consensus state,
// which is used for key-value pair verification.
func (cs *ConsensusState) GetRoot() exported.Root {
	return cs.GetProxyConsensusState().GetRoot()
}

// GetTimestamp returns the timestamp (in nanoseconds) of the consensus state
func (cs *ConsensusState) GetTimestamp() uint64 {
	return cs.GetProxyConsensusState().GetTimestamp()
}

func (cs *ConsensusState) ValidateBasic() error {
	return cs.GetProxyConsensusState().ValidateBasic()
}
