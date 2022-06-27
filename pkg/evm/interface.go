package evm

import "github.com/multiverse-dev/saiya/pkg/core/native"

type NativeContracts interface {
	Contracts() *native.Contracts
}
