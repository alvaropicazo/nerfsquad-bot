package main

import (
	"flag"
	"os"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/rs/zerolog"
)

func main() {
	var ns NSReceiver
	var platform string
	var help bool
	log := zerolog.New(
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339},
	).Level(zerolog.TraceLevel).With().Timestamp().Caller().Logger()

	if len(os.Args) == 0 {
		ns.Log.Error().Msg("Arguments not passed")
		os.Exit(1)
	}
	flag.BoolVar(&help, "help", false, "Help")
	flag.StringVar(&platform, "p", "raydium", "Provide a platform to be tracked and used")
	flag.Parse()
	if help {
		flag.PrintDefaults()
		os.Exit(1)
	}

	//Initializations
	ns.PersonalWallet.MintQuantityHashMap = make(map[solana.PublicKey]float64)
	ns.PersonalWallet.TokenAccountHashMap = make(map[solana.PublicKey]solana.PublicKey)
	ns.ExternalWallet.MintQuantityHashMap = make(map[solana.PublicKey]float64)
	ns.ExternalWallet.TokenAccountHashMap = make(map[solana.PublicKey]solana.PublicKey)

	ns.Log = log

	keys_obj, err := get_api_key()
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

	// instantnodes_key := keys_obj["keys"].(map[string]interface{})["instantnodes"].(string)
	helius_key := keys_obj["keys"].(map[string]interface{})["helius"].(string)
	coingecko_key := keys_obj["keys"].(map[string]interface{})["coingecko"].(string)
	// endpoint := apis_obj["url"].(map[string]interface{})["instantnodes"].(string) + instantnodes_key
	helius_endpoint := apis_obj["url"].(map[string]interface{})["helius"].(string) + helius_key
	send_transactions_api_url := apis_obj["url"].(map[string]interface{})["send_transactions_app"].(string)
	ns.Client = rpc.New(helius_endpoint)
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

	ticker := time.NewTicker(20 * time.Minute)

	// Use a goroutine to run the function periodically to keep updating SOL price
	go func() {
		for range ticker.C { // Channel receives a tick every 20 minutes
			err := ns.get_current_solana_price(coingecko_key)
			if err != nil {
				ns.Log.Error().Msg(err.Error())
			}
		}
	}()

	//First time getting txs to get latest signature and start from there
	signature_starting_point, err := ns.get_starting_point(pubKeyExternalWallet)
	if err != nil {
		ns.Log.Error().Msg(err.Error())
		os.Exit(1)
	}
	// signature_starting_point = solana.MustSignatureFromBase58("EJaE5UndjBWFDuFViqxjM2YbGoGC1DnbMosGicenM6SWUwMLGJ2aiNhn7wpGiKAu9rrGNjFcWpAKgCPnPQGsC74")

	//Slippage
	ns.Slippage = 0.2
	ns.Log.Debug().Msg("Initial starting point: " + signature_starting_point.String())
	//infinite loop checking for new transactions
	for {
		available, tx_available, new_starting_point, _ := ns.check_new_tx_available(pubKeyExternalWallet, signature_starting_point, dex_wallets, platform)
		signature_starting_point = new_starting_point
		if available {
			ns.Log.Info().Msg("Available transactions found")
			ns.Mu.Lock()
			ns.Log.Info().Msg("Running thread to replicate transactions")
			//go thread to process transaction
			go ns.replicate_transaction(tx_available, pubKeyExternalWallet, personalKeyWallet, send_transactions_api_url, platform)
		}
		time.Sleep(time.Second * 3)
	}
}
