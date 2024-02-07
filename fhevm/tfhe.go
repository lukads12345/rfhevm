// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package fhevm

/*
#cgo linux CFLAGS: -O3 -I../tfhe-rs/target/release -I../tfhe-rs/target/release/deps
#cgo linux LDFLAGS: -L../tfhe-rs/target/release -l:libtfhe.a -L../tfhe-rs/target/release/deps -l:libtfhe_c_api_dynamic_buffer.a -lm
#cgo darwin CFLAGS: -O3 -I../tfhe-rs/target/release -I../tfhe-rs/target/release/deps
#cgo darwin LDFLAGS: -framework Security -L../tfhe-rs/target/release -ltfhe -L../tfhe-rs/target/release/deps -ltfhe_c_api_dynamic_buffer -lm

#include "tfhe_wrappers.h"

*/
import "C"

import (
	_ "embed"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func toDynamicBufferView(in []byte) C.DynamicBufferView {
	return C.DynamicBufferView{
		pointer: (*C.uint8_t)(unsafe.Pointer(&in[0])),
		length:  (C.size_t)(len(in)),
	}
}

// Expanded TFHE ciphertext sizes by type, in bytes.
var expandedFheCiphertextSize map[FheUintType]uint

// Compact TFHE ciphertext sizes by type, in bytes.
var compactFheCiphertextSize map[FheUintType]uint

// server key: evaluation key
var sks unsafe.Pointer

// client key: secret key
var cks unsafe.Pointer

// public key
var pks unsafe.Pointer
var pksHash common.Hash

// Generate keys for the fhevm (sks, cks, psk)
func generateFhevmKeys() (unsafe.Pointer, unsafe.Pointer, unsafe.Pointer) {
	var keys = C.generate_fhevm_keys()
	return keys.sks, keys.cks, keys.pks
}

func allGlobalKeysPresent() bool {
	return sks != nil && cks != nil && pks != nil
}

func initGlobalKeysWithNewKeys() {
	sks, cks, pks = generateFhevmKeys()
	initCiphertextSizes()
}

func initCiphertextSizes() {
	expandedFheCiphertextSize = make(map[FheUintType]uint)
	compactFheCiphertextSize = make(map[FheUintType]uint)

	expandedFheCiphertextSize[FheUint8] = uint(len(new(tfheCiphertext).trivialEncrypt(*big.NewInt(0), FheUint8).serialize()))
	expandedFheCiphertextSize[FheUint16] = uint(len(new(tfheCiphertext).trivialEncrypt(*big.NewInt(0), FheUint16).serialize()))
	expandedFheCiphertextSize[FheUint32] = uint(len(new(tfheCiphertext).trivialEncrypt(*big.NewInt(0), FheUint32).serialize()))
	expandedFheCiphertextSize[FheUint64] = uint(len(new(tfheCiphertext).trivialEncrypt(*big.NewInt(0), FheUint64).serialize()))

	compactFheCiphertextSize[FheUint8] = uint(len(encryptAndSerializeCompact(0, FheUint8)))
	compactFheCiphertextSize[FheUint16] = uint(len(encryptAndSerializeCompact(0, FheUint16)))
	compactFheCiphertextSize[FheUint32] = uint(len(encryptAndSerializeCompact(0, FheUint32)))
	compactFheCiphertextSize[FheUint64] = uint(len(encryptAndSerializeCompact(0, FheUint64)))
}

func InitGlobalKeysFromFiles(keysDir string) error {
	if _, err := os.Stat(keysDir); os.IsNotExist(err) {
		return fmt.Errorf("init_keys: global keys directory doesn't exist (FHEVM_GO_KEYS_DIR): %s", keysDir)
	}
	// read keys from files
	var sksPath = path.Join(keysDir, "sks")
	sksBytes, err := os.ReadFile(sksPath)
	if err != nil {
		return err
	}
	var pksPath = path.Join(keysDir, "pks")
	pksBytes, err := os.ReadFile(pksPath)
	if err != nil {
		return err
	}

	sks = C.deserialize_server_key(toDynamicBufferView(sksBytes))

	pksHash = crypto.Keccak256Hash(pksBytes)
	pks = C.deserialize_compact_public_key(toDynamicBufferView(pksBytes))

	initCiphertextSizes()

	fmt.Println("INFO: global keys loaded from: " + keysDir)

	return nil
}

// initialize keys automatically only if FHEVM_GO_KEYS_DIR is set
func init() {
	var keysDirPath, present = os.LookupEnv("FHEVM_GO_KEYS_DIR")
	if present {
		err := InitGlobalKeysFromFiles(keysDirPath)
		if err != nil {
			panic(err)
		}
		fmt.Println("INFO: global keys are initialized automatically using FHEVM_GO_KEYS_DIR env variable")
	} else {
		fmt.Println("INFO: global keys aren't initialized automatically (FHEVM_GO_KEYS_DIR env variable not set)")
	}
}

func serialize(ptr unsafe.Pointer, t FheUintType) ([]byte, error) {
	out := &C.DynamicBuffer{}
	var ret C.int
	switch t {
	case FheUint8:
		ret = C.serialize_fhe_uint8(ptr, out)
	case FheUint16:
		ret = C.serialize_fhe_uint16(ptr, out)
	case FheUint32:
		ret = C.serialize_fhe_uint32(ptr, out)
	case FheUint64:
		ret = C.serialize_fhe_uint64(ptr, out)
	default:
		panic("serialize: unexpected ciphertext type")
	}
	if ret != 0 {
		return nil, errors.New("serialize: failed to serialize a ciphertext")
	}
	ser := C.GoBytes(unsafe.Pointer(out.pointer), C.int(out.length))
	C.destroy_dynamic_buffer(out)
	return ser, nil
}

func serializePublicKey(pks unsafe.Pointer) ([]byte, error) {
	out := &C.DynamicBuffer{}
	var ret C.int
	ret = C.serialize_compact_public_key(pks, out)
	if ret != 0 {
		return nil, errors.New("serialize: failed to serialize public key")
	}
	ser := C.GoBytes(unsafe.Pointer(out.pointer), C.int(out.length))
	C.destroy_dynamic_buffer(out)
	return ser, nil
}

// Represents a TFHE ciphertext type, i.e. its bit capacity.
type FheUintType uint8

const (
	FheUint8  FheUintType = 0
	FheUint16 FheUintType = 1
	FheUint32 FheUintType = 2
	FheUint64 FheUintType = 3
)

// Represents an expanded TFHE ciphertext.
type tfheCiphertext struct {
	serialization []byte
	hash          *common.Hash
	fheUintType   FheUintType
}

// Deserializes a TFHE ciphertext.
func (ct *tfheCiphertext) deserialize(in []byte, t FheUintType) error {
	switch t {
	case FheUint8:
		ptr := C.deserialize_fhe_uint8(toDynamicBufferView((in)))
		if ptr == nil {
			return errors.New("FheUint8 ciphertext deserialization failed")
		}
		C.destroy_fhe_uint8(ptr)
	case FheUint16:
		ptr := C.deserialize_fhe_uint16(toDynamicBufferView((in)))
		if ptr == nil {
			return errors.New("FheUint16 ciphertext deserialization failed")
		}
		C.destroy_fhe_uint16(ptr)
	case FheUint32:
		ptr := C.deserialize_fhe_uint32(toDynamicBufferView((in)))
		if ptr == nil {
			return errors.New("FheUint32 ciphertext deserialization failed")
		}
		C.destroy_fhe_uint32(ptr)
	case FheUint64:
		ptr := C.deserialize_fhe_uint64(toDynamicBufferView((in)))
		if ptr == nil {
			return errors.New("FheUint64 ciphertext deserialization failed")
		}
		C.destroy_fhe_uint64(ptr)
	default:
		panic("deserialize: unexpected ciphertext type")
	}
	ct.fheUintType = t
	ct.serialization = in
	ct.computeHash()
	return nil
}

// Deserializes a compact TFHE ciphetext.
// Note: After the compact TFHE ciphertext has been serialized, subsequent calls to serialize()
// will produce non-compact ciphertext serialziations.
func (ct *tfheCiphertext) deserializeCompact(in []byte, t FheUintType) error {
	switch t {
	case FheUint8:
		ptr := C.deserialize_compact_fhe_uint8(toDynamicBufferView((in)))
		if ptr == nil {
			return errors.New("compact FheUint8 ciphertext deserialization failed")
		}
		var err error
		ct.serialization, err = serialize(ptr, t)
		C.destroy_fhe_uint8(ptr)
		if err != nil {
			return err
		}
	case FheUint16:
		ptr := C.deserialize_compact_fhe_uint16(toDynamicBufferView((in)))
		if ptr == nil {
			return errors.New("compact FheUint16 ciphertext deserialization failed")
		}
		var err error
		ct.serialization, err = serialize(ptr, t)
		C.destroy_fhe_uint16(ptr)
		if err != nil {
			return err
		}
	case FheUint32:
		ptr := C.deserialize_compact_fhe_uint32(toDynamicBufferView((in)))
		if ptr == nil {
			return errors.New("compact FheUint32 ciphertext deserialization failed")
		}
		var err error
		ct.serialization, err = serialize(ptr, t)
		C.destroy_fhe_uint32(ptr)
		if err != nil {
			return err
		}
	case FheUint64:
		ptr := C.deserialize_compact_fhe_uint64(toDynamicBufferView((in)))
		if ptr == nil {
			return errors.New("compact FheUint64 ciphertext deserialization failed")
		}
		var err error
		ct.serialization, err = serialize(ptr, t)
		C.destroy_fhe_uint64(ptr)
		if err != nil {
			return err
		}
	default:
		panic("deserializeCompact: unexpected ciphertext type")
	}
	ct.fheUintType = t
	ct.computeHash()
	return nil
}

// Encrypts a value as a TFHE ciphertext, using the compact public FHE key.
// The resulting ciphertext is automaticaly expanded.
func (ct *tfheCiphertext) encrypt(value big.Int, t FheUintType) *tfheCiphertext {
	var ptr unsafe.Pointer
	var err error
	switch t {
	case FheUint8:
		ptr = C.public_key_encrypt_fhe_uint8(pks, C.uint8_t(value.Uint64()))
		ct.serialization, err = serialize(ptr, t)
		C.destroy_fhe_uint8(ptr)
		if err != nil {
			panic(err)
		}
	case FheUint16:
		ptr = C.public_key_encrypt_fhe_uint16(pks, C.uint16_t(value.Uint64()))
		ct.serialization, err = serialize(ptr, t)
		C.destroy_fhe_uint16(ptr)
		if err != nil {
			panic(err)
		}
	case FheUint32:
		ptr = C.public_key_encrypt_fhe_uint32(pks, C.uint32_t(value.Uint64()))
		ct.serialization, err = serialize(ptr, t)
		C.destroy_fhe_uint32(ptr)
		if err != nil {
			panic(err)
		}
	case FheUint64:
		ptr = C.public_key_encrypt_fhe_uint64(pks, C.uint64_t(value.Uint64()))
		ct.serialization, err = serialize(ptr, t)
		C.destroy_fhe_uint64(ptr)
		if err != nil {
			panic(err)
		}
	default:
		panic("encrypt: unexpected ciphertext type")
	}
	ct.fheUintType = t
	ct.computeHash()
	return ct
}

func (ct *tfheCiphertext) trivialEncrypt(value big.Int, t FheUintType) *tfheCiphertext {
	var ptr unsafe.Pointer
	var err error
	switch t {
	case FheUint8:
		ptr = C.trivial_encrypt_fhe_uint8(sks, C.uint8_t(value.Uint64()))
		ct.serialization, err = serialize(ptr, t)
		C.destroy_fhe_uint8(ptr)
		if err != nil {
			panic(err)
		}
	case FheUint16:
		ptr = C.trivial_encrypt_fhe_uint16(sks, C.uint16_t(value.Uint64()))
		ct.serialization, err = serialize(ptr, t)
		C.destroy_fhe_uint16(ptr)
		if err != nil {
			panic(err)
		}
	case FheUint32:
		ptr = C.trivial_encrypt_fhe_uint32(sks, C.uint32_t(value.Uint64()))
		ct.serialization, err = serialize(ptr, t)
		C.destroy_fhe_uint32(ptr)
		if err != nil {
			panic(err)
		}
	case FheUint64:
		ptr = C.trivial_encrypt_fhe_uint64(sks, C.uint64_t(value.Uint64()))
		ct.serialization, err = serialize(ptr, t)
		C.destroy_fhe_uint64(ptr)
		if err != nil {
			panic(err)
		}
	default:
		panic("trivialEncrypt: unexpected ciphertext type")
	}
	ct.fheUintType = t
	ct.computeHash()
	return ct
}

func (ct *tfheCiphertext) serialize() []byte {
	return ct.serialization
}

func (ct *tfheCiphertext) executeUnaryCiphertextOperation(rhs *tfheCiphertext,
	op8 func(ct unsafe.Pointer) unsafe.Pointer,
	op16 func(ct unsafe.Pointer) unsafe.Pointer,
	op32 func(ct unsafe.Pointer) unsafe.Pointer,
	op64 func(ct unsafe.Pointer) unsafe.Pointer) (*tfheCiphertext, error) {

	res := new(tfheCiphertext)
	res.fheUintType = ct.fheUintType
	res_ser := &C.DynamicBuffer{}
	switch ct.fheUintType {
	case FheUint8:
		ct_ptr := C.deserialize_fhe_uint8(toDynamicBufferView((ct.serialization)))
		if ct_ptr == nil {
			return nil, errors.New("8 bit unary op deserialization failed")
		}
		res_ptr := op8(ct_ptr)
		C.destroy_fhe_uint8(ct_ptr)
		if res_ptr == nil {
			return nil, errors.New("8 bit unary op failed")
		}
		ret := C.serialize_fhe_uint8(res_ptr, res_ser)
		C.destroy_fhe_uint8(res_ptr)
		if ret != 0 {
			return nil, errors.New("8 bit unary op serialization failed")
		}
		res.serialization = C.GoBytes(unsafe.Pointer(res_ser.pointer), C.int(res_ser.length))
		C.destroy_dynamic_buffer(res_ser)
	case FheUint16:
		ct_ptr := C.deserialize_fhe_uint16(toDynamicBufferView((ct.serialization)))
		if ct_ptr == nil {
			return nil, errors.New("16 bit unary op deserialization failed")
		}
		res_ptr := op16(ct_ptr)
		C.destroy_fhe_uint16(ct_ptr)
		if res_ptr == nil {
			return nil, errors.New("16 bit op failed")
		}
		ret := C.serialize_fhe_uint16(res_ptr, res_ser)
		C.destroy_fhe_uint16(res_ptr)
		if ret != 0 {
			return nil, errors.New("16 bit unary op serialization failed")
		}
		res.serialization = C.GoBytes(unsafe.Pointer(res_ser.pointer), C.int(res_ser.length))
		C.destroy_dynamic_buffer(res_ser)
	case FheUint32:
		ct_ptr := C.deserialize_fhe_uint32(toDynamicBufferView((ct.serialization)))
		if ct_ptr == nil {
			return nil, errors.New("32 bit unary op deserialization failed")
		}
		res_ptr := op32(ct_ptr)
		C.destroy_fhe_uint32(ct_ptr)
		if res_ptr == nil {
			return nil, errors.New("32 bit op failed")
		}
		ret := C.serialize_fhe_uint32(res_ptr, res_ser)
		C.destroy_fhe_uint32(res_ptr)
		if ret != 0 {
			return nil, errors.New("32 bit unary op serialization failed")
		}
		res.serialization = C.GoBytes(unsafe.Pointer(res_ser.pointer), C.int(res_ser.length))
		C.destroy_dynamic_buffer(res_ser)
	case FheUint64:
		ct_ptr := C.deserialize_fhe_uint64(toDynamicBufferView((ct.serialization)))
		if ct_ptr == nil {
			return nil, errors.New("64 bit unary op deserialization failed")
		}
		res_ptr := op64(ct_ptr)
		C.destroy_fhe_uint64(ct_ptr)
		if res_ptr == nil {
			return nil, errors.New("64 bit op failed")
		}
		ret := C.serialize_fhe_uint64(res_ptr, res_ser)
		C.destroy_fhe_uint64(res_ptr)
		if ret != 0 {
			return nil, errors.New("64 bit unary op serialization failed")
		}
		res.serialization = C.GoBytes(unsafe.Pointer(res_ser.pointer), C.int(res_ser.length))
		C.destroy_dynamic_buffer(res_ser)
	default:
		panic("unary op unexpected ciphertext type")
	}
	res.computeHash()
	return res, nil
}

func (lhs *tfheCiphertext) executeBinaryCiphertextOperation(rhs *tfheCiphertext,
	op8 func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer,
	op16 func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer,
	op32 func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer,
	op64 func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer) (*tfheCiphertext, error) {
	if lhs.fheUintType != rhs.fheUintType {
		return nil, errors.New("binary operations are only well-defined for identical types")
	}

	res := new(tfheCiphertext)
	res.fheUintType = lhs.fheUintType
	res_ser := &C.DynamicBuffer{}
	switch lhs.fheUintType {
	case FheUint8:
		lhs_ptr := C.deserialize_fhe_uint8(toDynamicBufferView((lhs.serialization)))
		if lhs_ptr == nil {
			return nil, errors.New("8 bit binary op deserialization failed")
		}
		rhs_ptr := C.deserialize_fhe_uint8(toDynamicBufferView((rhs.serialization)))
		if rhs_ptr == nil {
			C.destroy_fhe_uint8(lhs_ptr)
			return nil, errors.New("8 bit binary op deserialization failed")
		}
		res_ptr := op8(lhs_ptr, rhs_ptr)
		C.destroy_fhe_uint8(lhs_ptr)
		C.destroy_fhe_uint8(rhs_ptr)
		if res_ptr == nil {
			return nil, errors.New("8 bit binary op failed")
		}
		ret := C.serialize_fhe_uint8(res_ptr, res_ser)
		C.destroy_fhe_uint8(res_ptr)
		if ret != 0 {
			return nil, errors.New("8 bit binary op serialization failed")
		}
		res.serialization = C.GoBytes(unsafe.Pointer(res_ser.pointer), C.int(res_ser.length))
		C.destroy_dynamic_buffer(res_ser)
	case FheUint16:
		lhs_ptr := C.deserialize_fhe_uint16(toDynamicBufferView((lhs.serialization)))
		if lhs_ptr == nil {
			return nil, errors.New("16 bit binary op deserialization failed")
		}
		rhs_ptr := C.deserialize_fhe_uint16(toDynamicBufferView((rhs.serialization)))
		if rhs_ptr == nil {
			C.destroy_fhe_uint16(lhs_ptr)
			return nil, errors.New("16 bit binary op deserialization failed")
		}
		res_ptr := op16(lhs_ptr, rhs_ptr)
		C.destroy_fhe_uint16(lhs_ptr)
		C.destroy_fhe_uint16(rhs_ptr)
		if res_ptr == nil {
			return nil, errors.New("16 bit binary op failed")
		}
		ret := C.serialize_fhe_uint16(res_ptr, res_ser)
		C.destroy_fhe_uint16(res_ptr)
		if ret != 0 {
			return nil, errors.New("16 bit binary op serialization failed")
		}
		res.serialization = C.GoBytes(unsafe.Pointer(res_ser.pointer), C.int(res_ser.length))
		C.destroy_dynamic_buffer(res_ser)
	case FheUint32:
		lhs_ptr := C.deserialize_fhe_uint32(toDynamicBufferView((lhs.serialization)))
		if lhs_ptr == nil {
			return nil, errors.New("32 bit binary op deserialization failed")
		}
		rhs_ptr := C.deserialize_fhe_uint32(toDynamicBufferView((rhs.serialization)))
		if rhs_ptr == nil {
			C.destroy_fhe_uint32(lhs_ptr)
			return nil, errors.New("32 bit binary op deserialization failed")
		}
		res_ptr := op32(lhs_ptr, rhs_ptr)
		C.destroy_fhe_uint32(lhs_ptr)
		C.destroy_fhe_uint32(rhs_ptr)
		if res_ptr == nil {
			return nil, errors.New("32 bit binary op failed")
		}
		ret := C.serialize_fhe_uint32(res_ptr, res_ser)
		C.destroy_fhe_uint32(res_ptr)
		if ret != 0 {
			return nil, errors.New("32 bit binary op serialization failed")
		}
		res.serialization = C.GoBytes(unsafe.Pointer(res_ser.pointer), C.int(res_ser.length))
		C.destroy_dynamic_buffer(res_ser)
	case FheUint64:
		lhs_ptr := C.deserialize_fhe_uint64(toDynamicBufferView((lhs.serialization)))
		if lhs_ptr == nil {
			return nil, errors.New("64 bit binary op deserialization failed")
		}
		rhs_ptr := C.deserialize_fhe_uint64(toDynamicBufferView((rhs.serialization)))
		if rhs_ptr == nil {
			C.destroy_fhe_uint64(lhs_ptr)
			return nil, errors.New("64 bit binary op deserialization failed")
		}
		res_ptr := op64(lhs_ptr, rhs_ptr)
		C.destroy_fhe_uint64(lhs_ptr)
		C.destroy_fhe_uint64(rhs_ptr)
		if res_ptr == nil {
			return nil, errors.New("64 bit binary op failed")
		}
		ret := C.serialize_fhe_uint64(res_ptr, res_ser)
		C.destroy_fhe_uint64(res_ptr)
		if ret != 0 {
			return nil, errors.New("64 bit binary op serialization failed")
		}
		res.serialization = C.GoBytes(unsafe.Pointer(res_ser.pointer), C.int(res_ser.length))
		C.destroy_dynamic_buffer(res_ser)
	default:
		panic("binary op unexpected ciphertext type")
	}
	res.computeHash()
	return res, nil
}

func (first *tfheCiphertext) executeTernaryCiphertextOperation(lhs *tfheCiphertext, rhs *tfheCiphertext,
	op8 func(first unsafe.Pointer, lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer,
	op16 func(first unsafe.Pointer, lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer,
	op32 func(first unsafe.Pointer, lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer,
	op64 func(first unsafe.Pointer, lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer) (*tfheCiphertext, error) {
	if lhs.fheUintType != rhs.fheUintType {
		return nil, errors.New("ternary operations are only well-defined for identical types")
	}

	res := new(tfheCiphertext)
	res.fheUintType = lhs.fheUintType
	res_ser := &C.DynamicBuffer{}
	switch lhs.fheUintType {
	case FheUint8:
		lhs_ptr := C.deserialize_fhe_uint8(toDynamicBufferView((lhs.serialization)))
		if lhs_ptr == nil {
			return nil, errors.New("8 bit binary op deserialization failed")
		}
		rhs_ptr := C.deserialize_fhe_uint8(toDynamicBufferView((rhs.serialization)))
		if rhs_ptr == nil {
			C.destroy_fhe_uint8(lhs_ptr)
			return nil, errors.New("8 bit binary op deserialization failed")
		}
		first_ptr := C.deserialize_fhe_uint8(toDynamicBufferView((first.serialization)))
		if first_ptr == nil {
			C.destroy_fhe_uint8(lhs_ptr)
			C.destroy_fhe_uint8(rhs_ptr)
			return nil, errors.New("8 bit binary op deserialization failed")
		}
		res_ptr := op8(first_ptr, lhs_ptr, rhs_ptr)
		C.destroy_fhe_uint8(lhs_ptr)
		C.destroy_fhe_uint8(rhs_ptr)
		if res_ptr == nil {
			return nil, errors.New("8 bit binary op failed")
		}
		ret := C.serialize_fhe_uint8(res_ptr, res_ser)
		C.destroy_fhe_uint8(res_ptr)
		if ret != 0 {
			return nil, errors.New("8 bit binary op serialization failed")
		}
		res.serialization = C.GoBytes(unsafe.Pointer(res_ser.pointer), C.int(res_ser.length))
		C.destroy_dynamic_buffer(res_ser)
	case FheUint16:
		lhs_ptr := C.deserialize_fhe_uint16(toDynamicBufferView((lhs.serialization)))
		if lhs_ptr == nil {
			return nil, errors.New("16 bit binary op deserialization failed")
		}
		rhs_ptr := C.deserialize_fhe_uint16(toDynamicBufferView((rhs.serialization)))
		if rhs_ptr == nil {
			C.destroy_fhe_uint16(lhs_ptr)
			return nil, errors.New("16 bit binary op deserialization failed")
		}
		first_ptr := C.deserialize_fhe_uint8(toDynamicBufferView((first.serialization)))
		if first_ptr == nil {
			C.destroy_fhe_uint8(lhs_ptr)
			C.destroy_fhe_uint8(rhs_ptr)
			return nil, errors.New("8 bit binary op deserialization failed")
		}
		res_ptr := op16(first_ptr, lhs_ptr, rhs_ptr)
		C.destroy_fhe_uint16(lhs_ptr)
		C.destroy_fhe_uint16(rhs_ptr)
		if res_ptr == nil {
			return nil, errors.New("16 bit binary op failed")
		}
		ret := C.serialize_fhe_uint16(res_ptr, res_ser)
		C.destroy_fhe_uint16(res_ptr)
		if ret != 0 {
			return nil, errors.New("16 bit binary op serialization failed")
		}
		res.serialization = C.GoBytes(unsafe.Pointer(res_ser.pointer), C.int(res_ser.length))
		C.destroy_dynamic_buffer(res_ser)
	case FheUint32:
		lhs_ptr := C.deserialize_fhe_uint32(toDynamicBufferView((lhs.serialization)))
		if lhs_ptr == nil {
			return nil, errors.New("32 bit binary op deserialization failed")
		}
		rhs_ptr := C.deserialize_fhe_uint32(toDynamicBufferView((rhs.serialization)))
		if rhs_ptr == nil {
			C.destroy_fhe_uint32(lhs_ptr)
			return nil, errors.New("32 bit binary op deserialization failed")
		}
		first_ptr := C.deserialize_fhe_uint8(toDynamicBufferView((first.serialization)))
		if first_ptr == nil {
			C.destroy_fhe_uint8(lhs_ptr)
			C.destroy_fhe_uint8(rhs_ptr)
			return nil, errors.New("8 bit binary op deserialization failed")
		}
		res_ptr := op32(first_ptr, lhs_ptr, rhs_ptr)
		C.destroy_fhe_uint32(lhs_ptr)
		C.destroy_fhe_uint32(rhs_ptr)
		if res_ptr == nil {
			return nil, errors.New("32 bit binary op failed")
		}
		ret := C.serialize_fhe_uint32(res_ptr, res_ser)
		C.destroy_fhe_uint32(res_ptr)
		if ret != 0 {
			return nil, errors.New("32 bit binary op serialization failed")
		}
		res.serialization = C.GoBytes(unsafe.Pointer(res_ser.pointer), C.int(res_ser.length))
		C.destroy_dynamic_buffer(res_ser)
	case FheUint64:
		lhs_ptr := C.deserialize_fhe_uint64(toDynamicBufferView((lhs.serialization)))
		if lhs_ptr == nil {
			return nil, errors.New("64 bit binary op deserialization failed")
		}
		rhs_ptr := C.deserialize_fhe_uint64(toDynamicBufferView((rhs.serialization)))
		if rhs_ptr == nil {
			C.destroy_fhe_uint64(lhs_ptr)
			return nil, errors.New("64 bit binary op deserialization failed")
		}
		first_ptr := C.deserialize_fhe_uint8(toDynamicBufferView((first.serialization)))
		if first_ptr == nil {
			C.destroy_fhe_uint8(lhs_ptr)
			C.destroy_fhe_uint8(rhs_ptr)
			return nil, errors.New("8 bit binary op deserialization failed")
		}
		res_ptr := op64(first_ptr, lhs_ptr, rhs_ptr)
		C.destroy_fhe_uint64(lhs_ptr)
		C.destroy_fhe_uint64(rhs_ptr)
		if res_ptr == nil {
			return nil, errors.New("64 bit binary op failed")
		}
		ret := C.serialize_fhe_uint64(res_ptr, res_ser)
		C.destroy_fhe_uint64(res_ptr)
		if ret != 0 {
			return nil, errors.New("64 bit binary op serialization failed")
		}
		res.serialization = C.GoBytes(unsafe.Pointer(res_ser.pointer), C.int(res_ser.length))
		C.destroy_dynamic_buffer(res_ser)
	default:
		panic("binary op unexpected ciphertext type")
	}
	res.computeHash()
	return res, nil
}

func (lhs *tfheCiphertext) executeBinaryScalarOperation(rhs uint64,
	op8 func(lhs unsafe.Pointer, rhs C.uint8_t) unsafe.Pointer,
	op16 func(lhs unsafe.Pointer, rhs C.uint16_t) unsafe.Pointer,
	op32 func(lhs unsafe.Pointer, rhs C.uint32_t) unsafe.Pointer,
	op64 func(lhs unsafe.Pointer, rhs C.uint64_t) unsafe.Pointer) (*tfheCiphertext, error) {
	res := new(tfheCiphertext)
	res.fheUintType = lhs.fheUintType
	res_ser := &C.DynamicBuffer{}
	switch lhs.fheUintType {
	case FheUint8:
		lhs_ptr := C.deserialize_fhe_uint8(toDynamicBufferView((lhs.serialization)))
		if lhs_ptr == nil {
			return nil, errors.New("8 bit scalar op deserialization failed")
		}
		scalar := C.uint8_t(rhs)
		res_ptr := op8(lhs_ptr, scalar)
		C.destroy_fhe_uint8(lhs_ptr)
		if res_ptr == nil {
			return nil, errors.New("8 bit scalar op failed")
		}
		ret := C.serialize_fhe_uint8(res_ptr, res_ser)
		C.destroy_fhe_uint8(res_ptr)
		if ret != 0 {
			return nil, errors.New("8 bit scalar op serialization failed")
		}
		res.serialization = C.GoBytes(unsafe.Pointer(res_ser.pointer), C.int(res_ser.length))
		C.destroy_dynamic_buffer(res_ser)
	case FheUint16:
		lhs_ptr := C.deserialize_fhe_uint16(toDynamicBufferView((lhs.serialization)))
		if lhs_ptr == nil {
			return nil, errors.New("16 bit scalar op deserialization failed")
		}
		scalar := C.uint16_t(rhs)
		res_ptr := op16(lhs_ptr, scalar)
		C.destroy_fhe_uint16(lhs_ptr)
		if res_ptr == nil {
			return nil, errors.New("16 bit scalar op failed")
		}
		ret := C.serialize_fhe_uint16(res_ptr, res_ser)
		C.destroy_fhe_uint16(res_ptr)
		if ret != 0 {
			return nil, errors.New("16 bit scalar op serialization failed")
		}
		res.serialization = C.GoBytes(unsafe.Pointer(res_ser.pointer), C.int(res_ser.length))
		C.destroy_dynamic_buffer(res_ser)
	case FheUint32:
		lhs_ptr := C.deserialize_fhe_uint32(toDynamicBufferView((lhs.serialization)))
		if lhs_ptr == nil {
			return nil, errors.New("32 bit scalar op deserialization failed")
		}
		scalar := C.uint32_t(rhs)
		res_ptr := op32(lhs_ptr, scalar)
		C.destroy_fhe_uint32(lhs_ptr)
		if res_ptr == nil {
			return nil, errors.New("32 bit scalar op failed")
		}
		ret := C.serialize_fhe_uint32(res_ptr, res_ser)
		C.destroy_fhe_uint32(res_ptr)
		if ret != 0 {
			return nil, errors.New("32 bit scalar op serialization failed")
		}
		res.serialization = C.GoBytes(unsafe.Pointer(res_ser.pointer), C.int(res_ser.length))
		C.destroy_dynamic_buffer(res_ser)
	case FheUint64:
		lhs_ptr := C.deserialize_fhe_uint64(toDynamicBufferView((lhs.serialization)))
		if lhs_ptr == nil {
			return nil, errors.New("64 bit scalar op deserialization failed")
		}
		scalar := C.uint64_t(rhs)
		res_ptr := op64(lhs_ptr, scalar)
		C.destroy_fhe_uint64(lhs_ptr)
		if res_ptr == nil {
			return nil, errors.New("64 bit scalar op failed")
		}
		ret := C.serialize_fhe_uint64(res_ptr, res_ser)
		C.destroy_fhe_uint64(res_ptr)
		if ret != 0 {
			return nil, errors.New("64 bit scalar op serialization failed")
		}
		res.serialization = C.GoBytes(unsafe.Pointer(res_ser.pointer), C.int(res_ser.length))
		C.destroy_dynamic_buffer(res_ser)
	default:
		panic("scalar op unexpected ciphertext type")
	}
	res.computeHash()
	return res, nil
}

func (lhs *tfheCiphertext) add(rhs *tfheCiphertext) (*tfheCiphertext, error) {
	return lhs.executeBinaryCiphertextOperation(rhs,
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.add_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.add_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.add_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.add_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) scalarAdd(rhs uint64) (*tfheCiphertext, error) {
	return lhs.executeBinaryScalarOperation(rhs,
		func(lhs unsafe.Pointer, rhs C.uint8_t) unsafe.Pointer {
			return C.scalar_add_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint16_t) unsafe.Pointer {
			return C.scalar_add_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint32_t) unsafe.Pointer {
			return C.scalar_add_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint64_t) unsafe.Pointer {
			return C.scalar_add_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) sub(rhs *tfheCiphertext) (*tfheCiphertext, error) {
	return lhs.executeBinaryCiphertextOperation(rhs,
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.sub_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.sub_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.sub_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.sub_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) scalarSub(rhs uint64) (*tfheCiphertext, error) {
	return lhs.executeBinaryScalarOperation(rhs,
		func(lhs unsafe.Pointer, rhs C.uint8_t) unsafe.Pointer {
			return C.scalar_sub_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint16_t) unsafe.Pointer {
			return C.scalar_sub_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint32_t) unsafe.Pointer {
			return C.scalar_sub_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint64_t) unsafe.Pointer {
			return C.scalar_sub_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) mul(rhs *tfheCiphertext) (*tfheCiphertext, error) {
	return lhs.executeBinaryCiphertextOperation(rhs,
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.mul_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.mul_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.mul_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.mul_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) scalarMul(rhs uint64) (*tfheCiphertext, error) {
	return lhs.executeBinaryScalarOperation(rhs,
		func(lhs unsafe.Pointer, rhs C.uint8_t) unsafe.Pointer {
			return C.scalar_mul_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint16_t) unsafe.Pointer {
			return C.scalar_mul_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint32_t) unsafe.Pointer {
			return C.scalar_mul_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint64_t) unsafe.Pointer {
			return C.scalar_mul_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) scalarDiv(rhs uint64) (*tfheCiphertext, error) {
	return lhs.executeBinaryScalarOperation(rhs,
		func(lhs unsafe.Pointer, rhs C.uint8_t) unsafe.Pointer {
			return C.scalar_div_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint16_t) unsafe.Pointer {
			return C.scalar_div_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint32_t) unsafe.Pointer {
			return C.scalar_div_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint64_t) unsafe.Pointer {
			return C.scalar_div_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) scalarRem(rhs uint64) (*tfheCiphertext, error) {
	return lhs.executeBinaryScalarOperation(rhs,
		func(lhs unsafe.Pointer, rhs C.uint8_t) unsafe.Pointer {
			return C.scalar_rem_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint16_t) unsafe.Pointer {
			return C.scalar_rem_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint32_t) unsafe.Pointer {
			return C.scalar_rem_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint64_t) unsafe.Pointer {
			return C.scalar_rem_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) bitand(rhs *tfheCiphertext) (*tfheCiphertext, error) {
	return lhs.executeBinaryCiphertextOperation(rhs,
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.bitand_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.bitand_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.bitand_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.bitand_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) bitor(rhs *tfheCiphertext) (*tfheCiphertext, error) {
	return lhs.executeBinaryCiphertextOperation(rhs,
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.bitor_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.bitor_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.bitor_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.bitor_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) bitxor(rhs *tfheCiphertext) (*tfheCiphertext, error) {
	return lhs.executeBinaryCiphertextOperation(rhs,
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.bitxor_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.bitxor_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.bitxor_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.bitxor_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) shl(rhs *tfheCiphertext) (*tfheCiphertext, error) {
	return lhs.executeBinaryCiphertextOperation(rhs,
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.shl_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.shl_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.shl_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.shl_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) scalarShl(rhs uint64) (*tfheCiphertext, error) {
	return lhs.executeBinaryScalarOperation(rhs,
		func(lhs unsafe.Pointer, rhs C.uint8_t) unsafe.Pointer {
			return C.scalar_shl_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint16_t) unsafe.Pointer {
			return C.scalar_shl_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint32_t) unsafe.Pointer {
			return C.scalar_shl_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint64_t) unsafe.Pointer {
			return C.scalar_shl_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) shr(rhs *tfheCiphertext) (*tfheCiphertext, error) {
	return lhs.executeBinaryCiphertextOperation(rhs,
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.shr_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.shr_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.shr_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.shr_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) scalarShr(rhs uint64) (*tfheCiphertext, error) {
	return lhs.executeBinaryScalarOperation(rhs,
		func(lhs unsafe.Pointer, rhs C.uint8_t) unsafe.Pointer {
			return C.scalar_shr_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint16_t) unsafe.Pointer {
			return C.scalar_shr_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint32_t) unsafe.Pointer {
			return C.scalar_shr_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint64_t) unsafe.Pointer {
			return C.scalar_shr_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) eq(rhs *tfheCiphertext) (*tfheCiphertext, error) {
	return lhs.executeBinaryCiphertextOperation(rhs,
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.eq_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.eq_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.eq_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.eq_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) scalarEq(rhs uint64) (*tfheCiphertext, error) {
	return lhs.executeBinaryScalarOperation(rhs,
		func(lhs unsafe.Pointer, rhs C.uint8_t) unsafe.Pointer {
			return C.scalar_eq_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint16_t) unsafe.Pointer {
			return C.scalar_eq_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint32_t) unsafe.Pointer {
			return C.scalar_eq_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint64_t) unsafe.Pointer {
			return C.scalar_eq_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) ne(rhs *tfheCiphertext) (*tfheCiphertext, error) {
	return lhs.executeBinaryCiphertextOperation(rhs,
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.ne_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.ne_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.ne_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.ne_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) scalarNe(rhs uint64) (*tfheCiphertext, error) {
	return lhs.executeBinaryScalarOperation(rhs,
		func(lhs unsafe.Pointer, rhs C.uint8_t) unsafe.Pointer {
			return C.scalar_ne_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint16_t) unsafe.Pointer {
			return C.scalar_ne_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint32_t) unsafe.Pointer {
			return C.scalar_ne_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint64_t) unsafe.Pointer {
			return C.scalar_ne_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) ge(rhs *tfheCiphertext) (*tfheCiphertext, error) {
	return lhs.executeBinaryCiphertextOperation(rhs,
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.ge_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.ge_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.ge_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.ge_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) scalarGe(rhs uint64) (*tfheCiphertext, error) {
	return lhs.executeBinaryScalarOperation(rhs,
		func(lhs unsafe.Pointer, rhs C.uint8_t) unsafe.Pointer {
			return C.scalar_ge_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint16_t) unsafe.Pointer {
			return C.scalar_ge_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint32_t) unsafe.Pointer {
			return C.scalar_ge_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint64_t) unsafe.Pointer {
			return C.scalar_ge_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) gt(rhs *tfheCiphertext) (*tfheCiphertext, error) {
	return lhs.executeBinaryCiphertextOperation(rhs,
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.gt_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.gt_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.gt_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.gt_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) scalarGt(rhs uint64) (*tfheCiphertext, error) {
	return lhs.executeBinaryScalarOperation(rhs,
		func(lhs unsafe.Pointer, rhs C.uint8_t) unsafe.Pointer {
			return C.scalar_gt_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint16_t) unsafe.Pointer {
			return C.scalar_gt_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint32_t) unsafe.Pointer {
			return C.scalar_gt_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint64_t) unsafe.Pointer {
			return C.scalar_gt_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) le(rhs *tfheCiphertext) (*tfheCiphertext, error) {
	return lhs.executeBinaryCiphertextOperation(rhs,
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.le_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.le_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.le_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.le_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) scalarLe(rhs uint64) (*tfheCiphertext, error) {
	return lhs.executeBinaryScalarOperation(rhs,
		func(lhs unsafe.Pointer, rhs C.uint8_t) unsafe.Pointer {
			return C.scalar_le_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint16_t) unsafe.Pointer {
			return C.scalar_le_fhe_uint16(lhs, rhs, sks)

		},
		func(lhs unsafe.Pointer, rhs C.uint32_t) unsafe.Pointer {
			return C.scalar_le_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint64_t) unsafe.Pointer {
			return C.scalar_le_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) lt(rhs *tfheCiphertext) (*tfheCiphertext, error) {
	return lhs.executeBinaryCiphertextOperation(rhs,
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.lt_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.lt_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.lt_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.lt_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) scalarLt(rhs uint64) (*tfheCiphertext, error) {
	return lhs.executeBinaryScalarOperation(rhs,
		func(lhs unsafe.Pointer, rhs C.uint8_t) unsafe.Pointer {
			return C.scalar_lt_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint16_t) unsafe.Pointer {
			return C.scalar_lt_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint32_t) unsafe.Pointer {
			return C.scalar_lt_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint64_t) unsafe.Pointer {
			return C.scalar_lt_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) min(rhs *tfheCiphertext) (*tfheCiphertext, error) {
	return lhs.executeBinaryCiphertextOperation(rhs,
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.min_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.min_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.min_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.min_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) scalarMin(rhs uint64) (*tfheCiphertext, error) {
	return lhs.executeBinaryScalarOperation(rhs,
		func(lhs unsafe.Pointer, rhs C.uint8_t) unsafe.Pointer {
			return C.scalar_min_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint16_t) unsafe.Pointer {
			return C.scalar_min_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint32_t) unsafe.Pointer {
			return C.scalar_min_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint64_t) unsafe.Pointer {
			return C.scalar_min_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) max(rhs *tfheCiphertext) (*tfheCiphertext, error) {
	return lhs.executeBinaryCiphertextOperation(rhs,
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.max_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.max_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.max_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.max_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) scalarMax(rhs uint64) (*tfheCiphertext, error) {
	return lhs.executeBinaryScalarOperation(rhs,
		func(lhs unsafe.Pointer, rhs C.uint8_t) unsafe.Pointer {
			return C.scalar_max_fhe_uint8(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint16_t) unsafe.Pointer {
			return C.scalar_max_fhe_uint16(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint32_t) unsafe.Pointer {
			return C.scalar_max_fhe_uint32(lhs, rhs, sks)
		},
		func(lhs unsafe.Pointer, rhs C.uint64_t) unsafe.Pointer {
			return C.scalar_max_fhe_uint64(lhs, rhs, sks)
		})
}

func (lhs *tfheCiphertext) neg() (*tfheCiphertext, error) {
	return lhs.executeUnaryCiphertextOperation(lhs,
		func(lhs unsafe.Pointer) unsafe.Pointer {
			return C.neg_fhe_uint8(lhs, sks)
		},
		func(lhs unsafe.Pointer) unsafe.Pointer {
			return C.neg_fhe_uint16(lhs, sks)
		},
		func(lhs unsafe.Pointer) unsafe.Pointer {
			return C.neg_fhe_uint32(lhs, sks)
		},
		func(lhs unsafe.Pointer) unsafe.Pointer {
			return C.neg_fhe_uint64(lhs, sks)
		})
}

func (lhs *tfheCiphertext) not() (*tfheCiphertext, error) {
	return lhs.executeUnaryCiphertextOperation(lhs,
		func(lhs unsafe.Pointer) unsafe.Pointer {
			return C.not_fhe_uint8(lhs, sks)
		},
		func(lhs unsafe.Pointer) unsafe.Pointer {
			return C.not_fhe_uint16(lhs, sks)
		},
		func(lhs unsafe.Pointer) unsafe.Pointer {
			return C.not_fhe_uint32(lhs, sks)
		},
		func(lhs unsafe.Pointer) unsafe.Pointer {
			return C.not_fhe_uint64(lhs, sks)
		})
}

func (condition *tfheCiphertext) ifThenElse(lhs *tfheCiphertext, rhs *tfheCiphertext) (*tfheCiphertext, error) {
	return condition.executeTernaryCiphertextOperation(lhs, rhs,
		func(condition unsafe.Pointer, lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.if_then_else_fhe_uint8(condition, lhs, rhs, sks)
		},
		func(condition unsafe.Pointer, lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.if_then_else_fhe_uint16(condition, lhs, rhs, sks)
		},
		func(condition unsafe.Pointer, lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.if_then_else_fhe_uint32(condition, lhs, rhs, sks)
		},
		func(condition unsafe.Pointer, lhs unsafe.Pointer, rhs unsafe.Pointer) unsafe.Pointer {
			return C.if_then_else_fhe_uint64(condition, lhs, rhs, sks)
		})
}

func (ct *tfheCiphertext) castTo(castToType FheUintType) (*tfheCiphertext, error) {
	if ct.fheUintType == castToType {
		return nil, errors.New("casting to same type is not supported")
	}

	res := new(tfheCiphertext)
	res.fheUintType = castToType

	switch ct.fheUintType {
	case FheUint8:
		switch castToType {
		case FheUint16:
			from_ptr := C.deserialize_fhe_uint8(toDynamicBufferView(ct.serialization))
			if from_ptr == nil {
				return nil, errors.New("castTo failed to deserialize FheUint8 ciphertext")
			}
			to_ptr := C.cast_8_16(from_ptr, sks)
			C.destroy_fhe_uint8(from_ptr)
			if to_ptr == nil {
				return nil, errors.New("castTo failed to cast FheUint8 to FheUint16")
			}
			var err error
			res.serialization, err = serialize(to_ptr, castToType)
			C.destroy_fhe_uint16(to_ptr)
			if err != nil {
				return nil, err
			}
		case FheUint32:
			from_ptr := C.deserialize_fhe_uint8(toDynamicBufferView(ct.serialization))
			if from_ptr == nil {
				return nil, errors.New("castTo failed to deserialize FheUint8 ciphertext")
			}
			to_ptr := C.cast_8_32(from_ptr, sks)
			C.destroy_fhe_uint8(from_ptr)
			if to_ptr == nil {
				return nil, errors.New("castTo failed to cast FheUint8 to FheUint32")
			}
			var err error
			res.serialization, err = serialize(to_ptr, castToType)
			C.destroy_fhe_uint32(to_ptr)
			if err != nil {
				return nil, err
			}
		case FheUint64:
			from_ptr := C.deserialize_fhe_uint8(toDynamicBufferView(ct.serialization))
			if from_ptr == nil {
				return nil, errors.New("castTo failed to deserialize FheUint8 ciphertext")
			}
			to_ptr := C.cast_8_64(from_ptr, sks)
			C.destroy_fhe_uint8(from_ptr)
			if to_ptr == nil {
				return nil, errors.New("castTo failed to cast FheUint8 to FheUint64")
			}
			var err error
			res.serialization, err = serialize(to_ptr, castToType)
			C.destroy_fhe_uint64(to_ptr)
			if err != nil {
				return nil, err
			}
		default:
			panic("castTo: unexpected type to cast to")
		}
	case FheUint16:
		switch castToType {
		case FheUint8:
			from_ptr := C.deserialize_fhe_uint16(toDynamicBufferView(ct.serialization))
			if from_ptr == nil {
				return nil, errors.New("castTo failed to deserialize FheUint16 ciphertext")
			}
			to_ptr := C.cast_16_8(from_ptr, sks)
			C.destroy_fhe_uint16(from_ptr)
			if to_ptr == nil {
				return nil, errors.New("castTo failed to cast FheUint16 to FheUint8")
			}
			var err error
			res.serialization, err = serialize(to_ptr, castToType)
			C.destroy_fhe_uint8(to_ptr)
			if err != nil {
				return nil, err
			}
		case FheUint32:
			from_ptr := C.deserialize_fhe_uint16(toDynamicBufferView(ct.serialization))
			if from_ptr == nil {
				return nil, errors.New("castTo failed to deserialize FheUint16 ciphertext")
			}
			to_ptr := C.cast_16_32(from_ptr, sks)
			C.destroy_fhe_uint16(from_ptr)
			if to_ptr == nil {
				return nil, errors.New("castTo failed to cast FheUint16 to FheUint32")
			}
			var err error
			res.serialization, err = serialize(to_ptr, castToType)
			C.destroy_fhe_uint32(to_ptr)
			if err != nil {
				return nil, err
			}
		case FheUint64:
			from_ptr := C.deserialize_fhe_uint16(toDynamicBufferView(ct.serialization))
			if from_ptr == nil {
				return nil, errors.New("castTo failed to deserialize FheUint16 ciphertext")
			}
			to_ptr := C.cast_16_64(from_ptr, sks)
			C.destroy_fhe_uint16(from_ptr)
			if to_ptr == nil {
				return nil, errors.New("castTo failed to cast FheUint16 to FheUint64")
			}
			var err error
			res.serialization, err = serialize(to_ptr, castToType)
			C.destroy_fhe_uint64(to_ptr)
			if err != nil {
				return nil, err
			}
		default:
			panic("castTo: unexpected type to cast to")
		}
	case FheUint32:
		switch castToType {
		case FheUint8:
			from_ptr := C.deserialize_fhe_uint32(toDynamicBufferView(ct.serialization))
			if from_ptr == nil {
				return nil, errors.New("castTo failed to deserialize FheUint32 ciphertext")
			}
			to_ptr := C.cast_32_8(from_ptr, sks)
			C.destroy_fhe_uint32(from_ptr)
			if to_ptr == nil {
				return nil, errors.New("castTo failed to cast FheUint32 to FheUint8")
			}
			var err error
			res.serialization, err = serialize(to_ptr, castToType)
			C.destroy_fhe_uint8(to_ptr)
			if err != nil {
				return nil, err
			}
		case FheUint16:
			from_ptr := C.deserialize_fhe_uint32(toDynamicBufferView(ct.serialization))
			if from_ptr == nil {
				return nil, errors.New("castTo failed to deserialize FheUint32 ciphertext")
			}
			to_ptr := C.cast_32_16(from_ptr, sks)
			C.destroy_fhe_uint32(from_ptr)
			if to_ptr == nil {
				return nil, errors.New("castTo failed to cast FheUint32 to FheUint16")
			}
			var err error
			res.serialization, err = serialize(to_ptr, castToType)
			C.destroy_fhe_uint16(to_ptr)
			if err != nil {
				return nil, err
			}
		case FheUint64:
			from_ptr := C.deserialize_fhe_uint32(toDynamicBufferView(ct.serialization))
			if from_ptr == nil {
				return nil, errors.New("castTo failed to deserialize FheUint32 ciphertext")
			}
			to_ptr := C.cast_32_64(from_ptr, sks)
			C.destroy_fhe_uint32(from_ptr)
			if to_ptr == nil {
				return nil, errors.New("castTo failed to cast FheUint32 to FheUint64")
			}
			var err error
			res.serialization, err = serialize(to_ptr, castToType)
			C.destroy_fhe_uint64(to_ptr)
			if err != nil {
				return nil, err
			}
		default:
			panic("castTo: unexpected type to cast to")
		}
	case FheUint64:
		switch castToType {
		case FheUint8:
			from_ptr := C.deserialize_fhe_uint64(toDynamicBufferView(ct.serialization))
			if from_ptr == nil {
				return nil, errors.New("castTo failed to deserialize FheUint64 ciphertext")
			}
			to_ptr := C.cast_64_8(from_ptr, sks)
			C.destroy_fhe_uint64(from_ptr)
			if to_ptr == nil {
				return nil, errors.New("castTo failed to cast FheUint64 to FheUint8")
			}
			var err error
			res.serialization, err = serialize(to_ptr, castToType)
			C.destroy_fhe_uint8(to_ptr)
			if err != nil {
				return nil, err
			}
		case FheUint16:
			from_ptr := C.deserialize_fhe_uint64(toDynamicBufferView(ct.serialization))
			if from_ptr == nil {
				return nil, errors.New("castTo failed to deserialize FheUint64 ciphertext")
			}
			to_ptr := C.cast_64_16(from_ptr, sks)
			C.destroy_fhe_uint64(from_ptr)
			if to_ptr == nil {
				return nil, errors.New("castTo failed to cast FheUint64 to FheUint16")
			}
			var err error
			res.serialization, err = serialize(to_ptr, castToType)
			C.destroy_fhe_uint16(to_ptr)
			if err != nil {
				return nil, err
			}
		case FheUint32:
			from_ptr := C.deserialize_fhe_uint64(toDynamicBufferView(ct.serialization))
			if from_ptr == nil {
				return nil, errors.New("castTo failed to deserialize FheUint64 ciphertext")
			}
			to_ptr := C.cast_64_32(from_ptr, sks)
			C.destroy_fhe_uint64(from_ptr)
			if to_ptr == nil {
				return nil, errors.New("castTo failed to cast FheUint64 to FheUint32")
			}
			var err error
			res.serialization, err = serialize(to_ptr, castToType)
			C.destroy_fhe_uint32(to_ptr)
			if err != nil {
				return nil, err
			}
		default:
			panic("castTo: unexpected type to cast to")
		}
	}
	res.computeHash()
	return res, nil
}

func (ct *tfheCiphertext) decrypt() (big.Int, error) {
	if cks == nil {
		return *new(big.Int).SetUint64(0), errors.New("cks is not initialized")
	}
	var value uint64
	var ret C.int
	switch ct.fheUintType {
	case FheUint8:
		ptr := C.deserialize_fhe_uint8(toDynamicBufferView(ct.serialization))
		if ptr == nil {
			return *new(big.Int).SetUint64(0), errors.New("failed to deserialize FheUint8")
		}
		var result C.uint8_t
		ret = C.decrypt_fhe_uint8(cks, ptr, &result)
		C.destroy_fhe_uint8(ptr)
		value = uint64(result)
	case FheUint16:
		ptr := C.deserialize_fhe_uint16(toDynamicBufferView(ct.serialization))
		if ptr == nil {
			return *new(big.Int).SetUint64(0), errors.New("failed to deserialize FheUint16")
		}
		var result C.uint16_t
		ret = C.decrypt_fhe_uint16(cks, ptr, &result)
		C.destroy_fhe_uint16(ptr)
		value = uint64(result)
	case FheUint32:
		ptr := C.deserialize_fhe_uint32(toDynamicBufferView(ct.serialization))
		if ptr == nil {
			return *new(big.Int).SetUint64(0), errors.New("failed to deserialize FheUint32")
		}
		var result C.uint32_t
		ret = C.decrypt_fhe_uint32(cks, ptr, &result)
		C.destroy_fhe_uint32(ptr)
		value = uint64(result)
	case FheUint64:
		ptr := C.deserialize_fhe_uint64(toDynamicBufferView(ct.serialization))
		if ptr == nil {
			return *new(big.Int).SetUint64(0), errors.New("failed to deserialize FheUint64")
		}
		var result C.uint64_t
		ret = C.decrypt_fhe_uint64(cks, ptr, &result)
		C.destroy_fhe_uint64(ptr)
		value = uint64(result)
	default:
		panic("decrypt: unexpected ciphertext type")
	}
	if ret != 0 {
		return *new(big.Int).SetUint64(0), errors.New("decrypt failed")
	}
	return *new(big.Int).SetUint64(value), nil
}

func (ct *tfheCiphertext) computeHash() {
	hash := common.BytesToHash(crypto.Keccak256(ct.serialization))
	ct.hash = &hash
}

func (ct *tfheCiphertext) getHash() common.Hash {
	if ct.hash != nil {
		return *ct.hash
	}
	ct.computeHash()
	return *ct.hash
}

func isValidType(t byte) bool {
	if uint8(t) < uint8(FheUint8) || uint8(t) > uint8(FheUint64) {
		return false
	}
	return true
}

func encryptAndSerializeCompact(value uint64, fheUintType FheUintType) []byte {
	out := &C.DynamicBuffer{}
	switch fheUintType {
	case FheUint8:
		C.public_key_encrypt_and_serialize_fhe_uint8_list(pks, C.uint8_t(value), out)
	case FheUint16:
		C.public_key_encrypt_and_serialize_fhe_uint16_list(pks, C.uint16_t(value), out)
	case FheUint32:
		C.public_key_encrypt_and_serialize_fhe_uint32_list(pks, C.uint32_t(value), out)
	case FheUint64:
		C.public_key_encrypt_and_serialize_fhe_uint64_list(pks, C.uint64_t(value), out)
	}

	ser := C.GoBytes(unsafe.Pointer(out.pointer), C.int(out.length))
	C.destroy_dynamic_buffer(out)
	return ser
}
