package main

import (
	"sync"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
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
}

type TransactionToSend struct {
	Type       string
	SolAmount  float64
	MintAmount float64
	MintName   solana.PublicKey
}

type ControlConcurrency struct {
	Mu              sync.Mutex
	ExternalBalance uint64
	PersonalWallet  PersonalWalletData
	Client          *rpc.Client
}

type PersonalWalletData struct {
	PersonalBalance     uint64
	MintQuantityHashMap map[interface{}]float64
}
