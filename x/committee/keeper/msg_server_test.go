package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/0glabs/0g-chain/app"
	"github.com/0glabs/0g-chain/chaincfg"
	"github.com/0glabs/0g-chain/x/committee/keeper"
	"github.com/0glabs/0g-chain/x/committee/types"
)

//NewDistributionGenesisWithPool creates a default distribution genesis state with some coins in the community pool.
//func NewDistributionGenesisWithPool(communityPoolCoins sdk.Coins) app.GenesisState {
//gs := distribution.DefaultGenesisState()
//gs.FeePool = distribution.FeePool{CommunityPool: sdk.NewDecCoinsFromCoins(communityPoolCoins...)}
//return app.GenesisState{distribution.ModuleName: distribution.ModuleCdc.MustMarshalJSON(gs)}
//}

type MsgServerTestSuite struct {
	suite.Suite

	app       app.TestApp
	keeper    keeper.Keeper
	msgServer types.MsgServer
	ctx       sdk.Context
	addresses []sdk.AccAddress

	communityPoolAmt sdk.Coins
}

func (suite *MsgServerTestSuite) SetupTest() {
	_, suite.addresses = app.GeneratePrivKeyAddressPairs(5)
	suite.app = app.NewTestApp()
	suite.keeper = suite.app.GetCommitteeKeeper()
	suite.msgServer = keeper.NewMsgServerImpl(suite.keeper)
	encodingCfg := app.MakeEncodingConfig()
	cdc := encodingCfg.Marshaler

	memberCommittee, err := types.NewMemberCommittee(
		1,
		"This committee is for testing.",
		suite.addresses[:3],
		[]types.Permission{&types.GodPermission{}},
		sdk.MustNewDecFromStr("0.5"),
		time.Hour*24*7,
		types.TALLY_OPTION_FIRST_PAST_THE_POST,
	)
	suite.Require().NoError(err)

	firstBlockTime := time.Date(1998, time.January, 1, 1, 0, 0, 0, time.UTC)
	testGenesis := types.NewGenesisState(
		3,
		[]types.Committee{memberCommittee},
		[]types.Proposal{},
		[]types.Vote{},
	)
	suite.communityPoolAmt = sdk.NewCoins(chaincfg.MakeCoinForEvmDenom(1000000000000000))
	suite.app.InitializeFromGenesisStates(
		app.GenesisState{types.ModuleName: cdc.MustMarshalJSON(testGenesis)},
		// TODO: not used?
		// NewDistributionGenesisWithPool(suite.communityPoolAmt),
	)
	suite.ctx = suite.app.NewContext(true, tmproto.Header{Height: 1, Time: firstBlockTime})
}

func (suite *MsgServerTestSuite) TestSubmitProposalMsg_ValidUpgrade() {
	msg, err := types.NewMsgSubmitProposal(
		upgradetypes.NewSoftwareUpgradeProposal(
			"A Title",
			"A description of this proposal.",
			upgradetypes.Plan{
				Name:   "emergency-shutdown-1", // identifier for the upgrade
				Height: 100000,
				Info:   "Some information about the shutdown.",
			},
		),
		suite.addresses[0],
		1,
	)
	suite.Require().NoError(err)

	res, err := suite.msgServer.SubmitProposal(sdk.WrapSDKContext(suite.ctx), msg)

	suite.NoError(err)
	_, found := suite.keeper.GetProposal(suite.ctx, res.ProposalID)
	suite.True(found)
}

// TODO: create a unregisted proto for tests?
func (suite *MsgServerTestSuite) TestSubmitProposalMsg_Unregistered() {
	var committeeID uint64 = 1
	msg, err := types.NewMsgSubmitProposal(
		&UnregisteredPubProposal{},
		suite.addresses[0],
		committeeID,
	)
	suite.Require().NoError(err)

	_, err = suite.msgServer.SubmitProposal(sdk.WrapSDKContext(suite.ctx), msg)

	suite.Error(err)
	suite.Empty(
		suite.keeper.GetProposalsByCommittee(suite.ctx, committeeID),
		"proposal found when none should exist",
	)
}

func TestMsgServerTestSuite(t *testing.T) {
	suite.Run(t, new(MsgServerTestSuite))
}
