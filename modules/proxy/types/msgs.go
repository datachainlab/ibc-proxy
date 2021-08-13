package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

var (
	_ sdk.Msg = (*MsgProxyConnectionOpenTry)(nil)
)

func NewMsgProxyConnectionOpenTry(
	connectionID string,
	upstreamClientID string,
	upstreamPrefix commitmenttypes.MerklePrefix,
	connection connectiontypes.ConnectionEnd,
	clientState exported.ClientState,
	consensusState exported.ConsensusState,
	proofInit []byte,
	proofClient []byte,
	proofConsensus []byte,
	proofHeight clienttypes.Height,
	consensusHeight clienttypes.Height,
	signer string,
) (*MsgProxyConnectionOpenTry, error) {
	anyClient, err := clienttypes.PackClientState(clientState)
	if err != nil {
		panic(err)
	}
	anyConsensus, err := clienttypes.PackConsensusState(consensusState)
	if err != nil {
		panic(err)
	}
	return &MsgProxyConnectionOpenTry{
		ConnectionId:     connectionID,
		UpstreamClientId: upstreamClientID,
		UpstreamPrefix:   upstreamPrefix,
		Connection:       connection,
		ClientState:      anyClient,
		ConsensusState:   anyConsensus,
		ProofInit:        proofInit,
		ProofClient:      proofClient,
		ProofConsensus:   proofConsensus,
		ProofHeight:      proofHeight,
		ConsensusHeight:  consensusHeight,
		Signer:           signer,
	}, nil
}

func (msg MsgProxyConnectionOpenTry) ValidateBasic() error {
	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgProxyConnectionOpenTry) GetSigners() []sdk.AccAddress {
	accAddr, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{accAddr}
}
