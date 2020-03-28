package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ququzone/ckb-sdk-go/crypto/secp256k1"
	"github.com/ququzone/ckb-sdk-go/rpc"
	"github.com/ququzone/ckb-sdk-go/transaction"
	"github.com/ququzone/ckb-sdk-go/types"
	"github.com/ququzone/ckb-sdk-go/utils"
	"log"
)

func main() {
	b, _ := hexutil.Decode("0x6a242b57227484e904b4e08ba96f19a623c367dcbd18675ec6f2a71a0ff4ec26")
	fmt.Println(b)

	client, err := rpc.Dial("http://127.0.0.1:8115")
	if err != nil {
		log.Fatalf("create rpc client error: %v", err)
	}

	key, err := secp256k1.HexToKey("9a2b104a02ba9a5959c51368e3c4b38c503c0f519c04d202d4709aeb6a1158f7")
	if err != nil {
		log.Fatalf("import private key error: %v", err)
	}

	scripts, err := utils.NewSystemScripts(client)
	if err != nil {
		log.Fatalf("load system script error: %v", err)
	}

	bider, err := key.Script(scripts)
	pixelID, err := hexutil.Decode("0xcd64ecc1fa2570073cbe9b2dfda7974288b564f323b4cd07e9d84fef22d62661")
	official, err := hexutil.Decode("0xedcda9513fa030ce4308e29245a22c022d0443bb")
	owner, err := hexutil.Decode("0xedcda9513fa030ce4308e29245a22c022d0443bb")

	tx := transaction.NewSecp256k1SingleSigTx(scripts)

	// pixel canvas
	tx.CellDeps = append(tx.CellDeps, &types.CellDep{
		OutPoint: &types.OutPoint{
			TxHash: types.HexToHash("0x57c2344716e4ac7ef23fe84d9ebe9bf6f51079347c8f7e7796eba1dc22903b28"),
			Index:  0,
		},
		DepType: types.DepTypeCode,
	})
	// pixel lock
	tx.CellDeps = append(tx.CellDeps, &types.CellDep{
		OutPoint: &types.OutPoint{
			TxHash: types.HexToHash("0x42bbf1806f8baf8bd6b16c0682157dc717c3d021644aae108e03e452479199b1"),
			Index:  0,
		},
		DepType: types.DepTypeCode,
	})

	// pixel canvas
	tx.Outputs = append(tx.Outputs, &types.CellOutput{
		Capacity: uint64(18000000000),
		Lock: &types.Script{
			CodeHash: types.HexToHash("0xe959ac726354858d598c9ea1ceb5f617e409b1b0a4a3baa25aa08b6da7b95091"),
			HashType: types.HashTypeType,
			Args:     bider.Args,
		},
		Type: &types.Script{
			CodeHash: types.HexToHash("0x295c725e14ddd32019d09b1a72876d688d494281a1a973aa19eaf9a9d2e84bd1"),
			HashType: types.HashTypeData,
			Args:     pixelID,
		},
	})
	tx.OutputsData = append(tx.OutputsData, []byte{'0', '0', '1', '2', '3'})
	// secp256k1
	tx.Outputs = append(tx.Outputs, &types.CellOutput{
		Capacity: uint64(6100000000),
		Lock: &types.Script{
			CodeHash: types.HexToHash("0x9bd7e06f3ecf4be0f2fcd2188b23f1b9fcc88e5d4b65a8637b17723bbda3cce8"),
			HashType: types.HashTypeType,
			Args:     official,
		},
	})
	tx.OutputsData = append(tx.OutputsData, []byte{})
	// origin pixel lock
	tx.Outputs = append(tx.Outputs, &types.CellOutput{
		Capacity: uint64(18000000000),
		Lock: &types.Script{
			CodeHash: types.HexToHash("0xe959ac726354858d598c9ea1ceb5f617e409b1b0a4a3baa25aa08b6da7b95091"),
			HashType: types.HashTypeType,
			Args:     owner,
		},
	})
	tx.OutputsData = append(tx.OutputsData, []byte{})
	// change
	tx.Outputs = append(tx.Outputs, &types.CellOutput{
		Capacity: uint64(767299980000),
		Lock: bider,
	})
	tx.OutputsData = append(tx.OutputsData, []byte{})

	// pixel canvas
	_, _, err = transaction.AddInputsForTransaction(tx, []*types.Cell{
		{
			OutPoint: &types.OutPoint{
				TxHash: types.HexToHash("0xf23b95472358dd731b71f53dbd19031347a8d1bce8ba4968978ab94d78f6f44a"),
				Index: 0,
			},
		},
	})
	// pay
	group, witnessArgs, err := transaction.AddInputsForTransaction(tx, []*types.Cell{
		{
			OutPoint: &types.OutPoint{
				TxHash: types.HexToHash("0x59f1fca75408ea690c6055022ccfa8961e782cbfff16ce083fd15216191543e3"),
				Index: 3,
			},
		},
	})
	err = transaction.SingleSignTransaction(tx, group, witnessArgs, key)
	if err != nil {
		log.Fatalf("sign transaction error: %v", err)
	}

	jtx, _ := json.Marshal(tx)
	fmt.Println(string(jtx))

	hash, err := client.SendTransaction(context.Background(), tx)
	if err != nil {
		log.Fatalf("send transaction error: %v", err)
	}

	fmt.Println(hash.String())
}
