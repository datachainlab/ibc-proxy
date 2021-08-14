package types

import (
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

const (
	// ModuleName defines the IBC Proxy
	ModuleName = "proxy"

	Version = "proxy-1"

	PortID = "proxy"

	// StoreKey is the store key string for IBC proxy
	StoreKey = ModuleName

	RouterKey = ModuleName

	QuerierRoute = ModuleName
)

// ProxyKey returns the store key for a proxy state
func ProxyKey(upstreamPrefix exported.Prefix, upstreamClientID string, key []byte) []byte {
	return append(append([]byte(upstreamClientID+"/"), string(upstreamPrefix.Bytes())+"/"...), key...)
}

// ProxyClientStateKey returns the store key for the proxy client state of a particular
// client.
func ProxyClientStateKey(upstreamPrefix exported.Prefix, upstreamClientID string, clientID string) []byte {
	return ProxyKey(upstreamPrefix, upstreamClientID, host.FullClientStateKey(clientID))
}

// ProxyConsensusStateKey returns the store key for the proxy consensus state of a particular
// client.
func ProxyConsensusStateKey(upstreamPrefix exported.Prefix, upstreamClientID string, clientID string, consensusHeight exported.Height) []byte {
	return ProxyKey(upstreamPrefix, upstreamClientID, host.FullConsensusStateKey(clientID, consensusHeight))
}

// ProxyConnectionKey returns the store key for a particular proxy connection
func ProxyConnectionKey(upstreamPrefix exported.Prefix, upstreamClientID string, connectionID string) []byte {
	return ProxyKey(upstreamPrefix, upstreamClientID, host.ConnectionKey(connectionID))
}

// ProxyChannelKey returns the store key for a particular proxy channel
func ProxyChannelKey(upstreamPrefix exported.Prefix, upstreamClientID string, portID string, channelID string) []byte {
	return ProxyKey(upstreamPrefix, upstreamClientID, host.ChannelKey(portID, channelID))
}

// ProxyPacketCommitmentKey returns the store key of under which a proxy packet commitment
// is stored
func ProxyPacketCommitmentKey(upstreamPrefix exported.Prefix, upstreamClientID string, portID string, channelID string, sequence uint64) []byte {
	return ProxyKey(upstreamPrefix, upstreamClientID, host.PacketCommitmentKey(portID, channelID, sequence))
}

// ProxyPacketAcknowledgementKey returns the store key of under which a proxy packet
// acknowledgement is stored
func ProxyAcknowledgementKey(upstreamPrefix exported.Prefix, upstreamClientID string, portID string, channelID string, sequence uint64) []byte {
	return ProxyKey(upstreamPrefix, upstreamClientID, host.PacketAcknowledgementKey(portID, channelID, sequence))
}
