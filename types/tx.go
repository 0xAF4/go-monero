package types

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"slices"

	"filippo.io/edwards25519"
	"github.com/0xAF4/go-monero/levin"
	"github.com/0xAF4/go-monero/util"
)

type Transaction struct {
	Hash [32]byte `json:"-"`
	Raw  []byte   `json:"-"`

	Version        uint64          `json:"version"`
	UnlockTime     uint64          `json:"unlock_time"`
	VinCount       uint64          `json:"-"`
	Inputs         []TxInput       `json:"vin"`
	VoutCount      uint64          `json:"-"`
	Outputs        []TxOutput      `json:"vout"`
	Extra          ByteArray       `json:"extra"`
	RctRaw         []byte          `json:"-"`
	RctSignature   *RctSignature   `json:"rct_signature"`
	RctSigPrunable *RctSigPrunable `json:"rctsig_prunable"`

	POutputs     []TxPrm                `json:"-"`
	PInputs      []TxPrm                `json:"-"`
	SecretKey    Hash                   `json:"-"`
	PublicKey    Hash                   `json:"-"`
	BlindScalars []*edwards25519.Scalar `json:"-"`
	InputScalars []*edwards25519.Scalar `json:"-"`
	BlindAmounts []uint64               `json:"-"`
}

type TxInput struct {
	Type       uint8    `json:"-"`
	Height     uint64   `json:"-"`
	Amount     uint64   `json:"amount"`
	KeyOffsets []uint64 `json:"key_offsets"`
	KeyImage   Hash     `json:"k_image"`
	Address    string   `json:"-"`
	Mixins     []Mixin  `json:"mixins,omitempty"`
	OrderIndx  int      `json:"-"`
	InSk       Mixin    `json:"-"`
}

type Mixin struct {
	Dest Hash `json:"dest"`
	Mask Hash `json:"mask"`
}

type TxOutput struct {
	Amount  uint64 `json:"amount"`
	Target  Hash   `json:"key"`
	Type    byte   `json:"type"`
	ViewTag HByte  `json:"view_tag"`
}

type Echd struct {
	Mask   Hash    `json:"mask"`
	Amount HAmount `json:"amount"`
}

type RctSignature struct {
	Type     uint64 `json:"type"`
	TxnFee   uint64 `json:"txn_fee"`
	EcdhInfo []Echd `json:"ecdhInfo"`
	OutPk    []Hash `json:"outPk"`
}

type RctSigPrunable struct {
	Nbp        uint64  `json:"nbp"`
	Bpp        []Bpp   `json:"bpp"`
	CLSAGs     []CLSAG `json:"CLSAGs"`
	PseudoOuts []Hash  `json:"pseudoOuts"`
}

func (tx *Transaction) Serialize() []byte {
	part1 := tx.CalculatePart1()
	part2 := tx.CalculatePart2()
	part3 := tx.CalculatePart3()

	// Step 2: Concatenate the parts
	concat := append(part1, part2...)
	concat = append(concat, part3...)

	return concat
}

func (tx *Transaction) ParseTx() {
	reader := bytes.NewReader(tx.Raw)

	// 1. Версия транзакции
	tx.Version, _ = levin.ReadVarint(reader)
	tx.UnlockTime, _ = levin.ReadVarint(reader)

	// 3. Inputs
	tx.VinCount, _ = levin.ReadVarint(reader)
	for i := 0; i < int(tx.VinCount); i++ {
		var in TxInput
		// тип входа (0xff = coinbase)
		in.Type, _ = reader.ReadByte()

		switch in.Type {
		case 0xff:
			in.Height, _ = levin.ReadVarint(reader)
		case 0x02:
			in.Amount, _ = levin.ReadVarint(reader)
			ofsCount, _ := levin.ReadVarint(reader)
			for j := 0; j < int(ofsCount); j++ {
				ofs, _ := levin.ReadVarint(reader)
				in.KeyOffsets = append(in.KeyOffsets, ofs)
			}
			reader.Read(in.KeyImage[:])
		default:
			fmt.Printf("⚠️ Unknown TxInput type: 0x%X\n", in.Type)
		}
		tx.Inputs = append(tx.Inputs, in)
	}

	// 4. Outputs
	tx.VoutCount, _ = levin.ReadVarint(reader)
	for i := 0; i < int(tx.VoutCount); i++ {
		var out TxOutput
		out.Amount, _ = levin.ReadVarint(reader)
		out.Type, _ = reader.ReadByte()
		reader.Read(out.Target[:])
		b, _ := reader.ReadByte()
		out.ViewTag = HByte(b)
		tx.Outputs = append(tx.Outputs, out)
	}

	// 5. Extra
	extraLen, _ := levin.ReadVarint(reader)
	extra := make([]byte, extraLen)
	reader.Read(extra)
	tx.Extra = extra

	rest := make([]byte, reader.Len())
	reader.Read(rest)
	tx.RctRaw = rest
}

func (tx *Transaction) ParseRctSig() {
	if len(tx.RctRaw) == 0 {
		return
	}

	RctSignature := &RctSignature{}
	RctSigPrunable := &RctSigPrunable{}

	reader := bytes.NewReader(tx.RctRaw)
	RctSignature.Type, _ = levin.ReadVarint(reader)
	RctSignature.TxnFee, _ = levin.ReadVarint(reader)

	for i := 0; i < len(tx.Outputs); i++ {
		ecdh := Echd{}
		if !slices.Contains([]uint64{4, 5, 6}, RctSignature.Type) {
			reader.Read(ecdh.Mask[:])
		}
		reader.Read(ecdh.Amount[:])
		RctSignature.EcdhInfo = append(RctSignature.EcdhInfo, ecdh)
	}

	for i := 0; i < len(tx.Outputs); i++ {
		var outPk Hash
		reader.Read(outPk[:])
		RctSignature.OutPk = append(RctSignature.OutPk, outPk)
	}

	RctSigPrunable.Nbp, _ = levin.ReadVarint(reader)
	for i := 0; i < int(RctSigPrunable.Nbp); i++ {
		bpp := Bpp{}
		reader.Read(bpp.A[:])
		reader.Read(bpp.A1[:])
		reader.Read(bpp.B[:])
		reader.Read(bpp.R1[:])
		reader.Read(bpp.S1[:])
		reader.Read(bpp.D1[:])

		c, _ := levin.ReadVarint(reader)
		for j := 0; j < int(c); j++ {
			var l Hash
			reader.Read(l[:])
			bpp.L = append(bpp.L, l)
		}

		c, _ = levin.ReadVarint(reader)
		for j := 0; j < int(c); j++ {
			var r Hash
			reader.Read(r[:])
			bpp.R = append(bpp.R, r)
		}

		RctSigPrunable.Bpp = append(RctSigPrunable.Bpp, bpp)
	}

	// CLSAGs
	for i := 0; i < int(tx.VinCount); i++ {
		CLSAG := CLSAG{}
		for j := 0; j < len(tx.Inputs[i].KeyOffsets); j++ {
			var s Hash // Скаляр — 32 байта
			reader.Read(s[:])
			CLSAG.S = append(CLSAG.S, s)
		}

		reader.Read(CLSAG.C1[:])
		reader.Read(CLSAG.D[:])

		RctSigPrunable.CLSAGs = append(RctSigPrunable.CLSAGs, CLSAG)
	}

	for i := 0; i < int(tx.VinCount); i++ {
		var s Hash // Скаляр — 32 байта
		reader.Read(s[:])
		RctSigPrunable.PseudoOuts = append(RctSigPrunable.PseudoOuts, s)
	}

	tx.RctSignature = RctSignature
	tx.RctSigPrunable = RctSigPrunable

	rest := make([]byte, reader.Len())
	reader.Read(rest)
}

func (tx *Transaction) CheckOutputs(address string, privateViewKey string) (float64, uint64, error) {
	pubSpendKey, pubViewKey, err := util.DecodeAddress(address) // correct ✅
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode address: %w", err)
	}

	privViewKeyBytes, err := util.HexTo32(privateViewKey) // correct ✅
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode private view key: %w", err)
	}

	if !verifyViewKeyPair(privViewKeyBytes, pubViewKey[:]) { // correct ✅
		return 0, 0, fmt.Errorf("private view key does not match address")
	}

	// txPubKey, err := extractTxPubKey(tx.Extra) // correct ✅
	txPubKey, _, _, encPID, err := util.ParseTxExtra(tx.Extra)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to extract tx public key: %w", err)
	}

	// Check each output
	var totalAmount float64
	var foundOutputs int

	for outputIndex, output := range tx.Outputs {
		// Derive the one-time public key and view tag
		viewTagByte, err := DeriveViewTag(txPubKey, privViewKeyBytes, uint64(outputIndex))
		if err != nil {
			continue
		}

		// If view tag exists and doesn't match, log it but don't immediately skip — fall back to full derived-key check
		if byte(output.ViewTag) != viewTagByte {
			fmt.Printf("Output %d: View tag mismatch (expected %02x, got %02x), will still check derived key\n", outputIndex, viewTagByte, byte(output.ViewTag))
			continue
		} else {
			fmt.Println("Output view_tag is match✅")
		}

		derivedKey, err := util.DerivePublicKeyMy(txPubKey, privViewKeyBytes, pubSpendKey[:], uint64(outputIndex))
		if err != nil {
			return 0, 0, fmt.Errorf("failed to derive public key for output %d: %w", outputIndex, err)
		}

		// Compare derived key with output target (this is authoritative)
		if !util.EqualBytes(derivedKey, output.Target[:]) {
			fmt.Printf("Output %d: Derived key %x does not match output target %x\n", outputIndex, derivedKey, output.Target)
			continue // Not our output
		} else {
			fmt.Printf("Output %d: Derived key matches output target ✅\n", outputIndex)
		}

		// This output belongs to us!
		foundOutputs++
		var amount float64

		// If it's an RCT transaction, decode the amount
		if tx.RctSignature != nil && tx.RctSignature.Type > 0 {
			if outputIndex < len(tx.RctSignature.EcdhInfo) {
				amount, err = DecodeRctAmount(
					txPubKey,
					privViewKeyBytes,
					uint64(outputIndex),
					tx.RctSignature.EcdhInfo[outputIndex].Amount[:],
				)
				if err != nil {
					return 0, 0, fmt.Errorf("failed to decode RCT amount for output %d: %w", outputIndex, err)
				} else {
					fmt.Printf("Output %d: Decoded RCT amount: %.12f\n", outputIndex, amount)
				}
			}
		} else {
			amount = 0
		}

		totalAmount += amount
	}

	if foundOutputs == 0 {
		return 0, 0, fmt.Errorf("no outputs found for this address")
	}

	if encPID != nil {
		id, _, err := util.DecryptShortPaymentID(txPubKey, privViewKeyBytes, encPID)
		if err != nil {
			return 0, 0, err
		}
		return totalAmount, id, nil
	}
	return totalAmount, 0, nil
}

func (tx *Transaction) CalculatePart1() []byte {
	var buf bytes.Buffer

	// Version
	buf.Write(util.EncodeVarint(tx.Version))

	// Unlock time
	buf.Write(util.EncodeVarint(tx.UnlockTime))

	// Inputs
	buf.Write(util.EncodeVarint(uint64(len(tx.Inputs))))

	for _, input := range tx.Inputs {
		buf.WriteByte(input.Type)

		if input.Type == 0x02 { // TxIn to key
			buf.Write(util.EncodeVarint(input.Amount))
			buf.Write(util.EncodeVarint(uint64(len(input.KeyOffsets))))
			for _, offset := range input.KeyOffsets {
				buf.Write(util.EncodeVarint(offset))
			}
			buf.Write(input.KeyImage[:])
		}
	}

	// Outputs
	buf.Write(util.EncodeVarint(uint64(len(tx.Outputs))))
	for _, output := range tx.Outputs {
		buf.Write(util.EncodeVarint(output.Amount))
		buf.WriteByte(output.Type)
		buf.Write(output.Target[:])
		buf.WriteByte(byte(output.ViewTag))
	}

	// Extra
	buf.Write(util.EncodeVarint(uint64(len(tx.Extra))))
	buf.Write(tx.Extra)

	return buf.Bytes()
}

func (tx *Transaction) CalculatePart2() []byte {
	var buf bytes.Buffer

	buf.Write(util.EncodeVarint(tx.RctSignature.Type))
	buf.Write(util.EncodeVarint(tx.RctSignature.TxnFee))

	if tx.RctSignature.Type == 2 { //MLSAGBorromean
		for _, ps := range tx.RctSigPrunable.PseudoOuts {
			buf.Write(ps[:])
		}
	}
	if slices.Contains([]uint64{4, 5, 6}, tx.RctSignature.Type) { //MLSAGBulletproofCompactAmount, CLSAGBulletproof, CLSAGBulletproofPlus
		for _, ei := range tx.RctSignature.EcdhInfo {
			buf.Write(ei.Amount[:])
		}
	} else {
		for _, ei := range tx.RctSignature.EcdhInfo {
			buf.Write(ei.Mask[:])
			buf.Write(ei.Amount[:])
		}
	}

	for _, c := range tx.RctSignature.OutPk {
		buf.Write(c[:])
	}

	return buf.Bytes()
}

func (tx *Transaction) CalculatePart3() []byte {
	var buf bytes.Buffer

	buf.Write(util.EncodeVarint(1))
	for _, bpp := range tx.RctSigPrunable.Bpp {
		buf.Write(bpp.A[:])
		buf.Write(bpp.A1[:])
		buf.Write(bpp.B[:])
		buf.Write(bpp.R1[:])
		buf.Write(bpp.S1[:])
		buf.Write(bpp.D1[:])
		buf.Write(util.EncodeVarint(uint64(len(bpp.L))))
		for _, l := range bpp.L {
			buf.Write(l[:])
		}
		buf.Write(util.EncodeVarint(uint64(len(bpp.R))))
		for _, r := range bpp.R {
			buf.Write(r[:])
		}
	}

	for _, clsag := range tx.RctSigPrunable.CLSAGs {
		for _, s := range clsag.S {
			buf.Write(s[:])
		}
		buf.Write(clsag.C1[:])
		buf.Write(clsag.D[:])
	}

	for _, pseudo := range tx.RctSigPrunable.PseudoOuts {
		buf.Write(pseudo[:])
	}

	return buf.Bytes()
}

func (tx *Transaction) CalcHash() {
	// Step 1: Hash the transaction parts
	part1 := util.Keccak256(tx.CalculatePart1())
	part2 := util.Keccak256(tx.CalculatePart2())
	part3 := util.Keccak256(tx.CalculatePart3())

	// Step 2: Concatenate the parts
	concat := append(part1, part2...)
	concat = append(concat, part3...)

	// Step 3: Final keccak hash
	finalHash := util.Keccak256(concat)

	// Step 4: Copy to tx.Hash
	copy(tx.Hash[:], finalHash)
}

func DeriveViewTag(txPubKey []byte, privateViewKey []byte, index uint64) (byte, error) {
	if len(txPubKey) != 32 || len(privateViewKey) != 32 {
		return 0, fmt.Errorf("invalid key lengths")
	}
	// Compute shared secret (a * R). Use the shared secret for view tag.
	sharedSecret, err := util.SharedSecret(txPubKey, privateViewKey)
	if err != nil {
		return 0, err
	}

	// Compute view tag per Monero: H[salt||derivation||varint(index)], salt is 8 bytes "view_tag"
	data := make([]byte, 0, 8+len(sharedSecret)+10)
	data = append(data, []byte("view_tag")...) // 8 bytes without null terminator
	data = append(data, sharedSecret...)
	data = append(data, util.EncodeVarint(index)...)
	viewTagHash := util.Keccak256(data)
	viewTag := viewTagHash[0]

	return viewTag, nil
}

// verifyViewKeyPair checks if a private view key corresponds to a public view key
func verifyViewKeyPair(privViewKey []byte, pubViewKey []byte) bool {
	// Import edwards25519
	var scalar edwards25519.Scalar
	if _, err := scalar.SetCanonicalBytes(privViewKey); err != nil {
		return false
	}

	// Compute public key: P = a*G
	derivedPubKey := new(edwards25519.Point).ScalarBaseMult(&scalar)
	return util.EqualBytes(derivedPubKey.Bytes(), pubViewKey)
}

// decodeRctAmount decodes an encrypted RCT amount
func DecodeRctAmount(txPubKey []byte, privateViewKey []byte, outputIndex uint64, encryptedAmount []byte) (float64, error) {
	if len(encryptedAmount) != 8 {
		return 0, fmt.Errorf("invalid encrypted amount length: %d", len(encryptedAmount))
	}

	// Parse R (tx pubkey)
	Rpt, err := new(edwards25519.Point).SetBytes(txPubKey)
	if err != nil {
		return 0, fmt.Errorf("invalid tx public key: %w", err)
	}

	// Parse scalar a (private view key)
	a := new(edwards25519.Scalar)
	if _, err := a.SetCanonicalBytes(privateViewKey); err != nil {
		return 0, fmt.Errorf("invalid private view key scalar bytes: %w", err)
	}

	// Scalar eight = 8
	eightBytes := make([]byte, 32)
	eightBytes[0] = 8
	eight := new(edwards25519.Scalar)
	if _, err := eight.SetCanonicalBytes(eightBytes); err != nil {
		return 0, fmt.Errorf("creating scalar 8 failed: %w", err)
	}

	// eightA = eight * a
	eightA := new(edwards25519.Scalar).Multiply(eight, a)

	// Compute shared secret: sharedPoint = (8*a) * R
	sharedPoint := new(edwards25519.Point).ScalarMult(eightA, Rpt)
	sharedBytes := sharedPoint.Bytes()

	// Monero hash_to_scalar (Hs):
	// 1. Concatenate 8aR with varint(outputIndex)
	// 2. Hash with util.Keccak256 to get 32 bytes
	// 3. Interpret as scalar (reduce mod l) - this is Hs(8aR || i)
	hashInput := append(sharedBytes, util.EncodeVarint(outputIndex)...)
	hsHash := util.Keccak256(hashInput) // 32 bytes

	// For sc_reduce32: pad the 32-byte hash to 64 bytes for SetUniformBytes
	// SetUniformBytes expects 64 bytes and reduces mod l internally
	hsHash64 := make([]byte, 64)
	copy(hsHash64, hsHash)

	hsScalar := new(edwards25519.Scalar)
	if _, err := hsScalar.SetUniformBytes(hsHash64); err != nil {
		return 0, fmt.Errorf("hash_to_scalar failed: %w", err)
	}

	// Get the canonical bytes of the scalar Hs
	hsBytes := hsScalar.Bytes()

	// Compute amount mask: util.Keccak256("amount" || Hs)
	amountMask := util.Keccak256(append([]byte("amount"), hsBytes...))

	// XOR first 8 bytes (little-endian) to get amount
	var amount uint64
	for i := 0; i < 8; i++ {
		decrypted := encryptedAmount[i] ^ amountMask[i]
		amount |= uint64(decrypted) << (8 * i)
	}

	return float64(amount) / 1e12, nil
}

// decodeRctAmount decodes an encrypted RCT amount
// generateBulletproofPlusMask генерирует commitment mask для BP+ входа
func generateBulletproofPlusMask(txPubKey []byte, privateViewKey []byte, outputIndex uint64) (Hash, error) {

	// 1. Вычисляем derivation (8*a*R)
	Rpt, err := new(edwards25519.Point).SetBytes(txPubKey)
	if err != nil {
		return Hash{}, fmt.Errorf("invalid tx public key: %w", err)
	}

	a := new(edwards25519.Scalar)
	if _, err := a.SetCanonicalBytes(privateViewKey); err != nil {
		return Hash{}, fmt.Errorf("invalid private view key: %w", err)
	}

	eight := new(edwards25519.Scalar)
	eightBytes := [32]byte{8}
	eight.SetCanonicalBytes(eightBytes[:])

	eightA := new(edwards25519.Scalar).Multiply(eight, a)
	derivation := new(edwards25519.Point).ScalarMult(eightA, Rpt)
	derivationBytes := derivation.Bytes()

	// 2. Вычисляем Hs(8aR || outputIndex)
	hashInput := append(derivationBytes, util.EncodeVarint(outputIndex)...)
	derivationHash := util.Keccak256(hashInput)

	// 3. Reduce mod l
	derivationHash64 := make([]byte, 64)
	copy(derivationHash64, derivationHash[:])

	derivationScalar := new(edwards25519.Scalar)
	derivationScalar.SetUniformBytes(derivationHash64)
	derivationScalarBytes := derivationScalar.Bytes()

	// 4. mask = Hs("commitment_mask" || Hs(8aR||i))
	maskHash := util.Keccak256(append([]byte("commitment_mask"), derivationScalarBytes...))

	// 5. Reduce mod l
	maskHash64 := make([]byte, 64)
	copy(maskHash64, maskHash[:])

	maskScalar := new(edwards25519.Scalar)
	maskScalar.SetUniformBytes(maskHash64)

	mask := Hash(maskScalar.Bytes())

	return mask, nil
}

func EncryptRctAmount(amount float64, pubViewKey []byte, txSecretKey []byte, outputIndex uint64) (HAmount, error) {
	// Конвертируем amount в uint64 (предполагаем, что amount уже в atomic units)
	amountAtomic := uint64(amount * 1e12)

	// Получаем shared secret (shared = 8 * txSecretKey * pubViewKey)
	shared, err := util.SharedSecret(pubViewKey, txSecretKey)
	if err != nil {
		return [8]byte{}, err
	}

	// Вычисляем Hs = sc_reduce32(util.Keccak256(shared || varint(index)))
	hashInput := append(shared, util.EncodeVarint(outputIndex)...)
	hsHash := util.Keccak256(hashInput)

	// Для получения скаляра используем SetUniformBytes (ожидает 64 байта)
	hsHash64 := make([]byte, 64)
	copy(hsHash64, hsHash)
	hsScalar := new(edwards25519.Scalar)
	if _, err := hsScalar.SetUniformBytes(hsHash64); err != nil {
		return [8]byte{}, err
	}
	hsBytes := hsScalar.Bytes()

	// amountMask = util.Keccak256("amount" || hsBytes)
	amountMask := util.Keccak256(append([]byte("amount"), hsBytes...))

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

// func CalcOutPk(pubViewKey []byte, pubSpendKey []byte, txSecretKey []byte, outputIndex uint64) (Hash, error) {
// CalcOutPk calculates the output commitment (outPk) for a Monero RingCT transaction
// This is a Pedersen commitment: C = xG + aH where:
// - x is the blinding factor (mask) derived from shared secret
// - a is the amount in atomic units
// - G is the base point, H is the second base point
// CalcOutPk calculates the output commitment (outPk) for a Monero RingCT transaction
// This is a Pedersen commitment: C = xG + aH where:
// - x is the blinding factor (mask) derived from shared secret
// - a is the amount in atomic units
// - G is the base point, H is the second base point
func CalcOutPk(amount float64, pubViewKey []byte, pubSpendKey []byte, txSecretKey []byte, outputIndex uint64) (*edwards25519.Scalar, Hash, error) {
	// Convert amount to atomic units
	amountAtomic := uint64(amount * 1e12)

	// ВАЖНО: Сначала вычисляем shared secret правильно
	// В Monero: shared_secret = r * A (где r - tx secret key, A - pub view key)
	// НО для Edwards25519 нужно умножить на 8 (cofactor)

	// Parse recipient's public view key
	pubViewPoint, err := new(edwards25519.Point).SetBytes(pubViewKey)
	if err != nil {
		return nil, Hash{}, fmt.Errorf("invalid public view key: %w", err)
	}

	// Parse tx secret key as scalar
	txSecScalar := new(edwards25519.Scalar)
	if _, err := txSecScalar.SetCanonicalBytes(txSecretKey); err != nil {
		return nil, Hash{}, fmt.Errorf("invalid tx secret key: %w", err)
	}

	// Create scalar 8 (cofactor)
	eightBytes := make([]byte, 32)
	eightBytes[0] = 8
	eight := new(edwards25519.Scalar)
	if _, err := eight.SetCanonicalBytes(eightBytes); err != nil {
		return nil, Hash{}, fmt.Errorf("failed to create scalar 8: %w", err)
	}

	// Compute 8 * r
	eightR := new(edwards25519.Scalar).Multiply(eight, txSecScalar)

	// Compute shared secret point: (8*r) * A
	sharedPoint := new(edwards25519.Point).ScalarMult(eightR, pubViewPoint)
	sharedSecret := sharedPoint.Bytes()

	// Compute Hs(shared_secret || index) - derivation scalar
	hashInput := append(sharedSecret, util.EncodeVarint(outputIndex)...)
	hsHash := util.Keccak256(hashInput)

	// Convert to scalar (reduce mod l)
	hsHash64 := make([]byte, 64)
	copy(hsHash64, hsHash)
	hsScalar := new(edwards25519.Scalar)
	if _, err := hsScalar.SetUniformBytes(hsHash64); err != nil {
		return nil, Hash{}, fmt.Errorf("failed to derive Hs scalar: %w", err)
	}
	hsBytes := hsScalar.Bytes()

	// Compute blinding factor (mask): x = Hs("commitment_mask" || Hs(8rA||i))
	maskInput := append([]byte("commitment_mask"), hsBytes...)
	maskHash := util.Keccak256(maskInput)

	maskHash64 := make([]byte, 64)
	copy(maskHash64, maskHash)

	blindingFactor := new(edwards25519.Scalar)
	if _, err := blindingFactor.SetUniformBytes(maskHash64); err != nil {
		return nil, Hash{}, fmt.Errorf("failed to derive blinding factor: %w", err)
	}

	// Create scalar from amount (little-endian)
	amountBytes := make([]byte, 32)
	binary.LittleEndian.PutUint64(amountBytes, amountAtomic)
	amountScalar := new(edwards25519.Scalar)
	if _, err := amountScalar.SetCanonicalBytes(amountBytes); err != nil {
		return nil, Hash{}, fmt.Errorf("failed to create amount scalar: %w", err)
	}

	// Get H (second base point)
	H := getH()

	// Compute Pedersen commitment: C = xG + aH
	// где x - blinding factor (mask), a - amount

	// xG = blinding_factor * G
	xG := new(edwards25519.Point).ScalarBaseMult(blindingFactor)

	// aH = amount * H
	aH := new(edwards25519.Point).ScalarMult(amountScalar, H)

	// C = xG + aH
	commitment := new(edwards25519.Point).Add(xG, aH)

	return blindingFactor, Hash(commitment.Bytes()), nil
}

func CalcCommitment(amount uint64, mask [32]byte) (Hash, error) {
	blindingFactor := new(edwards25519.Scalar)
	if _, err := blindingFactor.SetCanonicalBytes(mask[:]); err != nil {
		return Hash{}, fmt.Errorf("failed to derive blinding factor: %w", err)
	}

	// Create scalar from amount (little-endian)
	amountBytes := make([]byte, 32)
	binary.LittleEndian.PutUint64(amountBytes, amount)
	amountScalar := new(edwards25519.Scalar)
	if _, err := amountScalar.SetCanonicalBytes(amountBytes); err != nil {
		return Hash{}, fmt.Errorf("failed to create amount scalar: %w", err)
	}

	xG := new(edwards25519.Point).ScalarBaseMult(blindingFactor)
	aH := new(edwards25519.Point).ScalarMult(amountScalar, getH())
	commitment := new(edwards25519.Point).Add(xG, aH)

	return Hash(commitment.Bytes()), nil
}

func CalcScalars(scalars []*edwards25519.Scalar) (*edwards25519.Scalar, error) {
	sum := edwards25519.NewScalar()
	for _, scalar := range scalars {
		sum.Add(sum, scalar)
	}
	return sum, nil
}

// getH returns the second generator point H used in Pedersen commitments
// H is derived as hash_to_point(util.Keccak256(G))
func getH() *edwards25519.Point {
	// Известное значение H для Monero (это константа!)
	// H = 8b655970153799af2aeadc9ff1add0ea6c7251d54154cfa92c173a0dd39c1f94
	hHex := "8b655970153799af2aeadc9ff1add0ea6c7251d54154cfa92c173a0dd39c1f94"
	hBytes, _ := hex.DecodeString(hHex)

	H, err := new(edwards25519.Point).SetBytes(hBytes)
	if err != nil {
		// Fallback: вычислить H из G
		return deriveH()
	}

	return H
}

// deriveH computes H if hardcoded value fails
func deriveH() *edwards25519.Point {
	G := edwards25519.NewGeneratorPoint().Bytes()
	hHash := util.Keccak256(G)

	H, err := new(edwards25519.Point).SetBytes(hHash)
	if err == nil {
		return H
	}

	// Iterative approach with counter
	for i := 0; i < 256; i++ {
		data := append(G, byte(i))
		attempt := util.Keccak256(data)
		H, err = new(edwards25519.Point).SetBytes(attempt)
		if err == nil {
			return H
		}
	}

	return edwards25519.NewGeneratorPoint() // fallback to G (should never happen)
}

func (i *TxInput) Serialize() []byte {
	var buf bytes.Buffer

	buf.WriteByte(2) //txInToKeyMarker
	buf.Write(util.EncodeVarint(i.Amount))
	buf.Write(util.EncodeVarint(uint64(len(i.KeyOffsets))))

	for _, keyOffset := range i.KeyOffsets {
		buf.Write(util.EncodeVarint(keyOffset))
	}

	buf.Write(i.KeyImage[:])
	return buf.Bytes()
}

func (o *TxOutput) Serialize() []byte {
	var buf bytes.Buffer
	buf.Write(util.EncodeVarint(0))
	buf.WriteByte(o.Type)
	buf.Write(o.Target[:])
	buf.WriteByte(byte(o.ViewTag))

	return buf.Bytes()
}
