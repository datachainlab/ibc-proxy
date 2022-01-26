package types

import (
	"fmt"
	"math/big"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

type ProxyClientI interface {
	VerifyBlockTime(
		store sdk.KVStore,
		cdc codec.BinaryCodec,
		height exported.Height,
		prefix exported.Prefix,
		upstreamHeight exported.Height,
		proof []byte,
		timestamp uint64,
	) error
}

var GlobalProxyClientRegistry = NewProxyClientRegistry()

type ProxyClientRegistry struct {
	builders map[string]ProxyClientBuilder
	sealed   bool
}

func NewProxyClientRegistry() *ProxyClientRegistry {
	return &ProxyClientRegistry{builders: make(map[string]ProxyClientBuilder)}
}

func (pr *ProxyClientRegistry) RegisterBuilder(builder ProxyClientBuilder) {
	if pr.sealed {
		panic(fmt.Errorf("the registry is already sealed"))
	}
	clientType := builder.ClientType()
	builder, ok := pr.builders[clientType]
	if !ok {
		panic(fmt.Errorf("the clientType '%v' already exists", clientType))
	}
	pr.builders[clientType] = builder
}

func (pr *ProxyClientRegistry) Seal() {
	if pr.sealed {
		panic(fmt.Errorf("the registry is already sealed"))
	}
	pr.sealed = true
}

func (pr ProxyClientRegistry) MustGet(clientType string) ProxyClientBuilder {
	builder, ok := pr.builders[clientType]
	if !ok {
		panic(fmt.Errorf("the clientType '%v' not found", clientType))
	}
	return builder
}

type ProxyClientBuilder interface {
	ClientType() string
	Build(exported.ClientState) (ProxyClientI, error)
}

var _ exported.Height = (*UpstreamHeight)(nil)

// ZeroHeight is a helper function which returns an uninitialized height.
func ZeroHeight() UpstreamHeight {
	return UpstreamHeight{}
}

// NewHeight is a constructor for the IBC height type
func NewHeight(revisionNumber, revisionHeight uint64) UpstreamHeight {
	return UpstreamHeight{
		RevisionNumber: revisionNumber,
		RevisionHeight: revisionHeight,
	}
}

// GetRevisionNumber returns the revision-number of the height
func (h UpstreamHeight) GetRevisionNumber() uint64 {
	return h.RevisionNumber
}

// GetRevisionHeight returns the revision-height of the height
func (h UpstreamHeight) GetRevisionHeight() uint64 {
	return h.RevisionHeight
}

// Compare implements a method to compare two heights. When comparing two heights a, b
// we can call a.Compare(b) which will return
// -1 if a < b
// 0  if a = b
// 1  if a > b
//
// It first compares based on revision numbers, whichever has the higher revision number is the higher height
// If revision number is the same, then the revision height is compared
func (h UpstreamHeight) Compare(other exported.Height) int64 {
	// height, ok := other.(UpstreamHeight)
	// if !ok {
	// 	panic(fmt.Sprintf("cannot compare against invalid height type: %T. expected height type: %T", other, h))
	// }
	height := NewHeight(other.GetRevisionNumber(), other.GetRevisionHeight())
	var a, b big.Int
	if h.RevisionNumber != height.RevisionNumber {
		a.SetUint64(h.RevisionNumber)
		b.SetUint64(height.RevisionNumber)
	} else {
		a.SetUint64(h.RevisionHeight)
		b.SetUint64(height.RevisionHeight)
	}
	return int64(a.Cmp(&b))
}

// LT Helper comparison function returns true if h < other
func (h UpstreamHeight) LT(other exported.Height) bool {
	return h.Compare(other) == -1
}

// LTE Helper comparison function returns true if h <= other
func (h UpstreamHeight) LTE(other exported.Height) bool {
	cmp := h.Compare(other)
	return cmp <= 0
}

// GT Helper comparison function returns true if h > other
func (h UpstreamHeight) GT(other exported.Height) bool {
	return h.Compare(other) == 1
}

// GTE Helper comparison function returns true if h >= other
func (h UpstreamHeight) GTE(other exported.Height) bool {
	cmp := h.Compare(other)
	return cmp >= 0
}

// EQ Helper comparison function returns true if h == other
func (h UpstreamHeight) EQ(other exported.Height) bool {
	return h.Compare(other) == 0
}

// String returns a string representation of UpstreamHeight
func (h UpstreamHeight) String() string {
	return fmt.Sprintf("%d-%d", h.RevisionNumber, h.RevisionHeight)
}

// Decrement will return a new height with the RevisionHeight decremented
// If the RevisionHeight is already at lowest value (1), then false success flag is returend
func (h UpstreamHeight) Decrement() (decremented exported.Height, success bool) {
	if h.RevisionHeight == 0 {
		return UpstreamHeight{}, false
	}
	return NewHeight(h.RevisionNumber, h.RevisionHeight-1), true
}

// Increment will return a height with the same revision number but an
// incremented revision height
func (h UpstreamHeight) Increment() exported.Height {
	return NewHeight(h.RevisionNumber, h.RevisionHeight+1)
}

// IsZero returns true if height revision and revision-height are both 0
func (h UpstreamHeight) IsZero() bool {
	return h.RevisionNumber == 0 && h.RevisionHeight == 0
}
