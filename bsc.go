package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var StartedTime int64

func getAccountAuth(client *ethclient.Client, accountAddress string) *bind.TransactOpts {

	privateKey, err := crypto.HexToECDSA(accountAddress)
	if err != nil {
		panic(err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		panic("invalid key")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	//fetch the last use nonce of account
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		panic(err)
	}
	fmt.Println("nounce=", nonce)
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		panic(err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		panic(err)
	}
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)      // in wei
	auth.GasLimit = uint64(3000000) // in units
	auth.GasPrice = big.NewInt(1000000)

	return auth
}

func initBsc() {
	StartedTime = time.Now().Unix() * 1000

	client, err := ethclient.Dial("wss://cold-alien-scion.bsc.discover.quiknode.pro/b80be7c1662c2485ee5d9508c442e0b79200afa7/")
	if err != nil {
		log.Fatal(err)
		logTelegram(err.Error())
	}

	headers := make(chan *types.Header)
	sub, err := client.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		log.Fatal(err)
		logTelegram(err.Error())
	}

	for {
		select {
		case err := <-sub.Err():
			log.Fatal(err)
			logTelegram(err.Error())
		case header := <-headers:
			// fmt.Println(header.Hash().Hex()) // 0xbc10defa8dda384c96a17640d84de5578804945d347072e091b4e5f390ddea7f

			block, err := client.BlockByNumber(context.Background(), header.Number)
			if err != nil {
				log.Println(err)
				logTelegram(err.Error())
			} else {
				for _, t := range block.Transactions() {
					ca := common.HexToAddress("0xa174E60Ef8b3b1FA7c71BB91d685191E915BaaED")
					if t.To() != nil && *t.To() == ca {
						contractABI, err := abi.JSON(strings.NewReader(GetLocalABI("./store.abi")))
						if err != nil {
							log.Fatal(err)
							logTelegram(err.Error())
						}

						log.Println(prettyPrint(t))

						addr := DecodeTransactionInputData(&contractABI, t.Data())
						log.Println(addr)

						sa := StoreAddress

						pricedb, err := getData2("%s__nodePrice", &sa)
						if err != nil {
							log.Fatal(err)
							logTelegram(err.Error())
						}

						tierdb, err := getData2("%s__nodeTier", &sa)
						if err != nil {
							log.Fatal(err)
							logTelegram(err.Error())
						}

						priceChanged := false
						price := new(big.Int).Mul(big.NewInt(10000000000000000), big.NewInt(pricedb.(int64)))
						val := t.Value()
						amountTotal := uint64(0)

						for val.Cmp(big.NewInt(0)) == 1 {
							bigamt := new(big.Int).Div(val, price)
							amount := bigamt.Uint64()

							if amount > uint64(tierdb.(int64)) {
								amount = uint64(tierdb.(int64))
								amountTotal += amount
								price = new(big.Int).Add(price, big.NewInt(10000000000000000))
								priceChanged = true
							} else {
								amountTotal += amount
							}

							valTier := new(big.Int).Mul(price, big.NewInt(int64(amount)))
							val = new(big.Int).Sub(val, valTier)
						}

						if priceChanged {
							newPrice := price.Int64()
							err := dataTransaction("%s__nodePrice", nil, &newPrice, nil)
							if err != nil {
								log.Println(err)
								logTelegram(err.Error())
							}
						}

						newTier := int64(0)

						if amountTotal > uint64(tierdb.(int64)) {
							newTier = (int64(amountTotal) - tierdb.(int64)) % 10
						} else {
							newTier = tierdb.(int64) - int64(amountTotal)
						}

						err = dataTransaction("%s__nodeTier", nil, &newTier, nil)
						if err != nil {
							log.Println(err)
							logTelegram(err.Error())
						}

						blockchain := "BSC"

						key := blockchain + Sep + t.Hash().String()
						data, err := getData(key)

						tdb := &Transaction{}
						db.First(t, &Transaction{TxID: t.Hash().String()})

						if err == nil && (data == nil || !data.(bool)) && tdb.ID == 0 && !tdb.Processed {
							if block.Time()*1000 > uint64(StartedTime) {
								// addr, amount := DecodeTransactionInputData(&contractABI, t.Data())
								// log.Println(block.Time())
								// log.Println(mon.StartedTime)
								if len(addr) > 0 && amountTotal > 0 && strings.HasPrefix(addr, "3A") {
									err := sendAsset(amountTotal, NodeTokenId, addr, t.Hash().String())
									if err == nil {
										done := true
										dataTransaction(key, nil, nil, &done)

										tdb.TxID = t.Hash().String()
										tdb.Processed = true
										tdb.Type = blockchain
										db.Save(tdb)
									}

									chainID, err := client.NetworkID(context.Background())
									if err != nil {
										log.Println(err)
										logTelegram(err.Error())
									}

									// m, err := t.AsMessage(types.NewEIP155Signer(chainID))
									// if err != nil {
									// 	log.Println(err)
									// 	logTelegram(err.Error())
									// }
									// sender := m.From().Hex()

									from, err := types.Sender(types.NewLondonSigner(chainID), t)
									if err != nil {
										fmt.Println(err) // 0x0fD081e3Bb178dc45c0cb23202069ddA57064258
										logTelegram(err.Error())
									}
									logTelegram(fmt.Sprintf("New NODE minted: %s %s %d", from.Hex(), addr, amountTotal))
								}
							}
						}
					}
				}
			}
		}
	}
}

func GetLocalABI(path string) string {
	abiFile, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer abiFile.Close()

	result, err := io.ReadAll(abiFile)
	if err != nil {
		log.Fatal(err)
	}
	return string(result)
}

func DecodeTransactionInputData(contractABI *abi.ABI, data []byte) string {
	addr := ""
	// The first 4 bytes of the t represent the ID of the method in the ABI
	// https://docs.soliditylang.org/en/v0.5.3/abi-spec.html#function-selector
	methodSigData := data[:4]
	method, err := contractABI.MethodById(methodSigData)
	if err != nil {
		log.Fatal(err)
	}

	inputsSigData := data[4:]
	inputsMap := make(map[string]interface{})
	if err := method.Inputs.UnpackIntoMap(inputsMap, inputsSigData); err != nil {
		log.Fatal(err)
	}

	if method.Name == "mintNode" {
		addr = inputsMap["addr"].(string)
	}

	return addr
}
