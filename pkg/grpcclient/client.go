// Package grpcclient provides high-performance gRPC client for chain interaction
package grpcclient

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	orderbooktypes "github.com/openalpha/perp-dex/x/orderbook/types"
)

// Config holds gRPC client configuration
type Config struct {
	GRPCAddr       string
	ChainID        string
	AccountNumber  uint64
	GasLimit       uint64
	GasPrice       string
	PoolSize       int           // Connection pool size
	Timeout        time.Duration // Request timeout
	RetryAttempts  int           // Retry attempts on failure
	BatchSize      int           // Max transactions per batch
}

// DefaultConfig returns optimized default configuration
func DefaultConfig() *Config {
	return &Config{
		GRPCAddr:       "localhost:9090",
		ChainID:        "perpdex-1",
		AccountNumber:  0,
		GasLimit:       200000,
		GasPrice:       "0.001usdc",
		PoolSize:       10,
		Timeout:        5 * time.Second,
		RetryAttempts:  3,
		BatchSize:      100,
	}
}

// Client is a high-performance gRPC client with connection pooling
type Client struct {
	config    *Config
	pool      []*grpc.ClientConn
	poolIndex uint64
	mu        sync.RWMutex

	// Cached signer info
	privKey   cryptotypes.PrivKey
	pubKey    cryptotypes.PubKey
	address   sdk.AccAddress
	sequence  uint64
	seqMu     sync.Mutex

	// Metrics
	txCount     uint64
	successCount uint64
	failCount   uint64
	totalLatency int64

	// TX encoder
	txConfig client.TxConfig
}

// NewClient creates a new high-performance gRPC client
func NewClient(config *Config, privKeyHex string) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Decode private key
	privKeyBytes, err := hex.DecodeString(privKeyHex)
	if err != nil {
		return nil, fmt.Errorf("decode private key: %w", err)
	}

	privKey := &secp256k1.PrivKey{Key: privKeyBytes}
	pubKey := privKey.PubKey()
	address := sdk.AccAddress(pubKey.Address())

	c := &Client{
		config:   config,
		pool:     make([]*grpc.ClientConn, config.PoolSize),
		privKey:  privKey,
		pubKey:   pubKey,
		address:  address,
		sequence: 0,
	}

	// Initialize connection pool
	for i := 0; i < config.PoolSize; i++ {
		conn, err := grpc.Dial(
			config.GRPCAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(1024*1024*10), // 10MB
				grpc.MaxCallSendMsgSize(1024*1024*10),
			),
		)
		if err != nil {
			return nil, fmt.Errorf("connect to gRPC: %w", err)
		}
		c.pool[i] = conn
	}

	return c, nil
}

// getConn returns a connection from the pool (round-robin)
func (c *Client) getConn() *grpc.ClientConn {
	idx := atomic.AddUint64(&c.poolIndex, 1) % uint64(len(c.pool))
	return c.pool[idx]
}

// nextSequence atomically increments and returns the next sequence number
func (c *Client) nextSequence() uint64 {
	c.seqMu.Lock()
	defer c.seqMu.Unlock()
	seq := c.sequence
	c.sequence++
	return seq
}

// PlaceOrderResult contains the result of a place order operation
type PlaceOrderResult struct {
	TxHash  string
	Success bool
	Latency time.Duration
	Error   error
}

// PlaceOrder places a single order with minimal latency
func (c *Client) PlaceOrder(ctx context.Context, marketID, side, orderType, price, quantity string) *PlaceOrderResult {
	start := time.Now()
	result := &PlaceOrderResult{}

	atomic.AddUint64(&c.txCount, 1)

	// Build message
	msg := &orderbooktypes.MsgPlaceOrder{
		Trader:    c.address.String(),
		MarketId:  marketID,
		Side:      parseSide(side),
		OrderType: parseOrderType(orderType),
		Price:     price,
		Quantity:  quantity,
	}

	// Get sequence
	seq := c.nextSequence()

	// Build and sign transaction in memory
	txBytes, err := c.buildSignedTx(msg, seq)
	if err != nil {
		result.Error = err
		result.Latency = time.Since(start)
		atomic.AddUint64(&c.failCount, 1)
		return result
	}

	// Broadcast via gRPC
	conn := c.getConn()
	txClient := NewTxServiceClient(conn)

	resp, err := txClient.BroadcastTx(ctx, &BroadcastTxRequest{
		TxBytes: txBytes,
		Mode:    BroadcastMode_BROADCAST_MODE_ASYNC,
	})

	result.Latency = time.Since(start)
	atomic.AddInt64(&c.totalLatency, int64(result.Latency))

	if err != nil {
		result.Error = err
		atomic.AddUint64(&c.failCount, 1)
		return result
	}

	if resp.TxResponse.Code == 0 {
		result.Success = true
		result.TxHash = resp.TxResponse.TxHash
		atomic.AddUint64(&c.successCount, 1)
	} else {
		result.Error = fmt.Errorf("tx failed: %s", resp.TxResponse.RawLog)
		atomic.AddUint64(&c.failCount, 1)
	}

	return result
}

// BatchOrder represents an order in a batch
type BatchOrder struct {
	MarketID  string
	Side      string
	OrderType string
	Price     string
	Quantity  string
}

// BatchPlaceOrders places multiple orders in a single transaction
func (c *Client) BatchPlaceOrders(ctx context.Context, orders []BatchOrder) *PlaceOrderResult {
	start := time.Now()
	result := &PlaceOrderResult{}

	if len(orders) == 0 {
		result.Error = fmt.Errorf("no orders to place")
		return result
	}

	if len(orders) > c.config.BatchSize {
		result.Error = fmt.Errorf("batch size %d exceeds max %d", len(orders), c.config.BatchSize)
		return result
	}

	atomic.AddUint64(&c.txCount, uint64(len(orders)))

	// Build messages
	msgs := make([]sdk.Msg, len(orders))
	for i, order := range orders {
		msgs[i] = &orderbooktypes.MsgPlaceOrder{
			Trader:    c.address.String(),
			MarketId:  order.MarketID,
			Side:      parseSide(order.Side),
			OrderType: parseOrderType(order.OrderType),
			Price:     order.Price,
			Quantity:  order.Quantity,
		}
	}

	// Get sequence
	seq := c.nextSequence()

	// Build and sign multi-message transaction
	txBytes, err := c.buildSignedTxMulti(msgs, seq)
	if err != nil {
		result.Error = err
		result.Latency = time.Since(start)
		atomic.AddUint64(&c.failCount, uint64(len(orders)))
		return result
	}

	// Broadcast
	conn := c.getConn()
	txClient := NewTxServiceClient(conn)

	resp, err := txClient.BroadcastTx(ctx, &BroadcastTxRequest{
		TxBytes: txBytes,
		Mode:    BroadcastMode_BROADCAST_MODE_ASYNC,
	})

	result.Latency = time.Since(start)
	atomic.AddInt64(&c.totalLatency, int64(result.Latency))

	if err != nil {
		result.Error = err
		atomic.AddUint64(&c.failCount, uint64(len(orders)))
		return result
	}

	if resp.TxResponse.Code == 0 {
		result.Success = true
		result.TxHash = resp.TxResponse.TxHash
		atomic.AddUint64(&c.successCount, uint64(len(orders)))
	} else {
		result.Error = fmt.Errorf("batch tx failed: %s", resp.TxResponse.RawLog)
		atomic.AddUint64(&c.failCount, uint64(len(orders)))
	}

	return result
}

// buildSignedTx builds and signs a transaction in memory
func (c *Client) buildSignedTx(msg sdk.Msg, sequence uint64) ([]byte, error) {
	return c.buildSignedTxMulti([]sdk.Msg{msg}, sequence)
}

// buildSignedTxMulti builds and signs a multi-message transaction
func (c *Client) buildSignedTxMulti(msgs []sdk.Msg, sequence uint64) ([]byte, error) {
	// This is a simplified implementation
	// In production, use proper tx building from cosmos-sdk

	// Create tx builder
	txBuilder := c.txConfig.NewTxBuilder()

	// Set messages
	if err := txBuilder.SetMsgs(msgs...); err != nil {
		return nil, err
	}

	// Set fee
	fee := sdk.NewCoins(sdk.NewCoin("usdc", sdk.NewInt(int64(c.config.GasLimit)*10)))
	txBuilder.SetFeeAmount(fee)
	txBuilder.SetGasLimit(c.config.GasLimit * uint64(len(msgs)))

	// Sign
	sigV2 := signing.SignatureV2{
		PubKey: c.pubKey,
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: nil,
		},
		Sequence: sequence,
	}

	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return nil, err
	}

	// Get sign bytes
	signerData := authsigning.SignerData{
		ChainID:       c.config.ChainID,
		AccountNumber: c.config.AccountNumber,
		Sequence:      sequence,
	}

	signBytes, err := c.txConfig.SignModeHandler().GetSignBytes(
		signing.SignMode_SIGN_MODE_DIRECT,
		signerData,
		txBuilder.GetTx(),
	)
	if err != nil {
		return nil, err
	}

	// Sign
	signature, err := c.privKey.Sign(signBytes)
	if err != nil {
		return nil, err
	}

	// Set signature
	sigV2.Data = &signing.SingleSignatureData{
		SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
		Signature: signature,
	}

	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return nil, err
	}

	// Encode
	return c.txConfig.TxEncoder()(txBuilder.GetTx())
}

// GetMetrics returns current client metrics
func (c *Client) GetMetrics() (txCount, successCount, failCount uint64, avgLatency time.Duration) {
	txCount = atomic.LoadUint64(&c.txCount)
	successCount = atomic.LoadUint64(&c.successCount)
	failCount = atomic.LoadUint64(&c.failCount)

	if successCount > 0 {
		avgLatency = time.Duration(atomic.LoadInt64(&c.totalLatency) / int64(successCount))
	}
	return
}

// ResetMetrics resets all metrics
func (c *Client) ResetMetrics() {
	atomic.StoreUint64(&c.txCount, 0)
	atomic.StoreUint64(&c.successCount, 0)
	atomic.StoreUint64(&c.failCount, 0)
	atomic.StoreInt64(&c.totalLatency, 0)
}

// Close closes all connections in the pool
func (c *Client) Close() error {
	for _, conn := range c.pool {
		if err := conn.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Helper functions
func parseSide(s string) orderbooktypes.Side {
	switch s {
	case "buy", "BUY":
		return orderbooktypes.Side_SIDE_BUY
	case "sell", "SELL":
		return orderbooktypes.Side_SIDE_SELL
	default:
		return orderbooktypes.Side_SIDE_BUY
	}
}

func parseOrderType(t string) orderbooktypes.OrderType {
	switch t {
	case "limit", "LIMIT":
		return orderbooktypes.OrderType_ORDER_TYPE_LIMIT
	case "market", "MARKET":
		return orderbooktypes.OrderType_ORDER_TYPE_MARKET
	default:
		return orderbooktypes.OrderType_ORDER_TYPE_LIMIT
	}
}

// Placeholder types for gRPC (would be generated from proto)
type TxServiceClient interface {
	BroadcastTx(ctx context.Context, req *BroadcastTxRequest, opts ...grpc.CallOption) (*BroadcastTxResponse, error)
}

type BroadcastTxRequest struct {
	TxBytes []byte
	Mode    BroadcastMode
}

type BroadcastMode int

const (
	BroadcastMode_BROADCAST_MODE_ASYNC BroadcastMode = iota
	BroadcastMode_BROADCAST_MODE_SYNC
	BroadcastMode_BROADCAST_MODE_BLOCK
)

type BroadcastTxResponse struct {
	TxResponse *TxResponse
}

type TxResponse struct {
	TxHash string
	Code   uint32
	RawLog string
}

func NewTxServiceClient(conn *grpc.ClientConn) TxServiceClient {
	return &txServiceClient{conn: conn}
}

type txServiceClient struct {
	conn *grpc.ClientConn
}

func (c *txServiceClient) BroadcastTx(ctx context.Context, req *BroadcastTxRequest, opts ...grpc.CallOption) (*BroadcastTxResponse, error) {
	// Implementation would use actual gRPC call
	return &BroadcastTxResponse{
		TxResponse: &TxResponse{
			TxHash: "placeholder",
			Code:   0,
		},
	}, nil
}
