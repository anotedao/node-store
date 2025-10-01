package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Monitor struct {
	balance *big.Int
}

func (m *Monitor) monitorBsc() {
	for {
		m.checkBsc()

		time.Sleep(time.Second * MonitorTick)
	}
}

func (m *Monitor) checkBsc() {
	// client, err := ethclient.Dial("wss://bsc-mainnet.core.chainstack.com/916554e2d2df25f9f4dc8a6b35e5735f")
	client, err := ethclient.Dial("wss://ethereum-hoodi.core.chainstack.com/26c6784a0f915a9e597a418b0643ada4")
	if err != nil {
		log.Println(err)
	}

	header, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}

	// height := header.Number.Int64()

	owner, err := common.NewMixedcaseAddressFromString(ContractOwner)
	if err != nil {
		log.Fatal(err)
	}

	balance, err := client.BalanceAt(context.Background(), owner.Address(), header.Number)
	if err != nil {
		log.Fatal(err)
	}

	if m.balance != nil && balance.Cmp(m.balance) == 1 {
		for i := 0; i < 100; i++ {
			ibig := big.NewInt(int64(i))
			chbig := new(big.Int)
			chbig.Sub(header.Number, ibig)
			block, err := client.BlockByNumber(context.Background(), chbig)
			if err != nil {
				log.Println(err)
			}

			trs := block.Transactions()

			log.Println(chbig.String())

			for _, t := range trs {
				ca := common.HexToAddress(ContractTestnet)
				if t.To() != nil && *t.To() == ca {
					contractABI, err := abi.JSON(strings.NewReader(GetLocalABI("./store.abi")))
					if err != nil {
						log.Fatal(err)
						logTelegram(err.Error())

						contractABI, err = abi.JSON(strings.NewReader(GetLocalABI("/persistent/store.abi")))
						if err != nil {
							log.Println(err)
							logTelegram(err.Error())
						}
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

					bigamt := new(big.Int).Div(val, price)
					amount := bigamt.Uint64()
					tier := uint64(tierdb.(int64))

					if amount > tier {
						valTier := big.NewInt(1)
						for val.Cmp(big.NewInt(0)) == 1 && valTier.Cmp(big.NewInt(0)) == 1 {
							// bigamt := new(big.Int).Div(val, price)
							amount = tier
							amountTotal += amount

							valTier = new(big.Int).Mul(price, big.NewInt(int64(amount)))
							val = new(big.Int).Sub(val, valTier)

							if val.Cmp(big.NewInt(0)) == 1 {
								price = new(big.Int).Add(price, big.NewInt(10000000000000000))
								priceChanged = true
							} else {
								val = big.NewInt(0)
							}

							tier = new(big.Int).Div(val, price).Uint64()

							log.Println(valTier.String())
							log.Println(val.String())
							log.Println(price.String())
							log.Println(amount)
						}
					} else {
						amountTotal += amount
					}

					newTier := int64(0)

					if amountTotal > uint64(tierdb.(int64)) {
						newTier = 10 - int64(amount)
					} else {
						newTier = tierdb.(int64) - int64(amountTotal)
					}

					if newTier == 0 {
						newTier = 10
						if !priceChanged {
							price = new(big.Int).Add(price, big.NewInt(10000000000000000))
							priceChanged = true
						}
					}

					err = dataTransaction("%s__nodeTier", nil, &newTier, nil)
					if err != nil {
						log.Println(err)
						logTelegram(err.Error())
					}

					if priceChanged {
						newPrice := new(big.Int).Div(price, big.NewInt(10000000000000000)).Int64()
						err := dataTransaction("%s__nodePrice", nil, &newPrice, nil)
						if err != nil {
							log.Println(err)
							logTelegram(err.Error())
						}
					}

					blockchain := "BSC"

					key := blockchain + Sep + t.Hash().String()
					data, err := getData(key)

					tdb := &Transaction{}
					db.First(tdb, &Transaction{TxID: t.Hash().String()})

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

						tdb = nil
					}
				}
			}

			time.Sleep(50 * time.Millisecond)
		}
	}

	m.balance = balance
}

func (m *Monitor) start() {
	m.monitorBsc()
}

func initMonitor() {
	m := &Monitor{}
	m.start()
}
