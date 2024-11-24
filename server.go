package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var current_date int64

type ControlConcurrency struct {
	mu sync.Mutex
}

func format_data(tx_available []interface{}) ([]interface{}, error) {
	// var external_wallet_balance, personal_wallet_balance int
	// //First we get total balance from external wallet
	// external_wallet_balance = 100
	// //Now we get our current personal wallet balance
	// personal_wallet_balance = 20
	// //Latest transaction data

	// for i := range tx_available {

	// }
	//An array of tx already formatted is returned
	return nil, nil
}

// main function which calls submethods to recreate tx
func (m *ControlConcurrency) replicate_transaction(tx_available []interface{}) error {

	_, err := format_data(tx_available)
	if err != nil {
		return err
	}

	//Use Raydium API to send data
	//Freeing channel so other thread can insert new txs to replicate

	m.mu.Unlock()
	return nil
}

func check_new_tx_available(solscan_url string, wallet_address string, api_token string, current_date int64) (bool, []interface{}) {

	req, err := http.NewRequest("GET", solscan_url+wallet_address, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("token", api_token) //<--add your header

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// print all the response headers
	fmt.Println("Response headers:")
	for k, v := range resp.Header {
		fmt.Printf("%s: %s\n", k, v)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	//Parse data into object
	fmt.Println(string(data))

	return true, nil
}

func main() {
	current_date = time.Now().UnixMilli()
	var mu ControlConcurrency
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

	//infinite loop checking for new transactions
	for {
		available := false
		available, tx_available := check_new_tx_available(apis_obj["url"].(map[string]interface{})["solscan"].(string), wallet_obj["external_wallet"].(string), api_key, current_date)
		if available {
			available = false
			mu.mu.Lock()
			//go thread to process transaction
			go mu.replicate_transaction(tx_available)
		}
	}

}
