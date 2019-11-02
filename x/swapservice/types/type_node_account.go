package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"gitlab.com/thorchain/bepswap/thornode/common"
)

// NodeStatus Represent the Node status
type NodeStatus uint8

// As soon as user donate a certain amount of asset(defined later)
// their node adddress will be whitelisted
// once we discover their observer had send tx in to statechain , then their status will be standby
// once we rotate them in , then they will be active
const (
	Unknown NodeStatus = iota
	WhiteListed
	Standby
	Ready
	Active
	Disabled
)

var nodeStatusStr = map[string]NodeStatus{
	"unknown":     Unknown,
	"whitelisted": WhiteListed,
	"standby":     Standby,
	"ready":       Ready,
	"active":      Active,
	"disabled":    Disabled,
}

// String implement stringer
func (ps NodeStatus) String() string {
	for key, item := range nodeStatusStr {
		if item == ps {
			return key
		}
	}
	return ""
}

func (ps NodeStatus) Valid() error {
	if ps.String() == "" {
		return fmt.Errorf("invalid node status")
	}
	return nil
}

// MarshalJSON marshal NodeStatus to JSON in string form
func (ps NodeStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(ps.String())
}

// UnmarshalJSON convert string form back to NodeStatus
func (ps *NodeStatus) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); nil != err {
		return err
	}
	*ps = GetNodeStatus(s)
	return nil
}

// GetNodeStatus from string
func GetNodeStatus(ps string) NodeStatus {
	for key, item := range nodeStatusStr {
		if strings.EqualFold(key, ps) {
			return item
		}
	}

	return Unknown
}

// NodeAccount represent node
type NodeAccount struct {
	NodeAddress sdk.AccAddress `json:"node_address"` // Thor address which is an operator address
	Status      NodeStatus     `json:"status"`
	Accounts    TrustAccount   `json:"accounts"`
	Bond        sdk.Uint       `json:"bond"`
	BondAddress common.Address `json:"bond_address"` // BNB Address to send bond from. It also indicates the operator address to whilelist and associate.
	// start from when this node account is in current status
	// StatusSince field is important , it has been used to sort node account , used for validator rotation
	StatusSince    int64         `json:"status_since"`
	ObserverActive bool          `json:"observer_active"`
	SignerActive   bool          `json:"signer_active"`
	Version        common.Amount `json:"version"`
}

// NewNodeAccount create new instance of NodeAccount
func NewNodeAccount(nodeAddress sdk.AccAddress, status NodeStatus, accounts TrustAccount, bond sdk.Uint, bondAddress common.Address, height int64) NodeAccount {
	na := NodeAccount{
		NodeAddress: nodeAddress,
		Accounts:    accounts,
		Bond:        bond,
		BondAddress: bondAddress,
		Version:     common.ZeroAmount,
	}
	na.UpdateStatus(status, height)
	return na
}

// IsEmpty decide whether NodeAccount is empty
func (n NodeAccount) IsEmpty() bool {
	return n.NodeAddress.Empty()
}

// IsValid check whether NodeAccount has all necessary values
func (n NodeAccount) IsValid() error {
	if n.NodeAddress.Empty() {
		return errors.New("node bep address is empty")
	}
	if n.BondAddress.IsEmpty() {
		return errors.New("bond address is empty")
	}
	return n.Accounts.IsValid()
}

// UpdateStatus change the status of node account, in the mean time update StatusSince field
func (n *NodeAccount) UpdateStatus(status NodeStatus, height int64) {
	n.Status = status
	n.StatusSince = height
}

// Equals compare two node account, to see whether they are equal
func (n NodeAccount) Equals(n1 NodeAccount) bool {
	if n.NodeAddress.Equals(n1.NodeAddress) &&
		n.Accounts.Equals(n1.Accounts) &&
		n.BondAddress.Equals(n1.BondAddress) &&
		n.Bond.Equal(n1.Bond) &&
		n.Version.Equals(n1.Version) {
		return true
	}
	return false
}

// String implement fmt.Stringer interface
func (n NodeAccount) String() string {
	sb := strings.Builder{}
	sb.WriteString("node:" + n.NodeAddress.String() + "\n")
	sb.WriteString("status:" + n.Status.String() + "\n")
	sb.WriteString("account:" + n.Accounts.String() + "\n")
	sb.WriteString("bond:" + n.Bond.String() + "\n")
	sb.WriteString("version:" + n.Version.String() + "\n")
	sb.WriteString("bond address:" + n.BondAddress.String() + "\n")
	return sb.String()
}

// NodeAccounts just a list of NodeAccount
type NodeAccounts []NodeAccount

// IsTrustAccount validate whether the given account address is an observer address
func (nodeAccounts NodeAccounts) IsTrustAccount(addr sdk.AccAddress) bool {
	for _, na := range nodeAccounts {
		if na.Status == Active && na.Accounts.ObserverBEPAddress.Equals(addr) {
			return true
		}
	}
	return false
}

// NodeAccount sort interface , it will sort by StatusSince field, and then by SignerBNBAddress
func (nodeAccounts NodeAccounts) Less(i, j int) bool {
	if nodeAccounts[i].StatusSince < nodeAccounts[j].StatusSince {
		return true
	}
	if nodeAccounts[i].StatusSince > nodeAccounts[j].StatusSince {
		return false
	}
	return nodeAccounts[i].Accounts.SignerBNBAddress.String() < nodeAccounts[j].Accounts.SignerBNBAddress.String()
}
func (nodeAccounts NodeAccounts) Len() int { return len(nodeAccounts) }
func (nodeAccounts NodeAccounts) Swap(i, j int) {
	nodeAccounts[i], nodeAccounts[j] = nodeAccounts[j], nodeAccounts[i]
}

func (nodeAccounts NodeAccounts) After(addr common.Address) NodeAccount {
	idx := 0
	for i, na := range nodeAccounts {
		if na.Accounts.SignerBNBAddress.Equals(addr) {
			idx = i
			break
		}
	}
	if idx+1 < len(nodeAccounts) {
		return nodeAccounts[idx+1]
	}
	return nodeAccounts[0]
}

// First return the first item in the slice
func (nodeAccounts NodeAccounts) First() NodeAccount {
	if len(nodeAccounts) > 0 {
		return nodeAccounts[0]
	}
	return NodeAccount{}
}
