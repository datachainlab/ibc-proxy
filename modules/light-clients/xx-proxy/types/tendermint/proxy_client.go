package tendermint

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
	tmtypes "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
	"github.com/datachainlab/ibc-proxy/modules/light-clients/xx-proxy/types"
)

type ProxyClientBuilder struct{}

var _ types.ProxyClientBuilder = (*ProxyClientBuilder)(nil)

func (ProxyClientBuilder) ClientType() string {
	return exported.Tendermint
}

func (pb ProxyClientBuilder) Build(clientState exported.ClientState) (types.ProxyClientI, error) {
	cs, ok := clientState.(*tmtypes.ClientState)
	if !ok {
		return nil, sdkerrors.Wrapf(clienttypes.ErrInvalidClientType, "expected type %T, got %T", &tmtypes.ClientState{}, clientState)
	}
	return ProxyClient{clientState: *cs}, nil
}

type ProxyClient struct {
	clientState tmtypes.ClientState
}

var _ types.ProxyClientI = (*ProxyClient)(nil)

func (pc ProxyClient) VerifyBlockTime(
	store sdk.KVStore,
	cdc codec.BinaryCodec,
	height exported.Height,
	prefix exported.Prefix,
	upstreamHeight exported.Height,
	proof []byte,
	timestamp uint64,
) error {
	merkleProof, provingConsensusState, err := produceVerificationArgs(store, cdc, pc.clientState, height, prefix, proof)
	if err != nil {
		return err
	}

	clientPrefixedPath := commitmenttypes.NewMerklePath(fmt.Sprintf("block/%s", upstreamHeight.String()))
	path, err := commitmenttypes.ApplyPrefix(prefix, clientPrefixedPath)
	if err != nil {
		return err
	}

	return merkleProof.VerifyMembership(pc.clientState.ProofSpecs, provingConsensusState.GetRoot(), path, sdk.Uint64ToBigEndian(timestamp))
}

// produceVerificationArgs perfoms the basic checks on the arguments that are
// shared between the verification functions and returns the unmarshalled
// merkle proof, the consensus state and an error if one occurred.
func produceVerificationArgs(
	store sdk.KVStore,
	cdc codec.BinaryCodec,
	cs tmtypes.ClientState,
	height exported.Height,
	prefix exported.Prefix,
	proof []byte,
) (merkleProof commitmenttypes.MerkleProof, consensusState *tmtypes.ConsensusState, err error) {
	if cs.GetLatestHeight().LT(height) {
		return commitmenttypes.MerkleProof{}, nil, sdkerrors.Wrapf(
			sdkerrors.ErrInvalidHeight,
			"client state height < proof height (%d < %d), please ensure the client has been updated", cs.GetLatestHeight(), height,
		)
	}

	if prefix == nil {
		return commitmenttypes.MerkleProof{}, nil, sdkerrors.Wrap(commitmenttypes.ErrInvalidPrefix, "prefix cannot be empty")
	}

	if proof == nil {
		return commitmenttypes.MerkleProof{}, nil, sdkerrors.Wrap(commitmenttypes.ErrInvalidProof, "proof cannot be empty")
	}

	if err = cdc.Unmarshal(proof, &merkleProof); err != nil {
		return commitmenttypes.MerkleProof{}, nil, sdkerrors.Wrap(commitmenttypes.ErrInvalidProof, "failed to unmarshal proof into commitment merkle proof")
	}

	consensusState, err = tmtypes.GetConsensusState(store, cdc, height)
	if err != nil {
		return commitmenttypes.MerkleProof{}, nil, sdkerrors.Wrap(err, "please ensure the proof was constructed against a height that exists on the client")
	}

	return merkleProof, consensusState, nil
}
