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
