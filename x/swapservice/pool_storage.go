package swapservice

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"gitlab.com/thorchain/bepswap/common"
)

// PoolStorage allow us to access the pool struct from key values store
// this is an interface thus we could write unit tests
type poolStorage interface {
	PoolExist(ctx sdk.Context, ticker common.Ticker) bool

	GetPool(ctx sdk.Context, ticker common.Ticker) Pool
	SetPool(ctx sdk.Context, ps Pool)

	GetStakerPool(ctx sdk.Context, stakerID common.BnbAddress) (StakerPool, error)
	SetStakerPool(ctx sdk.Context, stakerID common.BnbAddress, sp StakerPool)

	GetPoolStaker(ctx sdk.Context, ticker common.Ticker) (PoolStaker, error)
	SetPoolStaker(ctx sdk.Context, ticker common.Ticker, ps PoolStaker)

	GetAdminConfigValue(ctx sdk.Context, key AdminConfigKey, addr sdk.AccAddress) (string, error)

	GetAdminConfigStakerAmtInterval(ctx sdk.Context, addr sdk.AccAddress) common.Amount

	AddIncompleteEvents(ctx sdk.Context, event Event)
}
