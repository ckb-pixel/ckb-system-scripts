package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"

	"github.com/ququzone/ckb-sdk-go/crypto/secp256k1"
	"github.com/ququzone/ckb-sdk-go/rpc"
	"github.com/ququzone/ckb-sdk-go/transaction"
	"github.com/ququzone/ckb-sdk-go/types"
	"github.com/ququzone/ckb-sdk-go/utils"
)

func main() {
	dataFile, err := os.Open("./pixel_canvas")
	if err != nil {
		log.Fatalf("load data file error: %v", err)
	}
	defer dataFile.Close()

	dataInfo, err := dataFile.Stat()
	if err != nil {
		log.Fatalf("load data info error: %v", err)
	}

	data := make([]byte, dataInfo.Size())
	_, err = dataFile.Read(data)
	if err != nil {
		log.Fatalf("read data file error: %v", err)
	}

	client, err := rpc.Dial("http://127.0.0.1:8115")
	if err != nil {
		log.Fatalf("create rpc client error: %v", err)
	}

	key, err := secp256k1.HexToKey("ff1f91f7a63893d2f5a1bd424b139718ff6b0eb66853ace772e7a25250ce635f")
	if err != nil {
		log.Fatalf("import private key error: %v", err)
	}

	scripts, err := utils.NewSystemScripts(client)
	if err != nil {
		log.Fatalf("load system script error: %v", err)
	}

	change, err := key.Script(scripts)

	capacity := uint64(dataInfo.Size()/1000+1) * 1000 * uint64(math.Pow10(8))
	// fee rate lower than min_fee_rate: 1000 shannons/KB
	fee := uint64(5280)

	cellCollector := utils.NewCellCollector(client, change, capacity+fee)
	cells, total, err := cellCollector.Collect()
	if err != nil {
		log.Fatalf("collect cell error: %v", err)
	}

	if total < capacity+fee {
		log.Fatalf("insufficient capacity: %d < %d", total, capacity+fee)
	}

	tx := transaction.NewSecp256k1SingleSigTx(scripts)
	tx.Outputs = append(tx.Outputs, &types.CellOutput{
		Capacity: uint64(capacity),
		Lock: &types.Script{
			CodeHash: types.HexToHash("0x9bd7e06f3ecf4be0f2fcd2188b23f1b9fcc88e5d4b65a8637b17723bbda3cce8"),
			HashType: types.HashTypeType,
			Args:     change.Args,
		},
	})
	tx.OutputsData = append(tx.OutputsData, data)
	if total-capacity+fee > 0 {
		tx.Outputs = append(tx.Outputs, &types.CellOutput{
			Capacity: total - capacity - fee,
			Lock: &types.Script{
				CodeHash: types.HexToHash("0x9bd7e06f3ecf4be0f2fcd2188b23f1b9fcc88e5d4b65a8637b17723bbda3cce8"),
				HashType: types.HashTypeType,
				Args:     change.Args,
			},
		})
		tx.OutputsData = append(tx.OutputsData, []byte{})
	}

	group, witnessArgs, err := transaction.AddInputsForTransaction(tx, cells)
	if err != nil {
		log.Fatalf("add inputs to transaction error: %v", err)
	}

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
