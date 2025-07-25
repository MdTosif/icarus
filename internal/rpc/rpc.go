package rpc

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/ethclient"
)

// GetChainID connects to the given Ethereum JSON-RPC endpoint and returns the chain ID as *big.Int.
// rpcURL: e.g. "https://mainnet.infura.io/v3/YOUR-PROJECT-ID" or "http://localhost:8545".
func GetChainID(client *ethclient.Client, ctx context.Context) (*big.Int, error) {
   


    // NetworkID returns the chain ID (it uses eth_chainId under the hood)
    chainID, err := client.NetworkID(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get chain ID: %w", err)
    }
    return chainID, nil
}