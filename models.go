package main

import (
	"sync"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/rs/zerolog"
)

type DexWallets struct {
	DexArray []string
}

type TransactionFormatted struct {
	Type       string
	SolAmount  float64
	MintAmount float64
	MintName   solana.PublicKey
	MintPre    float64
	ProgramId  solana.PublicKey
	PoolId     solana.PublicKey
}

type TransactionToSend struct {
	Type                 string           `json:"type"`
	SolAmount            float64          `json:"solAmount"`
	MintAmount           float64          `json:"mintAmount"`
	MintName             solana.PublicKey `json:"mintName"`
	Slippage             float64          `json:"slippage"`
	ProgramId            solana.PublicKey `json:"programId"`
	TokenAccountExternal solana.PublicKey `json:"tokenAccountExternal"`
	TokenAccountPersonal solana.PublicKey `json:"tokenAccountPersonal"`
	CurrentPrice         float64          `json:"currentPrice"`
}

type NSReceiver struct {
	Mu              sync.Mutex
	ExternalBalance uint64
	PersonalWallet  PersonalWalletData
	Client          *rpc.Client
	Log             zerolog.Logger
	ExternalWallet  ExternalWalletData
	SolPrice        float64
	Slippage        float64
}

type PersonalWalletData struct {
	PersonalBalance     float64
	MintQuantityHashMap map[solana.PublicKey]float64
	TokenAccountHashMap map[solana.PublicKey]solana.PublicKey
}

type ExternalWalletData struct {
	PersonalBalance     float64
	MintQuantityHashMap map[solana.PublicKey]float64
	TokenAccountHashMap map[solana.PublicKey]solana.PublicKey
}

type TokenInfo struct {
	Solana map[string]float64 `json:"solana"`
}

type TxResponse struct {
	Message string `json:"message"`
}
