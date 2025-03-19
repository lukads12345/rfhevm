package crypto

import (
	"PureChain/common"
	evm "PureChain/crypto"
)

// CreateProtectedStorageAddress creates an ethereum contract address for protected storage
// given the corresponding contract address
func CreateProtectedStorageContractAddress(b common.Address) common.Address {
	return evm.CreateAddress(b, 0)
}
