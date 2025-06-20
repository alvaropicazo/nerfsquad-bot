package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/enescakir/emoji"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

var zero uint64

// main function which calls submethods to recreate tx
func (ns *NSReceiver) replicate_transaction(tx_available []TransactionFormatted, pubKeyExternalWallet solana.PublicKey, personalKeyWallet solana.PublicKey, send_transactions_api_url string) error {

	res, err := ns.format_data(tx_available, pubKeyExternalWallet, personalKeyWallet)
	if err != nil {
		return err
	}

	client := &http.Client{}

	for _, tx_to_send := range res {
		var txResponse TxResponse
		//Call external component to interact with raydium
		bytes_slice, err := json.Marshal(tx_to_send)
		if err != nil {
			return err
		}
		r := bytes.NewReader(bytes_slice)
		ns.Log.Info().Msg("Request to Send Transaction Server")
		req, err := http.NewRequest(http.MethodPost, send_transactions_api_url, r)
		if err != nil {
			ns.Log.Error().Msg(err.Error())
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}

		defer resp.Body.Close()
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		err = json.Unmarshal(responseBody, &txResponse)
		if err != nil {
			ns.Log.Error().Msg(err.Error())
		}

		icon := ""
		if tx_to_send.Type == "BUY" {
			icon = string(emoji.GreenCircle) + " "
		} else {
			icon = string(emoji.RedCircle) + " "
		}

		//Send transaction result data to Telegram group chat
		ns.send_telegram_updates(icon + txResponse.Message)
	}

	//Update balance from external wallet and our wallet
	err = ns.get_balance(ns.Client, pubKeyExternalWallet, personalKeyWallet)
	if err != nil {
		ns.Log.Error().Msg(err.Error())
		os.Exit(1)
	}
	//Freeing channel so other thread can insert new txs to replicate
	ns.Mu.Unlock()
	return nil
}

// Checks new available transaction with a starting point given. If new available transactions, creates an object that will be used
// by replicate_transaction() in order to send it to the external service component to interact with Raydium
func (ns *NSReceiver) check_new_tx_available(wallet_address solana.PublicKey, api_token string, sig solana.Signature, dex_wallets DexWallets) (bool, []TransactionFormatted, solana.Signature, error) {
	var result []TransactionFormatted

	//Get newer signatures which are the ones to be tracked, if any.
	out_new_sigs, err := ns.Client.GetSignaturesForAddress(
		context.TODO(),
		wallet_address,
		&rpc.GetSignaturesForAddressOpts{Until: sig, Commitment: "processed"}, //replace with signature param
	)
	if err != nil {
		ns.Log.Error().Msg(err.Error())
		return false, nil, sig, err
	}

	signatures := []solana.Signature{}
	left := 0
	right := len(out_new_sigs) - 1
	for left < right {
		out_new_sigs[left], out_new_sigs[right] = out_new_sigs[right], out_new_sigs[left]
		left++
		right--
	}

	for _, new_sig := range out_new_sigs {
		signatures = append(signatures, new_sig.Signature)
	}

	check_point := sig
	for i, signature := range signatures {
		out, err := ns.Client.GetTransaction(
			context.TODO(),
			signature,
			&rpc.GetTransactionOpts{
				MaxSupportedTransactionVersion: &zero,
			},
		)
		if err != nil {
			ns.Log.Info().Msg("Failed due to request limits, checkpoint is: " + check_point.String())
			marsh, _ := json.Marshal(result)
			ns.Log.Info().Msg("Transactions available (should be more as we are returning due to request limit error): " + string(marsh))
			ns.Log.Error().Msg(err.Error())
			if i == 0 {
				return false, result, check_point, err
			} else {
				if len(result) != 0 {
					return true, result, check_point, err
				} else {
					return false, result, check_point, err
				}
			}
		}

		check := false
		operation := ""

		sol_tx_pre := 0.0  //what the DEX had before the tx
		sol_tx_pos := 0.0  //what the DEX had after the tx
		mint_tx_pre := 0.0 //amount of mint owned by external wallet before the tx
		mint_tx_pos := 0.0 // amount of mint owned by the external wallet after the tx
		wallet_dex_found := ""
		preTransactions := []rpc.TokenBalance{}
		postTransactions := []rpc.TokenBalance{}
		programId := solana.PublicKey{}

		//For checking if it is a buy or a sell

		//Enhance
		//Make sure first if this array contains txs made to one of our targeted dex
		for _, tx_pre := range out.Meta.PreTokenBalances {
			for _, wallet := range dex_wallets.DexArray {
				if tx_pre.Owner.ToPointer().String() == wallet {
					amount, _ := strconv.ParseFloat(tx_pre.UiTokenAmount.UiAmountString, 64)
					if amount > 0.0 {
						wallet_dex_found = wallet
						if tx_pre.Mint.ToPointer().String() == "So11111111111111111111111111111111111111112" {
							sol_tx_pre = amount
							check = true
							break
						}
					}
				}
			}
			if check {
				break
			}
		}

		if check {
			for _, tx_pre := range out.Meta.PreTokenBalances {
				if *tx_pre.Owner == wallet_address {
					preTransactions = append(preTransactions, tx_pre)
					amount, _ := strconv.ParseFloat(tx_pre.UiTokenAmount.UiAmountString, 64)
					mint_tx_pre = amount
					programId = *tx_pre.ProgramId
				}
			}

			for _, tx_post := range out.Meta.PostTokenBalances {
				if *tx_post.Owner == wallet_address {
					postTransactions = append(postTransactions, tx_post)
					amount, _ := strconv.ParseFloat(tx_post.UiTokenAmount.UiAmountString, 64)
					mint_tx_pos = amount
				} else if *tx_post.Owner == solana.MustPublicKeyFromBase58(wallet_dex_found) {
					amount, _ := strconv.ParseFloat(tx_post.UiTokenAmount.UiAmountString, 64)
					if tx_post.Mint.ToPointer().String() == "So11111111111111111111111111111111111111112" {
						sol_tx_pos = amount
						if sol_tx_pos > sol_tx_pre {
							operation = "BUY" //DEX has more SOL, which means the external wallet has bought something.
						} else {
							operation = "SELL"
						}
					}
				}
			}
		}

		if len(preTransactions) > 0 || len(postTransactions) > 0 {
			transaction_formated := TransactionFormatted{}
			transaction_formated.MintName = postTransactions[0].Mint
			transaction_formated.Type = operation
			if operation == "BUY" {
				amount_obtained := mint_tx_pos - mint_tx_pre
				sol_spent := sol_tx_pos - sol_tx_pre
				transaction_formated.MintAmount = amount_obtained
				transaction_formated.SolAmount = sol_spent
				transaction_formated.ProgramId = programId
			} else if operation == "SELL" {
				//Important to keep pre_mint to apply percentages if the external wallet has not sold 100% of the token
				transaction_formated.MintPre = mint_tx_pre
				amount_obtained := mint_tx_pre - mint_tx_pos
				sol_obtained := sol_tx_pre - sol_tx_pos
				transaction_formated.MintAmount = amount_obtained
				transaction_formated.SolAmount = sol_obtained
				transaction_formated.MintPre = mint_tx_pre
				transaction_formated.ProgramId = programId
			}
			if transaction_formated.MintAmount == 0 && transaction_formated.SolAmount == 0 && transaction_formated.MintPre == 0 {
				continue
			} else {
				result = append(result, transaction_formated)
			}
		}
		check_point = signature
	}
	if len(result) != 0 {
		marsh, _ := json.Marshal(result)
		ns.Log.Info().Msg("Transactions available: " + string(marsh))
		return true, result, check_point, nil
	} else {
		return false, result, check_point, nil
	}
}
