package ibctesting

import (
	"fmt"

	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/modules/core/03-connection/types"
	"github.com/cosmos/ibc-go/modules/core/04-channel/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/exported"
	proxytypes "github.com/datachainlab/ibc-proxy/modules/proxy/types"
	"github.com/datachainlab/ibc-proxy/testing/simapp"
	"github.com/stretchr/testify/require"
)

func (coord *Coordinator) InitProxy(
	source, counterparty *TestChain,
	clientType string,
) (clientID string, err error) {
	clientID, err = coord.CreateClient(source, counterparty, clientType)
	if err != nil {
		return clientID, err
	}
	err = source.App.(*simapp.SimApp).IBCProxyKeeper.EnableProxy(source.GetContext(), clientID)
	coord.CommitBlock(source)
	return clientID, err
}

// steps:
// - setup connection
// - setup channel (port=proxy)
// - relay the proxy request packet
func (coord *Coordinator) SetupProxy(
	downstream, proxy *TestChain,
	upstreamClientID string,
) (string, error) {
	// A: downstream, B: proxy

	clientA, clientB := coord.SetupClients(downstream, proxy, exported.Tendermint)
	connA, connB := coord.CreateConnection(downstream, proxy, clientA, clientB)
	chanA, _ := coord.CreateChannel(downstream, proxy, connA, connB, proxytypes.PortID, proxytypes.PortID, types.UNORDERED)

	// begin proxy handshake

	packet, proxyClient, err := downstream.App.(*simapp.SimApp).IBCProxyKeeper.SendProxyRequest(
		downstream.GetContext(),
		chanA.PortID, chanA.ID,
		upstreamClientID,
		clienttypes.NewHeight(0, 110), 0,
	)
	if err != nil {
		return "", err
	}

	ack := proxy.App.(*simapp.SimApp).IBCProxyKeeper.OnRecvPacket(proxy.GetContext(), *packet)
	err = downstream.App.(*simapp.SimApp).IBCProxyKeeper.OnAcknowledgementPacket(
		downstream.GetContext(), *packet, ack.(channeltypes.Acknowledgement),
	)
	if err != nil {
		return proxyClient, err
	}

	coord.CommitBlock(downstream, proxy)

	return proxyClient, nil
}

func (chain *TestChain) UpdateProxyClient(proxy *TestChain, proxyClientID string) error {
	header, err := chain.ConstructUpdateTMClientHeader(proxy, proxyClientID)
	if err != nil {
		return err
	}
	msg, err := clienttypes.NewMsgUpdateClient(proxyClientID, header, chain.SenderAccount.GetAddress().String())
	if err != nil {
		return err
	}
	return chain.sendMsgs(msg)
}

func (coord *Coordinator) CreateConnectionWithProxy(
	chainA, chainB *TestChain,
	clientA, clientB string,
	proxies ProxyPair,
) (*TestConnection, *TestConnection) {

	connA, connB, err := coord.ConnOpenInitWithProxy(chainA, chainB, clientA, clientB, proxies)
	require.NoError(coord.t, err)

	err = coord.ConnOpenTryWithProxy(chainB, chainA, connB, connA, proxies.Swap())
	require.NoError(coord.t, err)

	err = coord.ConnOpenAckWithProxy(chainA, chainB, connA, connB, proxies)
	require.NoError(coord.t, err)

	err = coord.ConnOpenConfirmWithProxy(chainB, chainA, connB, connA, proxies.Swap())
	require.NoError(coord.t, err)

	return connA, connB
}

type ProxyInfo struct {
	Chain            *TestChain
	ClientID         string
	UpstreamClientID string
}

type ProxyPair [2]*ProxyInfo

func (pair ProxyPair) Swap() ProxyPair {
	pair[0], pair[1] = pair[1], pair[0]
	return pair
}

func (coord *Coordinator) ConnOpenInitWithProxy(
	source, counterparty *TestChain,
	clientID, counterpartyClientID string, proxies ProxyPair,
) (*TestConnection, *TestConnection, error) {
	if proxies[1] == nil {
		return coord.ConnOpenInit(source, counterparty, clientID, counterpartyClientID)
	}

	sourceConnection := source.AddTestConnection(clientID, counterpartyClientID)
	counterpartyConnection := counterparty.AddTestConnection(counterpartyClientID, clientID)

	// initialize connection on source
	if err := source.ConnectionOpenInit(counterparty, sourceConnection, counterpartyConnection); err != nil {
		return sourceConnection, counterpartyConnection, err
	}
	coord.IncrementTime()

	// update source client on counterparty connection
	if err := coord.UpdateClient(
		proxies[1].Chain, source,
		proxies[1].UpstreamClientID, exported.Tendermint,
	); err != nil {
		return sourceConnection, counterpartyConnection, err
	}

	return sourceConnection, counterpartyConnection, nil
}

func (coord *Coordinator) ConnOpenTryWithProxy(
	source, counterparty *TestChain,
	sourceConnection, counterpartyConnection *TestConnection,
	proxies ProxyPair,
) error {
	if proxies[0] == nil {
		if err := source.ConnectionOpenTry(counterparty, sourceConnection, counterpartyConnection); err != nil {
			return err
		}
		coord.IncrementTime()
	} else {
		// source: downstream, counterparty: upstream
		proxy := proxies[0].Chain

		proxyConnection := connectiontypes.NewConnectionEnd(
			connectiontypes.INIT,
			counterpartyConnection.ClientID,
			connectiontypes.Counterparty{
				ClientId:     proxies[0].ClientID,
				ConnectionId: "",
				Prefix:       counterparty.GetPrefix(),
			},
			[]*connectiontypes.Version{ConnectionVersion},
			DefaultDelayPeriod,
		)

		counterpartyClient, proofClient := counterparty.QueryClientStateProof(counterpartyConnection.ClientID)

		connectionKey := host.ConnectionKey(counterpartyConnection.ID)
		proofInit, proofHeight := counterparty.QueryProof(connectionKey)

		proofConsensus, consensusHeight := counterparty.QueryConsensusStateProof(counterpartyConnection.ClientID)

		consensusState, found := counterparty.GetConsensusState(counterpartyConnection.ClientID, consensusHeight)
		if !found {
			return fmt.Errorf("consensusState '%v-%v' not found", counterpartyConnection.ClientID, consensusHeight)
		}

		err := proxy.App.(*simapp.SimApp).IBCProxyKeeper.ConnOpenTry(
			proxy.GetContext(),
			counterpartyConnection.ID,
			proxies[0].UpstreamClientID,
			proxyConnection,
			counterpartyClient, proofInit, proofClient, proofConsensus, proofHeight, consensusHeight, consensusState,
		)
		if err != nil {
			return err
		}
		coord.CommitBlock(proxy)
		coord.CommitBlock(proxy)

		if err := source.UpdateProxyClient(proxy, proxies[0].ClientID); err != nil {
			return err
		}
		coord.CommitBlock(source)

		{
			client, proofClient := proxy.QueryProxiedClientStateProof(counterpartyConnection.ClientID, proxies[0].UpstreamClientID)
			proofInit, proofHeight := proxy.QueryProxiedConnectionStateProof(counterpartyConnection.ID, proxies[0].UpstreamClientID)
			proofConsensus, consensusHeight := proxy.QueryProxiedConsensusStateProof(counterpartyConnection.ClientID, proxies[0].UpstreamClientID)

			msg := connectiontypes.NewMsgConnectionOpenTry(
				"", sourceConnection.ClientID, counterpartyConnection.ID, counterpartyConnection.ClientID, client, // testing doesn't use flexible selection
				counterparty.GetPrefix(), []*connectiontypes.Version{ConnectionVersion}, DefaultDelayPeriod,
				proofInit, proofClient, proofConsensus,
				proofHeight, consensusHeight,
				source.SenderAccount.GetAddress().String(),
			)
			if _, err := source.SendMsgs(msg); err != nil {
				return err
			}
			coord.CommitBlock(source)
		}
	}

	if proxies[1] == nil {
		return coord.UpdateClient(
			counterparty, source, counterpartyConnection.ClientID, exported.Tendermint,
		)
	} else {
		return coord.UpdateClient(
			proxies[1].Chain, source, proxies[1].UpstreamClientID, exported.Tendermint,
		)
	}
}

func (coord *Coordinator) ConnOpenAckWithProxy(
	source, counterparty *TestChain,
	sourceConnection, counterpartyConnection *TestConnection,
	proxies ProxyPair,
) error {
	if proxies[0] == nil {
		if err := source.ConnectionOpenAck(counterparty, sourceConnection, counterpartyConnection); err != nil {
			return err
		}
		coord.IncrementTime()
	} else {
		// source: downstream, counterparty: upstream
		proxy := proxies[0].Chain

		proxyConnection := connectiontypes.NewConnectionEnd(
			connectiontypes.TRYOPEN,
			counterpartyConnection.ClientID,
			connectiontypes.Counterparty{
				ClientId:     sourceConnection.ClientID,
				ConnectionId: sourceConnection.ID,
				Prefix:       counterparty.GetPrefix(),
			},
			[]*connectiontypes.Version{ConnectionVersion},
			DefaultDelayPeriod,
		)

		{
			counterpartyClient, proofClient := counterparty.QueryClientStateProof(counterpartyConnection.ClientID)

			connectionKey := host.ConnectionKey(counterpartyConnection.ID)
			proofTry, proofHeight := counterparty.QueryProof(connectionKey)

			proofConsensus, consensusHeight := counterparty.QueryConsensusStateProof(counterpartyConnection.ClientID)

			consensusState, found := counterparty.GetConsensusState(counterpartyConnection.ClientID, consensusHeight)
			if !found {
				return fmt.Errorf("consensusState '%v-%v' not found", counterpartyConnection.ClientID, consensusHeight)
			}

			err := proxy.App.(*simapp.SimApp).IBCProxyKeeper.ConnOpenACK(
				proxy.GetContext(),
				counterpartyConnection.ID,
				proxies[0].UpstreamClientID,
				proxyConnection,
				counterpartyClient,
				ConnectionVersion,
				proofTry,
				proofClient,
				proofConsensus,
				proofHeight,
				consensusHeight,
				consensusState,
			)
			if err != nil {
				return err
			}
			coord.CommitBlock(proxy)
			coord.CommitBlock(proxy)
		}

		if err := source.UpdateProxyClient(proxy, proxies[0].ClientID); err != nil {
			return err
		}
		coord.CommitBlock(source)
		// XXX
		coord.CommitBlock(source)

		{ // callerA calls connOpenAck with proxied proof
			client, proofClient := proxy.QueryProxiedClientStateProof(counterpartyConnection.ClientID, proxies[0].UpstreamClientID)
			proofTry, proofHeight := proxy.QueryProxiedConnectionStateProof(counterpartyConnection.ID, proxies[0].UpstreamClientID)
			proofConsensus, consensusHeight := proxy.QueryProxiedConsensusStateProof(counterpartyConnection.ClientID, proxies[0].UpstreamClientID)

			msg := connectiontypes.NewMsgConnectionOpenAck(
				sourceConnection.ID, counterpartyConnection.ID, client, // testing doesn't use flexible selection
				proofTry, proofClient, proofConsensus,
				proofHeight, consensusHeight,
				proxyConnection.Versions[0],
				source.SenderAccount.GetAddress().String(),
			)
			if _, err := source.SendMsgs(msg); err != nil {
				return err
			}
			coord.CommitBlock(source)
		}
	}

	if proxies[1] == nil {
		return coord.UpdateClient(
			counterparty, source, counterpartyConnection.ClientID, exported.Tendermint,
		)
	} else {
		return coord.UpdateClient(
			proxies[1].Chain, source, proxies[1].UpstreamClientID, exported.Tendermint,
		)
	}
}

func (coord *Coordinator) ConnOpenConfirmWithProxy(
	source, counterparty *TestChain,
	sourceConnection, counterpartyConnection *TestConnection,
	proxies ProxyPair,
) error {
	if proxies[0] == nil {
		if err := source.ConnectionOpenConfirm(counterparty, sourceConnection, counterpartyConnection); err != nil {
			return err
		}
		coord.IncrementTime()
	} else {
		// source: downstream, counterparty: upstream
		proxy := proxies[0].Chain

		proxyConnection := connectiontypes.NewConnectionEnd(
			connectiontypes.OPEN,
			counterpartyConnection.ClientID,
			connectiontypes.Counterparty{
				ClientId:     sourceConnection.ClientID,
				ConnectionId: sourceConnection.ID,
				Prefix:       counterparty.GetPrefix(),
			},
			[]*connectiontypes.Version{ConnectionVersion},
			DefaultDelayPeriod,
		)

		{
			connectionKey := host.ConnectionKey(counterpartyConnection.ID)
			proofAck, proofHeight := counterparty.QueryProof(connectionKey)

			err := proxy.App.(*simapp.SimApp).IBCProxyKeeper.ConnOpenConfirm(
				proxy.GetContext(),
				counterpartyConnection.ID,
				proxies[0].UpstreamClientID,
				proxyConnection,
				proofAck,
				proofHeight,
			)
			if err != nil {
				return err
			}
			coord.CommitBlock(proxy)
			coord.CommitBlock(proxy)
		}

		if err := source.UpdateProxyClient(proxy, proxies[0].ClientID); err != nil {
			return err
		}
		coord.CommitBlock(source)

		{
			proofAck, proofHeight := proxy.QueryProxiedConnectionStateProof(counterpartyConnection.ID, proxies[0].UpstreamClientID)

			msg := connectiontypes.NewMsgConnectionOpenConfirm(
				sourceConnection.ID, proofAck, proofHeight,
				source.SenderAccount.GetAddress().String(),
			)
			if _, err := source.SendMsgs(msg); err != nil {
				return err
			}
			coord.CommitBlock(source)
		}
	}

	if proxies[1] == nil {
		return coord.UpdateClient(
			counterparty, source, counterpartyConnection.ClientID, exported.Tendermint,
		)
	} else {
		return coord.UpdateClient(
			proxies[1].Chain, source, proxies[1].UpstreamClientID, exported.Tendermint,
		)
	}
}

func (chain *TestChain) QueryProxiedClientStateProof(clientID string, upstreamClientID string) (exported.ClientState, []byte) {
	// retrieve client state to provide proof for
	clientState, found := chain.App.(*simapp.SimApp).IBCProxyKeeper.GetClientStateCommitment(
		chain.GetContext(),
		clientID,
		upstreamClientID,
	)
	require.True(chain.t, found)

	clientKey := withProxyPrefix(upstreamClientID, host.FullClientStateKey(clientID))

	proofClient, _ := chain.QueryProof(clientKey)

	return clientState, proofClient
}

func (chain *TestChain) QueryProxiedConsensusStateProof(clientID string, upstreamClientID string) ([]byte, clienttypes.Height) {
	clientState, found := chain.App.(*simapp.SimApp).IBCProxyKeeper.GetClientStateCommitment(
		chain.GetContext(),
		clientID,
		upstreamClientID,
	)
	require.True(chain.t, found)
	consensusHeight := clientState.GetLatestHeight().(clienttypes.Height)
	consensusKey := withProxyPrefix(upstreamClientID, host.FullConsensusStateKey(clientID, consensusHeight))
	proofConsensus, _ := chain.QueryProof(consensusKey)

	return proofConsensus, consensusHeight
}

func (chain *TestChain) QueryProxiedConnectionStateProof(connectionID string, upstreamClientID string) ([]byte, clienttypes.Height) {
	connectionKey := withProxyPrefix(upstreamClientID, host.ConnectionKey(connectionID))
	return chain.QueryProof(connectionKey)
}

func withProxyPrefix(upstreamClientID string, key []byte) []byte {
	return append([]byte(upstreamClientID+"/"), key...)
}
