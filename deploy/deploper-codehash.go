package main

import (
	"encoding/hex"
	"fmt"
	"github.com/ququzone/ckb-sdk-go/crypto/blake2b"
	"log"
	"os"
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

	hash, _ := blake2b.Blake256(data)

	fmt.Println(hex.EncodeToString(hash))
}
