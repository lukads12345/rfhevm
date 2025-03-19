package fhevm

import "PureChain/common"

type Contract interface {
	Address() common.Address
}
