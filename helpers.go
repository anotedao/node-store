package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/mr-tron/base58/base58"
	"github.com/wavesplatform/gowaves/pkg/client"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func prettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}

func getCallerInfo() (info string) {

	// pc, file, lineNo, ok := runtime.Caller(2)
	_, file, lineNo, ok := runtime.Caller(2)
	if !ok {
		info = "runtime.Caller() failed"
		return
	}
	// funcName := runtime.FuncForPC(pc).Name()
	fileName := path.Base(file) // The Base function returns the last element of the path
	return fmt.Sprintf("%s:%d: ", fileName, lineNo)
}

func logTelegram(message string) {
	message = "store:" + getCallerInfo() + url.PathEscape(url.QueryEscape(message))

	_, err := http.Get(fmt.Sprintf("http://localhost:5006/log/%s", message))
	if err != nil {
		log.Println(err)
	}
}

func sendAsset(amount uint64, assetId string, recipient string, attachment string) error {
	var networkByte byte
	var nodeURL string
	// var assetBytes []byte

	networkByte = 55
	nodeURL = AnoteNodeURL

	// Create sender's public key from BASE58 string
	sender, err := crypto.NewPublicKeyFromBase58(conf.PublicKey)
	if err != nil {
		log.Println(err)
		logTelegram(err.Error())
		return err
	}

	// Create sender's private key from BASE58 string
	sk, err := crypto.NewSecretKeyFromBase58(conf.PrivateKey)
	if err != nil {
		log.Println(err)
		logTelegram(err.Error())
		return err
	}

	// Current time in milliseconds
	ts := time.Now().Unix() * 1000

	asset, err := proto.NewOptionalAssetFromString(assetId)
	if err != nil {
		log.Println(err)
		logTelegram(err.Error())
		return err
	}

	assetW, err := proto.NewOptionalAssetFromString("")
	if err != nil {
		log.Println(err)
		logTelegram(err.Error())
		return err
	}

	rec, err := proto.NewRecipientFromString(recipient)
	if err != nil {
		log.Println(err)
		logTelegram(err.Error())
		return err
	}

	att, err := proto.NewAttachmentFromBase58(base58.Encode([]byte(attachment)))
	if err != nil {
		log.Println(err)
		logTelegram(err.Error())
		return err
	}

	tr := proto.NewUnsignedTransferWithSig(sender, *asset, *assetW, uint64(ts), amount, Fee, rec, att)

	err = tr.Sign(networkByte, sk)
	if err != nil {
		log.Println(err)
		logTelegram(err.Error())
		return err
	}

	// Create new HTTP client to send the transaction to public TestNet nodes
	client, err := client.NewClient(client.Options{BaseUrl: nodeURL, Client: &http.Client{}})
	if err != nil {
		log.Println(err)
		logTelegram(err.Error())
		return err
	}

	// Context to cancel the request execution on timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// // Send the transaction to the network
	_, err = client.Transactions.Broadcast(ctx, tr)
	if err != nil {
		log.Println(err)
		logTelegram(err.Error())
		return err
	}

	return nil
}

func getData(key string) (interface{}, error) {
	dkr, err := anc.AddressesDataKey(anoteAddress, key)
	if err != nil {
		if strings.Contains(fmt.Sprintf("%s", err.Error()), "304") {
			return nil, nil
		} else {
			return nil, err
		}
	}
	return dkr.Value, nil
}

func getData2(key string, address *string) (interface{}, error) {
	var a proto.WavesAddress

	wc, err := client.NewClient(client.Options{BaseUrl: AnoteNodeURL, Client: &http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2: true,
			// MaxConnsPerHost:   -1,
			MaxIdleConnsPerHost: -1,
			DisableKeepAlives:   true,
		},
	}})
	if err != nil {
		log.Println(err)
		logTelegram(err.Error())
	}

	if address == nil {
		pk, err := crypto.NewPublicKeyFromBase58(conf.PublicKey)
		if err != nil {
			return nil, err
		}

		a, err = proto.NewAddressFromPublicKey(55, pk)
		if err != nil {
			return nil, err
		}
	} else {
		a, err = proto.NewAddressFromString(*address)
		if err != nil {
			return nil, err
		}
	}

	ad, _, err := wc.Addresses.AddressesDataKey(context.Background(), a, key)
	if err != nil {
		return nil, err
	}

	if ad.GetValueType().String() == "string" {
		return ad.ToProtobuf().GetStringValue(), nil
	}

	if ad.GetValueType().String() == "boolean" {
		return ad.ToProtobuf().GetBoolValue(), nil
	}

	if ad.GetValueType().String() == "integer" {
		return ad.ToProtobuf().GetIntValue(), nil
	}

	return "", nil
}

func dataTransaction(key string, valueStr *string, valueInt *int64, valueBool *bool) error {
	// Create sender's public key from BASE58 string
	sender, err := crypto.NewPublicKeyFromBase58(conf.PublicKey)
	if err != nil {
		log.Println(err)
		logTelegram(err.Error())
		return err
	}

	// Create sender's private key from BASE58 string
	sk, err := crypto.NewSecretKeyFromBase58(conf.PrivateKey)
	if err != nil {
		log.Println(err)
		logTelegram(err.Error())
		return err
	}

	// Current time in milliseconds
	ts := time.Now().Unix() * 1000

	// tr := proto.NewUnsignedData(2, sender, Fee, uint64(ts))
	tr := proto.NewUnsignedDataWithProofs(2, sender, Fee, uint64(ts))

	if valueStr == nil && valueInt == nil && valueBool == nil {
		tr.Entries = append(tr.Entries,
			&proto.DeleteDataEntry{
				Key: key,
			},
		)
	}

	if valueStr != nil {
		tr.Entries = append(tr.Entries,
			&proto.StringDataEntry{
				Key:   key,
				Value: *valueStr,
			},
		)
	}

	if valueInt != nil {
		tr.Entries = append(tr.Entries,
			&proto.IntegerDataEntry{
				Key:   key,
				Value: *valueInt,
			},
		)
	}

	if valueBool != nil {
		tr.Entries = append(tr.Entries,
			&proto.BooleanDataEntry{
				Key:   key,
				Value: *valueBool,
			},
		)
	}

	err = tr.Sign(55, sk)
	if err != nil {
		log.Println(err)
		logTelegram(err.Error())
		return err
	}

	// Create new HTTP client to send the transaction to public TestNet nodes
	cl, err := client.NewClient(client.Options{BaseUrl: AnoteNodeURL, Client: &http.Client{}})
	if err != nil {
		log.Println(err)
		logTelegram(err.Error())
		return err
	}

	// Context to cancel the request execution on timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// // Send the transaction to the network
	_, err = cl.Transactions.Broadcast(ctx, tr)
	if err != nil {
		log.Println(err)
		logTelegram(err.Error())
		return err
	}

	return nil
}
