package keeper_test

import (
	"testing"

	ibctesting "github.com/datachainlab/ibc-proxy/testing"
	"github.com/stretchr/testify/suite"
)

type KeeperTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain
	chainD *ibctesting.TestChain
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 4)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(0))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainC = suite.coordinator.GetChain(ibctesting.GetChainID(2))
	suite.chainD = suite.coordinator.GetChain(ibctesting.GetChainID(3))
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}
