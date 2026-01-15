package levin

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

const (
	BoostSerializeTypeInt64 byte = 0x1
	BoostSerializeTypeInt32 byte = 0x2
	BoostSerializeTypeInt16 byte = 0x3
	BoostSerializeTypeInt8  byte = 0x4

	BoostSerializeTypeUint64 byte = 0x5
	BoostSerializeTypeUint32 byte = 0x6
	BoostSerializeTypeUint16 byte = 0x7
	BoostSerializeTypeUint8  byte = 0x8

	BoostSerializeTypeDouble byte = 0x9

	BoostSerializeTypeString byte = 0x0a
	BoostSerializeTypeBool   byte = 0x0b
	BoostSerializeTypeObject byte = 0x0c
	BoostSerializeTypeArray  byte = 0xd

	BoostSerializeFlagArray byte = 0x80
)

type BoostByte byte

func (v BoostByte) Bytes() []byte {
	return []byte{
		BoostSerializeTypeUint8,
		byte(v),
	}
}

type BoostUint8 uint8

func (v BoostUint8) Bytes() []byte {
	PutUint8 := func(b []byte, v uint8) {
		_ = b[0]
		b[0] = v
	}

	b := []byte{
		BoostSerializeTypeUint8, 0x00,
	}
	PutUint8(b[1:], uint8(v))
	return b
}

type BoostUint16 uint16

func (v BoostUint16) Bytes() []byte {
	b := []byte{
		BoostSerializeTypeUint16,
		0x00, 0x00,
	}
	binary.LittleEndian.PutUint16(b[1:], uint16(v))
	return b
}

type BoostUint32 uint32

func (v BoostUint32) Bytes() []byte {
	b := []byte{
		BoostSerializeTypeUint32,
		0x00, 0x00, 0x00, 0x00,
	}
	binary.LittleEndian.PutUint32(b[1:], uint32(v))
	return b
}

type BoostUint64 uint64

func (v BoostUint64) Bytes() []byte {
	b := []byte{
		BoostSerializeTypeUint64,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}

	binary.LittleEndian.PutUint64(b[1:], uint64(v))

	return b
}

type BoostString string

func (v BoostString) Bytes() []byte {
	b := []byte{BoostSerializeTypeString}

	varInB, err := VarIn(len(v))
	if err != nil {
		panic(fmt.Errorf("varin '%d': %w", len(v), err))
	}

	return append(b, append(varInB, []byte(v)...)...)
}

type BoostHash string

func (v BoostHash) Bytes() []byte {
	const HASH_SIZE = 32

	hashBytes, err := hex.DecodeString(string(v))
	if err != nil {
		return nil
	}

	if len(hashBytes) != HASH_SIZE {
		return nil
	}

	result := make([]byte, 0, HASH_SIZE)
	result = append(result, hashBytes...)

	prefix := []byte{BoostSerializeTypeString, BoostSerializeFlagArray}
	result = append(prefix, result...)

	return result
}

type BoostBlockIds []string

func (blockIds BoostBlockIds) Bytes() []byte {
	if len(blockIds) == 0 {
		return []byte{}
	}

	temp := make([]byte, 0, len(blockIds)*HASH_SIZE)
	for _, blockId := range blockIds {
		hashBytes, err := hex.DecodeString(blockId)
		if err != nil || len(hashBytes) != HASH_SIZE {
			return nil
		}

		temp = append(temp, hashBytes...)
	}

	payloadSize := len(MainnetGenesisTxByte) + len(temp)
	varInB, err := VarIn(payloadSize)
	if err != nil || len(varInB) != 2 {
		return nil
	}

	prefix := []byte{
		BoostSerializeTypeString,
		varInB[0],
		varInB[1],
	}

	result := make([]byte, 0, payloadSize+3)
	result = append(result, prefix...)
	result = append(result, temp...)
	result = append(result, MainnetGenesisTxByte...)

	return result
}

type BoostTxIDs []uint64

func (txIds BoostTxIDs) Bytes() []byte {
	if len(txIds) == 0 {
		return []byte{}
	}

	// Каждый uint64 = 8 байтов (БЕЗ типа)
	temp := make([]byte, 0, len(txIds)*8)
	for _, txId := range txIds {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, txId)
		temp = append(temp, b...)
	}

	payloadSize := len(temp)
	varInB, err := VarIn(payloadSize)
	if err != nil {
		return nil
	}

	var buf bytes.Buffer
	buf.WriteByte(BoostSerializeTypeString)
	buf.Write(varInB)
	buf.Write(temp)

	return buf.Bytes()
}

// Добавьте этот код в конец файла levin/boost.go

type BoostUint64Array []uint64

func (uints BoostUint64Array) Bytes() []byte {
	if len(uints) == 0 {
		return []byte{
			BoostSerializeTypeUint64 | BoostSerializeFlagArray,
			0x00, // varint для длины 0
		}
	}

	var buf bytes.Buffer

	// Тип: uint64 array
	buf.WriteByte(BoostSerializeTypeUint64 | BoostSerializeFlagArray)

	// Длина массива в varint
	varInB, err := VarIn(len(uints))
	if err != nil {
		panic(fmt.Errorf("varin for array length: %w", err))
	}
	buf.Write(varInB)

	// Каждый uint64 как varint (не как фиксированные 8 байт!)
	for _, val := range uints {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, val)
		buf.Write(b)
	}

	return buf.Bytes()
}

type BoostBlock []string

func (blockIds BoostBlock) Bytes() []byte {
	if len(blockIds) == 0 {
		return []byte{}
	}

	temp := make([]byte, 0, len(blockIds)*HASH_SIZE)
	for _, blockId := range blockIds {
		hashBytes, err := hex.DecodeString(blockId)
		if err != nil || len(hashBytes) != HASH_SIZE {
			return nil
		}

		temp = append(temp, hashBytes...)
	}

	payloadSize := len(temp)
	varInB, err := VarIn(payloadSize)
	if err != nil {
		return nil
	}

	var buf bytes.Buffer
	buf.WriteByte(BoostSerializeTypeString)
	buf.Write(varInB)
	buf.Write(temp)

	return buf.Bytes()
}
