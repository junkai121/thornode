package types

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"gitlab.com/thorchain/bepswap/thornode/common"
)

type AdminConfigKey string

const (
	UnknownKey                AdminConfigKey = "Unknown"
	GSLKey                    AdminConfigKey = "GSL"
	StakerAmtIntervalKey      AdminConfigKey = "StakerAmtInterval"
	MinValidatorBondKey       AdminConfigKey = "MinValidatorBond"
	WhiteListGasAssetKey      AdminConfigKey = "WhiteListGasAsset"      // How much gas asset we mint and send it to the newly whitelisted bep address
	DesireValidatorSetKey     AdminConfigKey = "DesireValidatorSet"     // how much validators we would like to have. Replace with Dynamic N.
	RotatePerBlockHeightKey   AdminConfigKey = "RotatePerBlockHeight"   // how many blocks we try to rotate validators
	ValidatorsChangeWindowKey AdminConfigKey = "ValidatorsChangeWindow" // when should we open the rotate window, nominate validators, and identify who should be out
	PoolRefundGasKey          AdminConfigKey = "PoolRefundGas"          // when we move assets from one pool to another , we leave this amount of BNB behind, thus we could refund customer if they send fund to the previous pool
	DefaultPoolStatus         AdminConfigKey = "DefaultPoolStatus"
)

func (k AdminConfigKey) String() string {
	return string(k)
}

func (k AdminConfigKey) IsValidKey() bool {
	key := GetAdminConfigKey(k.String())
	return key != UnknownKey
}
func GetAdminConfigKey(key string) AdminConfigKey {
	switch key {
	case string(GSLKey):
		return GSLKey
	case string(StakerAmtIntervalKey):
		return StakerAmtIntervalKey
	case string(MinValidatorBondKey):
		return MinValidatorBondKey
	case string(WhiteListGasAssetKey):
		return WhiteListGasAssetKey
	case string(DesireValidatorSetKey):
		return DesireValidatorSetKey
	case string(RotatePerBlockHeightKey):
		return RotatePerBlockHeightKey
	case string(ValidatorsChangeWindowKey):
		return ValidatorsChangeWindowKey
	case string(PoolRefundGasKey):
		return PoolRefundGasKey
	case string(DefaultPoolStatus):
		return DefaultPoolStatus
	default:
		return UnknownKey
	}
}

func (k AdminConfigKey) Default() string {
	switch k {
	case GSLKey:
		return "0.3"
	case StakerAmtIntervalKey:
		return "100"
	case MinValidatorBondKey:
		return sdk.NewUint(common.One * 10).String()
	case WhiteListGasAssetKey:
		return "1000bep"
	case DesireValidatorSetKey:
		return "4"
	case RotatePerBlockHeightKey:
		return "17280" // with 5 second block times, this is a day
	case ValidatorsChangeWindowKey:
		return "1200" // one hour
	case PoolRefundGasKey:
		return strconv.Itoa(common.One / 10)
	case DefaultPoolStatus:
		return "Enabled"
	default:
		return ""
	}
}

// Ensure the value for a given key is a valid
func (k AdminConfigKey) ValidValue(value string) error {
	var err error
	switch k {
	case GSLKey, StakerAmtIntervalKey:
		_, err = common.NewAmount(value)
	case MinValidatorBondKey:
		_, err = sdk.ParseUint(value)
	case WhiteListGasAssetKey:
		_, err = sdk.ParseCoins(value)
	case DesireValidatorSetKey, RotatePerBlockHeightKey, ValidatorsChangeWindowKey, PoolRefundGasKey: // int64
		_, err = strconv.ParseInt(value, 10, 64)
	}
	return err
}

type AdminConfig struct {
	Key     AdminConfigKey `json:"key"`
	Value   string         `json:"value"`
	Address sdk.AccAddress `json:"address"`
}

func NewAdminConfig(key AdminConfigKey, value string, address sdk.AccAddress) AdminConfig {
	return AdminConfig{
		Key:     key,
		Value:   value,
		Address: address,
	}
}

func (c AdminConfig) Empty() bool {
	return c.Key == ""
}

func (c AdminConfig) Valid() error {
	if c.Address.Empty() {
		return fmt.Errorf("address cannot be empty")
	}
	if c.Key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	if !c.Key.IsValidKey() {
		return fmt.Errorf("key not recognized")
	}
	if c.Value == "" {
		return fmt.Errorf("value cannot be empty")
	}
	if err := c.Key.ValidValue(c.Value); err != nil {
		return err
	}
	return nil
}

func (c AdminConfig) DbKey() string {
	return fmt.Sprintf("%s_%s", c.Key.String(), c.Address.String())
}

func (c AdminConfig) String() string {
	return fmt.Sprintf("Config: %s --> %s", c.Key, c.Value)
}
