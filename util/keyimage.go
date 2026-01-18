package util

import (
	"encoding/binary"
	"fmt"

	"filippo.io/edwards25519"
)

func CreateKeyImage(pubSpendKey, secSpendKey, secViewKey, txPubKey *Key, outIndex uint64) (*Key, *Key, error) {
	derivation, ok := GenerateKeyDerivation(txPubKey, secViewKey)
	if !ok {
		return nil, nil, fmt.Errorf("generate key derivation failed")
	}

	derivedPubKey, ok := DerivePublicKey(&derivation, outIndex, pubSpendKey)
	if !ok {
		return nil, nil, fmt.Errorf("derive public key failed")
	}

	derivedPriKey := DeriveSecretKey(&derivation, outIndex, secSpendKey)
	if *derivedPriKey.PubKey() != derivedPubKey {
		return nil, nil, fmt.Errorf("derived secret key doesn't match derived public key")
	}

	keyImage := GenerateKeyImage(&derivedPriKey)
	return &keyImage, &derivedPriKey, nil
}

func GenerateKeyDerivation(pubKey, secKey *Key) (keyDerivation Key, ok bool) {
	point := new(ExtendedGroupElement)
	ok = point.FromBytes(pubKey)
	if !ok {
		return
	}

	point2 := new(ProjectiveGroupElement)
	GeScalarMult(point2, secKey, point)

	point3 := new(CompletedGroupElement)
	GeMul8(point3, point2)
	point3.ToProjective(point2)
	point2.ToBytes(&keyDerivation)
	ok = true
	return
}

func Uint64ToBytes(num uint64) (result []byte) {
	for ; num >= 0x80; num >>= 7 {
		result = append(result, byte((num&0x7f)|0x80))
	}
	result = append(result, byte(num))
	return
}

func derivationToScalar(derivation *Key, outIndex uint64) (scalar Key) {
	data := append((*derivation)[:], Uint64ToBytes(outIndex)...)
	scalar = Key(Keccak256(data))
	ScReduce32(&scalar)
	return
}

func DerivePublicKey(derivation *Key, outIndex uint64, base *Key) (derivedKey Key, ok bool) {
	point1 := new(ExtendedGroupElement)
	ok = point1.FromBytes(base)
	if !ok {
		return
	}

	scalar := derivationToScalar(derivation, outIndex)
	point2 := new(ExtendedGroupElement)
	GeScalarMultBase(point2, &scalar)

	point3 := new(CachedGroupElement)
	point2.ToCached(point3)

	point4 := new(CompletedGroupElement)
	geAdd(point4, point1, point3)

	point5 := new(ProjectiveGroupElement)
	point4.ToProjective(point5)
	point5.ToBytes(&derivedKey)
	ok = true
	return
}

func DeriveViewTag(derivation *Key, index uint64) (byte, error) {
	sharedSecret := derivation.ToBytes2()

	// Compute view tag per Monero: H[salt||derivation||varint(index)], salt is 8 bytes "view_tag"
	data := make([]byte, 0, 8+len(sharedSecret)+10)
	data = append(data, []byte("view_tag")...) // 8 bytes without null terminator
	data = append(data, sharedSecret...)
	data = append(data, EncodeVarint(index)...)
	viewTagHash := Keccak256(data)
	viewTag := viewTagHash[0]

	return viewTag, nil
}

func DeriveSecretKey(derivation *Key, outIndex uint64, base *Key) (derivedKey Key) {
	scalar := derivationToScalar(derivation, outIndex)
	ScAdd(&derivedKey, base, &scalar)
	return
}

func EncryptRctAmount(derivation *Key, outputIndex uint64, amount float64) ([8]byte, error) {
	// Конвертируем amount в uint64 (предполагаем, что amount уже в atomic units)
	amountAtomic := XmrToAtomic(amount, 1e12)

	// Получаем shared secret (shared = 8 * txSecretKey * pubViewKey)
	shared := derivation.ToBytes2()

	// Вычисляем Hs = sc_reduce32(util.Keccak256(shared || varint(index)))
	hashInput := append(shared, EncodeVarint(outputIndex)...)
	hsHash := Keccak256(hashInput)

	// Для получения скаляра используем SetUniformBytes (ожидает 64 байта)
	hsHash64 := make([]byte, 64)
	copy(hsHash64, hsHash)
	hsScalar := new(edwards25519.Scalar)
	if _, err := hsScalar.SetUniformBytes(hsHash64); err != nil {
		return [8]byte{}, err
	}
	hsBytes := hsScalar.Bytes()

	// amountMask = util.Keccak256("amount" || hsBytes)
	amountMask := Keccak256(append([]byte("amount"), hsBytes...))

	// Конвертируем amount в байты (little-endian)
	amountBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(amountBytes, amountAtomic)

	// XOR первых 8 байт маски
	var encrypted [8]byte
	for i := 0; i < 8; i++ {
		encrypted[i] = amountBytes[i] ^ amountMask[i]
	}

	return encrypted, nil
}

func GenerateKeyImage(privKey *Key) (keyImage Key) {
	point := privKey.PubKey().HashToEC()
	keyImagePoint := new(ProjectiveGroupElement)
	GeScalarMult(keyImagePoint, privKey, point)
	// convert key Image point from Projective to Extended
	// in order to precompute
	keyImagePoint.ToBytes(&keyImage)
	return
}
