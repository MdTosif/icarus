package ethwallet

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mdtosif/icarus/internal/logger"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
	"github.com/tyler-smith/go-bip39"
)

// WalletInfo holds the derived address and private key hex string.
type WalletInfo struct {
	Address    common.Address    // Hex address, e.g., "0x..."
	PrivateKey *ecdsa.PrivateKey // Hex of the private key (64 bytes hex, without 0x prefix)
	Client     *ethclient.Client
	mu         *sync.Mutex
	WaitMilis  int
	Failed     int
	Success    int
}

// DeriveEthereumWalletsFromMnemonic derives `count` Ethereum wallets from the given mnemonic and optional passphrase.
//
// mnemonic: BIP-39 mnemonic phrase (12/15/18/21/24 words).
// passphrase: optional passphrase for the seed; often "" if not used.
// count: how many addresses to derive starting at index 0.
//
// Returns a slice of WalletInfo of length `count`, or an error.
func DeriveEthereumWalletsFromMnemonic(mnemonic string, count int, client *ethclient.Client, waitMilis int) ([]*WalletInfo, error) {
	if count <= 0 {
		return nil, errors.New("count must be > 0")
	}

	// 1. Validate mnemonic
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, fmt.Errorf("invalid mnemonic")
	}

	// 2. Create HD wallet
	// hdwallet.NewFromMnemonic will internally create seed = BIP39.NewSeed(mnemonic, passphrase)
	wallet, err := hdwallet.NewFromMnemonic(mnemonic)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet from mnemonic: %w", err)
	}

	wallets := make([]*WalletInfo, 0, count)

	// 3. For each index, derive path m/44'/60'/0'/0/i
	for i := 0; i < count; i++ {
		// Format the derivation path
		// Example path: m/44'/60'/0'/0/0, m/44'/60'/0'/0/1, etc.
		derivationPath := fmt.Sprintf("m/44'/60'/0'/0/%d", i)
		path, err := hdwallet.ParseDerivationPath(derivationPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse derivation path %s: %w", derivationPath, err)
		}

		account, err := wallet.Derive(path, false)
		if err != nil {
			return nil, fmt.Errorf("failed to derive account at index %d: %w", i, err)
		}

		// 4. Get the private key for this account
		privKey, err := wallet.PrivateKey(account)
		if err != nil {
			return nil, fmt.Errorf("failed to get private key for index %d: %w", i, err)
		}

		wallets = append(wallets, &WalletInfo{
			Address:    account.Address,
			PrivateKey: privKey,
			Client:     client,
			mu:         &sync.Mutex{},
			Failed:     0,
			Success:    0,
			WaitMilis:  waitMilis,
		})
	}

	return wallets, nil
}

// GetBalanceWei connects to the given Ethereum JSON-RPC endpoint (rpcURL),
// and returns the balance in Wei for the provided hex address string.
// rpcURL: e.g. "https://mainnet.infura.io/v3/YOUR-PROJECT-ID" or "ws://..."
// addressHex: e.g. "0x742d35Cc6634C0532925a3b844Bc454e4438f44e"
//
// Returns the balance as *big.Int in Wei, or an error.
func GetBalanceWei(rpcURL, addressHex string) (*big.Int, error) {
	// 1. Validate address format
	if !common.IsHexAddress(addressHex) {
		return nil, fmt.Errorf("invalid Ethereum address: %s", addressHex)
	}
	addr := common.HexToAddress(addressHex)

	// 2. Create a context with timeout for dialing and RPC calls
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 3. Dial the RPC endpoint
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum RPC at %s: %w", rpcURL, err)
	}
	defer client.Close()

	// 4. Query balance at latest block (pass nil for block number)
	balanceWei, err := client.BalanceAt(ctx, addr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance for address %s: %w", addressHex, err)
	}

	return balanceWei, nil
}

// WeiToEther converts a balance in Wei (*big.Int) to Ether (*big.Float).
// It returns a *big.Float representing balance / 1e18.
// Note: big.Float uses arbitrary precision; here we set sufficient precision for Ethereum values.
func WeiToEther(balanceWei *big.Int) *big.Float {
	// Create a big.Float from the Wei
	fWei := new(big.Float).SetInt(balanceWei)
	// Divide by 1e18
	ethValue := new(big.Float).Quo(fWei, big.NewFloat(1e18))
	return ethValue
}

// GetBalanceEther is a convenience wrapper: returns the balance as a decimal string in Ether.
// It internally calls GetBalanceWei and converts to Ether string.
// Returns something like "0.123456789012345678".
func (wallet *WalletInfo) GetBalanceEther(rpcURL string) (string, error) {
	balanceWei, err := GetBalanceWei(rpcURL, wallet.Address.Hex())
	if err != nil {
		return "", err
	}
	ethValue := WeiToEther(balanceWei)
	// Format with necessary precision.
	// Note: .Text('f', 18) prints exactly 18 decimal places.
	// You can trim trailing zeros if desired.
	str := ethValue.Text('f', 18)
	// Optionally trim trailing zeros and dot:
	str = trimTrailingZeros(str)
	return str, nil
}

// trimTrailingZeros removes trailing zeros and possibly the decimal point if integer.
// E.g. "1.230000000000000000" -> "1.23"; "2.000000000000000000" -> "2"
func trimTrailingZeros(s string) string {
	// Find decimal point
	dot := -1
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			dot = i
			break
		}
	}
	if dot < 0 {
		return s
	}
	// Trim trailing zeros
	i := len(s) - 1
	for i > dot && s[i] == '0' {
		i--
	}
	// If the last char is '.', remove it too
	if i == dot {
		i--
	}
	return s[:i+1]
}

// SendEIP1559ETHTransfer sends an EIP-1559 transaction from the wallet at nonceIncrease,
// with tipCap, maxFeeCap, and gasLimit. The recipient is the same as the wallet's address.
// The transaction has a value of 100000 Wei.
// Returns the signed transaction, or an error.
func (wallet *WalletInfo) SendEIP1559ETHTransfer(chainId *big.Int, nonceIncrease uint64, tipCap *big.Int, maxFeeCap *big.Int, gasLimit uint64, value int64) (*types.Transaction, error) {

	txData := &types.DynamicFeeTx{
		ChainID:   chainId,
		Nonce:     nonceIncrease,
		GasTipCap: tipCap,
		GasFeeCap: maxFeeCap,
		Gas:       gasLimit,
		To:        &wallet.Address,
		Value:     big.NewInt((value)),
		Data:      nil,
		// AccessList: nil,
	}
	tx := types.NewTx(txData)

	signedTx, err := types.SignTx(tx, types.NewLondonSigner(chainId), wallet.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign tx: %w", err)
	}

	return signedTx, nil
}

func (wallet *WalletInfo) SendEIP1559ETHTransferInBatch(chainId *big.Int, batch int) ([]*types.Transaction, error) {
	client := wallet.Client
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	nonce, err := client.PendingNonceAt(ctx, wallet.Address)
	if err != nil {
		logger.Errorf("failed to get nonce: %v", err)
		return nil, err
	}

	msg := ethereum.CallMsg{
		From:  wallet.Address,
		To:    &wallet.Address,
		Value: big.NewInt(100000000000),
		Data:  nil,
	}

	gasLimit, err := wallet.Client.EstimateGas(ctx, msg)
	if err != nil {
		logger.Errorf("failed to estimate gas: %v", err)
		return nil, err
	}

	// Optionally add buffer:
	gasLimit += 1000

	tipCap, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		logger.Errorf("failed to suggest tip cap: %v", err)
		return nil, err
	}

	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		logger.Errorf("failed to fetch latest header: %v", err)
		return nil, err
	}

	if header.BaseFee == nil {
		logger.Errorf("node does not return base fee (non-EIP-1559?)")
		return nil,	err
	}

	baseFee := header.BaseFee

	maxFeeCap := new(big.Int).Add(
		new(big.Int).Mul(baseFee, big.NewInt(2)),
		tipCap,
	)

	var txs []*types.Transaction

	for i := (0); i < batch; i++ {
		tx, err := wallet.SendEIP1559ETHTransfer(chainId, nonce+uint64(i), tipCap, maxFeeCap, gasLimit, 10000)
		if err != nil {
			logger.Errorf("failed to create transaction: %v", err)

		} else {
			logger.Debugf("Transaction created successfully: %d/%d", i, batch)
			txs = append(txs, tx)
		}
	}

	logger.Infof("Sending %d transactions...", len(txs))

	return txs, nil
}
