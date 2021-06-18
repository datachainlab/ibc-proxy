package types

import (
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
)

func NewProxyRequestPacketData(upstreamClientID, proxyClientID string) ProxyRequestPacketData {
	return ProxyRequestPacketData{
		UpstreamClientId: upstreamClientID,
		ProxyClientId:    proxyClientID,
	}
}

func NewProxyRequestAcknowledgement(status Status, proxyPrefix, ibcPrefix commitmenttypes.MerklePrefix) ProxyRequestAcknowledgement {
	return ProxyRequestAcknowledgement{
		Status:      status,
		ProxyPrefix: &proxyPrefix,
		IbcPrefix:   &ibcPrefix,
	}
}
