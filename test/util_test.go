package test

import (
	"fmt"
	"testing"

	"github.com/0xAF4/go-monero/util"
)

func Test_RandomScalar(t *testing.T) {
	util.SetTest(true)
	scalar := util.RandomScalar()
	fmt.Printf("Scalar: %x\n", scalar.ToBytes())
}

func Test_DerivePublicKey(t *testing.T) {
	derivation, _ := util.ParseKeyFromHex("2d282437dc6ad4c123eaa5657a233b6a4398ad820eb6704329679b05e720d283")
	m_spend_public_key, _ := util.ParseKeyFromHex("c731e50450ef611bfadbc5e873c758688719394b7ac837c33b0b0e94f3cdb705")
	key, _ := util.DerivePublicKey(&derivation, 0, &m_spend_public_key)
	fmt.Printf("Key: %x\n", key)
}
