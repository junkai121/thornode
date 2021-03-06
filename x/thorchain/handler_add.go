package thorchain

import (
	"fmt"

	"github.com/blang/semver"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keep"
)

// AddHandler is to handle Add message
type AddHandler struct {
	keeper                keep.Keeper
	versionedEventManager VersionedEventManager
}

// NewAddHandler create a new instance of AddHandler
func NewAddHandler(keeper keep.Keeper, versionedEventManager VersionedEventManager) AddHandler {
	return AddHandler{
		keeper:                keeper,
		versionedEventManager: versionedEventManager,
	}
}

// Run it the main entry point to execute Ack logic
func (ah AddHandler) Run(ctx sdk.Context, m sdk.Msg, version semver.Version, _ constants.ConstantValues) sdk.Result {
	msg, ok := m.(MsgAdd)
	if !ok {
		return errInvalidMessage.Result()
	}
	ctx.Logger().Info(fmt.Sprintf("receive msg add %s", msg.Tx.ID))
	if err := ah.validate(ctx, msg, version); err != nil {
		ctx.Logger().Error("msg add failed validation", "error", err)
		return err.Result()
	}
	if err := ah.handle(ctx, msg, version); err != nil {
		ctx.Logger().Error("fail to process msg add", "error", err)
		return err.Result()
	}

	return sdk.Result{
		Code:      sdk.CodeOK,
		Codespace: DefaultCodespace,
	}
}

func (ah AddHandler) validate(ctx sdk.Context, msg MsgAdd, version semver.Version) sdk.Error {
	if version.GTE(semver.MustParse("0.1.0")) {
		return ah.validateV1(ctx, msg)
	}
	return errBadVersion
}

func (ah AddHandler) validateV1(ctx sdk.Context, msg MsgAdd) sdk.Error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}
	if !isSignedByActiveNodeAccounts(ctx, ah.keeper, msg.GetSigners()) {
		return sdk.ErrUnauthorized("Not authorized")
	}
	return nil
}

// handleMsgAdd
func (ah AddHandler) handle(ctx sdk.Context, msg MsgAdd, version semver.Version) sdk.Error {
	pool, err := ah.keeper.GetPool(ctx, msg.Asset)
	if err != nil {
		return sdk.ErrInternal(fmt.Errorf("fail to get pool for (%s): %w", msg.Asset, err).Error())
	}
	if pool.Asset.IsEmpty() {
		return sdk.ErrUnknownRequest(fmt.Sprintf("pool %s not exist", msg.Asset.String()))
	}
	if msg.AssetAmount.GT(sdk.ZeroUint()) {
		pool.BalanceAsset = pool.BalanceAsset.Add(msg.AssetAmount)
	}
	if msg.RuneAmount.GT(sdk.ZeroUint()) {
		pool.BalanceRune = pool.BalanceRune.Add(msg.RuneAmount)
	}

	if err := ah.keeper.SetPool(ctx, pool); err != nil {
		return sdk.ErrInternal(fmt.Errorf("fail to set pool(%s): %w", pool, err).Error())
	}
	eventMgr, err := ah.versionedEventManager.GetEventManager(ctx, version)
	if err != nil {
		return errFailGetEventManager
	}
	// emit event
	addEvt := NewEventAdd(pool.Asset, msg.Tx)
	if err := eventMgr.EmitAddEvent(ctx, ah.keeper, addEvt); err != nil {
		return sdk.NewError(DefaultCodespace, CodeFailSaveEvent, "fail to save add events")
	}
	return nil
}
