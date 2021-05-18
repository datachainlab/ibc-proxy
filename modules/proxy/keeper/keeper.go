package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/exported"
	"github.com/datachainlab/ibc-proxy/modules/proxy/types"
)

type Keeper struct {
	proxyStoreKey sdk.StoreKey
	ibcStoreKey   sdk.StoreKey
	cdc           codec.BinaryCodec

	clientKeeper     types.ClientKeeper
	connectionKeeper types.ConnectionKeeper
	channelKeeper    types.ChannelKeeper
	scopedKeeper     capabilitykeeper.ScopedKeeper
	portKeeper       types.PortKeeper
}

func NewKeeper(cdc codec.BinaryCodec, proxyStoreKey, ibcStoreKey sdk.StoreKey, clientKeeper types.ClientKeeper, connectionKeeper types.ConnectionKeeper, channelKeeper types.ChannelKeeper, scopedKeeper capabilitykeeper.ScopedKeeper, portKeeper types.PortKeeper) Keeper {
	return Keeper{
		proxyStoreKey: proxyStoreKey,
		ibcStoreKey:   ibcStoreKey,
		cdc:           cdc,

		clientKeeper:     clientKeeper,
		connectionKeeper: connectionKeeper,
		channelKeeper:    channelKeeper,
		scopedKeeper:     scopedKeeper,
		portKeeper:       portKeeper,
	}
}

// GetCommitmentPrefix returns the IBC connection store prefix as a commitment
// Prefix
func (k Keeper) GetCommitmentPrefix() exported.Prefix {
	return commitmenttypes.NewMerklePrefix([]byte(k.proxyStoreKey.Name()))
}

func (k Keeper) EnableProxy(ctx sdk.Context, clientID string) error {
	_, found := k.clientKeeper.GetClientState(ctx, clientID)
	if !found {
		return fmt.Errorf("clientID '%v' not found", clientID)
	}
	store := ctx.KVStore(k.proxyStoreKey)
	if store.Has([]byte(clientID)) {
		return fmt.Errorf("clientID '%v' already exists", clientID)
	}
	store.Set([]byte(clientID), []byte{1})
	return nil
}

func (k Keeper) IsProxyEnabled(ctx sdk.Context, clientID string) bool {
	return ctx.KVStore(k.proxyStoreKey).Has([]byte(clientID))
}

// AuthenticateCapability wraps the scopedKeeper's AuthenticateCapability function
func (k Keeper) AuthenticateCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) bool {
	return k.scopedKeeper.AuthenticateCapability(ctx, cap, name)
}

// ClaimCapability allows the transfer module that can claim a capability that IBC module
// passes to it
func (k Keeper) ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error {
	return k.scopedKeeper.ClaimCapability(ctx, cap, name)
}

// IsBound checks if the transfer module is already bound to the desired port
func (k Keeper) IsBound(ctx sdk.Context, portID string) bool {
	_, ok := k.scopedKeeper.GetCapability(ctx, host.PortPath(portID))
	return ok
}

// BindPort defines a wrapper function for the ort Keeper's function in
// order to expose it to module's InitGenesis function
func (k Keeper) BindPort(ctx sdk.Context, portID string) error {
	cap := k.portKeeper.BindPort(ctx, portID)
	return k.ClaimCapability(ctx, cap, host.PortPath(portID))
}
