package types

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math/bits"
	"slices"

	"github.com/0xAF4/go-monero/levin"
	"github.com/0xAF4/go-monero/util"
)

type Block struct {
	block []byte   `json:"-"`
	tx    [][]byte `json:"-"`

	MajorVersion      uint8  `json:"major_version"`
	MinorVersion      uint8  `json:"minor_version"`
	BlockHeight       uint64 `json:"height"`
	Timestamp         uint64 `json:"timestamp"`
	PreviousBlockHash Hash   `json:"prev_id"`
	Nonce             uint32 `json:"nonce"`
	MinerTx           struct {
		Version    uint64     `json:"version"`
		UnlockTime uint64     `json:"unlock_time"`
		VinCount   uint64     `json:"vin_count"`
		InputType  byte       `json:"input_type"`
		Height     uint64     `json:"height"`
		OutputNum  uint64     `json:"-"`
		Outs       []TxOutput `json:"vout"`
		ExtraSize  uint64     `json:"-"`
		Extra      ByteArray  `json:"extra"`
	} `json:"miner_tx"`

	TxsCount uint64         `json:"txs_count"`
	TXs      []*Transaction `json:"-"`
}

const (
	TxOutToKey       = 2
	TxOutToTaggedKey = 3
)

func NewBlock() *Block {
	return &Block{}
}

func (b *Block) SetBlockData(data []byte) {
	b.block = data
}

func (b *Block) InsertTx(data []byte) {
	b.tx = append(b.tx, data)
}

func (block *Block) FullfillBlockHeader() error {
	if len(block.block) < 43 {
		return fmt.Errorf("block data too short: %d bytes", len(block.block))
	}

	reader := bytes.NewReader(block.block)
	//----
	block.MajorVersion, _ = levin.ReadUint8(reader)
	block.MinorVersion, _ = levin.ReadUint8(reader)
	//----
	timestamp, _ := levin.ReadVarint(reader)
	block.Timestamp = timestamp
	//----
	reader.Read(block.PreviousBlockHash[:])
	binary.Read(reader, binary.LittleEndian, &block.Nonce)
	//----
	block.MinerTx.Version, _ = levin.ReadVarint(reader)
	block.MinerTx.UnlockTime, _ = levin.ReadVarint(reader)
	block.MinerTx.VinCount, _ = levin.ReadVarint(reader)
	block.MinerTx.InputType, _ = reader.ReadByte()
	block.MinerTx.Height, _ = levin.ReadVarint(reader)
	block.MinerTx.OutputNum, _ = levin.ReadVarint(reader)
	block.BlockHeight = block.MinerTx.Height
	//----
	outs := []TxOutput{}
	for i := 1; i <= int(block.MinerTx.OutputNum); i++ {
		out := TxOutput{}
		out.Amount, _ = levin.ReadVarint(reader)
		out.Type, _ = reader.ReadByte()
		reader.Read(out.Target[:])
		b, _ := reader.ReadByte()
		out.ViewTag = HByte(b)
		outs = append(outs, out)
	}
	block.MinerTx.Outs = outs
	//----
	block.MinerTx.ExtraSize, _ = levin.ReadVarint(reader)
	extra := make([]byte, block.MinerTx.ExtraSize)
	reader.Read(extra)
	block.MinerTx.Extra = extra
	reader.Seek(1, io.SeekCurrent)
	//----
	block.TxsCount, _ = levin.ReadVarint(reader)
	for i := 0; i <= int(block.TxsCount)-1; i++ {
		tx := &Transaction{
			Raw: block.tx[i],
		}
		reader.Read(tx.Hash[:])
		block.TXs = append(block.TXs, tx)
	}

	return nil
}

func (b *Block) getBlockHeader() []byte {
	var buf bytes.Buffer

	buf.WriteByte(b.MajorVersion)
	buf.WriteByte(b.MinorVersion)
	timestampBytes := util.EncodeVarint(b.Timestamp)
	buf.Write(timestampBytes)
	buf.Write(b.PreviousBlockHash[:])
	nonceBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(nonceBytes, b.Nonce)
	buf.Write(nonceBytes)

	return buf.Bytes()
}

func leafHash(data []Hash) Hash {
	switch len(data) {
	case 0:
		panic("unsupported length")
	case 1:
		return data[0]
	default:
		//only hash the next two items
		var buf bytes.Buffer
		buf.Write(data[0][:])
		buf.Write(data[1][:])
		return Hash(util.Keccak256(buf.Bytes()))
	}
}

func PreviousPowerOfTwo(x uint64) int {
	if x == 0 {
		return 0
	}
	return 1 << (bits.Len64(x) - 1)
}

func (b *Block) calcMerkleRoot(t []Hash) (rootHash Hash) {

	count := len(t)
	if count <= 2 {
		return leafHash(t)
	}

	pow2cnt := PreviousPowerOfTwo(uint64(count))
	offset := pow2cnt*2 - count

	temporaryTree := make([]Hash, pow2cnt)
	copy(temporaryTree, t[:offset])

	//TODO: maybe can be done zero-alloc
	//temporaryTree := t[:max(pow2cnt, offset)]

	offsetTree := temporaryTree[offset:]
	for i := range offsetTree {
		offsetTree[i] = leafHash(t[offset+i*2:])
	}

	for pow2cnt >>= 1; pow2cnt > 1; pow2cnt >>= 1 {
		for i := range temporaryTree[:pow2cnt] {
			temporaryTree[i] = leafHash(temporaryTree[i*2:])
		}
	}

	rootHash = leafHash(temporaryTree)

	return
}

func (b *Block) CalculateMinerBuff() []byte {
	c := b.MinerTx
	var buf bytes.Buffer
	buf.Write(util.EncodeVarint(c.Version))
	buf.Write(util.EncodeVarint(c.UnlockTime))
	buf.Write(util.EncodeVarint(c.VinCount))
	buf.WriteByte(c.InputType)
	buf.Write(util.EncodeVarint(c.Height))

	buf.Write(util.EncodeVarint(uint64(len(c.Outs))))
	for _, o := range c.Outs {
		buf.Write(util.EncodeVarint(o.Amount))
		buf.WriteByte(o.Type)
		if slices.Contains([]byte{TxOutToTaggedKey, TxOutToKey}, o.Type) {
			buf.Write(o.Target[:])
		}
		if o.Type == TxOutToTaggedKey {
			buf.WriteByte(byte(o.ViewTag))
		}
	}

	buf.Write(util.EncodeVarint(uint64(len(c.Extra))))
	buf.Write(c.Extra)
	return buf.Bytes()
}

func (b *Block) CalculateMinerTxHash() []byte {
	txHashingBlob := make([]byte, 96)
	buff := b.CalculateMinerBuff()

	copy(txHashingBlob, util.Keccak256(buff))
	copy(txHashingBlob[32:], util.Keccak256([]byte{0}))

	return util.Keccak256(txHashingBlob)
}

func (b *Block) GetHashingBlob() []byte {
	var (
		hashingblob []byte
		header      []byte
		merkleroot  []byte
		txcount     []byte
	)

	header = b.getBlockHeader()

	merkleTree := make([]Hash, b.TxsCount+1)
	merkleTree[0] = Hash(b.CalculateMinerTxHash())
	for i, tx := range b.TXs {
		merkleTree[i+1] = tx.Hash
	}
	hash := b.calcMerkleRoot(merkleTree)
	merkleroot = hash[:]

	txcount = util.EncodeVarint(b.TxsCount + 1)

	hashingblob = append(hashingblob, header...)
	hashingblob = append(hashingblob, merkleroot...)
	hashingblob = append(hashingblob, txcount...)
	return hashingblob
}

func (b *Block) GetBlockId() string {
	var varIntBuf [binary.MaxVarintLen64]byte
	hashingblob := b.GetHashingBlob()
	data := varIntBuf[:binary.PutUvarint(varIntBuf[:], uint64(len(hashingblob)))]

	final := []byte{}
	final = append(final, data...)
	final = append(final, hashingblob...)

	return hex.EncodeToString(util.Keccak256(final))
}
