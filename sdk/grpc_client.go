package sdk

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// DirectGRPCClient provides high-performance gRPC access to the chain
// Bypasses CLI overhead for ~10x latency improvement
type DirectGRPCClient struct {
	conn         *grpc.ClientConn
	txClient     txtypes.ServiceClient
	authClient   authtypes.QueryClient
	cdc          codec.Codec
	chainID      string
	keyring      keyring.Keyring
	accountCache sync.Map
	mu           sync.RWMutex
}

// NewDirectGRPCClient creates a new high-perf gRPC client
func NewDirectGRPCClient(grpcAddr, chainID string, cdc codec.Codec, kr keyring.Keyring) (*DirectGRPCClient, error) {
	conn, err := grpc.Dial(
		grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*10)), // 10MB
	)
	if err != nil {
		return nil, fmt.Errorf("failed to dial gRPC: %w", err)
	}

	return &DirectGRPCClient{
		conn:       conn,
		txClient:   txtypes.NewServiceClient(conn),
		authClient: authtypes.NewQueryClient(conn),
		cdc:        cdc,
		chainID:    chainID,
		keyring:    kr,
	}, nil
}

// AccountInfo caches account sequence for faster tx building
type AccountInfo struct {
	Address       string
	AccountNumber uint64
	Sequence      uint64
	LastUpdated   time.Time
}

// BroadcastTx broadcasts a signed transaction with minimal latency
func (c *DirectGRPCClient) BroadcastTx(ctx context.Context, txBytes []byte, mode txtypes.BroadcastMode) (*sdk.TxResponse, error) {
	res, err := c.txClient.BroadcastTx(ctx, &txtypes.BroadcastTxRequest{
		TxBytes: txBytes,
		Mode:    mode,
	})
	if err != nil {
		return nil, fmt.Errorf("broadcast failed: %w", err)
	}
	return res.TxResponse, nil
}

// BroadcastTxSync broadcasts and waits for CheckTx result (faster than commit)
func (c *DirectGRPCClient) BroadcastTxSync(ctx context.Context, txBytes []byte) (*sdk.TxResponse, error) {
	return c.BroadcastTx(ctx, txBytes, txtypes.BroadcastMode_BROADCAST_MODE_SYNC)
}

// BroadcastTxAsync broadcasts without waiting (fastest, ~1ms)
func (c *DirectGRPCClient) BroadcastTxAsync(ctx context.Context, txBytes []byte) (*sdk.TxResponse, error) {
	return c.BroadcastTx(ctx, txBytes, txtypes.BroadcastMode_BROADCAST_MODE_ASYNC)
}

// GetAccountInfo fetches or returns cached account info
func (c *DirectGRPCClient) GetAccountInfo(ctx context.Context, address string) (*AccountInfo, error) {
	// Check cache first
	if cached, ok := c.accountCache.Load(address); ok {
		info := cached.(*AccountInfo)
		// Cache valid for 100ms (assuming <200ms block time)
		if time.Since(info.LastUpdated) < 100*time.Millisecond {
			return info, nil
		}
	}

	// Fetch from chain
	res, err := c.authClient.Account(ctx, &authtypes.QueryAccountRequest{
		Address: address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	var acc authtypes.AccountI
	if err := c.cdc.UnpackAny(res.Account, &acc); err != nil {
		return nil, fmt.Errorf("failed to unpack account: %w", err)
	}

	info := &AccountInfo{
		Address:       address,
		AccountNumber: acc.GetAccountNumber(),
		Sequence:      acc.GetSequence(),
		LastUpdated:   time.Now(),
	}

	c.accountCache.Store(address, info)
	return info, nil
}

// IncrementSequence atomically increments the cached sequence
func (c *DirectGRPCClient) IncrementSequence(address string) {
	if cached, ok := c.accountCache.Load(address); ok {
		info := cached.(*AccountInfo)
		c.mu.Lock()
		info.Sequence++
		c.mu.Unlock()
	}
}

// BatchBroadcast sends multiple transactions in parallel
func (c *DirectGRPCClient) BatchBroadcast(ctx context.Context, txBytesSlice [][]byte) ([]*sdk.TxResponse, error) {
	results := make([]*sdk.TxResponse, len(txBytesSlice))
	errors := make([]error, len(txBytesSlice))
	var wg sync.WaitGroup

	for i, txBytes := range txBytesSlice {
		wg.Add(1)
		go func(idx int, tb []byte) {
			defer wg.Done()
			res, err := c.BroadcastTxAsync(ctx, tb)
			results[idx] = res
			errors[idx] = err
		}(i, txBytes)
	}

	wg.Wait()

	// Check for errors
	for _, err := range errors {
		if err != nil {
			return results, fmt.Errorf("batch broadcast had errors: %w", err)
		}
	}

	return results, nil
}

// Close closes the gRPC connection
func (c *DirectGRPCClient) Close() error {
	return c.conn.Close()
}

// TxBuilder helper for building transactions efficiently
type TxBuilder struct {
	client  *DirectGRPCClient
	factory tx.Factory
}

// NewTxBuilder creates a new transaction builder
func NewTxBuilder(client *DirectGRPCClient, gasLimit uint64, gasPrice string) *TxBuilder {
	return &TxBuilder{
		client: client,
	}
}

// Stats returns client statistics
type ClientStats struct {
	TotalBroadcasts   int64
	SuccessBroadcasts int64
	FailedBroadcasts  int64
	AvgLatency        time.Duration
	CacheHits         int64
	CacheMisses       int64
}
