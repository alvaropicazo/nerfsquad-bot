package main

import (
	"context"
	"os"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	"gopkg.in/yaml.v3"
)

// Calculates the quantit of x token that should be transferred according to our balance.
func (m *ControlConcurrency) format_data(tx_available []TransactionFormatted) ([]TransactionToSend, error) {
	res := []TransactionToSend{}
	for _, tx := range tx_available {
		tx_to_send := TransactionToSend{}
		if tx.Type == "BUY" {
			percentage_external := tx.SolAmount / float64(m.ExternalBalance)
			sol_to_spend := m.PersonalWallet.PersonalBalance * uint64(percentage_external)
			tx_to_send.SolAmount = float64(sol_to_spend)

		} else if tx.Type == "SELL" {
			percentage_external := tx.MintAmount / tx.MintPre //If its 1, all stake was sold.
			tokens_to_sell := m.PersonalWallet.MintQuantityHashMap[tx.MintName] * percentage_external
			tx_to_send.MintAmount = tokens_to_sell

		}
		tx_to_send.Type = tx.Type
		tx_to_send.MintName = tx.MintName
	}

	return res, nil
}

// Retrieves the keys to be used for the apis.
func get_api_key() (string, error) {
	keys_file := make(map[string]interface{})

	keys_byte, err := os.ReadFile("./config/keys.yaml")
	if err != nil {
		return "yamlFile.Get err #%v ", err
	}
	err = yaml.Unmarshal(keys_byte, keys_file)
	if err != nil {
		return "Unmarshal: %v", err
	}

	return keys_file["keys"].([]interface{})[0].(string), nil
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

// Gets the signature as a starting point, any transaction newer to it should be checked.
func get_starting_point(wallet_address solana.PublicKey, api_token string, current_date int64, client *rpc.Client) (solana.Signature, error) {
	out, err := client.GetSignaturesForAddress(
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
func (m *ControlConcurrency) get_balance(client *rpc.Client, external_wallet_address solana.PublicKey, personal_wallet_address solana.PublicKey) error {
	out_sol, err := client.GetBalance(
		context.TODO(),
		external_wallet_address,
		rpc.CommitmentFinalized,
	)
	if err != nil {
		panic(err)
	}

	out_sol_personal, err := client.GetBalance(
		context.TODO(),
		personal_wallet_address,
		rpc.CommitmentFinalized,
	)
	if err != nil {
		return err
	}

	m.PersonalWallet.PersonalBalance = out_sol_personal.Value
	m.ExternalBalance = out_sol.Value

	err = m.get_token_accounts(personal_wallet_address, client)
	if err != nil {
		return err
	}
	//spew.Dump(out_sol.Value) // total lamports on the account; 1 sol = 1000000000 lamports

	// var lamportsOnAccount = new(big.Float).SetUint64(uint64(out_sol.Value))
	// // Convert lamports to sol:
	// var solBalance = new(big.Float).Quo(lamportsOnAccount, new(big.Float).SetUint64(solana.LAMPORTS_PER_SOL))

	return nil
}

// Gets the different token accounts from the tracked wallet.
func (m *ControlConcurrency) get_token_accounts(pubKey solana.PublicKey, client *rpc.Client) error {
	check_program_id, err := client.GetTokenAccountsByOwner(
		context.TODO(),
		pubKey,
		&rpc.GetTokenAccountsConfig{
			Mint: solana.WrappedSol.ToPointer(),
		},
		&rpc.GetTokenAccountsOpts{
			Encoding: solana.EncodingBase64Zstd,
		},
	)
	if err != nil {
		return err
	}
	program_id := check_program_id.Value[0].Account.Owner

	out, err := client.GetTokenAccountsByOwner(
		context.TODO(),
		pubKey,
		&rpc.GetTokenAccountsConfig{
			ProgramId: program_id.ToPointer(),
		},
		&rpc.GetTokenAccountsOpts{
			Encoding: solana.EncodingBase64Zstd,
		},
	)
	if err != nil {
		return err
	}

	for _, rawAccount := range out.Value {
		var tokAcc token.Account

		data := rawAccount.Account.Data.GetBinary()
		dec := bin.NewBinDecoder(data)
		err := dec.Decode(&tokAcc)
		if err != nil {
			return err
		}
		if tokAcc.Amount != 0 {
			m.PersonalWallet.MintQuantityHashMap[tokAcc.Mint] = float64(tokAcc.Amount)
		}
	}

	return nil
}
