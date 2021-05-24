package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
)

func NewProxyRequestPacketData(upstreamClientID, proxyClientID string) ProxyRequestPacketData {
	return ProxyRequestPacketData{
		UpstreamClientId: upstreamClientID,
		ProxyClientId:    proxyClientID,
	}
}

// // GetBytes is a helper for serialising
// func (pd ProxyRequestPacketData) GetBytes() []byte {
// 	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&pd))
// }

func NewProxyRequestAcknowledgement(status Status, prefix commitmenttypes.MerklePrefix, upstreamState UpstreamState) ProxyRequestAcknowledgement {
	return ProxyRequestAcknowledgement{
		Status:        status,
		Prefix:        &prefix,
		UpstreamState: &upstreamState,
	}
}

// func (ack ProxyRequestAcknowledgement) GetBytes() []byte {
// 	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&ack))
// }

func NewUpstreamState(height clienttypes.Height, clientState, consensusState *codectypes.Any) UpstreamState {
	return UpstreamState{
		Height:         height,
		ClientState:    clientState,
		ConsensusState: consensusState,
	}
}
