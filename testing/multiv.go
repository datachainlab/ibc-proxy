package ibctesting

import (
	"fmt"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/exported"
	multivtypes "github.com/datachainlab/ibc-proxy/modules/light-clients/xx-multiv/types"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

func (coord *Coordinator) CreateMultiVClient(
	source, counterparty *TestChain,
	clientType string,
) (clientID string, err error) {
	coord.CommitBlock(source, counterparty)

	clientID = source.NewClientID(clientType)

	err = source.CreateMultiVClient(counterparty, clientID, clientType)
	if err != nil {
		return "", err
	}

	coord.IncrementTime()

	return clientID, nil
}

func (coord *Coordinator) UpdateMultiVClient(
	source, counterparty *TestChain,
	clientID string,
) error {
	return source.UpdateMultiVClient(counterparty, clientID)
}

func (chain *TestChain) CreateMultiVClient(
	counterparty *TestChain,
	clientID string,
	wrappedClientType string,
) error {
	if wrappedClientType != exported.Tendermint {
		return fmt.Errorf("unsupported client type %v", wrappedClientType)
	}
	m := chain.ConstructMsgCreateClient(counterparty, clientID, wrappedClientType)

	consensusState, err := clienttypes.UnpackConsensusState(m.ConsensusState)
	if err != nil {
		return err
	}

	msg, err := clienttypes.NewMsgCreateClient(
		multivtypes.NewClientState(m.ClientState),
		consensusState, m.Signer,
	)
	if err != nil {
		return err
	}
	return chain.sendMsgs(msg)
}

func (chain *TestChain) UpdateMultiVClient(
	counterparty *TestChain,
	clientID string,
) error {
	return chain.UpdateTMClient(counterparty, clientID)
}

// chain: c1, counterparty: c0, counterpartyProxy: p0
// c0 -> p0 -> c1
// c1 -> c0
// proof-tree: c1 -> c0 (head)-> p0 (leaf)-> c1
func (chain *TestChain) ConnectionOpenTryWithProxy(
	counterparty *TestChain,
	connection, counterpartyConnection *TestConnection,
	counterpartyProxy ProxyInfo,
) error {
	head := counterparty.QueryMultiVHeadProof(counterpartyConnection.ClientID)
	upstreamClientState, proofClient := chain.QueryMultiVLeafClientProof(head, counterpartyProxy.UpstreamClientID, counterpartyProxy)
	_, proofConsensus, upstreamConsensusHeight := chain.QueryMultiVLeafConsensusProof(head, counterpartyProxy.UpstreamClientID, counterpartyProxy)
	proofInit, proofHeight := counterparty.QueryProof(host.ConnectionKey(counterpartyConnection.ID))

	msg := connectiontypes.NewMsgConnectionOpenTry(
		"", connection.ClientID, // does not support handshake continuation
		counterpartyConnection.ID, counterpartyConnection.ClientID,
		upstreamClientState, counterparty.GetPrefix(), []*connectiontypes.Version{ConnectionVersion}, DefaultDelayPeriod,
		proofInit, proofClient, proofConsensus,
		proofHeight, upstreamConsensusHeight,
		chain.SenderAccount.GetAddress().String(),
	)
	return chain.sendMsgs(msg)
}

// chain: c0, counterparty: c1, counterpartyProxy: p1
// c0 -> c1
// c1 -> p1 -> c0
// proof-tree: c0 -> c1 -> p1 -> c0
func (chain *TestChain) ConnectionOpenAckWithProxy(
	counterparty *TestChain,
	connection, counterpartyConnection *TestConnection,
	counterpartyProxy ProxyInfo,
) error {
	head := counterparty.QueryMultiVHeadProof(counterpartyConnection.ClientID)
	upstreamClientState, proofClient := chain.QueryMultiVLeafClientProof(head, counterpartyProxy.UpstreamClientID, counterpartyProxy)
	_, proofConsensus, upstreamConsensusHeight := chain.QueryMultiVLeafConsensusProof(head, counterpartyProxy.UpstreamClientID, counterpartyProxy)
	proofTry, proofHeight := counterparty.QueryProof(host.ConnectionKey(counterpartyConnection.ID))

	msg := connectiontypes.NewMsgConnectionOpenAck(
		connection.ID, counterpartyConnection.ID, upstreamClientState, // testing doesn't use flexible selection
		proofTry, proofClient, proofConsensus,
		proofHeight, upstreamConsensusHeight,
		ConnectionVersion,
		chain.SenderAccount.GetAddress().String(),
	)
	return chain.sendMsgs(msg)
}

func (chain *TestChain) QueryAnyClientStateProof(clientID string) (*codectypes.Any, []byte) {
	cs, proof := chain.QueryClientStateProof(clientID)
	any, err := clienttypes.PackClientState(cs)
	if err != nil {
		panic(err)
	}
	return any, proof
}

func (chain *TestChain) QueryAnyConsensusStateProof(clientID string) (*codectypes.Any, []byte, clienttypes.Height) {
	proof, consensusHeight := chain.QueryConsensusStateProof(clientID)
	consensusState, found := chain.GetConsensusState(clientID, consensusHeight)
	if !found {
		panic(fmt.Errorf("consensusState not found: %v %v", clientID, consensusHeight))
	}
	any, err := clienttypes.PackConsensusState(consensusState)
	if err != nil {
		panic(err)
	}
	return any, proof, consensusHeight
}

func (chain *TestChain) QueryMultiVHeadProof(clientID string) *multivtypes.HeadProof {
	proxyClientState, proxyProofClient := chain.QueryAnyClientStateProof(clientID)
	proxyConsensusState, proxyProofConsensus, consensusHeight := chain.QueryAnyConsensusStateProof(clientID)

	return &multivtypes.HeadProof{
		ClientProof:     proxyProofClient,
		ClientState:     proxyClientState,
		ConsensusProof:  proxyProofConsensus,
		ConsensusState:  proxyConsensusState,
		ConsensusHeight: consensusHeight,
	}
}

func (chain *TestChain) QueryMultiVLeafClientProof(head *multivtypes.HeadProof, upstreamClientID string, proxy ProxyInfo) (exported.ClientState, []byte) {
	cs, err := clienttypes.UnpackClientState(head.ClientState)
	if err != nil {
		panic(err)
	}
	h := cs.GetLatestHeight()
	upstreamClientState, upstreamClientProof, upstreamProofHeight := proxy.Chain.queryClientStateProof(upstreamClientID, int64(h.GetRevisionHeight())-1)

	leafClient := &multivtypes.LeafClientProof{
		Proof:       upstreamClientProof,
		ProofHeight: upstreamProofHeight,
	}
	proofClient := chain.makeClientStateProof(head, leafClient)
	return upstreamClientState, proofClient
}

func (chain *TestChain) QueryMultiVLeafConsensusProof(head *multivtypes.HeadProof, upstreamClientID string, proxy ProxyInfo) (exported.ConsensusState, []byte, clienttypes.Height) {
	cs, err := clienttypes.UnpackClientState(head.ClientState)
	if err != nil {
		panic(err)
	}
	h := cs.GetLatestHeight()
	upstreamConsensusProof, upstreamConsensusHeight, upstreamProofHeight := proxy.Chain.queryConsensusStateProof(proxy.UpstreamClientID, int64(h.GetRevisionHeight())-1)
	leafConsensus := &multivtypes.LeafConsensusProof{
		Proof:           upstreamConsensusProof,
		ProofHeight:     upstreamProofHeight,
		ConsensusHeight: upstreamConsensusHeight,
	}
	proofConsensus := chain.makeConsensusStateProof(head, leafConsensus)
	consensusState, found := proxy.Chain.GetConsensusState(proxy.UpstreamClientID, upstreamConsensusHeight)
	if !found {
		panic("consensusState not found")
	}
	return consensusState, proofConsensus, upstreamConsensusHeight
}

func (chain *TestChain) makeClientStateProof(
	head *multivtypes.HeadProof,
	leafClient *multivtypes.LeafClientProof,
	branches ...*multivtypes.BranchProof,
) []byte {
	var mp multivtypes.MultiProof

	mp.Proofs = append(mp.Proofs, &multivtypes.Proof{
		Proof: &multivtypes.Proof_Head{Head: head},
	})
	for _, branch := range branches {
		mp.Proofs = append(mp.Proofs, &multivtypes.Proof{
			Proof: &multivtypes.Proof_Branch{Branch: branch},
		})
	}
	mp.Proofs = append(mp.Proofs, &multivtypes.Proof{
		Proof: &multivtypes.Proof_LeafClient{LeafClient: leafClient},
	})

	any, err := codectypes.NewAnyWithValue(&mp)
	if err != nil {
		panic(err)
	}
	bz, err := chain.App.AppCodec().Marshal(any)
	if err != nil {
		panic(err)
	}
	return bz
}

func (chain *TestChain) makeConsensusStateProof(
	head *multivtypes.HeadProof,
	leafConsensus *multivtypes.LeafConsensusProof,
	branches ...*multivtypes.BranchProof,
) []byte {
	var mp multivtypes.MultiProof

	mp.Proofs = append(mp.Proofs, &multivtypes.Proof{
		Proof: &multivtypes.Proof_Head{Head: head},
	})
	for _, branch := range branches {
		mp.Proofs = append(mp.Proofs, &multivtypes.Proof{
			Proof: &multivtypes.Proof_Branch{Branch: branch},
		})
	}
	mp.Proofs = append(mp.Proofs, &multivtypes.Proof{
		Proof: &multivtypes.Proof_LeafConsensus{LeafConsensus: leafConsensus},
	})

	any, err := codectypes.NewAnyWithValue(&mp)
	if err != nil {
		panic(err)
	}
	bz, err := chain.App.AppCodec().Marshal(any)
	if err != nil {
		panic(err)
	}
	return bz
}

// QueryClientStateProof performs and abci query for a client state
// stored with a given clientID and returns the ClientState along with the proof
func (chain *TestChain) queryClientStateProof(clientID string, height int64) (exported.ClientState, []byte, clienttypes.Height) {
	// retrieve client state to provide proof for
	clientState, found := chain.App.GetIBCKeeper().ClientKeeper.GetClientState(chain.GetContext(), clientID)
	require.True(chain.t, found)

	clientKey := host.FullClientStateKey(clientID)
	proofClient, proofHeight := chain.queryProof(clientKey, height)

	return clientState, proofClient, proofHeight
}

// QueryConsensusStateProof performs an abci query for a consensus state
// stored on the given clientID. The proof and consensusHeight are returned.
func (chain *TestChain) queryConsensusStateProof(clientID string, height int64) ([]byte, clienttypes.Height, clienttypes.Height) {
	clientState := chain.GetClientState(clientID)

	consensusHeight := clientState.GetLatestHeight().(clienttypes.Height)
	consensusKey := host.FullConsensusStateKey(clientID, consensusHeight)
	proofConsensus, proofHeight := chain.queryProof(consensusKey, height)

	return proofConsensus, consensusHeight, proofHeight
}

// QueryProof performs an abci query with the given key and returns the proto encoded merkle proof
// for the query and the height at which the proof will succeed on a tendermint verifier.
func (chain *TestChain) queryProof(key []byte, height int64) ([]byte, clienttypes.Height) {
	res := chain.App.Query(abci.RequestQuery{
		Path:   fmt.Sprintf("store/%s/key", host.StoreKey),
		Height: height,
		Data:   key,
		Prove:  true,
	})

	merkleProof, err := commitmenttypes.ConvertProofs(res.ProofOps)
	require.NoError(chain.t, err)

	proof, err := chain.App.AppCodec().Marshal(&merkleProof)
	require.NoError(chain.t, err)

	revision := clienttypes.ParseChainID(chain.ChainID)

	// proof height + 1 is returned as the proof created corresponds to the height the proof
	// was created in the IAVL tree. Tendermint and subsequently the clients that rely on it
	// have heights 1 above the IAVL tree. Thus we return proof height + 1
	return proof, clienttypes.NewHeight(revision, uint64(res.Height)+1)
}
