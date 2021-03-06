package thorchain

import (
	"encoding/json"

	"github.com/blang/semver"
	sdk "github.com/cosmos/cosmos-sdk/types"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keep"
)

type HandlerOutboundTxSuite struct{}

type TestOutboundTxKeeper struct {
	KVStoreDummy
	activeNodeAccount NodeAccount
	vault             Vault
}

func (k *TestOutboundTxKeeper) GetNodeAccount(_ sdk.Context, addr sdk.AccAddress) (NodeAccount, error) {
	if k.activeNodeAccount.NodeAddress.Equals(addr) {
		return k.activeNodeAccount, nil
	}
	return NodeAccount{}, nil
}

var _ = Suite(&HandlerOutboundTxSuite{})

func (s *HandlerOutboundTxSuite) SetUpSuite(c *C) {
	SetupConfigForTest()
}

func (s *HandlerOutboundTxSuite) TestValidate(c *C) {
	ctx, _ := setupKeeperForTest(c)

	keeper := &TestOutboundTxKeeper{
		activeNodeAccount: GetRandomNodeAccount(NodeActive),
		vault:             GetRandomVault(),
	}

	handler := NewOutboundTxHandler(keeper, NewVersionedEventMgr())

	addr, err := keeper.vault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)

	ver := constants.SWVersion

	tx := NewObservedTx(common.Tx{
		ID:          GetRandomTxHash(),
		Chain:       common.BNBChain,
		Coins:       common.Coins{common.NewCoin(common.BNBAsset, sdk.NewUint(1*common.One))},
		Memo:        "",
		FromAddress: GetRandomBNBAddress(),
		ToAddress:   addr,
		Gas:         BNBGasFeeSingleton,
	}, 12, GetRandomPubKey())

	msgOutboundTx := NewMsgOutboundTx(tx, tx.Tx.ID, keeper.activeNodeAccount.NodeAddress)
	sErr := handler.validate(ctx, msgOutboundTx, ver)
	c.Assert(sErr, IsNil)

	// invalid version
	sErr = handler.validate(ctx, msgOutboundTx, semver.Version{})
	c.Assert(sErr, Equals, errBadVersion)

	// invalid msg
	msgOutboundTx = MsgOutboundTx{}
	sErr = handler.validate(ctx, msgOutboundTx, ver)
	c.Assert(sErr, NotNil)

	// not signed observer
	msgOutboundTx = NewMsgOutboundTx(tx, tx.Tx.ID, GetRandomBech32Addr())
	sErr = handler.validate(ctx, msgOutboundTx, ver)
	c.Assert(sErr.Code(), Equals, sdk.CodeUnauthorized)
}

type outboundTxHandlerTestHelper struct {
	ctx           sdk.Context
	pool          Pool
	version       semver.Version
	keeper        *outboundTxHandlerKeeperHelper
	asgardVault   Vault
	yggVault      Vault
	constAccessor constants.ConstantValues
	nodeAccount   NodeAccount
	inboundTx     ObservedTx
	toi           *TxOutItem
	event         Event
}

type outboundTxHandlerKeeperHelper struct {
	Keeper
	observeTxVoterErrHash common.TxID
	failGetPendingEvent   bool
	errGetTxOut           bool
	errGetNodeAccount     bool
	errGetPool            bool
	errSetPool            bool
	errSetNodeAccount     bool
	errGetVaultData       bool
	errSetVaultData       bool
	vault                 Vault
}

func newOutboundTxHandlerKeeperHelper(keeper keep.Keeper) *outboundTxHandlerKeeperHelper {
	return &outboundTxHandlerKeeperHelper{
		Keeper:                keeper,
		observeTxVoterErrHash: GetRandomTxHash(),
	}
}

func (k *outboundTxHandlerKeeperHelper) GetObservedTxVoter(ctx sdk.Context, hash common.TxID) (ObservedTxVoter, error) {
	if hash.Equals(k.observeTxVoterErrHash) {
		return ObservedTxVoter{}, kaboom
	}
	return k.Keeper.GetObservedTxVoter(ctx, hash)
}

func (k *outboundTxHandlerKeeperHelper) GetPendingEventID(ctx sdk.Context, hash common.TxID) ([]int64, error) {
	if k.failGetPendingEvent {
		return nil, kaboom
	}
	return k.Keeper.GetPendingEventID(ctx, hash)
}

func (k *outboundTxHandlerKeeperHelper) GetTxOut(ctx sdk.Context, height int64) (*TxOut, error) {
	if k.errGetTxOut {
		return nil, kaboom
	}
	return k.Keeper.GetTxOut(ctx, height)
}

func (k *outboundTxHandlerKeeperHelper) GetNodeAccountByPubKey(ctx sdk.Context, pk common.PubKey) (NodeAccount, error) {
	if k.errGetNodeAccount {
		return NodeAccount{}, kaboom
	}
	return k.Keeper.GetNodeAccountByPubKey(ctx, pk)
}

func (k *outboundTxHandlerKeeperHelper) GetPool(ctx sdk.Context, asset common.Asset) (Pool, error) {
	if k.errGetPool {
		return NewPool(), kaboom
	}
	return k.Keeper.GetPool(ctx, asset)
}

func (k *outboundTxHandlerKeeperHelper) SetPool(ctx sdk.Context, pool Pool) error {
	if k.errSetPool {
		return kaboom
	}
	return k.Keeper.SetPool(ctx, pool)
}

func (k *outboundTxHandlerKeeperHelper) SetNodeAccount(ctx sdk.Context, na NodeAccount) error {
	if k.errSetNodeAccount {
		return kaboom
	}
	return k.Keeper.SetNodeAccount(ctx, na)
}

func (k *outboundTxHandlerKeeperHelper) GetAsgardVaultsByStatus(ctx sdk.Context, status VaultStatus) (Vaults, error) {
	return k.Keeper.GetAsgardVaultsByStatus(ctx, status)
}

func (k *outboundTxHandlerKeeperHelper) GetVault(ctx sdk.Context, _ common.PubKey) (Vault, error) {
	return k.vault, nil
}

func (k *outboundTxHandlerKeeperHelper) SetVault(ctx sdk.Context, v Vault) error {
	k.vault = v
	return nil
}

func (k *outboundTxHandlerKeeperHelper) GetVaultData(ctx sdk.Context) (VaultData, error) {
	if k.errGetVaultData {
		return VaultData{}, kaboom
	}
	return k.Keeper.GetVaultData(ctx)
}

func (k *outboundTxHandlerKeeperHelper) SetVaultData(ctx sdk.Context, data VaultData) error {
	if k.errSetVaultData {
		return kaboom
	}
	return k.Keeper.SetVaultData(ctx, data)
}

// newOutboundTxHandlerTestHelper setup all the basic condition to test OutboundTxHandler
func newOutboundTxHandlerTestHelper(c *C) outboundTxHandlerTestHelper {
	ctx, k := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(1023)
	pool := NewPool()
	pool.Asset = common.BNBAsset
	pool.BalanceAsset = sdk.NewUint(100 * common.One)
	pool.BalanceRune = sdk.NewUint(100 * common.One)

	version := constants.SWVersion
	asgardVault := GetRandomVault()
	c.Assert(k.SetVault(ctx, asgardVault), IsNil)
	addr, err := asgardVault.PubKey.GetAddress(common.RuneAsset().Chain)
	yggVault := GetRandomVault()
	c.Assert(err, IsNil)

	tx := NewObservedTx(common.Tx{
		ID:          GetRandomTxHash(),
		Chain:       common.RuneAsset().Chain,
		Coins:       common.Coins{common.NewCoin(common.BNBAsset, sdk.NewUint(1*common.One))},
		Memo:        "SWAP:" + common.RuneAsset().String(),
		FromAddress: GetRandomRUNEAddress(),
		ToAddress:   addr,
		Gas:         BNBGasFeeSingleton,
	}, 12, GetRandomPubKey())

	voter := NewObservedTxVoter(tx.Tx.ID, make(ObservedTxs, 0))
	keeper := newOutboundTxHandlerKeeperHelper(k)
	voter.Height = ctx.BlockHeight()
	keeper.SetObservedTxVoter(ctx, voter)

	nodeAccount := GetRandomNodeAccount(NodeActive)
	nodeAccount.NodeAddress, err = yggVault.PubKey.GetThorAddress()
	c.Assert(err, IsNil)
	nodeAccount.Bond = sdk.NewUint(100 * common.One)
	nodeAccount.PubKeySet = common.NewPubKeySet(yggVault.PubKey, yggVault.PubKey)
	c.Assert(keeper.SetNodeAccount(ctx, nodeAccount), IsNil)

	c.Assert(keeper.SetPool(ctx, pool), IsNil)

	txOutStorage := NewTxOutStorageV1(keeper, NewEventMgr())
	constAccessor := constants.GetConstantValues(version)
	txOutStorage.NewBlock(ctx.BlockHeight(), constAccessor)
	toi := &TxOutItem{
		Chain:       common.BNBChain,
		ToAddress:   tx.Tx.FromAddress,
		VaultPubKey: yggVault.PubKey,
		Coin:        common.NewCoin(common.BNBAsset, sdk.NewUint(2*common.One)),
		Memo:        NewOutboundMemo(tx.Tx.ID).String(),
		InHash:      tx.Tx.ID,
	}
	result, err := txOutStorage.TryAddTxOutItem(ctx, toi)
	c.Assert(err, IsNil)
	c.Check(result, Equals, true)

	swapEvent := NewEventSwap(common.BNBAsset, sdk.NewUint(common.One), sdk.NewUint(common.One), sdk.NewUint(common.One), sdk.NewUint(common.One), GetRandomTx())
	buf, err := json.Marshal(swapEvent)
	c.Assert(err, IsNil)
	e := NewEvent(swapEvent.Type(), ctx.BlockHeight(), tx.Tx, buf, EventPending)
	c.Assert(keeper.UpsertEvent(ctx, e), IsNil)
	c.Assert(err, IsNil)
	c.Assert(result, Equals, true)
	return outboundTxHandlerTestHelper{
		ctx:           ctx,
		pool:          pool,
		version:       version,
		keeper:        keeper,
		asgardVault:   asgardVault,
		yggVault:      yggVault,
		nodeAccount:   nodeAccount,
		inboundTx:     tx,
		toi:           toi,
		constAccessor: constAccessor,
	}
}

func (s *HandlerOutboundTxSuite) TestOutboundTxHandlerShouldUpdateTxOut(c *C) {
	testCases := []struct {
		name           string
		messageCreator func(helper outboundTxHandlerTestHelper, tx ObservedTx) sdk.Msg
		runner         func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg sdk.Msg) sdk.Result
		expectedResult sdk.CodeType
	}{
		{
			name: "invalid message should return an error",
			messageCreator: func(helper outboundTxHandlerTestHelper, tx ObservedTx) sdk.Msg {
				return NewMsgNoOp(GetRandomObservedTx(), helper.nodeAccount.NodeAddress)
			},
			runner: func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg sdk.Msg) sdk.Result {
				return handler.Run(helper.ctx, msg, helper.version, helper.constAccessor)
			},
			expectedResult: CodeInvalidMessage,
		},
		{
			name: "if the version is lower than expected, it should return an error",
			messageCreator: func(helper outboundTxHandlerTestHelper, tx ObservedTx) sdk.Msg {
				return NewMsgOutboundTx(tx, tx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg sdk.Msg) sdk.Result {
				return handler.Run(helper.ctx, msg, semver.MustParse("0.0.1"), helper.constAccessor)
			},
			expectedResult: CodeBadVersion,
		},
		{
			name: "create a outbound tx with invalid observer account",
			messageCreator: func(helper outboundTxHandlerTestHelper, tx ObservedTx) sdk.Msg {
				return NewMsgOutboundTx(tx, tx.Tx.ID, GetRandomNodeAccount(NodeActive).NodeAddress)
			},
			runner: func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg sdk.Msg) sdk.Result {
				return handler.Run(helper.ctx, msg, semver.MustParse("0.2.0"), helper.constAccessor)
			},
			expectedResult: sdk.CodeUnauthorized,
		},
		{
			name: "fail to get observed TxVoter should result in an error",
			messageCreator: func(helper outboundTxHandlerTestHelper, tx ObservedTx) sdk.Msg {
				return NewMsgOutboundTx(tx, helper.keeper.observeTxVoterErrHash, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg sdk.Msg) sdk.Result {
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: sdk.CodeInternal,
		},
		{
			name: "fail to complete events should result in an error",
			messageCreator: func(helper outboundTxHandlerTestHelper, tx ObservedTx) sdk.Msg {
				return NewMsgOutboundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg sdk.Msg) sdk.Result {
				helper.keeper.failGetPendingEvent = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: sdk.CodeInternal,
		},
		{
			name: "fail to get txout should result in an error",
			messageCreator: func(helper outboundTxHandlerTestHelper, tx ObservedTx) sdk.Msg {
				return NewMsgOutboundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg sdk.Msg) sdk.Result {
				helper.keeper.errGetTxOut = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: sdk.CodeUnknownRequest,
		},
		{
			name: "fail to get node account should result in an error",
			messageCreator: func(helper outboundTxHandlerTestHelper, tx ObservedTx) sdk.Msg {
				tx.Tx.Coins = append(tx.Tx.Coins, common.NewCoin(common.BNBAsset, sdk.NewUint(common.One)))
				return NewMsgOutboundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg sdk.Msg) sdk.Result {
				helper.keeper.errGetNodeAccount = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: sdk.CodeInternal,
		},
		{
			name: "fail to get pool should result in an error",
			messageCreator: func(helper outboundTxHandlerTestHelper, tx ObservedTx) sdk.Msg {
				tx.Tx.Coins = append(tx.Tx.Coins, common.NewCoin(common.BNBAsset, sdk.NewUint(common.One)))
				return NewMsgOutboundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg sdk.Msg) sdk.Result {
				helper.keeper.errGetPool = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: sdk.CodeInternal,
		},
		{
			name: "fail to set pool should result in an error",
			messageCreator: func(helper outboundTxHandlerTestHelper, tx ObservedTx) sdk.Msg {
				tx.Tx.Coins = append(tx.Tx.Coins, common.NewCoin(common.BNBAsset, sdk.NewUint(common.One)))
				return NewMsgOutboundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg sdk.Msg) sdk.Result {
				helper.keeper.errSetPool = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: sdk.CodeInternal,
		},
		{
			name: "fail to set node account should result in an error",
			messageCreator: func(helper outboundTxHandlerTestHelper, tx ObservedTx) sdk.Msg {
				tx.Tx.Coins = append(tx.Tx.Coins, common.NewCoin(common.BNBAsset, sdk.NewUint(common.One)))
				return NewMsgOutboundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg sdk.Msg) sdk.Result {
				helper.keeper.errSetNodeAccount = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: sdk.CodeInternal,
		},
		{
			name: "fail to get vault data should result in an error",
			messageCreator: func(helper outboundTxHandlerTestHelper, tx ObservedTx) sdk.Msg {
				tx.Tx.Coins = append(tx.Tx.Coins, common.NewCoin(common.BNBAsset, sdk.NewUint(common.One)))
				tx.Tx.Coins = append(tx.Tx.Coins, common.NewCoin(common.RuneAsset(), sdk.NewUint(common.One*2)))
				return NewMsgOutboundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg sdk.Msg) sdk.Result {
				helper.keeper.errGetVaultData = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: sdk.CodeInternal,
		},
		{
			name: "fail to set vault data should result in an error",
			messageCreator: func(helper outboundTxHandlerTestHelper, tx ObservedTx) sdk.Msg {
				tx.Tx.Coins = append(tx.Tx.Coins, common.NewCoin(common.BNBAsset, sdk.NewUint(common.One)))
				tx.Tx.Coins = append(tx.Tx.Coins, common.NewCoin(common.RuneAsset(), sdk.NewUint(common.One*2)))
				return NewMsgOutboundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg sdk.Msg) sdk.Result {
				helper.keeper.errSetVaultData = true
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: sdk.CodeInternal,
		},
		{
			name: "valid outbound message, no event, no txout",
			messageCreator: func(helper outboundTxHandlerTestHelper, tx ObservedTx) sdk.Msg {
				return NewMsgOutboundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler OutboundTxHandler, helper outboundTxHandlerTestHelper, msg sdk.Msg) sdk.Result {
				return handler.Run(helper.ctx, msg, constants.SWVersion, helper.constAccessor)
			},
			expectedResult: sdk.CodeOK,
		},
	}

	for _, tc := range testCases {
		helper := newOutboundTxHandlerTestHelper(c)
		handler := NewOutboundTxHandler(helper.keeper, NewVersionedEventMgr())
		fromAddr, err := helper.yggVault.PubKey.GetAddress(common.BNBChain)
		c.Assert(err, IsNil)
		tx := NewObservedTx(common.Tx{
			ID:    GetRandomTxHash(),
			Chain: common.BNBChain,
			Coins: common.Coins{
				common.NewCoin(common.BNBAsset, sdk.NewUint(common.One)),
			},
			Memo:        NewOutboundMemo(helper.inboundTx.Tx.ID).String(),
			FromAddress: fromAddr,
			ToAddress:   helper.inboundTx.Tx.FromAddress,
			Gas:         BNBGasFeeSingleton,
		}, helper.ctx.BlockHeight(), helper.yggVault.PubKey)
		msg := tc.messageCreator(helper, tx)
		result := tc.runner(handler, helper, msg)
		c.Assert(result.Code, Equals, tc.expectedResult, Commentf("name: %s, Err: %s", tc.name, result.Log))
	}
}

func (s *HandlerOutboundTxSuite) TestOutboundTxNormalCase(c *C) {
	helper := newOutboundTxHandlerTestHelper(c)
	handler := NewOutboundTxHandler(helper.keeper, NewVersionedEventMgr())

	fromAddr, err := helper.yggVault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)
	tx := NewObservedTx(common.Tx{
		ID:    GetRandomTxHash(),
		Chain: common.BNBChain,
		Coins: common.Coins{
			common.NewCoin(common.BNBAsset, sdk.NewUint(common.One)),
		},
		Memo:        NewOutboundMemo(helper.inboundTx.Tx.ID).String(),
		FromAddress: fromAddr,
		ToAddress:   helper.inboundTx.Tx.FromAddress,
		Gas:         BNBGasFeeSingleton,
	}, helper.ctx.BlockHeight(), helper.yggVault.PubKey)
	// valid outbound message, with event, with txout
	outMsg := NewMsgOutboundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
	c.Assert(handler.Run(helper.ctx, outMsg, constants.SWVersion, helper.constAccessor).Code, Equals, sdk.CodeOK)
	// event should set to complete
	ev, err := helper.keeper.GetEvent(helper.ctx, 1)
	c.Assert(err, IsNil)
	c.Assert(ev.Status, Equals, EventSuccess)
	// txout should had been complete

	txOut, err := helper.keeper.GetTxOut(helper.ctx, helper.ctx.BlockHeight())
	c.Assert(err, IsNil)
	c.Assert(txOut.TxArray[0].OutHash.IsEmpty(), Equals, false)
}

func (s *HandlerOutboundTxSuite) TestOuboundTxHandlerSendExtraFundShouldBeSlashed(c *C) {
	helper := newOutboundTxHandlerTestHelper(c)
	handler := NewOutboundTxHandler(helper.keeper, NewVersionedEventMgr())
	fromAddr, err := helper.asgardVault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)
	tx := NewObservedTx(common.Tx{
		ID:    GetRandomTxHash(),
		Chain: common.BNBChain,
		Coins: common.Coins{
			common.NewCoin(common.RuneAsset(), sdk.NewUint(2*common.One)),
		},
		Memo:        NewOutboundMemo(helper.inboundTx.Tx.ID).String(),
		FromAddress: fromAddr,
		ToAddress:   helper.inboundTx.Tx.FromAddress,
		Gas:         BNBGasFeeSingleton,
	}, helper.ctx.BlockHeight(), helper.nodeAccount.PubKeySet.Secp256k1)
	expectedBond := helper.nodeAccount.Bond.Sub(sdk.NewUint(2 * common.One).MulUint64(3).QuoUint64(2))
	vaultData, err := helper.keeper.GetVaultData(helper.ctx)
	c.Assert(err, IsNil)
	expectedVaultTotalReserve := vaultData.TotalReserve.Add(sdk.NewUint(common.One * 2).QuoUint64(2))
	// valid outbound message, with event, with txout
	outMsg := NewMsgOutboundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
	c.Assert(handler.Run(helper.ctx, outMsg, constants.SWVersion, helper.constAccessor).Code, Equals, sdk.CodeOK)
	na, err := helper.keeper.GetNodeAccount(helper.ctx, helper.nodeAccount.NodeAddress)
	c.Assert(na.Bond.Equal(expectedBond), Equals, true)
	vaultData, err = helper.keeper.GetVaultData(helper.ctx)
	c.Assert(err, IsNil)
	c.Assert(vaultData.TotalReserve.Equal(expectedVaultTotalReserve), Equals, true)
}

func (s *HandlerOutboundTxSuite) TestOutboundTxHandlerSendAdditionalCoinsShouldBeSlashed(c *C) {
	helper := newOutboundTxHandlerTestHelper(c)
	handler := NewOutboundTxHandler(helper.keeper, NewVersionedEventMgr())
	fromAddr, err := helper.asgardVault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)
	tx := NewObservedTx(common.Tx{
		ID:    GetRandomTxHash(),
		Chain: common.BNBChain,
		Coins: common.Coins{
			common.NewCoin(common.RuneAsset(), sdk.NewUint(1*common.One)),
			common.NewCoin(common.BNBAsset, sdk.NewUint(1*common.One)),
		},
		Memo:        NewOutboundMemo(helper.inboundTx.Tx.ID).String(),
		FromAddress: fromAddr,
		ToAddress:   helper.inboundTx.Tx.FromAddress,
		Gas:         BNBGasFeeSingleton,
	}, helper.ctx.BlockHeight(), helper.nodeAccount.PubKeySet.Secp256k1)
	expectedBond := sdk.NewUint(9702970297)
	// slash one BNB, and one rune
	outMsg := NewMsgOutboundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
	c.Assert(handler.Run(helper.ctx, outMsg, constants.SWVersion, helper.constAccessor).Code, Equals, sdk.CodeOK)
	na, err := helper.keeper.GetNodeAccount(helper.ctx, helper.nodeAccount.NodeAddress)
	c.Assert(na.Bond.Equal(expectedBond), Equals, true, Commentf("%d/%d", na.Bond.Uint64(), expectedBond.Uint64()))
}

func (s *HandlerOutboundTxSuite) TestOutboundTxHandlerInvalidObservedTxVoterShouldSlash(c *C) {
	helper := newOutboundTxHandlerTestHelper(c)
	handler := NewOutboundTxHandler(helper.keeper, NewVersionedEventMgr())
	fromAddr, err := helper.asgardVault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)
	tx := NewObservedTx(common.Tx{
		ID:    GetRandomTxHash(),
		Chain: common.BNBChain,
		Coins: common.Coins{
			common.NewCoin(common.RuneAsset(), sdk.NewUint(1*common.One)),
			common.NewCoin(common.BNBAsset, sdk.NewUint(1*common.One)),
		},
		Memo:        NewOutboundMemo(helper.inboundTx.Tx.ID).String(),
		FromAddress: fromAddr,
		ToAddress:   helper.inboundTx.Tx.FromAddress,
		Gas:         BNBGasFeeSingleton,
	}, helper.ctx.BlockHeight(), helper.nodeAccount.PubKeySet.Secp256k1)

	expectedBond := sdk.NewUint(9702970297)
	vaultData, err := helper.keeper.GetVaultData(helper.ctx)
	c.Assert(err, IsNil)
	// expected 0.5 slashed RUNE be added to reserve
	expectedVaultTotalReserve := vaultData.TotalReserve.Add(sdk.NewUint(common.One).QuoUint64(2))
	pool, err := helper.keeper.GetPool(helper.ctx, common.BNBAsset)
	c.Assert(err, IsNil)
	poolBNB := common.SafeSub(pool.BalanceAsset, sdk.NewUint(common.One))

	// given the outbound tx doesn't have relevant OservedTxVoter in system ,
	// thus it should be slashed with 1.5 * the full amount of assets
	outMsg := NewMsgOutboundTx(tx, tx.Tx.ID, helper.nodeAccount.NodeAddress)
	na, _ := helper.keeper.GetNodeAccount(helper.ctx, helper.nodeAccount.NodeAddress)
	c.Assert(handler.Run(helper.ctx, outMsg, constants.SWVersion, helper.constAccessor).Code, Equals, sdk.CodeOK)
	na, err = helper.keeper.GetNodeAccount(helper.ctx, helper.nodeAccount.NodeAddress)
	c.Assert(na.Bond.Equal(expectedBond), Equals, true, Commentf("%d/%d", na.Bond.Uint64(), expectedBond.Uint64()))

	vaultData, err = helper.keeper.GetVaultData(helper.ctx)
	c.Assert(err, IsNil)
	c.Assert(vaultData.TotalReserve.Equal(expectedVaultTotalReserve), Equals, true)
	pool, err = helper.keeper.GetPool(helper.ctx, common.BNBAsset)
	c.Assert(err, IsNil)
	c.Assert(pool.BalanceRune.Equal(sdk.NewUint(10047029703)), Equals, true, Commentf("%d/%d", pool.BalanceRune.Uint64(), sdk.NewUint(10047029703)))
	c.Assert(pool.BalanceAsset.Equal(poolBNB), Equals, true, Commentf("%d/%d", pool.BalanceAsset.Uint64(), poolBNB.Uint64()))
}
