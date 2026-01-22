package test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/0xAF4/go-monero/rpc"
	"github.com/0xAF4/go-monero/types"
)

const (
	Address        = "49AvioLCdkk5gSXww99nEKJV3tsyBxszEZeywa7K3jQi1qcBhz4AJhPU6sroCaEsqDMJg1iG5Sv1z78u6vsa1fQCRaXGb1w"
	PrivateViewKey = "4fd69daf111e62ad6d64bfa3a529751db91eb35ef547e00d58ca1a99aee98209"
	TxID           = "fafe9b40569b1d04b68967ab7576a4037a50ce6859a08f84f3815fcc82817f4f"
)

func Test_Transaction_CheckBalance(t *testing.T) {
	client := rpc.NewDaemonRPCClient(timeout, 0, nil)

	resp, err := client.GetTransactions([]string{TxID})
	if err != nil {
		t.Fatalf("GetTransactions returned error: %v", err)
	}

	if resp == nil {
		t.Fatal("response is nil")
	}
	data := (*resp)[0]["data"].([]byte)

	transaction := types.Transaction{Raw: data}
	transaction.ParseTx()
	transaction.ParseRctSig()
	transaction.CalcHash()

	funds, paymentID, err := transaction.CheckOutputs(Address, PrivateViewKey)
	if err != nil {
		fmt.Printf("  - TX checkOutputs error: %s", err)
	} else {
		fmt.Printf("  - TX checkOutputs find in tx: %.12f; PaymentID: %d", funds, paymentID)
	}
}

const (
	hexTx           = "020001020010cfd0dd09cec38703e1f2a803e6eb6d92b5cc0e938999018cafcb0995852f9fcdce03fd878302d5fcb706efb1ef01f796ed03ade2bb02c4b4e003ae9dc3032fd8cf12571fa65756c7a5e3abd213b71ead89960cb7a4571bba109571cb2ce3020003b5c7b09c9235ab6c97b16626e9a1bd56590e8e1a7ed9c019fba0fa3c3d4aa28f08000304433e5edc78ea87e0e2827511051781faa930ca8b2e3056ff06f3c64d147700022c01f810207ee5b3f3500e65462d3db7107b118dd5500269610ade1add3046f9250a0209014a9ef716e901476f068094ebdc037eae81a0b4aea687a4987612515d958c219fd47a5d71899ed010175181b3182fc06279e550ba792f23ea09c82e91da2e66cda4cdd011448d9cf7af1e7fc2692d5a30f872e171fe669218bd9ee5112b3c01b2883de13a0e036fc94ee039f8b429fc6584efe9dc85f6130ac344d5d04f60a2103e05aa4e2d812577e597b25b9f9ee801377fcd1980f2aec1b629b8f98b67af0ef99178c1c08a42003899ed59efe45f85b3e6f4cd3670f9b093a62dd11a6008cff25cb31b0bcb73f601271ff9b97518452088b5693b3c11b76540f8a4b8a3036e5bed075926372e79b6fd448283a131162ff0775846297850ff5781ad41c809df67c1824c62c6988751edae2a88121048b3a5f3ba013d52306aa98fc59c960e075451ee334eb87de73ab4ab52b23930d15be17d3b007c67b1dbb5ce3b9bb37b9b6a65d84ef1098264cacb21b06b48ad3e9bfd523d1d68bf0d23a7e5c479bc9dcebbce10137902c442f887663bcec3a613f1c4e45728d801940f082917dec8f8892c29592d519cdca27a67c30784264d1740de761fcd8ef40e0cfe40e3dd6faa2ff9af309276d61219a97795ab5b0baaddbcf1a8089c43e26e4ea1e9bad80d72007ce02945fa596d06274186cb60953e351701a78c869bdb26ae9ab375c5f6e0c905db2a671e3079b97250c4ee2e7c381458e3920951954e49758d036063d10f0f07a8cb17f1a4156065b0230ac8ae611d39dc20cbf276711acf20f0333172f1e1283492089d898ed929ac9b17627f449911498ab55436219210c13fef3485590924367636945cff5c2794e3e18a97bb0c9d2cd736e69fc18276f5f031fbb0ea041282e65f436877fb90fbc7b635470a7a7d04d951e295a8b4c7b0b086cf8714b20d7aec78b660ec1c4f46a2bc7bd1d5001521c8e2b7073239681ffcc456ad122c71b26d411c73f439b679f3fe7beb3522c27e2b7a0c1675c4fe387104e29f59bf8920eab020fa1797e782680f8a47226b96401e3c40068c7d217d05df72fd645c4ee62bb45611c586ad697a18d604c1fa7256dc04927ffe645333fc683a97df0b0b6112c2d96196b1140ceef9f5cd3b7cfa9b788edd74221848ff5c1835f3afd608f14871089ead5fd7b8dcb9b6abe0734343a96c22833d2ac23c28f187673d640872eefded49567ee21dd98d6729fe32ecbae6cc0d4da99bf314eb1d0672d9a70cfbd684350337c9680200c595c649252f6df22b2a022033c0d455848171abae0267cc85cec3e869c2efa7369554e4228f45d87c552882bdc1234b7a9eb207c4087fad71a184d0201cb7d1c54afce0d72555c051ba6cc47d0253413bb2a8e636094dfff26ba1cebc51383da45a4b68cf17b23d2f01d55e5f224b8205bd55c3bb0c760c2c77f5f5b3297072a6e13525a2f622ee1a0ee510d1f091cd11a2601d7f0e505028fff7a2571b08d98ddcd3b1e853bf7595c1f4037c3fae44573040672e0d73579b5544e24df29ce9474ac12c96501121b9f0420e4d0f05766040de18020c7f47be4c328f8591d27a512328e9421026519bc0c695d0e301c2cdf99c07590ead10f723d592190b563fe59314095392e82f6b5b7e9d196c0fb9dd4012d9db05f03876c124ce82995fb0e6d035c8cb7390b1a47ef1a07ed605cba9b5f6c3c90e05098aa01f27fc586f47dd35d030aa7c40a262ef5e3c49c2611f1245c7d9780ba853edf22a6ebbafa32d50ff81d9bc571a241383d3d0c560dfeae8202b11d408698fb8271dccf2eb3da3fa47480e42fb4f879d9fb473b7a440832c873723f20ea79f458816f01f699d9a3579b88e369cff5c8a5847ce5b4f53ebc24fd7da8fd4ef089b7b2e4c28b6b51d3a7614a27c256609dd988eeb3fb74b99d5b69087ecd8"
	Address2        = "87NXDKzEGk1KhAwc6PfbES1L9kgkMCZHhMyvA4vZLnk5HLeusN859wmd6zpDp44n8xM4yQtGdWeNMg3ExA4Do3tNPwLhqzy"
	PrivateViewKey2 = "0ccdb326933cf18677cd09df73a147b2e7f6bbe4f22876ed0edbaebe2c691505"
)

func Test_Transaction_CheckBalanceFromHex(t *testing.T) {
	hex, _ := hex.DecodeString(hexTx)

	transaction := types.Transaction{Raw: hex}
	transaction.ParseTx()
	transaction.ParseRctSig()
	transaction.CalcHash()

	funds, paymentID, err := transaction.CheckOutputs(Address2, PrivateViewKey2)
	if err != nil {
		fmt.Printf("  - TX checkOutputs error: %s", err)
	} else {
		fmt.Printf("  - TX checkOutputs find in tx: %.12f; PaymentID: %d", funds, paymentID)
	}
}

func Test_Transaction_CreateEmptyTransaction(t *testing.T) {
	types.SetTest(true)
	tx := types.NewEmptyTransaction()
	fmt.Printf("SecretKey: %x\n", tx.SecretKey)
}
