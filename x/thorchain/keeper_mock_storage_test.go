package thorchain

import (
	"errors"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"gitlab.com/thorchain/thornode/common"

	"gitlab.com/thorchain/thornode/x/thorchain/types"
)

// MockPoolStorage implements PoolStorage interface, thus we can mock the error cases
type MockPoolStorage struct {
	KVStoreDummy
}

func (mps MockPoolStorage) PoolExist(ctx sdk.Context, asset common.Asset) bool {
	if asset.Equals(common.Asset{Chain: common.BNBChain, Symbol: "NOTEXIST", Ticker: "NOTEXIST"}) {
		return false
	}
	return true
}

func (mps MockPoolStorage) GetPool(ctx sdk.Context, asset common.Asset) types.Pool {
	if asset.Equals(common.Asset{Chain: common.BNBChain, Symbol: "NOTEXIST", Ticker: "NOTEXIST"}) {
		return types.Pool{}
	} else {
		return types.Pool{
			BalanceRune:  sdk.NewUint(100).MulUint64(common.One),
			BalanceAsset: sdk.NewUint(100).MulUint64(common.One),
			PoolUnits:    sdk.NewUint(100).MulUint64(common.One),
			Status:       types.Enabled,
			Asset:        asset,
		}
	}
}

func (mps MockPoolStorage) SetPool(ctx sdk.Context, ps types.Pool) {}

func (mps MockPoolStorage) GetStakerPool(ctx sdk.Context, stakerID common.Address) (types.StakerPool, error) {
	if strings.EqualFold(stakerID.String(), "NOTEXISTSTAKER") {
		return types.StakerPool{}, errors.New("you asked for it")
	}
	return types.NewStakerPool(stakerID), nil
}

func (mps MockPoolStorage) SetStakerPool(ctx sdk.Context, stakerID common.Address, sp types.StakerPool) {

}

func (mps MockPoolStorage) GetPoolStaker(ctx sdk.Context, asset common.Asset) (types.PoolStaker, error) {
	if asset.Equals(common.Asset{Chain: common.BNBChain, Symbol: "NOTEXISTSTICKER", Ticker: "NOTEXISTSTICKER"}) {
		return types.PoolStaker{}, errors.New("you asked for it")
	}
	return types.NewPoolStaker(asset, sdk.NewUint(100)), nil
}

func (mps MockPoolStorage) SetPoolStaker(ctx sdk.Context, asset common.Asset, ps types.PoolStaker) {}

func (mps MockPoolStorage) AddToLiquidityFees(ctx sdk.Context, ps types.Pool, fs sdk.Uint) error {
	return nil
}

func (mps MockPoolStorage) GetAdminConfigDefaultPoolStatus(ctx sdk.Context, addr sdk.AccAddress) types.PoolStatus {
	return types.Bootstrap
}

func (mps MockPoolStorage) GetAdminConfigValue(ctx sdk.Context, key types.AdminConfigKey, addr sdk.AccAddress) (string, error) {
	return "FOOBAR", nil
}
func (mps MockPoolStorage) GetAdminConfigStakerAmtInterval(ctx sdk.Context, addr sdk.AccAddress) common.Amount {
	return common.NewAmountFromFloat(100)
}

func (mps MockPoolStorage) AddIncompleteEvents(ctx sdk.Context, event types.Event) {}
func (mps MockPoolStorage) SetCompletedEvent(ctx sdk.Context, event types.Event)   {}

func (mps MockPoolStorage) GetLowestActiveVersion(ctx sdk.Context) int64 {
	return 0
}

func (mps MockPoolStorage) AddFeeToReserve(ctx sdk.Context, fee sdk.Uint) {}