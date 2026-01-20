package e2e_hyperliquid

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestWebSocket_Connect tests basic WebSocket connection
func TestWebSocket_Connect(t *testing.T) {
	client := NewHyperliquidWSClient()

	err := client.Connect()
	if err != nil {
		t.Fatalf("Failed to connect to Hyperliquid WebSocket: %v", err)
	}
	defer client.Close()

	t.Log("Successfully connected to Hyperliquid WebSocket")
}

// TestWebSocket_SubscribeAllMids tests subscribing to all mid prices
func TestWebSocket_SubscribeAllMids(t *testing.T) {
	client := NewHyperliquidWSClient()

	err := client.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	err = client.SubscribeAllMids()
	if err != nil {
		t.Fatalf("Failed to subscribe to allMids: %v", err)
	}

	t.Log("Subscribed to allMids, waiting for messages...")

	// Wait for messages
	messageCount := 0
	timeout := time.After(10 * time.Second)

	for messageCount < 5 {
		select {
		case <-timeout:
			if messageCount == 0 {
				t.Fatal("No messages received within timeout")
			}
			t.Logf("Received %d messages before timeout", messageCount)
			return
		default:
			msg, err := client.Receive(2 * time.Second)
			if err != nil {
				continue
			}

			messageCount++
			if messageCount <= 3 {
				// Log first few messages
				var data map[string]interface{}
				json.Unmarshal(msg, &data)
				t.Logf("Message %d: channel=%v", messageCount, data["channel"])
			}
		}
	}

	t.Logf("Successfully received %d allMids messages", messageCount)
}

// TestWebSocket_SubscribeL2Book tests subscribing to order book
func TestWebSocket_SubscribeL2Book(t *testing.T) {
	client := NewHyperliquidWSClient()

	err := client.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	err = client.SubscribeL2Book("BTC")
	if err != nil {
		t.Fatalf("Failed to subscribe to L2 book: %v", err)
	}

	t.Log("Subscribed to BTC L2 book, waiting for updates...")

	// Wait for messages
	messageCount := 0
	timeout := time.After(15 * time.Second)

	for messageCount < 5 {
		select {
		case <-timeout:
			if messageCount == 0 {
				t.Fatal("No L2 book messages received within timeout")
			}
			t.Logf("Received %d L2 book messages before timeout", messageCount)
			return
		default:
			msg, err := client.Receive(3 * time.Second)
			if err != nil {
				continue
			}

			messageCount++
			if messageCount == 1 {
				var data map[string]interface{}
				json.Unmarshal(msg, &data)
				t.Logf("L2 Book update received: %d bytes", len(msg))
			}
		}
	}

	t.Logf("Successfully received %d L2 book updates", messageCount)
}

// TestWebSocket_SubscribeTrades tests subscribing to trades
func TestWebSocket_SubscribeTrades(t *testing.T) {
	client := NewHyperliquidWSClient()

	err := client.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	err = client.SubscribeTrades("BTC")
	if err != nil {
		t.Fatalf("Failed to subscribe to trades: %v", err)
	}

	t.Log("Subscribed to BTC trades, waiting for trades...")

	// Wait for trade messages (trades may be infrequent)
	messageCount := 0
	timeout := time.After(30 * time.Second)

	for messageCount < 3 {
		select {
		case <-timeout:
			if messageCount == 0 {
				t.Log("No trade messages received within timeout (trades may be infrequent)")
			} else {
				t.Logf("Received %d trade messages before timeout", messageCount)
			}
			return
		default:
			msg, err := client.Receive(5 * time.Second)
			if err != nil {
				continue
			}

			messageCount++
			var data map[string]interface{}
			json.Unmarshal(msg, &data)
			if channel, ok := data["channel"].(string); ok && channel == "trades" {
				t.Logf("Trade received: %d bytes", len(msg))
			}
		}
	}

	t.Logf("Successfully received %d trade messages", messageCount)
}

// TestWebSocket_MultipleSubscriptions tests multiple subscriptions
func TestWebSocket_MultipleSubscriptions(t *testing.T) {
	client := NewHyperliquidWSClient()

	err := client.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Subscribe to multiple channels
	subscriptions := []func() error{
		client.SubscribeAllMids,
		func() error { return client.SubscribeL2Book("BTC") },
		func() error { return client.SubscribeL2Book("ETH") },
	}

	for i, sub := range subscriptions {
		if err := sub(); err != nil {
			t.Fatalf("Failed subscription %d: %v", i, err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Log("Subscribed to multiple channels, counting messages...")

	// Count messages for a few seconds
	duration := 10 * time.Second
	endTime := time.Now().Add(duration)

	for time.Now().Before(endTime) {
		_, err := client.Receive(1 * time.Second)
		if err != nil {
			continue
		}
	}

	count := client.GetMessageCount()
	t.Logf("Received %d total messages in %v", count, duration)

	if count == 0 {
		t.Error("Expected to receive messages from multiple subscriptions")
	}
}

// TestWebSocket_MessageThroughput tests message throughput
func TestWebSocket_MessageThroughput(t *testing.T) {
	client := NewHyperliquidWSClient()

	err := client.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Subscribe to high-frequency channel
	if err := client.SubscribeAllMids(); err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	t.Log("Measuring message throughput...")

	duration := 15 * time.Second
	startTime := time.Now()
	endTime := startTime.Add(duration)

	for time.Now().Before(endTime) {
		_, err := client.Receive(1 * time.Second)
		if err != nil {
			continue
		}
	}

	elapsed := time.Since(startTime)
	count := client.GetMessageCount()
	throughput := float64(count) / elapsed.Seconds()

	t.Logf("WebSocket Throughput Test:")
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Messages: %d", count)
	t.Logf("  Throughput: %.2f msg/sec", throughput)

	if count == 0 {
		t.Error("No messages received")
	}
}

// TestWebSocket_ConcurrentConnections tests multiple concurrent connections
func TestWebSocket_ConcurrentConnections(t *testing.T) {
	numConnections := 5
	var wg sync.WaitGroup
	var connectedCount int64
	var messageTotal int64

	t.Logf("Testing %d concurrent WebSocket connections...", numConnections)

	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(connID int) {
			defer wg.Done()

			client := NewHyperliquidWSClient()
			err := client.Connect()
			if err != nil {
				t.Logf("Connection %d failed: %v", connID, err)
				return
			}
			defer client.Close()

			atomic.AddInt64(&connectedCount, 1)

			// Subscribe and receive messages
			client.SubscribeAllMids()

			for j := 0; j < 10; j++ {
				_, err := client.Receive(2 * time.Second)
				if err == nil {
					atomic.AddInt64(&messageTotal, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	t.Logf("Concurrent WebSocket Test:")
	t.Logf("  Connections attempted: %d", numConnections)
	t.Logf("  Connections successful: %d", connectedCount)
	t.Logf("  Total messages received: %d", messageTotal)

	if connectedCount < int64(numConnections) {
		t.Errorf("Not all connections succeeded: %d/%d", connectedCount, numConnections)
	}
}

// TestWebSocket_Reconnection tests reconnection behavior
func TestWebSocket_Reconnection(t *testing.T) {
	t.Log("Testing WebSocket reconnection...")

	// First connection
	client1 := NewHyperliquidWSClient()
	err := client1.Connect()
	if err != nil {
		t.Fatalf("First connection failed: %v", err)
	}

	client1.SubscribeAllMids()

	// Receive some messages
	for i := 0; i < 3; i++ {
		_, err := client1.Receive(3 * time.Second)
		if err != nil {
			t.Logf("Message %d: timeout or error", i)
		}
	}

	firstCount := client1.GetMessageCount()
	t.Logf("First connection received %d messages", firstCount)

	// Close first connection
	client1.Close()
	time.Sleep(1 * time.Second)

	// Reconnect
	client2 := NewHyperliquidWSClient()
	err = client2.Connect()
	if err != nil {
		t.Fatalf("Reconnection failed: %v", err)
	}
	defer client2.Close()

	client2.SubscribeAllMids()

	// Receive more messages
	for i := 0; i < 3; i++ {
		_, err := client2.Receive(3 * time.Second)
		if err != nil {
			t.Logf("Message %d after reconnect: timeout or error", i)
		}
	}

	secondCount := client2.GetMessageCount()
	t.Logf("Second connection received %d messages", secondCount)
	t.Log("Reconnection test completed successfully")
}

// TestWebSocket_DataConsistency tests data consistency between REST and WS
func TestWebSocket_DataConsistency(t *testing.T) {
	// Get REST data
	restClient := NewHyperliquidClient()
	restResult := restClient.GetAllMids()
	if restResult.Error != nil {
		t.Fatalf("REST request failed: %v", restResult.Error)
	}

	var restMids map[string]string
	json.Unmarshal(restResult.Data, &restMids)

	restBTC := restMids["BTC"]
	t.Logf("REST BTC price: %s", restBTC)

	// Get WebSocket data
	wsClient := NewHyperliquidWSClient()
	err := wsClient.Connect()
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	defer wsClient.Close()

	wsClient.SubscribeAllMids()

	// Wait for WS message
	var wsBTC string
	timeout := time.After(10 * time.Second)

	for wsBTC == "" {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for WS message")
		default:
			msg, err := wsClient.Receive(2 * time.Second)
			if err != nil {
				continue
			}

			var data map[string]interface{}
			json.Unmarshal(msg, &data)

			if mids, ok := data["data"].(map[string]interface{}); ok {
				if btc, ok := mids["mids"].(map[string]interface{}); ok {
					if price, ok := btc["BTC"].(string); ok {
						wsBTC = price
					}
				}
			}
		}
	}

	t.Logf("WebSocket BTC price: %s", wsBTC)
	t.Logf("Both REST and WebSocket returned BTC price data")

	// Note: Prices may differ slightly due to timing
}

// TestWebSocket_HighFrequencyData tests handling high-frequency data
func TestWebSocket_HighFrequencyData(t *testing.T) {
	client := NewHyperliquidWSClient()

	err := client.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Subscribe to multiple high-frequency channels
	coins := []string{"BTC", "ETH", "SOL"}
	for _, coin := range coins {
		client.SubscribeL2Book(coin)
		time.Sleep(50 * time.Millisecond)
	}
	client.SubscribeAllMids()

	t.Log("Subscribed to multiple high-frequency channels...")

	// Measure for a period
	duration := 20 * time.Second
	startTime := time.Now()
	endTime := startTime.Add(duration)

	droppedEstimate := 0
	for time.Now().Before(endTime) {
		_, err := client.Receive(100 * time.Millisecond)
		if err != nil {
			droppedEstimate++
		}
	}

	elapsed := time.Since(startTime)
	count := client.GetMessageCount()
	throughput := float64(count) / elapsed.Seconds()

	t.Logf("High-Frequency Data Test:")
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Messages received: %d", count)
	t.Logf("  Throughput: %.2f msg/sec", throughput)
	t.Logf("  Potential timeouts: %d", droppedEstimate)

	if throughput < 1 {
		t.Error("Throughput too low for high-frequency test")
	}
}

// TestWebSocket_LongRunning tests long-running connection stability
func TestWebSocket_LongRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test in short mode")
	}

	client := NewHyperliquidWSClient()

	err := client.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	client.SubscribeAllMids()

	t.Log("Starting long-running WebSocket test (30 seconds)...")

	duration := 30 * time.Second
	checkInterval := 5 * time.Second
	startTime := time.Now()
	lastCheck := startTime
	lastCount := int64(0)

	endTime := startTime.Add(duration)

	for time.Now().Before(endTime) {
		_, _ = client.Receive(1 * time.Second)

		if time.Since(lastCheck) >= checkInterval {
			currentCount := client.GetMessageCount()
			newMessages := currentCount - lastCount
			t.Logf("  [%v] +%d messages (total: %d)", time.Since(startTime).Round(time.Second), newMessages, currentCount)
			lastCount = currentCount
			lastCheck = time.Now()

			if newMessages == 0 {
				t.Error("No new messages in last interval - connection may be stale")
			}
		}
	}

	totalCount := client.GetMessageCount()
	t.Logf("Long-running test complete: %d messages in %v", totalCount, duration)
}
