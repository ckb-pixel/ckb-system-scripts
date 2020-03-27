package main

import (
	"context"
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

	key, err := secp256k1.HexToKey("86c5661a58a0589009a600b9008ec083ddf65f0b8e194aa2b1d5178fbdf8122f")
	if err != nil {
		log.Fatalf("import private key error: %v", err)
	}

	scripts, err := utils.NewSystemScripts(client)
	if err != nil {
		log.Fatalf("load system script error: %v", err)
	}

	owner, err := key.Script(scripts)
	pixelID, err := hexutil.Decode("0xcd64ecc1fa2570073cbe9b2dfda7974288b564f323b4cd07e9d84fef22d62661")

	args, err := hexutil.Decode("0xedcda9513fa030ce4308e29245a22c022d0443bb")
	if err != nil {
		log.Fatalf("decode hex error: %v", err)
	}

	tx := transaction.NewSecp256k1SingleSigTx(scripts)

	tx.CellDeps = append(tx.CellDeps, &types.CellDep{
		OutPoint: &types.OutPoint{
			TxHash: types.HexToHash("0x57c2344716e4ac7ef23fe84d9ebe9bf6f51079347c8f7e7796eba1dc22903b28"),
			Index:  0,
		},
		DepType: types.DepTypeCode,
	})
	/*
	tx.CellDeps = append(tx.CellDeps, &types.CellDep{
		OutPoint: &types.OutPoint{
			TxHash: types.HexToHash("0x3615307e4f8e435113a53bf0500d34e4c4046db2b7877ba26a1955adc206d7ff"),
			Index:  0,
		},
		DepType: types.DepTypeCode,
	})*/

	// lock
	tx.Outputs = append(tx.Outputs, &types.CellOutput{
		Capacity: uint64(14200000000),
		Lock: &types.Script{
			CodeHash: types.HexToHash("0x9bd7e06f3ecf4be0f2fcd2188b23f1b9fcc88e5d4b65a8637b17723bbda3cce8"),
			HashType: types.HashTypeType,
			Args:     args,
		},
		Type: &types.Script{
			CodeHash: types.HexToHash("0x295c725e14ddd32019d09b1a72876d688d494281a1a973aa19eaf9a9d2e84bd1"),
			HashType: types.HashTypeData,
			Args:     pixelID,
		},
	})
	tx.OutputsData = append(tx.OutputsData, []byte{'0', '0', '2', '2', '2'})
	tx.Outputs = append(tx.Outputs, &types.CellOutput{
		Capacity: uint64(494142299870000),
		Lock: owner,
	})
	tx.OutputsData = append(tx.OutputsData, []byte{})

	group, witnessArgs, err := transaction.AddInputsForTransaction(tx, []*types.Cell{
		{
			OutPoint: &types.OutPoint{
				TxHash: types.HexToHash("0xc9186190d7d48f4b8fdcdecaec494296659a8d77b304afbd6ed90eec867223bc"),
				Index: 0,
			},
		}, {
			OutPoint: &types.OutPoint{
				TxHash: types.HexToHash("0x83efcccdf78155e98abf6159a15ef4434af057047bb37415dd8e1d9c5ceaa5b3"),
				Index: 1,
			},
		},
	})
	err = transaction.SingleSignTransaction(tx, group, witnessArgs, key)
	if err != nil {
		log.Fatalf("sign transaction error: %v", err)
	}
	hash, err := client.SendTransaction(context.Background(), tx)
	if err != nil {
		log.Fatalf("send transaction error: %v", err)
	}

	fmt.Println(hash.String())
}
