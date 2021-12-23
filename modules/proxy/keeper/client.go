package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

// caller: B
// CONTRACT: upstream is A, downstream is B, we are proxy(P)
func (k Keeper) ClientState(
	ctx sdk.Context,
	upstreamClientID string, // the client ID corresponding to A on P
	upstreamPrefix exported.Prefix,
	counterpartyClientID string,
	clientState exported.ClientState, // clientState that chainA has for chainB or proxy
	expectedConsensusState exported.ConsensusState,
	proofClient []byte, // proof that chainA stored a light client of chainB
	proofConsensus []byte, // proof that chainA stored chainB's consensus state at consensus height
	proofHeight exported.Height, // height at which relayer constructs proof of A storing connectionEnd in state
	consensusHeight exported.Height, // latest height of chain B which chain A has stored in its chain B client
) error {

	// Check that ChainA stored the clientState provided in the msg
	if err := k.VerifyAndProxyClientState(ctx, upstreamClientID, upstreamPrefix, counterpartyClientID, proofHeight, proofClient, clientState); err != nil {
		return err
	}

	// Check that ChainA stored the correct ConsensusState of chainB or proxy at the given consensusHeight
	if err := k.VerifyAndProxyClientConsensusState(
		ctx, upstreamClientID, upstreamPrefix, counterpartyClientID, proofHeight, consensusHeight, proofConsensus, expectedConsensusState,
	); err != nil {
		return err
	}

	return nil
}
