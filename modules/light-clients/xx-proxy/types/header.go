package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

var _ exported.Header = (*Header)(nil)

func (h *Header) ClientType() string {
	return ProxyClientType
}

func (h *Header) GetHeight() exported.Height {
	return h.GetProxyHeader().GetHeight()
}

func (h *Header) ValidateBasic() error {
	return nil
}

func (h *Header) GetProxyHeader() exported.Header {
	header, err := clienttypes.UnpackHeader(h.ProxyHeader)
	if err != nil {
		panic(err)
	}
	return header
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (h *Header) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	if err := unpacker.UnpackAny(h.ProxyHeader, new(exported.Header)); err != nil {
		return err
	}
	return nil
}
