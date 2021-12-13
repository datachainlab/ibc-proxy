package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/modules/core/04-channel/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
	ibctesting "github.com/datachainlab/ibc-proxy/testing"
)

func (suite *KeeperTestSuite) TestMultiV() {
	clientBA, err := suite.coordinator.CreateMultiVClient(suite.chainB, suite.chainA, exported.Tendermint)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.coordinator.UpdateClient(suite.chainB, suite.chainA, clientBA, exported.Tendermint))
}

// A(C) -> B, B -> A
// A: downstream, B: upstream, C: proxy
func (suite *KeeperTestSuite) TestOneSideProxy1() {
	clientCB, err := suite.coordinator.InitProxy(suite.chainC, suite.chainB, exported.Tendermint, false)
	suite.Require().NoError(err)

	// XXX increment the sequence...
	suite.coordinator.CreateClient(suite.chainB, suite.chainA, exported.Tendermint)
	clientBA, err := suite.coordinator.CreateMultiVClient(suite.chainB, suite.chainA, exported.Tendermint)
	suite.Require().NoError(err)

	// setup proxy
	clientAC, err := suite.coordinator.SetupProxy(suite.chainA, suite.chainC, clientCB)
	suite.Require().NoError(err)

	ppair := ibctesting.ProxyPair{{suite.chainC, clientAC, clientCB, suite.chainB.GetPrefix()}, nil}
	connA, connB := suite.coordinator.CreateConnectionWithProxy(suite.chainA, suite.chainB, clientAC, clientBA, ibctesting.TransferVersion, ppair)
	chanA, chanB := suite.coordinator.CreateChannelWithProxy(suite.chainA, suite.chainB, connA, connB, ibctesting.TransferPort, ibctesting.TransferPort, channeltypes.UNORDERED, ppair)
	suite.testHandleMsgTransfer(connA, connB, chanA, chanB, ppair)
}

// A -> B, B(C) -> A
// A: upstream, B: downstream, C: proxy
func (suite *KeeperTestSuite) TestOneSideProxy2() {
	clientCA, err := suite.coordinator.InitProxy(suite.chainC, suite.chainA, exported.Tendermint, false)
	suite.Require().NoError(err)

	// XXX increment the sequence...
	suite.coordinator.CreateClient(suite.chainA, suite.chainB, exported.Tendermint)
	clientAB, err := suite.coordinator.CreateMultiVClient(suite.chainA, suite.chainB, exported.Tendermint)
	suite.Require().NoError(err)

	// downstream creates a proxy client
	clientBC, err := suite.coordinator.SetupProxy(suite.chainB, suite.chainC, clientCA)
	suite.Require().NoError(err)

	ppair := ibctesting.ProxyPair{nil, {suite.chainC, clientBC, clientCA, suite.chainA.GetPrefix()}}
	connA, connB := suite.coordinator.CreateConnectionWithProxy(suite.chainA, suite.chainB, clientAB, clientBC, ibctesting.TransferVersion, ppair)
	chanA, chanB := suite.coordinator.CreateChannelWithProxy(suite.chainA, suite.chainB, connA, connB, ibctesting.TransferPort, ibctesting.TransferPort, types.UNORDERED, ppair)
	suite.testHandleMsgTransfer(connA, connB, chanA, chanB, ppair)
}

// A(C) -> B, B(D) -> A
// A: upstream/downstream, B: downstream/upstream, C: proxy for A, D: proxy for B
func (suite *KeeperTestSuite) TestBothSideProxy() {
	clientCB, err := suite.coordinator.InitProxy(suite.chainC, suite.chainB, exported.Tendermint, true)
	suite.Require().NoError(err)

	clientDA, err := suite.coordinator.InitProxy(suite.chainD, suite.chainA, exported.Tendermint, true)
	suite.Require().NoError(err)

	clientAC, err := suite.coordinator.SetupProxy(suite.chainA, suite.chainC, clientCB)
	suite.Require().NoError(err)

	clientBD, err := suite.coordinator.SetupProxy(suite.chainB, suite.chainD, clientDA)
	suite.Require().NoError(err)

	ppair := ibctesting.ProxyPair{{suite.chainC, clientAC, clientCB, suite.chainB.GetPrefix()}, {suite.chainD, clientBD, clientDA, suite.chainA.GetPrefix()}}
	connA, connB := suite.coordinator.CreateConnectionWithProxy(suite.chainA, suite.chainB, clientAC, clientBD, ibctesting.TransferVersion, ppair)
	chanA, chanB := suite.coordinator.CreateChannelWithProxy(suite.chainA, suite.chainB, connA, connB, ibctesting.TransferPort, ibctesting.TransferPort, channeltypes.UNORDERED, ppair)
	suite.testHandleMsgTransfer(connA, connB, chanA, chanB, ppair)
}

func (suite *KeeperTestSuite) testHandleMsgTransfer(connA, connB *ibctesting.TestConnection, chanA, chanB *ibctesting.TestChannel, proxies ibctesting.ProxyPair) {
	timeoutHeight := clienttypes.NewHeight(0, 110)
	coinToSendToB := sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))

	msg := transfertypes.NewMsgTransfer(chanA.PortID, chanA.ID, coinToSendToB, suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), timeoutHeight, 0)
	err := suite.coordinator.SendPacketWithProxy(suite.chainA, suite.chainB, connA, connB, proxies, msg)
	suite.Require().NoError(err) // message committed

	// relay send
	fungibleTokenPacket := transfertypes.NewFungibleTokenPacketData(coinToSendToB.Denom, coinToSendToB.Amount.Uint64(), suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String())
	packet := channeltypes.NewPacket(fungibleTokenPacket.GetBytes(), 1, chanA.PortID, chanA.ID, chanB.PortID, chanB.ID, timeoutHeight, 0)
	ack := channeltypes.NewResultAcknowledgement([]byte{byte(1)})

	err = suite.coordinator.RecvPacketWithProxy(
		suite.chainB, suite.chainA, connB, connA, packet, proxies.Swap(),
	)
	suite.Require().NoError(err) // relay committed

	err = suite.coordinator.AcknowledgePacketWithProxy(
		suite.chainA, suite.chainB, chanA, chanB, connA, connB, packet, ack.Acknowledgement(), proxies,
	)
	suite.Require().NoError(err) // ack committed
}
