package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

var current_date int64

func main() {
	current_date = time.Now().UnixMilli()
	var mu ControlConcurrency
	mu.PersonalWallet.MintQuantityHashMap = make(map[interface{}]float64)

	api_key, err := get_api_key()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	apis_obj, err := get_apis_obj()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	wallet_obj, err := get_wallet_obj()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	dex_wallets := DexWallets{}
	for _, wallet := range wallet_obj["dex"].([]interface{}) {
		dex_wallets.DexArray = append(dex_wallets.DexArray, wallet.(string))
	}

	endpoint := apis_obj["url"].(map[string]interface{})["instantnodes"].(string) + api_key
	mu.Client = rpc.New(endpoint)
	pubKeyExternalWallet := solana.MustPublicKeyFromBase58(wallet_obj["external_wallet"].(string))
	personalKeyExternalWallet := solana.MustPublicKeyFromBase58(wallet_obj["personal_wallet"].(string)) //change to personal
	err = mu.get_balance(mu.Client, pubKeyExternalWallet, personalKeyExternalWallet)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	//First time getting txs to get latest signature and start from there
	signature_starting_point, err := get_starting_point(pubKeyExternalWallet, api_key, current_date, mu.Client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(signature_starting_point)

	//infinite loop checking for new transactions
	for {
		available, tx_available, new_starting_point := check_new_tx_available(pubKeyExternalWallet, api_key, current_date, mu.Client, signature_starting_point, dex_wallets)
		signature_starting_point = new_starting_point
		time.Sleep(time.Second * 8)
		if available {
			mu.Mu.Lock()
			//go thread to process transaction
			go mu.replicate_transaction(tx_available, pubKeyExternalWallet, personalKeyExternalWallet)
		}
	}

}
