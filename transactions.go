package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// main function which calls submethods to recreate tx
func (m *ControlConcurrency) replicate_transaction(tx_available []TransactionFormatted, pubKeyExternalWallet solana.PublicKey, personalKeyExternalWallet solana.PublicKey) error {

	_, err := m.format_data(tx_available)
	if err != nil {
		return err
	}

	//Use Raydium API to send data
	//Update balance from external wallet and our wallet
	err = m.get_balance(m.Client, pubKeyExternalWallet, personalKeyExternalWallet)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	//Freeing channel so other thread can insert new txs to replicate
	m.Mu.Unlock()
	return nil
}

var zero uint64

func check_new_tx_available(wallet_address solana.PublicKey, api_token string, current_date int64, client *rpc.Client, signature solana.Signature, dex_wallets DexWallets) (bool, []TransactionFormatted, solana.Signature) {
	var result []TransactionFormatted
	var new_starting_point solana.Signature
	sig := solana.MustSignatureFromBase58("w32Vjwskr8oAYakEpABqQBcBDyTPwiSZMF9z42j8QFg7VjLMKCbSfknyRvGvBxccYm9S22a8fFU6fAXZStRhdqy")

	//Get newer signatures which are the ones to be tracked, if any.
	out_new_sigs, err := client.GetSignaturesForAddress(
		context.TODO(),
		wallet_address,
		&rpc.GetSignaturesForAddressOpts{Until: sig},
	)
	if err != nil {
		panic(err)
	}

	signatures := []solana.Signature{}
	for i, new_sig := range out_new_sigs {
		if i == 0 {
			new_starting_point = new_sig.Signature
		}
		signatures = append(signatures, new_sig.Signature)
	}
	fmt.Println("SIGNATURES:")
	fmt.Println(len(signatures))
	for _, signature := range signatures {
		out, err := client.GetTransaction(
			context.TODO(),
			signature,
			&rpc.GetTransactionOpts{
				MaxSupportedTransactionVersion: &zero,
			},
		)
		if err != nil {
			panic(err)
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
			//spew.Dump(out.Meta)

			for _, tx_pre := range out.Meta.PreTokenBalances {
				if *tx_pre.Owner == wallet_address {
					preTransactions = append(preTransactions, tx_pre)
					amount, _ := strconv.ParseFloat(tx_pre.UiTokenAmount.UiAmountString, 64)
					mint_tx_pre = amount
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

		if len(postTransactions) > 0 {
			transaction_formated := TransactionFormatted{}
			transaction_formated.Type = operation
			transaction_formated.MintName = preTransactions[0].Mint
			if operation == "BUY" {
				amount_obtained := mint_tx_pos - mint_tx_pre
				sol_spent := sol_tx_pos - sol_tx_pre
				transaction_formated.MintAmount = amount_obtained
				transaction_formated.SolAmount = sol_spent
			} else if operation == "SELL" {
				//Important to keep pre_mint to apply percentages if the external wallet has not sold 100% of the token
				transaction_formated.MintPre = mint_tx_pre
				amount_obtained := mint_tx_pre - mint_tx_pos
				sol_obtained := sol_tx_pre - sol_tx_pos
				transaction_formated.MintAmount = amount_obtained
				transaction_formated.SolAmount = sol_obtained
			}
			result = append(result, transaction_formated)
		}
	}
	fmt.Println(result)
	time.Sleep(time.Second * 6)
	return true, result, new_starting_point
}
