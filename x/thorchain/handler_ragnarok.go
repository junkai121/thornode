package thorchain

import (
	"fmt"

	"github.com/blang/semver"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keep"
)

type RagnarokHandler struct {
	keeper                keep.Keeper
	versionedEventManager VersionedEventManager
}

func NewRagnarokHandler(keeper keep.Keeper, versionedEventManager VersionedEventManager) RagnarokHandler {
	return RagnarokHandler{
		keeper:                keeper,
		versionedEventManager: versionedEventManager,
	}
}

func (h RagnarokHandler) Run(ctx sdk.Context, m sdk.Msg, version semver.Version, _ constants.ConstantValues) sdk.Result {
	msg, ok := m.(MsgRagnarok)
	if !ok {
		return errInvalidMessage.Result()
	}
	if err := h.validate(ctx, msg, version); err != nil {
		return sdk.ErrInternal(err.Error()).Result()
	}
	return h.handle(ctx, version, msg)
}

func (h RagnarokHandler) validate(ctx sdk.Context, msg MsgRagnarok, version semver.Version) error {
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.validateV1(ctx, msg)
	}
	ctx.Logger().Error(errInvalidVersion.Error())
	return errInvalidVersion
}

func (h RagnarokHandler) validateV1(ctx sdk.Context, msg MsgRagnarok) error {
	if err := msg.ValidateBasic(); nil != err {
		ctx.Logger().Error(err.Error())
		return err
	}

	if !isSignedByActiveNodeAccounts(ctx, h.keeper, msg.GetSigners()) {
		ctx.Logger().Error(notAuthorized.Error())
		return notAuthorized
	}
	return nil
}

func (h RagnarokHandler) handle(ctx sdk.Context, version semver.Version, msg MsgRagnarok) sdk.Result {
	ctx.Logger().Info("receive MsgRagnarok", "request tx hash", msg.Tx.Tx.ID)
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.handleV1(ctx, version, msg)
	}
	ctx.Logger().Error(errInvalidVersion.Error())
	return errBadVersion.Result()
}

func (h RagnarokHandler) slash(ctx sdk.Context, version semver.Version, tx ObservedTx) error {
	var returnErr error
	slasher, err := NewSlasher(h.keeper, version, h.versionedEventManager)
	if err != nil {
		return fmt.Errorf("fail to create slasher: %w", err)
	}
	for _, c := range tx.Tx.Coins {
		if err := slasher.SlashNodeAccount(ctx, tx.ObservedPubKey, c.Asset, c.Amount); err != nil {
			ctx.Logger().Error("fail to slash account", "error", err)
			returnErr = err
		}
	}
	return returnErr
}

func (h RagnarokHandler) handleV1(ctx sdk.Context, version semver.Version, msg MsgRagnarok) sdk.Result {
	// update txOut record with our TxID that sent funds out of the pool
	txOut, err := h.keeper.GetTxOut(ctx, msg.BlockHeight)
	if err != nil {
		ctx.Logger().Error("unable to get txOut record", "error", err)
		return sdk.ErrUnknownRequest(err.Error()).Result()
	}

	shouldSlash := true
	for i, tx := range txOut.TxArray {
		// ragnarok is the memo used by thorchain to identify fund returns to
		// bonders, LPs, and reserve contributors.
		// it use ragnarok:{block height} to mark a tx out caused by the ragnarok protocol
		// this type of tx out is special, because it doesn't have relevant tx
		// in to trigger it, it is trigger by thorchain itself.
		fromAddress, _ := tx.VaultPubKey.GetAddress(tx.Chain)
		if tx.InHash.Equals(common.BlankTxID) &&
			tx.OutHash.IsEmpty() &&
			msg.Tx.Tx.Coins.Contains(tx.Coin) &&
			tx.ToAddress.Equals(msg.Tx.Tx.ToAddress) &&
			fromAddress.Equals(msg.Tx.Tx.FromAddress) {

			txOut.TxArray[i].OutHash = msg.Tx.Tx.ID
			shouldSlash = false
			if err := h.keeper.SetTxOut(ctx, txOut); nil != err {
				ctx.Logger().Error("fail to save tx out", "error", err)
				return sdk.ErrInternal("fail to save tx out").Result()
			}

			break
		}
	}

	if shouldSlash {
		if err := h.slash(ctx, version, msg.Tx); err != nil {
			return sdk.ErrInternal("fail to slash account").Result()
		}
	}

	h.keeper.SetLastSignedHeight(ctx, msg.BlockHeight)

	return sdk.Result{
		Code:      sdk.CodeOK,
		Codespace: DefaultCodespace,
	}
}
