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
	client, err := rpc.Dial("http://127.0.0.1:8115")
	if err != nil {
		log.Fatalf("create rpc client error: %v", err)
	}

	key, err := secp256k1.HexToKey("5a59abed0d6fcbfb58b48dcabb4ebae9b91432a3fe1311964bc96b2e410d7da7")
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
	owner, err := hexutil.Decode("0x69b7667edbe08cf19413102fcadc53c67e34fb71")

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
	tx.OutputsData = append(tx.OutputsData, []byte{'0', '0', 'a', 'b', 'c'})

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
		Capacity: uint64(21600000000),
		Lock: &types.Script{
			CodeHash: types.HexToHash("0xe959ac726354858d598c9ea1ceb5f617e409b1b0a4a3baa25aa08b6da7b95091"),
			HashType: types.HashTypeType,
			Args:     owner,
		},
	})
	tx.OutputsData = append(tx.OutputsData, []byte{})
	// change
	tx.Outputs = append(tx.Outputs, &types.CellOutput{
		Capacity: uint64(483582199969000),
		Lock: bider,
	})
	tx.OutputsData = append(tx.OutputsData, []byte{})

	// pixel canvas
	_, _, err = transaction.AddInputsForTransaction(tx, []*types.Cell{
		{
			OutPoint: &types.OutPoint{
				TxHash: types.HexToHash("0x2db586414e050c19a5759e714bbbb7f948da7133b91b0ed4660091c42bec4b4b"),
				Index: 0,
			},
		},
	})
	// pay
	group, witnessArgs, err := transaction.AddInputsForTransaction(tx, []*types.Cell{
		{
			OutPoint: &types.OutPoint{
				TxHash: types.HexToHash("0x3615307e4f8e435113a53bf0500d34e4c4046db2b7877ba26a1955adc206d7ff"),
				Index: 1,
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
