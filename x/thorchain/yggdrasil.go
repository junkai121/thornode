package thorchain

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/constants"
)

func Fund(ctx sdk.Context, keeper Keeper, txOutStore TxOutStore, constAccessor constants.ConstantValues) error {

	// find total bonded
	totalBond := sdk.ZeroUint()
	nodeAccs, err := keeper.ListActiveNodeAccounts(ctx)
	if err != nil {
		return err
	}
	minimumNodesForYggdrasil := constAccessor.GetInt64Value(constants.MinimumNodesForYggdrasil)
	if int64(len(nodeAccs)) <= minimumNodesForYggdrasil {
		return nil
	}

	// Gather list of all pools
	pools, err := keeper.GetPools(ctx)
	if err != nil {
		return err
	}

	for _, na := range nodeAccs {
		totalBond = totalBond.Add(na.Bond)
	}

	// We don't want to check all Yggdrasil pools every time THORNode run this
	// function. So THORNode use modulus to determine which Ygg THORNode process. This
	// should behave as a "round robin" approach checking one Ygg per block.
	// With 100 Ygg pools, THORNode should check each pool every 8.33 minutes.
	na := nodeAccs[ctx.BlockHeight()%int64(len(nodeAccs))]

	// figure out if THORNode need to send them assets.
	// get a list of coin/amounts this yggdrasil pool should have, ideally.
	// TODO: We are assuming here that the pub key is Secp256K1
	ygg, err := keeper.GetVault(ctx, na.PubKeySet.Secp256k1)
	if err != nil {
		if !errors.Is(err, ErrVaultNotFound) {
			return fmt.Errorf("fail to get yggdrasil: %w", err)
		}
		ygg = NewVault(ctx.BlockHeight(), ActiveVault, YggdrasilVault, na.PubKeySet.Secp256k1)
		ygg.Membership = append(ygg.Membership, na.PubKeySet.Secp256k1)

		if err := keeper.SetVault(ctx, ygg); err != nil {
			return fmt.Errorf("fail to create yggdrasil pool: %w", err)
		}
	}
	if !ygg.IsYggdrasil() {
		return fmt.Errorf("this is not a Yggdrasil vault")
	}
	targetCoins, err := calcTargetYggCoins(pools, na.Bond, totalBond)
	if err != nil {
		return err
	}

	var sendCoins common.Coins
	// iterate over each target coin amount and figure if THORNode need to reimburse
	// a Ygg pool of this particular asset.
	for _, targetCoin := range targetCoins {
		yggCoin := ygg.GetCoin(targetCoin.Asset)
		// check if the amount the ygg pool has is less that 50% of what
		// they are suppose to have, ideally. We refill them if they drop
		// below this line
		if yggCoin.Amount.LT(targetCoin.Amount.QuoUint64(2)) {
			sendCoins = append(
				sendCoins,
				common.NewCoin(
					targetCoin.Asset,
					common.SafeSub(targetCoin.Amount, yggCoin.Amount),
				),
			)
		}

	}

	if len(sendCoins) > 0 {
		return sendCoinsToYggdrasil(ctx, keeper, sendCoins, ygg, txOutStore)
	}

	return nil
}

// sendCoinsToYggdrasil - adds outbound txs to send the given coins to a
// yggdrasil pool
func sendCoinsToYggdrasil(ctx sdk.Context, keeper Keeper, coins common.Coins, ygg Vault, txOutStore TxOutStore) error {
	for _, coin := range coins {
		to, err := ygg.PubKey.GetAddress(coin.Asset.Chain)
		if err != nil {
			return err
		}

		toi := &TxOutItem{
			Chain:     coin.Asset.Chain,
			ToAddress: to,
			Memo:      "yggdrasil+",
			Coin:      coin,
		}
		_, err = txOutStore.TryAddTxOutItem(ctx, toi)
		if err != nil {
			return err
		}
	}

	return nil
}

// calcTargetYggCoins - calculate the amount of coins of each pool a yggdrasil
// pool should have, relative to how much they have bonded (which should be
// target == bond / 2).
func calcTargetYggCoins(pools []Pool, yggBond, totalBond sdk.Uint) (common.Coins, error) {
	runeCoin := common.NewCoin(common.RuneAsset(), sdk.ZeroUint())
	var coins common.Coins

	// calculate total rune in our pools
	totalRune := sdk.ZeroUint()
	for _, pool := range pools {
		totalRune = totalRune.Add(pool.BalanceRune)
	}
	if totalRune.IsZero() {
		// if nothing is staked, no coins should be issued
		return nil, nil
	}

	// figure out what percentage of the bond this yggdrasil pool has. They
	// should get half of that value.
	targetRune := common.GetShare(yggBond, totalBond.Mul(sdk.NewUint(2)), totalRune)
	// check if more rune would be allocated to this pool than their bond allows
	if targetRune.GT(yggBond.QuoUint64(2)) {
		targetRune = yggBond.QuoUint64(2)
	}

	// track how much value (in rune) we've associated with this ygg pool. This
	// is here just to be absolutely sure THORNode never send too many assets to the
	// ygg by accident.
	counter := sdk.ZeroUint()
	for _, pool := range pools {
		runeAmt := common.GetShare(targetRune, totalRune, pool.BalanceRune)
		runeCoin.Amount = runeCoin.Amount.Add(runeAmt)
		assetAmt := common.GetShare(targetRune, totalRune, pool.BalanceAsset)
		if !assetAmt.IsZero() {
			// add rune amt (not asset since the two are considered to be equal)
			counter = counter.Add(runeAmt)

			coin := common.NewCoin(pool.Asset, assetAmt)
			coins = append(coins, coin)
		}
	}

	if !runeCoin.Amount.IsZero() {
		counter = counter.Add(runeCoin.Amount)
		coins = append(coins, runeCoin)
	}

	// ensure THORNode don't send too much value in coins to the ygg pool
	if counter.GT(yggBond.QuoUint64(2)) {
		return nil, errors.New("exceeded safe amounts of assets for given Yggdrasil pool")
	}

	return coins, nil
}
