package fhevm

import (
	"github.com/ethereum/go-ethereum/common"
)

type ScopeContext interface {
	GetMemory() Memory
	GetStack() Stack
	GetContract() Contract
}

type depthSet struct {
	m map[int]struct{}
}

func newDepthSet() *depthSet {
	s := &depthSet{}
	s.m = make(map[int]struct{})
	return s
}

func (s *depthSet) add(v int) {
	s.m[v] = struct{}{}
}

func (s *depthSet) del(v int) {
	delete(s.m, v)
}

func (s *depthSet) has(v int) bool {
	_, found := s.m[v]
	return found
}

func (s *depthSet) count() int {
	return len(s.m)
}

func (from *depthSet) clone() (to *depthSet) {
	to = newDepthSet()
	for k := range from.m {
		to.add(k)
	}
	return
}

type verifiedCiphertext struct {
	verifiedDepths *depthSet
	ciphertext     *tfheCiphertext
}

type PrivilegedMemory struct {
	// A map from a ciphertext hash to itself and stack depths at which it is verified
	VerifiedCiphertexts map[common.Hash]*verifiedCiphertext

	// All optimistic requires encountered up to that point in the txn execution
	OptimisticRequires []*tfheCiphertext
}

var PrivilegedMempory *PrivilegedMemory = &PrivilegedMemory{
	make(map[common.Hash]*verifiedCiphertext),
	make([]*tfheCiphertext, 0),
}

// Evaluate remaining optimistic requires when Interpreter.Run get an errStopToken
//
// This function is meant to be integrated as part of vm.EVMInterpreter.Run in the case
// there was an errStopToken
func EvalRemOptReqWhenStopToken(env EVMEnvironment) (err error) {
	err = nil
	// If we are finishing execution (about to go to from depth 1 to depth 0), evaluate
	// any remaining optimistic requires.
	if env.GetDepth() == 1 {
		result, evalErr := evaluateRemainingOptimisticRequires(env)
		if evalErr != nil {
			err = evalErr
		} else if !result {
			err = ErrExecutionReverted
		}
	}
	return err
}
