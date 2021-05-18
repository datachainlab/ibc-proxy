package keeper_test

import (
	"github.com/cosmos/ibc-go/modules/core/exported"
	ibctesting "github.com/datachainlab/ibc-proxy/testing"
)

// A(C) -> B, B -> A
// A: downstream, B: upstream, C: proxy
func (suite *KeeperTestSuite) TestConnection1() {
	clientCB, err := suite.coordinator.InitProxy(suite.chainC, suite.chainB, exported.Tendermint)
	suite.Require().NoError(err)

	// XXX increment the sequence...
	suite.coordinator.CreateClient(suite.chainB, suite.chainA, exported.Tendermint)
	clientBA, err := suite.coordinator.CreateClient(suite.chainB, suite.chainA, exported.Tendermint)
	suite.Require().NoError(err)

	// setup proxy
	clientAC, err := suite.coordinator.SetupProxy(suite.chainA, suite.chainC, clientCB)
	suite.Require().NoError(err)

	ppair := ibctesting.ProxyPair{{suite.chainC, clientAC, clientCB}, nil}
	suite.coordinator.CreateConnectionWithProxy(suite.chainA, suite.chainB, clientAC, clientBA, ppair)
}

// A -> B, B(C) -> A
// A: upstream, B: downstream, C: proxy
func (suite *KeeperTestSuite) TestConnection2() {
	clientCA, err := suite.coordinator.InitProxy(suite.chainC, suite.chainA, exported.Tendermint)
	suite.Require().NoError(err)

	// XXX increment the sequence...
	suite.coordinator.CreateClient(suite.chainA, suite.chainB, exported.Tendermint)
	clientAB, err := suite.coordinator.CreateClient(suite.chainA, suite.chainB, exported.Tendermint)
	suite.Require().NoError(err)

	// downstream creates a proxy client
	clientBC, err := suite.coordinator.SetupProxy(suite.chainB, suite.chainC, clientCA)
	suite.Require().NoError(err)

	ppair := ibctesting.ProxyPair{nil, {suite.chainC, clientBC, clientCA}}
	suite.coordinator.CreateConnectionWithProxy(suite.chainA, suite.chainB, clientAB, clientBC, ppair)
}

// A(C) -> B, B(D) -> A
// A: upstream/downstream, B: downstream/upstream, C: proxy for A, D: proxy for B
func (suite *KeeperTestSuite) TestConnection3() {
	clientCB, err := suite.coordinator.InitProxy(suite.chainC, suite.chainB, exported.Tendermint)
	suite.Require().NoError(err)

	clientDA, err := suite.coordinator.InitProxy(suite.chainD, suite.chainA, exported.Tendermint)
	suite.Require().NoError(err)

	clientAC, err := suite.coordinator.SetupProxy(suite.chainA, suite.chainC, clientCB)
	suite.Require().NoError(err)

	clientBD, err := suite.coordinator.SetupProxy(suite.chainB, suite.chainD, clientDA)
	suite.Require().NoError(err)

	ppair := ibctesting.ProxyPair{{suite.chainC, clientAC, clientCB}, {suite.chainD, clientBD, clientDA}}
	suite.coordinator.CreateConnectionWithProxy(suite.chainA, suite.chainB, clientAC, clientBD, ppair)
}
