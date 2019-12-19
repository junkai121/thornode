package types

import (
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"

	"gitlab.com/thorchain/thornode/common"
)

type VaultType string

const (
	UnknownVault   VaultType = "unknown"
	AsgardVault    VaultType = "asgard"
	YggdrasilVault VaultType = "yggdrasil"
)

// Vault
type Vault struct {
	PubKey common.PubKey `json:"pub_key"`
	Coins  common.Coins  `json:"coins"`
	Type   VaultType     `json:"type"`
}

type Vaults []Vault

func NewVault(vtype VaultType, pk common.PubKey) Vault {
	return Vault{
		PubKey: pk,
		Coins:  make(common.Coins, 0),
		Type:   vtype,
	}
}

func (v Vault) IsType(vtype VaultType) bool {
	return v.Type == vtype
}

func (v Vault) IsAsgard() bool {
	return v.IsType(AsgardVault)
}

func (v Vault) IsYggdrasil() bool {
	return v.IsType(YggdrasilVault)
}

func (v Vault) IsEmpty() bool {
	return v.PubKey.IsEmpty()
}

// IsValid check whether Vault has all necessary values
func (v Vault) IsValid() error {
	if v.PubKey.IsEmpty() {
		return errors.New("pubkey cannot be empty")
	}
	return nil
}

// HasFunds check whether the vault pool has fund
func (v Vault) HasFunds() bool {
	for _, coin := range v.Coins {
		if coin.Amount.GT(sdk.ZeroUint()) {
			return true
		}

	}
	return false
}

// Check if this vault has a particular asset
func (v Vault) HasAsset(asset common.Asset) bool {
	return !v.GetCoin(asset).Amount.IsZero()
}

func (v Vault) GetCoin(asset common.Asset) common.Coin {
	for _, coin := range v.Coins {
		if coin.Asset.Equals(asset) {
			return coin
		}
	}
	return common.NewCoin(asset, sdk.ZeroUint())
}

func (v *Vault) AddFunds(coins common.Coins) {
	for _, coin := range coins {
		if v.HasAsset(coin.Asset) {
			for i, ycoin := range v.Coins {
				if coin.Asset.Equals(ycoin.Asset) {
					v.Coins[i].Amount = ycoin.Amount.Add(coin.Amount)
				}
			}
		} else {
			v.Coins = append(v.Coins, coin)
		}
	}
}

func (v *Vault) SubFunds(coins common.Coins) {
	for _, coin := range coins {
		for i, ycoin := range v.Coins {
			if coin.Asset.Equals(ycoin.Asset) {
				// safeguard to protect against enter negative values
				if coin.Amount.GTE(ycoin.Amount) {
					coin.Amount = ycoin.Amount
				}
				v.Coins[i].Amount = common.SafeSub(ycoin.Amount, coin.Amount)
			}
		}
	}
}

func (vs Vaults) SortBy(sortBy common.Asset) Vaults {
	// use the vault pool with the highest quantity of our coin
	sort.Slice(vs[:], func(i, j int) bool {
		return vs[i].GetCoin(sortBy).Amount.GT(
			vs[j].GetCoin(sortBy).Amount,
		)
	})

	return vs
}
