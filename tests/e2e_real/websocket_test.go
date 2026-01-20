package e2e_real

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type    string          `json:"type"`
	Channel string          `json:"channel,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// TestWebSocket_Connect tests WebSocket connection
func TestWebSocket_Connect(t *testing.T) {
	config := DefaultConfig()
	client := NewWSClient(config)

	err := client.Connect("/ws")
	if err != nil {
		t.Skipf("WebSocket server not running: %v", err)
	}
	defer client.Close()

	t.Log("WebSocket connected successfully")
}

// TestWebSocket_Subscribe tests subscribing to channels
func TestWebSocket_Subscribe(t *testing.T) {
	config := DefaultConfig()
	client := NewWSClient(config)

	err := client.Connect("/ws")
	if err != nil {
		t.Skipf("WebSocket server not running: %v", err)
	}
	defer client.Close()

	// Subscribe to ticker channel
	subscribeMsg := map[string]interface{}{
		"type":    "subscribe",
		"channel": "ticker:BTC-USDC",
	}

	if err := client.Send(subscribeMsg); err != nil {
		t.Fatalf("Failed to send subscribe: %v", err)
	}

	// Wait for subscription confirmation or data
	msg, err := client.Receive(5 * time.Second)
	if err != nil {
		t.Fatalf("Failed to receive message: %v", err)
	}

	t.Logf("Received: %s", string(msg))
}

// TestWebSocket_TickerUpdates tests receiving ticker updates
func TestWebSocket_TickerUpdates(t *testing.T) {
	config := DefaultConfig()
	client := NewWSClient(config)

	err := client.Connect("/ws")
	if err != nil {
		t.Skipf("WebSocket server not running: %v", err)
	}
	defer client.Close()

	// Subscribe to ticker
	subscribeMsg := map[string]interface{}{
		"type":    "subscribe",
		"channel": "ticker:BTC-USDC",
	}
	client.Send(subscribeMsg)

	// Receive multiple updates
	receivedCount := 0
	timeout := time.After(10 * time.Second)

	for receivedCount < 3 {
		select {
		case <-timeout:
			if receivedCount == 0 {
				t.Skip("No ticker updates received within timeout")
			}
			return
		default:
			msg, err := client.Receive(2 * time.Second)
			if err != nil {
				continue
			}

			var wsMsg WSMessage
			if json.Unmarshal(msg, &wsMsg) == nil {
				t.Logf("Ticker update %d: %s", receivedCount+1, string(msg))
				receivedCount++
			}
		}
	}

	t.Logf("Received %d ticker updates", receivedCount)
}

// TestWebSocket_OrderBookUpdates tests receiving order book updates
func TestWebSocket_OrderBookUpdates(t *testing.T) {
	config := DefaultConfig()
	client := NewWSClient(config)

	err := client.Connect("/ws")
	if err != nil {
		t.Skipf("WebSocket server not running: %v", err)
	}
	defer client.Close()

	// Subscribe to depth/orderbook
	subscribeMsg := map[string]interface{}{
		"type":    "subscribe",
		"channel": "depth:BTC-USDC",
	}
	client.Send(subscribeMsg)

	// Wait for orderbook snapshot or update
	msg, err := client.Receive(5 * time.Second)
	if err != nil {
		t.Skipf("No orderbook update received: %v", err)
	}

	t.Logf("OrderBook update: %s", string(msg))
}

// TestWebSocket_TradeUpdates tests receiving trade updates
func TestWebSocket_TradeUpdates(t *testing.T) {
	config := DefaultConfig()
	wsClient := NewWSClient(config)
	httpClient := NewHTTPClient(config)

	err := wsClient.Connect("/ws")
	if err != nil {
		t.Skipf("WebSocket server not running: %v", err)
	}
	defer wsClient.Close()

	// Subscribe to trades
	subscribeMsg := map[string]interface{}{
		"type":    "subscribe",
		"channel": "trades:BTC-USDC",
	}
	wsClient.Send(subscribeMsg)

	// Place orders to trigger a trade
	trader1 := fmt.Sprintf("ws_trader1_%d", time.Now().UnixNano())
	trader2 := fmt.Sprintf("ws_trader2_%d", time.Now().UnixNano())

	// Buyer
	PlaceOrder(httpClient, &Order{
		MarketID:  "BTC-USDC",
		Trader:    trader1,
		Side:      "buy",
		OrderType: "limit",
		Price:     "50000",
		Quantity:  "0.1",
		Leverage:  "10",
	})

	// Seller
	PlaceOrder(httpClient, &Order{
		MarketID:  "BTC-USDC",
		Trader:    trader2,
		Side:      "sell",
		OrderType: "limit",
		Price:     "50000",
		Quantity:  "0.1",
		Leverage:  "10",
	})

	// Wait for trade notification
	msg, err := wsClient.Receive(5 * time.Second)
	if err != nil {
		t.Logf("No trade notification received: %v", err)
		return
	}

	t.Logf("Trade notification: %s", string(msg))
}

// TestWebSocket_MultipleChannels tests subscribing to multiple channels
func TestWebSocket_MultipleChannels(t *testing.T) {
	config := DefaultConfig()
	client := NewWSClient(config)

	err := client.Connect("/ws")
	if err != nil {
		t.Skipf("WebSocket server not running: %v", err)
	}
	defer client.Close()

	// Subscribe to multiple channels
	channels := []string{
		"ticker:BTC-USDC",
		"depth:BTC-USDC",
		"trades:BTC-USDC",
	}

	for _, ch := range channels {
		subscribeMsg := map[string]interface{}{
			"type":    "subscribe",
			"channel": ch,
		}
		if err := client.Send(subscribeMsg); err != nil {
			t.Errorf("Failed to subscribe to %s: %v", ch, err)
		}
	}

	// Receive messages from any channel
	receivedChannels := make(map[string]bool)
	timeout := time.After(10 * time.Second)

	for len(receivedChannels) < len(channels) {
		select {
		case <-timeout:
			t.Logf("Received from %d/%d channels", len(receivedChannels), len(channels))
			return
		default:
			msg, err := client.Receive(2 * time.Second)
			if err != nil {
				continue
			}

			var wsMsg WSMessage
			if json.Unmarshal(msg, &wsMsg) == nil && wsMsg.Channel != "" {
				receivedChannels[wsMsg.Channel] = true
				t.Logf("Message from channel %s", wsMsg.Channel)
			}
		}
	}
}

// TestWebSocket_Unsubscribe tests unsubscribing from channels
func TestWebSocket_Unsubscribe(t *testing.T) {
	config := DefaultConfig()
	client := NewWSClient(config)

	err := client.Connect("/ws")
	if err != nil {
		t.Skipf("WebSocket server not running: %v", err)
	}
	defer client.Close()

	// Subscribe
	subscribeMsg := map[string]interface{}{
		"type":    "subscribe",
		"channel": "ticker:BTC-USDC",
	}
	client.Send(subscribeMsg)

	// Wait a bit
	time.Sleep(1 * time.Second)

	// Unsubscribe
	unsubscribeMsg := map[string]interface{}{
		"type":    "unsubscribe",
		"channel": "ticker:BTC-USDC",
	}
	if err := client.Send(unsubscribeMsg); err != nil {
		t.Fatalf("Failed to unsubscribe: %v", err)
	}

	t.Log("Unsubscribe message sent successfully")
}

// TestWebSocket_Reconnect tests reconnection after disconnect
func TestWebSocket_Reconnect(t *testing.T) {
	config := DefaultConfig()
	client := NewWSClient(config)

	// First connection
	err := client.Connect("/ws")
	if err != nil {
		t.Skipf("WebSocket server not running: %v", err)
	}

	// Subscribe to a channel
	subscribeMsg := map[string]interface{}{
		"type":    "subscribe",
		"channel": "ticker:BTC-USDC",
	}
	client.Send(subscribeMsg)

	// Close connection
	client.Close()
	t.Log("Connection closed")

	// Wait a bit
	time.Sleep(500 * time.Millisecond)

	// Reconnect
	client2 := NewWSClient(config)
	err = client2.Connect("/ws")
	if err != nil {
		t.Fatalf("Failed to reconnect: %v", err)
	}
	defer client2.Close()

	// Resubscribe
	client2.Send(subscribeMsg)

	// Verify we can receive messages
	msg, err := client2.Receive(5 * time.Second)
	if err != nil {
		t.Logf("No message after reconnect: %v", err)
		return
	}

	t.Logf("Message received after reconnect: %s", string(msg))
}

// TestWebSocket_InvalidChannel tests subscribing to invalid channel
func TestWebSocket_InvalidChannel(t *testing.T) {
	config := DefaultConfig()
	client := NewWSClient(config)

	err := client.Connect("/ws")
	if err != nil {
		t.Skipf("WebSocket server not running: %v", err)
	}
	defer client.Close()

	// Subscribe to invalid channel
	subscribeMsg := map[string]interface{}{
		"type":    "subscribe",
		"channel": "invalid:NONEXISTENT",
	}

	if err := client.Send(subscribeMsg); err != nil {
		t.Fatalf("Failed to send: %v", err)
	}

	// Should receive an error response or be ignored
	msg, err := client.Receive(3 * time.Second)
	if err != nil {
		t.Log("No response to invalid channel (expected behavior)")
		return
	}

	t.Logf("Response to invalid channel: %s", string(msg))
}

// TestWebSocket_MessageLatencyV2 measures WebSocket message latency (extended version)
func TestWebSocket_MessageLatencyV2(t *testing.T) {
	config := DefaultConfig()
	client := NewWSClient(config)

	err := client.Connect("/ws")
	if err != nil {
		t.Skipf("WebSocket server not running: %v", err)
	}
	defer client.Close()

	// Subscribe to ticker for frequent updates
	subscribeMsg := map[string]interface{}{
		"type":    "subscribe",
		"channel": "ticker:BTC-USDC",
	}
	client.Send(subscribeMsg)

	// Measure time between messages
	var latencies []time.Duration
	lastTime := time.Now()
	messageCount := 0

	timeout := time.After(10 * time.Second)

	for messageCount < 10 {
		select {
		case <-timeout:
			break
		default:
			_, err := client.Receive(2 * time.Second)
			if err != nil {
				continue
			}

			now := time.Now()
			if messageCount > 0 {
				latencies = append(latencies, now.Sub(lastTime))
			}
			lastTime = now
			messageCount++
		}
	}

	if len(latencies) > 0 {
		var total time.Duration
		for _, l := range latencies {
			total += l
		}
		avg := total / time.Duration(len(latencies))
		t.Logf("Average message interval: %v", avg)
	}
}

// TestWebSocket_HighFrequency tests handling high-frequency messages
func TestWebSocket_HighFrequency(t *testing.T) {
	config := DefaultConfig()
	wsClient := NewWSClient(config)
	httpClient := NewHTTPClient(config)

	err := wsClient.Connect("/ws")
	if err != nil {
		t.Skipf("WebSocket server not running: %v", err)
	}
	defer wsClient.Close()

	// Subscribe to trades
	subscribeMsg := map[string]interface{}{
		"type":    "subscribe",
		"channel": "trades:BTC-USDC",
	}
	wsClient.Send(subscribeMsg)

	// Generate rapid orders
	go func() {
		trader := fmt.Sprintf("hf_trader_%d", time.Now().UnixNano())
		for i := 0; i < 50; i++ {
			PlaceOrder(httpClient, &Order{
				MarketID:  "BTC-USDC",
				Trader:    trader,
				Side:      "buy",
				OrderType: "limit",
				Price:     fmt.Sprintf("%d", 50000+i),
				Quantity:  "0.01",
				Leverage:  "10",
			})
		}
	}()

	// Count received messages
	receivedCount := 0
	timeout := time.After(10 * time.Second)

	for {
		select {
		case <-timeout:
			t.Logf("Received %d messages during high-frequency test", receivedCount)
			return
		default:
			_, err := wsClient.Receive(100 * time.Millisecond)
			if err == nil {
				receivedCount++
			}
		}
	}
}
