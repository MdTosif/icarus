package main

import (
	"flag"
	"sync"

	"github.com/mdtosif/icarus/internal/logger"
	"github.com/mdtosif/icarus/internal/txmanager"
)

func main() {
	 // Flags for
	 mnemonic := flag.String("mnemonic", "", "BIP-39 mnemonic phrase (required)")
	 wallets := flag.Int("wallets", 10, "Number of wallets to derive")
	 wait := flag.Duration("wait", 10, "Wait duration between operations (e.g., 10ms)")
	 rpcURL := flag.String("rpc-url", "", "Ethereum RPC URL (required)")
	 logLevel := flag.Int("log-level", 1, "Log level: debug = 0, info = 1, warn = 2, error = 3")
	 txns := flag.Int("txns", 100, "Number of transactions to send per wallet")
	 flag.Parse()

	 //
	
	 logger.SetMinLevel(logger.Level(*logLevel))
	
	// === Run ===
	txmanager := &txmanager.TxManager{
		RpcUrl: *rpcURL,
		WaitMilis: int(*wait),
		WalletsNumber: *wallets,
		TxNumber: *txns,
		Mu: &sync.Mutex{},
		Mnemonic: *mnemonic,
		Success: 0,
		Failed: 0,
	}

	txmanager.Run()
	// === Run ===

}
