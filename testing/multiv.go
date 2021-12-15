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
	clientType string, depth uint32,
) (string, error) {
	coord.CommitBlock(source, counterparty)
	clientID := source.NewClientID(clientType)
	if err := source.CreateMultiVClient(counterparty, clientID, clientType, depth); err != nil {
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
	clientType string,
	depth uint32,
) error {
	if clientType != exported.Tendermint {
		return fmt.Errorf("unsupported client type %v", clientType)
	}
	m := chain.ConstructMsgCreateClient(counterparty, clientID, clientType)

	consensusState, err := clienttypes.UnpackConsensusState(m.ConsensusState)
	if err != nil {
		return err
	}

	msg, err := clienttypes.NewMsgCreateClient(
		&multivtypes.ClientState{UnderlyingClientState: m.ClientState, Depth: depth},
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
// verification:
// 	c0 -> p0 -> c1
// 	c1 -> c0
// multi-proof: c1 -> c0 (head)-> p0 (leaf)-> c1
func (chain *TestChain) ConnectionOpenTryWithProxy(
	counterparty *TestChain,
	connection, counterpartyConnection *TestConnection,
	counterpartyProxy ProxyInfo,
) error {
	head := counterparty.QueryMultiVBranchProof(counterpartyConnection.ClientID)
	upstreamClientState, proofClient := counterpartyProxy.Chain.QueryMultiVLeafClientProof(head, counterpartyProxy.UpstreamClientID)
	_, proofConsensus, upstreamConsensusHeight := counterpartyProxy.Chain.QueryMultiVLeafConsensusProof(head, counterpartyProxy.UpstreamClientID)
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
// verification:
// 	c0 -> c1
// 	c1 -> p1 -> c0
// multi-proof: c0 -> c1 (head)-> p1 (leaf)-> c0
func (chain *TestChain) ConnectionOpenAckWithProxy(
	counterparty *TestChain,
	connection, counterpartyConnection *TestConnection,
	counterpartyProxy ProxyInfo,
) error {
	head := counterparty.QueryMultiVBranchProof(counterpartyConnection.ClientID)
	upstreamClientState, proofClient := counterpartyProxy.Chain.QueryMultiVLeafClientProof(head, counterpartyProxy.UpstreamClientID)
	_, proofConsensus, upstreamConsensusHeight := counterpartyProxy.Chain.QueryMultiVLeafConsensusProof(head, counterpartyProxy.UpstreamClientID)
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

func (chain *TestChain) QueryAnyClientStateProof(clientID string) (*codectypes.Any, exported.Height, []byte) {
	cs, proof, proofHeight := chain.queryClientStateProof(clientID, chain.App.LastBlockHeight()-1)
	any, err := clienttypes.PackClientState(cs)
	if err != nil {
		panic(err)
	}
	return any, proofHeight, proof
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

func (chain *TestChain) QueryMultiVBranchProof(clientID string) *multivtypes.Proof {
	proxyClientState, proofHeight, proxyProofClient := chain.QueryAnyClientStateProof(clientID)
	proxyConsensusState, proxyProofConsensus, consensusHeight := chain.QueryAnyConsensusStateProof(clientID)

	return &multivtypes.Proof{
		ClientProof:     proxyProofClient,
		ClientState:     proxyClientState,
		ConsensusProof:  proxyProofConsensus,
		ConsensusState:  proxyConsensusState,
		ConsensusHeight: consensusHeight,
		ProofHeight:     proofHeight.(clienttypes.Height),
	}
}

func (chain *TestChain) QueryMultiVLeafClientProof(head *multivtypes.Proof, upstreamClientID string) (exported.ClientState, []byte) {
	cs, err := clienttypes.UnpackClientState(head.ClientState)
	if err != nil {
		panic(err)
	}
	h := cs.GetLatestHeight()
	upstreamClientState, upstreamClientProof, upstreamProofHeight := chain.queryClientStateProof(upstreamClientID, int64(h.GetRevisionHeight())-1)
	leafClient := &multivtypes.LeafProof{
		Proof:       upstreamClientProof,
		ProofHeight: upstreamProofHeight,
	}
	proofClient := chain.makeMultiProof(head, nil, leafClient)
	return upstreamClientState, proofClient
}

func (chain *TestChain) QueryMultiVLeafConsensusProof(head *multivtypes.Proof, upstreamClientID string) (exported.ConsensusState, []byte, clienttypes.Height) {
	cs, err := clienttypes.UnpackClientState(head.ClientState)
	if err != nil {
		panic(err)
	}
	h := cs.GetLatestHeight()
	upstreamConsensusProof, upstreamConsensusHeight, upstreamProofHeight := chain.queryConsensusStateProof(upstreamClientID, int64(h.GetRevisionHeight())-1)
	leafConsensus := &multivtypes.LeafProof{
		Proof:       upstreamConsensusProof,
		ProofHeight: upstreamProofHeight,
	}
	proofConsensus := chain.makeMultiProof(head, nil, leafConsensus)
	consensusState, found := chain.GetConsensusState(upstreamClientID, upstreamConsensusHeight)
	if !found {
		panic("consensusState not found")
	}
	return consensusState, proofConsensus, upstreamConsensusHeight
}

func (chain *TestChain) makeMultiProof(
	head *multivtypes.Proof,
	branches []*multivtypes.Proof,
	leafClient *multivtypes.LeafProof,
) []byte {
	var mp multivtypes.MultiProof
	mp.Head = *head
	for _, branch := range branches {
		mp.Branches = append(mp.Branches, *branch)
	}
	mp.Leaf = *leafClient
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
