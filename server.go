package main

import (
	"os"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/rs/zerolog"
)

var current_date int64

func main() {
	current_date = time.Now().UnixMilli()
	var ns NSReceiver

	log := zerolog.New(
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339},
	).Level(zerolog.TraceLevel).With().Timestamp().Caller().Logger()

	//Initializations
	ns.PersonalWallet.MintQuantityHashMap = make(map[solana.PublicKey]float64)
	ns.PersonalWallet.TokenAccountHashMap = make(map[solana.PublicKey]solana.PublicKey)
	ns.ExternalWallet.MintQuantityHashMap = make(map[solana.PublicKey]float64)
	ns.ExternalWallet.TokenAccountHashMap = make(map[solana.PublicKey]solana.PublicKey)

	ns.Log = log

	api_key, err := get_api_key()
	if err != nil {
		ns.Log.Error().Msg(err.Error())
		os.Exit(1)
	}

	apis_obj, err := get_apis_obj()
	if err != nil {
		ns.Log.Error().Msg(err.Error())
		os.Exit(1)
	}

	wallet_obj, err := get_wallet_obj()
	if err != nil {
		ns.Log.Error().Msg(err.Error())
		os.Exit(1)
	}

	personal_wallet_obj, err := get_personal_wallet_obj()
	if err != nil {
		ns.Log.Error().Msg(err.Error())
		os.Exit(1)
	}

	dex_wallets := DexWallets{}
	for _, wallet := range wallet_obj["dex"].([]interface{}) {
		dex_wallets.DexArray = append(dex_wallets.DexArray, wallet.(string))
	}

	endpoint := apis_obj["url"].(map[string]interface{})["instantnodes"].(string) + api_key
	send_transactions_api_url := apis_obj["url"].(map[string]interface{})["send_transactions_app"].(string)
	ns.Client = rpc.New(endpoint)
	pubKeyExternalWallet := solana.MustPublicKeyFromBase58(wallet_obj["external_wallet"].(string))
	personalKeyWallet := solana.MustPublicKeyFromBase58(personal_wallet_obj["personal_wallet"].(string))

	ns.Log.Debug().Msg("Data retrieved from configuration files")
	ns.Log.Debug().Msg("External wallet: " + pubKeyExternalWallet.String())
	ns.Log.Debug().Msg("Personal wallet: " + personalKeyWallet.String())

	err = ns.get_balance(ns.Client, pubKeyExternalWallet, personalKeyWallet)
	if err != nil {
		ns.Log.Error().Msg(err.Error())
		os.Exit(1)
	}

	//First time getting txs to get latest signature and start from there
	signature_starting_point, err := ns.get_starting_point(pubKeyExternalWallet, api_key, current_date)
	if err != nil {
		ns.Log.Error().Msg(err.Error())
		os.Exit(1)
	}
	// signature_starting_point = solana.MustSignatureFromBase58("rQnbV5JiKvXv34GDpqAeYeCRZrqqm2xBujd5DfwNxSufRzfewvr3jGCSvH4czapJMoEN4PpAFMDooMB33J6aR5d")

	//Slippage
	slippage := 0.1
	ns.Log.Debug().Msg("Initial starting point: " + signature_starting_point.String())
	//infinite loop checking for new transactions
	for {
		ns.Log.Info().Msg("Starting to check new transactions available")
		available, tx_available, new_starting_point, _ := ns.check_new_tx_available(pubKeyExternalWallet, api_key, signature_starting_point, dex_wallets)
		signature_starting_point = new_starting_point
		if available {
			ns.Log.Info().Msg("Available transactions found")
			ns.Mu.Lock()
			ns.Log.Info().Msg("Running thread to replicate transactions")
			//go thread to process transaction
			go ns.replicate_transaction(tx_available, pubKeyExternalWallet, personalKeyWallet, send_transactions_api_url, slippage)
		}
		time.Sleep(time.Second * 20)
	}

}
