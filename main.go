// Package main is the entry point for the Icarus application, which manages
// multiple Ethereum wallet transactions in parallel.
package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/mdtosif/icarus/internal/logger"
	"github.com/mdtosif/icarus/internal/txmanager"
)

const (
	defaultWallets  = 10
	defaultWaitTime = 10 * time.Millisecond
	defaultTxCount  = 100
	defaultLogLevel = logger.INFO
)

// main is the entry point of the application.
// It parses command line flags, initializes the transaction manager, and starts the transaction processing.
func main() {
	// Command line flags
	mnemonic := flag.String(
		"mnemonic",
		"",
		"BIP-39 mnemonic phrase (required)",
	)
	wallets := flag.Int(
		"wallets",
		defaultWallets,
		"Number of wallet instances to create and manage",
	)
	wait := flag.Duration(
		"wait",
		defaultWaitTime,
		"Duration to wait between operations (e.g., 10ms, 1s)",
	)
	rpcURL := flag.String(
		"rpc-url",
		"",
		"Ethereum RPC URL (required)",
	)
	logLevel := flag.Int(
		"log-level",
		int(defaultLogLevel),
		"Log level: debug=0, info=1, warn=2, error=3",
	)
	txCount := flag.Int(
		"txns",
		defaultTxCount,
		"Number of transactions to send per wallet",
	)

	flag.Parse()

	// Input validation
	if *mnemonic == "" {
		fmt.Println("Error: mnemonic is required")
		flag.Usage()
		os.Exit(1)
	}

	if *rpcURL == "" {
		fmt.Println("Error: RPC URL is required")
		flag.Usage()
		os.Exit(1)
	}

	// Set the minimum log level
	logger.SetMinLevel(logger.Level(*logLevel))

	// Initialize transaction manager
	txManager := &txmanager.TxManager{
		RpcUrl:       *rpcURL,
		WaitMilis:    int(*wait / time.Millisecond),
		WalletsNumber: *wallets,
		TxNumber:     *txCount,
		Mu:           &sync.Mutex{},
		Mnemonic:     *mnemonic,
		Success:      0,
		Failed:       0,
	}

	// Start transaction processing
	txManager.Run()
}
