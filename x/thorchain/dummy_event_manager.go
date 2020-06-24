package thorchain

import (
	"github.com/blang/semver"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/x/thorchain/keep"
)

// DummyEventMgr used for test purpose , and it implement EventManager interface
type DummyEventMgr struct {
}

func NewDummyEventMgr() *DummyEventMgr {
	return &DummyEventMgr{}
}

func (m *DummyEventMgr) CompleteEvents(ctx sdk.Context, keeper keep.Keeper, height int64, txID common.TxID, txs common.Txs, eventStatus EventStatus) {
}

func (m *DummyEventMgr) EmitPoolEvent(ctx sdk.Context, keeper keep.Keeper, txIn common.TxID, status EventStatus, poolEvt EventPool) error {
	return nil
}

func (m *DummyEventMgr) EmitErrataEvent(ctx sdk.Context, keeper keep.Keeper, txIn common.TxID, errataEvent EventErrata) error {
	return nil
}

func (m *DummyEventMgr) EmitGasEvent(ctx sdk.Context, keeper keep.Keeper, gasEvent *EventGas) error {
	return nil
}

func (m *DummyEventMgr) EmitStakeEvent(ctx sdk.Context, keeper keep.Keeper, inTx common.Tx, stakeEvent EventStake) error {
	return nil
}

func (m *DummyEventMgr) EmitRewardEvent(ctx sdk.Context, keeper keep.Keeper, rewardEvt EventRewards) error {
	return nil
}

func (m *DummyEventMgr) EmitReserveEvent(ctx sdk.Context, keeper keep.Keeper, reserveEvent EventReserve) error {
	return nil
}

func (m *DummyEventMgr) EmitUnstakeEvent(ctx sdk.Context, keeper keep.Keeper, unstakeEvt EventUnstake) error {
	return nil
}

func (m *DummyEventMgr) EmitSwapEvent(ctx sdk.Context, keeper keep.Keeper, swap EventSwap) error {
	return nil
}

func (m *DummyEventMgr) EmitAddEvent(ctx sdk.Context, keeper keep.Keeper, addEvt EventAdd) error {
	return nil
}

func (m *DummyEventMgr) EmitRefundEvent(ctx sdk.Context, keeper keep.Keeper, refundEvt EventRefund, status EventStatus) error {
	return nil
}

func (m *DummyEventMgr) EmitBondEvent(ctx sdk.Context, keeper keep.Keeper, bondEvent EventBond) error {
	return nil
}

func (m *DummyEventMgr) EmitFeeEvent(ctx sdk.Context, keeper keep.Keeper, feeEvent EventFee) error {
	return nil
}

func (m *DummyEventMgr) EmitSlashEvent(ctx sdk.Context, keeper keep.Keeper, slashEvt EventSlash) error {
	return nil
}

func (m *DummyEventMgr) EmitOutboundEvent(ctx sdk.Context, outbound EventOutbound) error {
	return nil
}

type DummyVersionedEventMgr struct{}

func NewDummyVersionedEventMgr() *DummyVersionedEventMgr {
	return &DummyVersionedEventMgr{}
}

func (m *DummyVersionedEventMgr) GetEventManager(ctx sdk.Context, version semver.Version) (EventManager, error) {
	return NewDummyEventMgr(), nil
}
