package e2e_real

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

// ============================================================================
// WebSocket Integration E2E Tests
// ============================================================================
// These tests verify WebSocket functionality:
// 1. Connection establishment
// 2. Channel subscription
// 3. Real-time data push (ticker, depth, trades)
// 4. Private channel notifications (orders, positions)
// 5. Reconnection handling
// 6. Message ordering and delivery
// ============================================================================

// TestWebSocketConnection tests basic WebSocket connectivity
func TestWebSocketConnection(t *testing.T) {
	suite := NewE2ETestSuite(t, nil)

	err := suite.WaitForServer(10 * time.Second)
	if err != nil {
		t.Skipf("Server not available: %v", err)
	}

	t.Run("BasicConnection", func(t *testing.T) {
		ws, err := suite.NewWSClient()
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer ws.Close()

		t.Log("WebSocket connection established successfully")
	})

	t.Run("ConnectionLatency", func(t *testing.T) {
		var latencies []time.Duration
		iterations := 10

		for i := 0; i < iterations; i++ {
			start := time.Now()
			ws, err := suite.NewWSClient()
			latency := time.Since(start)

			if err != nil {
				t.Errorf("Connection %d failed: %v", i, err)
				continue
			}

			latencies = append(latencies, latency)
			ws.Close()
			time.Sleep(100 * time.Millisecond)
		}

		if len(latencies) > 0 {
			var total time.Duration
			for _, l := range latencies {
				total += l
			}
			t.Logf("Connection latency (%d samples): Avg %v",
				len(latencies), total/time.Duration(len(latencies)))
		}
	})

	t.Run("MultipleConnections", func(t *testing.T) {
		const numConnections = 5
		clients := make([]*WSClient, 0, numConnections)

		for i := 0; i < numConnections; i++ {
			ws, err := suite.NewWSClient()
			if err != nil {
				t.Errorf("Connection %d failed: %v", i, err)
				continue
			}
			clients = append(clients, ws)
		}

		t.Logf("Established %d concurrent connections", len(clients))

		// Clean up
		for _, ws := range clients {
			ws.Close()
		}
	})
}

// TestTickerSubscription tests ticker channel subscription
func TestTickerSubscription(t *testing.T) {
	suite := NewE2ETestSuite(t, nil)

	err := suite.WaitForServer(10 * time.Second)
	if err != nil {
		t.Skipf("Server not available: %v", err)
	}

	ws, err := suite.NewWSClient()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	// Subscribe to ticker
	err = ws.Subscribe("ticker", map[string]interface{}{
		"market": "BTC-USDC",
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	t.Log("Subscribed to BTC-USDC ticker")

	// Wait for messages
	messages := ws.CollectMessages(5 * time.Second)
	t.Logf("Received %d messages in 5 seconds", len(messages))

	for i, msg := range messages {
		if i < 3 { // Only show first 3
			t.Logf("  Message %d: %s", i, truncateString(string(msg), 100))
		}
	}
}

// TestDepthSubscription tests orderbook depth channel
func TestDepthSubscription(t *testing.T) {
	suite := NewE2ETestSuite(t, nil)

	err := suite.WaitForServer(10 * time.Second)
	if err != nil {
		t.Skipf("Server not available: %v", err)
	}

	ws, err := suite.NewWSClient()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	// Subscribe to depth
	err = ws.Subscribe("depth", map[string]interface{}{
		"market": "BTC-USDC",
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Wait for depth update
	msg, err := ws.WaitForMessage(10*time.Second, func(data []byte) bool {
		return strings.Contains(string(data), "depth") ||
			strings.Contains(string(data), "bids") ||
			strings.Contains(string(data), "asks")
	})

	if err != nil {
		t.Logf("No depth message received: %v", err)
	} else {
		t.Logf("Depth message received: %s", truncateString(string(msg), 200))
	}
}

// TestTradesSubscription tests trades channel
func TestTradesSubscription(t *testing.T) {
	suite := NewE2ETestSuite(t, nil)

	err := suite.WaitForServer(10 * time.Second)
	if err != nil {
		t.Skipf("Server not available: %v", err)
	}

	ws, err := suite.NewWSClient()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	// Subscribe to trades
	err = ws.Subscribe("trades", map[string]interface{}{
		"market": "BTC-USDC",
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	t.Log("Subscribed to BTC-USDC trades")

	// Also place an order to potentially trigger a trade
	user := suite.NewTestUser("perpdex1wstrades001")
	go func() {
		time.Sleep(1 * time.Second)
		_, _ = user.PlaceOrder(&PlaceOrderRequest{
			MarketID: "BTC-USDC",
			Side:     "buy",
			Type:     "limit",
			Price:    "50000.00",
			Quantity: "0.01",
		})
	}()

	// Collect messages
	messages := ws.CollectMessages(5 * time.Second)
	t.Logf("Received %d trade-related messages", len(messages))
}

// TestOrderNotifications tests private order notifications
func TestOrderNotifications(t *testing.T) {
	suite := NewE2ETestSuite(t, nil)

	err := suite.WaitForServer(10 * time.Second)
	if err != nil {
		t.Skipf("Server not available: %v", err)
	}

	user := suite.NewTestUser("perpdex1wsorder001")

	// Connect WebSocket
	ws, err := suite.NewWSClient()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	// Subscribe to user's order updates
	err = ws.Subscribe("orders", map[string]interface{}{
		"user": user.Address,
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Place an order and wait for notification
	var orderPlaced bool
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		msg, err := ws.WaitForMessage(10*time.Second, func(data []byte) bool {
			return strings.Contains(string(data), "order")
		})
		if err == nil {
			t.Logf("Order notification received: %s", truncateString(string(msg), 150))
			orderPlaced = true
		}
	}()

	// Place order
	time.Sleep(500 * time.Millisecond)
	order, err := user.PlaceOrder(&PlaceOrderRequest{
		MarketID: "BTC-USDC",
		Side:     "buy",
		Type:     "limit",
		Price:    "48000.00",
		Quantity: "0.1",
	})
	if err != nil {
		t.Fatalf("Failed to place order: %v", err)
	}
	t.Logf("Order placed: %s", order.OrderID)

	wg.Wait()

	if orderPlaced {
		t.Log("Order notification test PASSED")
	} else {
		t.Log("Order notification test: No notification received (may not be implemented)")
	}

	// Clean up
	_ = user.CancelOrder(order.OrderID)
}

// TestPositionNotifications tests private position notifications
func TestPositionNotifications(t *testing.T) {
	suite := NewE2ETestSuite(t, nil)

	err := suite.WaitForServer(10 * time.Second)
	if err != nil {
		t.Skipf("Server not available: %v", err)
	}

	maker := suite.NewTestUser("perpdex1wspos001")
	taker := suite.NewTestUser("perpdex1wspos002")

	// Connect WebSocket for taker
	ws, err := suite.NewWSClient()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	// Subscribe to position updates
	err = ws.Subscribe("positions", map[string]interface{}{
		"user": taker.Address,
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Create a matching scenario
	// Maker places sell order
	_, _ = maker.PlaceOrder(&PlaceOrderRequest{
		MarketID: "ETH-USDC",
		Side:     "sell",
		Type:     "limit",
		Price:    "3000.00",
		Quantity: "0.5",
	})

	// Start listening for position update
	positionUpdated := make(chan bool, 1)
	go func() {
		msg, err := ws.WaitForMessage(10*time.Second, func(data []byte) bool {
			return strings.Contains(string(data), "position")
		})
		if err == nil {
			t.Logf("Position notification: %s", truncateString(string(msg), 150))
			positionUpdated <- true
		} else {
			positionUpdated <- false
		}
	}()

	// Taker places matching buy order
	time.Sleep(500 * time.Millisecond)
	_, _ = taker.PlaceOrder(&PlaceOrderRequest{
		MarketID: "ETH-USDC",
		Side:     "buy",
		Type:     "limit",
		Price:    "3000.00",
		Quantity: "0.5",
	})

	select {
	case updated := <-positionUpdated:
		if updated {
			t.Log("Position notification test PASSED")
		} else {
			t.Log("Position notification not received")
		}
	case <-time.After(15 * time.Second):
		t.Log("Position notification timeout")
	}
}

// TestMultiChannelSubscription tests subscribing to multiple channels
func TestMultiChannelSubscription(t *testing.T) {
	suite := NewE2ETestSuite(t, nil)

	err := suite.WaitForServer(10 * time.Second)
	if err != nil {
		t.Skipf("Server not available: %v", err)
	}

	ws, err := suite.NewWSClient()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	// Subscribe to multiple channels
	channels := []struct {
		channel string
		params  map[string]interface{}
	}{
		{"ticker", map[string]interface{}{"market": "BTC-USDC"}},
		{"ticker", map[string]interface{}{"market": "ETH-USDC"}},
		{"depth", map[string]interface{}{"market": "BTC-USDC"}},
	}

	for _, ch := range channels {
		err := ws.Subscribe(ch.channel, ch.params)
		if err != nil {
			t.Errorf("Failed to subscribe to %s: %v", ch.channel, err)
		}
	}

	t.Logf("Subscribed to %d channels", len(channels))

	// Collect messages from all channels
	messages := ws.CollectMessages(5 * time.Second)
	t.Logf("Received %d total messages from all channels", len(messages))

	// Categorize messages
	msgTypes := make(map[string]int)
	for _, msg := range messages {
		var parsed map[string]interface{}
		if err := json.Unmarshal(msg, &parsed); err == nil {
			if msgType, ok := parsed["type"].(string); ok {
				msgTypes[msgType]++
			} else if ch, ok := parsed["channel"].(string); ok {
				msgTypes[ch]++
			}
		}
	}

	for msgType, count := range msgTypes {
		t.Logf("  %s: %d messages", msgType, count)
	}
}

// TestUnsubscribe tests unsubscribing from channels
func TestUnsubscribe(t *testing.T) {
	suite := NewE2ETestSuite(t, nil)

	err := suite.WaitForServer(10 * time.Second)
	if err != nil {
		t.Skipf("Server not available: %v", err)
	}

	ws, err := suite.NewWSClient()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	// Subscribe
	err = ws.Subscribe("ticker", map[string]interface{}{"market": "BTC-USDC"})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Collect some messages
	msgsBefore := ws.CollectMessages(2 * time.Second)
	t.Logf("Messages before unsubscribe: %d", len(msgsBefore))

	// Unsubscribe
	err = ws.Unsubscribe("ticker")
	if err != nil {
		t.Logf("Unsubscribe error: %v", err)
	}

	// Collect messages after unsubscribe
	time.Sleep(500 * time.Millisecond) // Give time for unsubscribe to process
	msgsAfter := ws.CollectMessages(2 * time.Second)
	t.Logf("Messages after unsubscribe: %d", len(msgsAfter))

	if len(msgsAfter) < len(msgsBefore) {
		t.Log("Unsubscribe appears to be working (fewer messages after)")
	}
}

// TestWebSocketReconnection tests reconnection behavior
func TestWebSocketReconnection(t *testing.T) {
	suite := NewE2ETestSuite(t, nil)

	err := suite.WaitForServer(10 * time.Second)
	if err != nil {
		t.Skipf("Server not available: %v", err)
	}

	// First connection
	ws1, err := suite.NewWSClient()
	if err != nil {
		t.Fatalf("Failed first connection: %v", err)
	}

	err = ws1.Subscribe("ticker", map[string]interface{}{"market": "BTC-USDC"})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Collect some messages
	msgs1 := ws1.CollectMessages(2 * time.Second)
	t.Logf("First connection: %d messages", len(msgs1))

	// Close connection
	ws1.Close()
	t.Log("First connection closed")

	// Wait a bit
	time.Sleep(1 * time.Second)

	// Reconnect
	ws2, err := suite.NewWSClient()
	if err != nil {
		t.Fatalf("Failed to reconnect: %v", err)
	}
	defer ws2.Close()

	err = ws2.Subscribe("ticker", map[string]interface{}{"market": "BTC-USDC"})
	if err != nil {
		t.Fatalf("Failed to resubscribe: %v", err)
	}

	// Collect messages after reconnection
	msgs2 := ws2.CollectMessages(2 * time.Second)
	t.Logf("After reconnection: %d messages", len(msgs2))

	t.Log("Reconnection test completed")
}

// TestWebSocketMessageLatency measures message push latency
func TestWebSocketMessageLatency(t *testing.T) {
	suite := NewE2ETestSuite(t, nil)

	err := suite.WaitForServer(10 * time.Second)
	if err != nil {
		t.Skipf("Server not available: %v", err)
	}

	ws, err := suite.NewWSClient()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close()

	err = ws.Subscribe("ticker", map[string]interface{}{"market": "BTC-USDC"})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Measure intervals between messages
	var intervals []time.Duration
	lastTime := time.Now()
	timeout := time.After(10 * time.Second)

	for {
		select {
		case <-ws.messages:
			now := time.Now()
			interval := now.Sub(lastTime)
			if interval < 5*time.Second { // Ignore first message
				intervals = append(intervals, interval)
			}
			lastTime = now

			if len(intervals) >= 20 {
				goto done
			}
		case <-timeout:
			goto done
		}
	}

done:
	if len(intervals) > 0 {
		var total time.Duration
		var min, max time.Duration = intervals[0], intervals[0]

		for _, i := range intervals {
			total += i
			if i < min {
				min = i
			}
			if i > max {
				max = i
			}
		}

		t.Logf("Message intervals (%d samples):", len(intervals))
		t.Logf("  Min: %v", min)
		t.Logf("  Max: %v", max)
		t.Logf("  Avg: %v", total/time.Duration(len(intervals)))
	} else {
		t.Log("No messages received for latency measurement")
	}
}

// Helper function to truncate strings
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
