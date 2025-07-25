package txmanager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	ethwallet "github.com/mdtosif/icarus/internal/account"
	"github.com/mdtosif/icarus/internal/logger"
)

type TxManager struct {
	RpcUrl        string
	WalletsNumber int
	TxNumber      int
	Mnemonic      string
	WaitMilis     int
	Wallets       []*ethwallet.WalletInfo
	Failed        int
	Success       int
	Mu            *sync.Mutex
}

func (t *TxManager) Run() {

	rpcURL := t.RpcUrl
	mnemonic := t.Mnemonic
	batch := t.TxNumber / t.WalletsNumber
	walletsNumber := t.WalletsNumber

	// Create a context with timeout to avoid hanging indefinitely
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		panic("failed to connect to RPC")
	}
	defer client.Close()

	chainId, err := client.NetworkID(ctx)
	if err != nil {
		fmt.Printf("failed to get chain ID")
	}

	wallets, _ := ethwallet.DeriveEthereumWalletsFromMnemonic(mnemonic, walletsNumber, client, t.WaitMilis)

	wg := sync.WaitGroup{}
	t.Wallets = wallets
	var txs []*types.Transaction

	for _, wallet := range wallets {
		wg.Add(1)
		go func() {
			defer wg.Done()
			balance, err := wallet.GetBalanceEther(rpcURL)
			if err != nil {
				logger.Errorf("failed to get balance for address %s: %v", wallet.Address, err)
			}

			fmt.Println(wallet.Address, balance)

			tx, err := wallet.SendEIP1559ETHTransferInBatch(chainId, (batch))
			if err != nil {
				logger.Errorf("failed to send transaction: %v", err)
			}
			t.Mu.Lock()
			defer t.Mu.Unlock()

			txs = append(txs, tx...)
		}()
	}

	wg.Wait()

	wg = sync.WaitGroup{}

	wg.Add(len(txs))
	logger.Infof("Transaction sent successfully: %d", len(txs))

	for _, tx := range txs {
		go func() {

			defer wg.Done()

			err := client.SendTransaction(context.Background(), tx)
			t.Mu.Lock()
			if err != nil {
				t.Failed++
				logger.Errorf("%d/%d failed to send transaction: %v", t.Failed, t.Failed+t.Success, err)
			} else {
				t.Success++
				logger.Debugf("%d/%d Transaction sent successfully: %v", t.Success, t.Failed+t.Success, tx.Hash())
			}
			t.Mu.Unlock()
		}()
		time.Sleep(time.Duration(t.WaitMilis) * time.Millisecond)
	}

	wg.Wait()

	success := t.Success
	failed := t.Failed

	logger.Infof("Total Success Count: %d/%d", success, success+failed)
	logger.Infof("Total Failed Count: %d/%d", failed, success+failed)
}
