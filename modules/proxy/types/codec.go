package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
	multivtypes "github.com/datachainlab/ibc-proxy/modules/light-clients/xx-multiv/types"
	proxytypes "github.com/datachainlab/ibc-proxy/modules/light-clients/xx-proxy/types"
)

// RegisterInterfaces register the ibc transfer module interfaces to protobuf
// Any.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgProxyConnectionOpenTry{},
		&MsgProxyConnectionOpenAck{},
		&MsgProxyConnectionOpenConfirm{},
		&MsgProxyChannelOpenTry{},
		&MsgProxyChannelOpenAck{},
		&MsgProxyChannelOpenConfirm{},
		&MsgProxyRecvPacket{},
		&MsgProxyAcknowledgePacket{},
	)
	registry.RegisterImplementations((*exported.ClientState)(nil), &proxytypes.ClientState{})
	registry.RegisterImplementations((*exported.ConsensusState)(nil), &proxytypes.ConsensusState{})
	multivtypes.RegisterInterfaces(registry)
}

var (
	// ModuleCdc references the global x/ibc-transfer module codec. Note, the codec
	// should ONLY be used in certain instances of tests and for JSON encoding.
	//
	// The actual codec used for serialization should be provided to x/ibc transfer and
	// defined at the application level.
	ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
)
