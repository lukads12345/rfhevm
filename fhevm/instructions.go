package fhevm

import (
	"bytes"
	"encoding/hex"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	crypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	ps "github.com/zama-ai/fhevm-go/crypto"
)

var zero = uint256.NewInt(0).Bytes32()

func newInt(buf []byte) *uint256.Int {
	i := uint256.NewInt(0)
	return i.SetBytes(buf)
}

// Ciphertext metadata is stored in protected storage, in a 32-byte slot.
// Currently, we only utilize 17 bytes from the slot.
type ciphertextMetadata struct {
	refCount    uint64
	length      uint64
	fheUintType fheUintType
}

func (m ciphertextMetadata) serialize() [32]byte {
	u := uint256.NewInt(0)
	u[0] = m.refCount
	u[1] = m.length
	u[2] = uint64(m.fheUintType)
	return u.Bytes32()
}

func (m *ciphertextMetadata) deserialize(buf [32]byte) *ciphertextMetadata {
	u := uint256.NewInt(0)
	u.SetBytes(buf[:])
	m.refCount = u[0]
	m.length = u[1]
	m.fheUintType = fheUintType(u[2])
	return m
}

func newCiphertextMetadata(buf [32]byte) *ciphertextMetadata {
	m := ciphertextMetadata{}
	return m.deserialize(buf)
}

func minUint64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// If references are still left, reduce refCount by 1. Otherwise, zero out the metadata and the ciphertext slots.
func garbageCollectProtectedStorage(flagHandleLocation common.Hash, handle common.Hash, protectedStorage common.Address, env EVMEnvironment) {
	// The location of ciphertext metadata is at Keccak256(handle). Doing so avoids attacks from users trying to garbage
	// collect arbitrary locations in protected storage. Hashing the handle makes it hard to find a preimage such that
	// it ends up in arbitrary non-zero places in protected stroage.
	metadataKey := crypto.Keccak256Hash(handle.Bytes())

	existingMetadataHash := env.GetState(protectedStorage, metadataKey)
	existingMetadataInt := newInt(existingMetadataHash.Bytes())
	if !existingMetadataInt.IsZero() {
		logger := env.GetLogger()

		// If no flag in protected storage for the location, ignore garbage collection.
		// Else, set the value at the location to zero.
		foundFlag := env.GetState(protectedStorage, flagHandleLocation)
		if !bytes.Equal(foundFlag.Bytes(), flag.Bytes()) {
			logger.Error("opSstore location flag not found for a ciphertext handle, ignoring garbage collection",
				"expectedFlag", hex.EncodeToString(flag[:]),
				"foundFlag", hex.EncodeToString(foundFlag[:]),
				"flagHandleLocation", hex.EncodeToString(flagHandleLocation[:]))
			return
		} else {
			env.SetState(protectedStorage, flagHandleLocation, zero)
		}

		metadata := newCiphertextMetadata(existingMetadataInt.Bytes32())
		if metadata.refCount == 1 {
			if env.IsCommitting() {
				logger.Info("opSstore garbage collecting ciphertext",
					"protectedStorage", hex.EncodeToString(protectedStorage[:]),
					"metadataKey", hex.EncodeToString(metadataKey[:]),
					"type", metadata.fheUintType,
					"len", metadata.length)
			}

			// Zero the metadata key-value.
			env.SetState(protectedStorage, metadataKey, zero)

			// Set the slot to the one after the metadata one.
			slot := newInt(metadataKey.Bytes())
			slot.AddUint64(slot, 1)

			// Zero the ciphertext slots.
			slotsToZero := metadata.length / 32
			if metadata.length > 0 && metadata.length < 32 {
				slotsToZero++
			}
			for i := uint64(0); i < slotsToZero; i++ {
				env.SetState(protectedStorage, slot.Bytes32(), zero)
				slot.AddUint64(slot, 1)
			}
		} else if metadata.refCount > 1 {
			if env.IsCommitting() {
				logger.Info("opSstore decrementing ciphertext refCount",
					"protectedStorage", hex.EncodeToString(protectedStorage[:]),
					"metadataKey", hex.EncodeToString(metadataKey[:]),
					"type", metadata.fheUintType,
					"len", metadata.length)
			}
			metadata.refCount--
			env.SetState(protectedStorage, existingMetadataHash, metadata.serialize())
		}
	}
}

func isVerifiedAtCurrentDepth(environment EVMEnvironment, ct *verifiedCiphertext) bool {
	return ct.verifiedDepths.has(environment.GetDepth())
}

// Returns a pointer to the ciphertext if the given hash points to a verified ciphertext.
// Else, it returns nil.
func getVerifiedCiphertextFromEVM(environment EVMEnvironment, ciphertextHash common.Hash) *verifiedCiphertext {
	ct, ok := environment.GetFhevmData().verifiedCiphertexts[ciphertextHash]
	if ok && isVerifiedAtCurrentDepth(environment, ct) {
		return ct
	}
	return nil
}

func verifyIfCiphertextHandle(handle common.Hash, env EVMEnvironment, contractAddress common.Address) error {
	ct, ok := env.GetFhevmData().verifiedCiphertexts[handle]
	if ok {
		// If already existing in memory, skip storage and import the same ciphertext at the current depth.
		//
		// Also works for gas estimation - we don't persist anything to protected storage during gas estimation.
		// However, ciphertexts remain in memory for the duration of the call, allowing for this lookup to find it.
		// Note that even if a ciphertext has an empty verification depth set, it still remains in memory.
		importCiphertextToEVM(env, ct.ciphertext)
		return nil
	}

	metadataKey := crypto.Keccak256Hash(handle.Bytes())
	protectedStorage := ps.CreateProtectedStorageContractAddress(contractAddress)
	metadataInt := newInt(env.GetState(protectedStorage, metadataKey).Bytes())
	if !metadataInt.IsZero() {
		metadata := newCiphertextMetadata(metadataInt.Bytes32())
		ctBytes := make([]byte, 0)
		left := metadata.length
		protectedSlotIdx := newInt(metadataKey.Bytes())
		protectedSlotIdx.AddUint64(protectedSlotIdx, 1)
		for {
			if left == 0 {
				break
			}
			bytes := env.GetState(protectedStorage, protectedSlotIdx.Bytes32())
			toAppend := minUint64(uint64(len(bytes)), left)
			left -= toAppend
			ctBytes = append(ctBytes, bytes[0:toAppend]...)
			protectedSlotIdx.AddUint64(protectedSlotIdx, 1)
		}

		ct := new(tfheCiphertext)
		err := ct.deserialize(ctBytes, metadata.fheUintType)
		if err != nil {
			msg := "opSload failed to deserialize a ciphertext"
			env.GetLogger().Error(msg, "err", err)
			return errors.New(msg)
		}
		importCiphertextToEVM(env, ct)
	}
	return nil
}

func OpSload(pc *uint64, env EVMEnvironment, scope ScopeContext) ([]byte, error) {
	loc := scope.GetStack().Peek()
	hash := common.Hash(loc.Bytes32())
	val := env.GetState(scope.GetContract().Address(), hash)
	if err := verifyIfCiphertextHandle(val, env, scope.GetContract().Address()); err != nil {
		return nil, err
	}
	loc.SetBytes(val.Bytes())
	return nil, nil
}

// An arbitrary constant value to flag locations in protected storage.
var flag = common.HexToHash("0xa145ffde0100a145ffde0100a145ffde0100a145ffde0100a145ffde0100fab3")

// If a verified ciphertext:
// * if the ciphertext does not exist in protected storage, persist it with a refCount = 1
// * if the ciphertexts exists in protected, bump its refCount by 1
func persistIfVerifiedCiphertext(flagHandleLocation common.Hash, handle common.Hash, protectedStorage common.Address, env EVMEnvironment) {
	verifiedCiphertext := getVerifiedCiphertextFromEVM(env, handle)
	if verifiedCiphertext == nil {
		return
	}
	logger := env.GetLogger()

	// Try to read ciphertext metadata from protected storage.
	metadataKey := crypto.Keccak256Hash(handle.Bytes())
	metadataInt := newInt(env.GetState(protectedStorage, metadataKey).Bytes())
	metadata := ciphertextMetadata{}

	// Set flag in protected storage to mark the location as containing a handle.
	env.SetState(protectedStorage, flagHandleLocation, flag)

	if metadataInt.IsZero() {
		// If no metadata, it means this ciphertext itself hasn't been persisted to protected storage yet. We do that as part of SSTORE.
		metadata.refCount = 1
		metadata.length = uint64(expandedFheCiphertextSize[verifiedCiphertext.ciphertext.fheUintType])
		metadata.fheUintType = verifiedCiphertext.ciphertext.fheUintType
		ciphertextSlot := newInt(metadataKey.Bytes())
		ciphertextSlot.AddUint64(ciphertextSlot, 1)
		if env.IsCommitting() {
			logger.Info("opSstore persisting new ciphertext",
				"protectedStorage", hex.EncodeToString(protectedStorage[:]),
				"handle", hex.EncodeToString(handle.Bytes()),
				"type", metadata.fheUintType,
				"len", metadata.length,
				"ciphertextSlot", hex.EncodeToString(ciphertextSlot.Bytes()))
		}
		ctPart32 := make([]byte, 32)
		partIdx := 0
		ctBytes := verifiedCiphertext.ciphertext.serialize()
		for i, b := range ctBytes {
			if i%32 == 0 && i != 0 {
				env.SetState(protectedStorage, ciphertextSlot.Bytes32(), common.BytesToHash(ctPart32))
				ciphertextSlot.AddUint64(ciphertextSlot, 1)
				ctPart32 = make([]byte, 32)
				partIdx = 0
			}
			ctPart32[partIdx] = b
			partIdx++
		}
		if len(ctPart32) != 0 {
			env.SetState(protectedStorage, ciphertextSlot.Bytes32(), common.BytesToHash(ctPart32))
		}
	} else {
		// If metadata exists, bump the refcount by 1.
		metadata = *newCiphertextMetadata(env.GetState(protectedStorage, metadataKey))
		metadata.refCount++
		if env.IsCommitting() {
			logger.Info("opSstore bumping refcount of existing ciphertext",
				"protectedStorage", hex.EncodeToString(protectedStorage[:]),
				"handle", hex.EncodeToString(handle.Bytes()),
				"type", metadata.fheUintType,
				"len", metadata.length,
				"refCount", metadata.refCount)
		}
	}
	// Save the metadata in protected storage.
	env.SetState(protectedStorage, metadataKey, metadata.serialize())
}

func OpSstore(pc *uint64, env EVMEnvironment, scope ScopeContext) ([]byte, error) {
	if env.IsReadOnly() {
		return nil, ErrWriteProtection
	}
	loc := scope.GetStack().Pop()
	locHash := common.BytesToHash(loc.Bytes())
	newVal := scope.GetStack().Pop()
	newValHash := common.BytesToHash(newVal.Bytes())
	oldValHash := env.GetState(scope.GetContract().Address(), common.Hash(loc.Bytes32()))
	// If the value is the same or if we are not going to commit, don't do anything to protected storage.
	if newValHash != oldValHash && env.IsCommitting() {
		protectedStorage := ps.CreateProtectedStorageContractAddress(scope.GetContract().Address())

		// Define flag location as keccak256(keccak256(loc)) in protected storage. Used to mark the location as containing a handle.
		// Note: We apply the hash function twice to make sure a flag location in protected storage cannot clash with a ciphertext
		// metadata location that is keccak256(keccak256(ciphertext)). Since a location is 32 bytes, it cannot clash with a well-formed
		// ciphertext. Therefore, there needs to be a hash collistion for a clash to happen. If hash is applied only once, there could
		// be a collision, since malicous users could store at loc = keccak256(ciphertext), making the flag clash with metadata.
		flagHandleLocation := crypto.Keccak256Hash(crypto.Keccak256Hash(locHash[:]).Bytes())

		// Since the old value is no longer stored in actual contract storage, run garbage collection on protected storage.
		garbageCollectProtectedStorage(flagHandleLocation, oldValHash, protectedStorage, env)

		// If a verified ciphertext, persist to protected storage.
		persistIfVerifiedCiphertext(flagHandleLocation, newValHash, protectedStorage, env)
	}
	// Set the SSTORE's value in the actual contract.
	env.SetState(scope.GetContract().Address(), loc.Bytes32(), newValHash)
	return nil, nil
}
