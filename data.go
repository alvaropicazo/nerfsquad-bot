package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/davecgh/go-spew/spew"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"gopkg.in/yaml.v3"
)

// Calculates the quantity of x token that should be transferred according to our balance.
func (ns *NSReceiver) format_data(tx_available []TransactionFormatted, pubKeyExternalWallet solana.PublicKey, personalKeyWallet solana.PublicKey) ([]TransactionToSend, error) {
	configs := []retry.Option{
		retry.Attempts(uint(1)),
		retry.OnRetry(func(n uint, err error) {
			log.Printf("Retry request %d to and get error: %v", n+1, err)
		}),
		retry.Delay(time.Second),
	}
	res := []TransactionToSend{}
	for _, tx := range tx_available {
		tx_to_send := TransactionToSend{}
		if tx.Type == "BUY" {
			if ns.PersonalWallet.PersonalBalance < 0.035 { //0.035 is the min amount to spend to be able to execute tx successfully
				str, _ := json.Marshal(ns.PersonalWallet.PersonalBalance)
				ns.send_telegram_updates("Tried to perform a BUY operation. Insufficient balance. Add more SOL. Current balance: " + string(str))
				continue
			}
			total := ns.ExternalWallet.PersonalBalance
			percentage_external := tx.SolAmount / total
			sol_to_spend := math.Max(ns.PersonalWallet.PersonalBalance*percentage_external, 0.035)
			mint_to_buy := tx.MintAmount * sol_to_spend / tx.SolAmount
			tx_to_send.MintAmount = mint_to_buy
			tx_to_send.Slippage = ns.Slippage //*sol_to_spend + sol_to_spend
			tx_to_send.SolAmount = convert_num(sol_to_spend)
			tx_to_send.TokenAccountPersonal = *solana.WrappedSol.ToPointer()
			tx_to_send.TokenAccountExternal = ns.ExternalWallet.TokenAccountHashMap[tx.MintName] //TODO: Remove it
		} else if tx.Type == "SELL" {
			err := retry.Do(
				func() error {
					err := ns.get_token_account_for_specific_mint(personalKeyWallet, tx.MintName.ToPointer(), true)
					if err != nil {
						ns.Log.Error().Msg(err.Error())
						return err
					}
					return nil
				},
				configs...,
			)
			if err != nil {
				ns.Log.Error().Msg(err.Error())
				return nil, err
			}
			if ns.PersonalWallet.MintQuantityHashMap[tx.MintName] == 0.0 {
				continue //we skip as we dont have anything to sell
			}
			//percentage_to_sell := tx.MintAmount / tx.MintPre //If its 1, all stake was sold for that token.
			mint_to_sell := float64(ns.PersonalWallet.MintQuantityHashMap[tx.MintName]) //* percentage_to_sell
			sol_to_receive := mint_to_sell * tx.SolAmount / tx.MintAmount
			tx_to_send.MintAmount = convert_num(mint_to_sell)
			tx_to_send.SolAmount = convert_num(sol_to_receive)
			if tx_to_send.SolAmount < 0.025 {
				ns.Log.Info().Msg("Due to low rates, half of the stake will be sold")
				mint_to_sell = float64(ns.PersonalWallet.MintQuantityHashMap[tx.MintName]) //* 0.6 //we sell 0.6 of it due to low numbers as we need to make sure tx are submitted correctly
				sol_to_receive = mint_to_sell * tx.SolAmount / tx.MintAmount
				tx_to_send.MintAmount = convert_num(mint_to_sell)
				tx_to_send.SolAmount = convert_num(sol_to_receive)
			}
			tx_to_send.Slippage = ns.Slippage
			tx_to_send.TokenAccountPersonal = ns.PersonalWallet.TokenAccountHashMap[tx.MintName]
			tx_to_send.TokenAccountExternal = *solana.WrappedSol.ToPointer()
		}
		tx_to_send.Type = tx.Type
		tx_to_send.ProgramId = tx.ProgramId
		tx_to_send.MintName = tx.MintName
		tx_to_send.CurrentPrice = ns.SolPrice
		res = append(res, tx_to_send)
	}
	marsh, _ := json.Marshal(res)
	ns.Log.Info().Msg("Transactions that will be sent: " + string(marsh))
	return res, nil
}

// Retrieves the keys to be used for the apis.
func get_api_key() (map[string]interface{}, error) {
	keys_file := make(map[string]interface{})

	keys_byte, err := os.ReadFile("./config/keys.yaml")
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(keys_byte, keys_file)
	if err != nil {
		return nil, err
	}

	return keys_file, nil
}

// Retrieves the apis that will be used to get data about the wallet to track / dex to send transactions.
func get_apis_obj() (map[string]interface{}, error) {
	apis_file := make(map[string]interface{})

	apis_byte, err := os.ReadFile("./config/apis.yaml")
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(apis_byte, apis_file)
	if err != nil {
		return nil, err
	}

	return apis_file, nil
}

// Retrieves the wallet that will be replicated and dex's wallets to filter transactions.
func get_wallet_obj() (map[interface{}]interface{}, error) {
	wallets_file := make(map[interface{}]interface{})

	wallets_byte, err := os.ReadFile("./config/wallets.yaml")
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(wallets_byte, wallets_file)
	if err != nil {
		return nil, err
	}

	return wallets_file, nil

}

// Retrieves the wallet that will be replicated and dex's wallets to filter transactions.
func get_personal_wallet_obj() (map[interface{}]interface{}, error) {
	wallets_file := make(map[interface{}]interface{})

	wallets_byte, err := os.ReadFile("./config/cryptokeys.yaml")
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(wallets_byte, wallets_file)
	if err != nil {
		return nil, err
	}

	return wallets_file, nil

}

// Gets the signature as a starting point, any transaction newer to it should be checked.
func (ns *NSReceiver) get_starting_point(wallet_address solana.PublicKey, api_token string, current_date int64) (solana.Signature, error) {
	out, err := ns.Client.GetSignaturesForAddress(
		context.TODO(),
		wallet_address,
		nil,
	)
	if err != nil {
		return solana.SignatureFromBytes(nil), err
	}

	if len(out) > 0 {
		return out[0].Signature, nil
	} else {
		return solana.SignatureFromBytes(nil), err
	}
}

// Retrieves SOL balance and mints of tokens owned by the tracked wallet.
func (ns *NSReceiver) get_balance(client *rpc.Client, external_wallet_address solana.PublicKey, personal_wallet_address solana.PublicKey) error {
	out_sol, err := client.GetBalance(
		context.TODO(),
		external_wallet_address,
		rpc.CommitmentFinalized,
	)
	if err != nil {
		return err
	}

	out_sol_personal, err := client.GetBalance(
		context.TODO(),
		personal_wallet_address,
		rpc.CommitmentFinalized,
	)
	if err != nil {
		return err
	}
	//Get SOL from our wallet and from the external wallet being tracked
	ns.PersonalWallet.PersonalBalance = float64(float64(out_sol_personal.Value) / 1000000000)
	ns.ExternalWallet.PersonalBalance = float64(float64(out_sol.Value) / 1000000000)

	val, _ := json.Marshal(ns.PersonalWallet)
	val2, _ := json.Marshal(ns.ExternalWallet)
	ns.Log.Debug().Msg(string(val))
	ns.Log.Debug().Msg(string(val2))

	return nil
}

// Gets the different token accounts from the tracked wallet and amount for a specific mint.
func (ns *NSReceiver) get_token_account_for_specific_mint(pubKey solana.PublicKey, mint *solana.PublicKey, ourWallet bool) error {
	out_mint, err := ns.Client.GetTokenAccountsByOwner(
		context.TODO(),
		pubKey,
		&rpc.GetTokenAccountsConfig{
			Mint: mint,
		},
		&rpc.GetTokenAccountsOpts{
			Encoding: solana.EncodingBase64Zstd,
		},
	)
	if err != nil {
		return err
	}
	// spew.Dump(out_mint)
	if len(out_mint.Value) == 0 {
		ns.Log.Info().Msg("No tokens available for account")
		return nil
	}

	out_tokenbalance, err := ns.Client.GetTokenAccountBalance(
		context.TODO(),
		out_mint.Value[0].Pubkey,
		rpc.CommitmentFinalized,
	)
	if err != nil {
		return err
	}

	if ourWallet {
		ns.PersonalWallet.MintQuantityHashMap[*mint] = float64(float64(*out_tokenbalance.Value.UiAmount))
		ns.PersonalWallet.TokenAccountHashMap[*mint] = out_mint.Value[0].Pubkey
		val, _ := json.Marshal(ns.PersonalWallet)
		ns.Log.Debug().Msg(string(val))
	} else {
		ns.ExternalWallet.MintQuantityHashMap[*mint] = float64(float64(*out_tokenbalance.Value.UiAmount))
		ns.ExternalWallet.TokenAccountHashMap[*mint] = out_mint.Value[0].Pubkey
		val2, _ := json.Marshal(ns.ExternalWallet)
		ns.Log.Debug().Msg(string(val2))
	}

	return nil
}

// Retrieves Token Supply for specific token.
func (ns *NSReceiver) get_token_supply(token_address solana.PublicKey) (float64, error) {
	out_sol, err := ns.Client.GetTokenSupply(
		context.TODO(),
		token_address,
		rpc.CommitmentFinalized,
	)
	if err != nil {
		return 0.0, err
	}

	token_supply := *out_sol.Value.UiAmount

	return token_supply, nil
}

func convert_num(number float64) float64 {
	decimalPlaces := fmt.Sprintf("%f", number-math.Floor(number)) // produces 0.xxxx0000
	step, _ := strconv.ParseFloat(decimalPlaces, 64)
	num := number - step
	extra := math.RoundToEven(num)
	finale := extra + step
	return finale
}

// Retrieves Token Supply for specific token.
func (ns *NSReceiver) get_current_solana_price(key string) error {

	url := "https://api.coingecko.com/api/v3/simple/price?ids=solana&vs_currencies=usd"

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("accept", "application/json")
	req.Header.Add("x-cg-demo-api-key", key)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var tokenInfo TokenInfo

	err = json.Unmarshal(body, &tokenInfo)
	if err != nil {
		return err
	}
	if tokenInfo.Solana["usd"] == float64(0) {
		return nil
	}
	ns.SolPrice = tokenInfo.Solana["usd"]

	return nil
}

// Retrieves Program accounts for specific DEX.
func (ns *NSReceiver) get_program_accounts(dex_address solana.PublicKey) error {
	out_sol, err := ns.Client.GetProgramAccounts(
		context.TODO(),
		dex_address,
	)
	if err != nil {
		return err
	}

	program_accounts := out_sol
	spew.Dump(program_accounts)

	return nil
}
