package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	storeprefix "github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/modules/core/exported"

	"github.com/datachainlab/ibc-proxy/modules/proxy/types"
)

type Keeper struct {
	proxyStoreKey sdk.StoreKey
	ibcStoreKey   sdk.StoreKey
	cdc           codec.BinaryCodec

	clientKeeper types.ClientKeeper
}

func NewKeeper(cdc codec.BinaryCodec, proxyStoreKey, ibcStoreKey sdk.StoreKey, clientKeeper types.ClientKeeper) Keeper {
	return Keeper{
		proxyStoreKey: proxyStoreKey,
		ibcStoreKey:   ibcStoreKey,
		cdc:           cdc,

		clientKeeper: clientKeeper,
	}
}

// GetCommitmentPrefix returns the IBC connection store prefix as a commitment
// Prefix
func (k Keeper) GetProxyCommitmentPrefix() exported.Prefix {
	prefix := commitmenttypes.NewMerklePrefix([]byte(k.proxyStoreKey.Name()))
	return &prefix
}

func (k Keeper) GetIBCCommitmentPrefix() exported.Prefix {
	prefix := commitmenttypes.NewMerklePrefix([]byte(k.ibcStoreKey.Name()))
	return &prefix
}

func (k Keeper) ProxyStore(ctx sdk.Context, upstreamPrefix exported.Prefix, upstreamClientID string) sdk.KVStore {
	return storeprefix.NewStore(ctx.KVStore(k.proxyStoreKey), append([]byte(upstreamClientID+"/"), string(upstreamPrefix.Bytes())+"/"...))
}

func (k Keeper) ProxyClientStore(ctx sdk.Context, upstreamPrefix exported.Prefix, upstreamClientID string, counterpartyClientIdentifier string) sdk.KVStore {
	clientPrefix := append([]byte("clients/"+counterpartyClientIdentifier), '/')
	return storeprefix.NewStore(k.ProxyStore(ctx, upstreamPrefix, upstreamClientID), clientPrefix)
}
