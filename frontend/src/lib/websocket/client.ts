/**
 * WebSocket Client for PerpDEX
 * Handles real-time data streaming with automatic reconnection
 */

export interface WSMessage {
  type: string;
  channel: string;
  data: any;
}

export interface Ticker {
  marketId: string;
  markPrice: string;
  indexPrice: string;
  lastPrice: string;
  high24h: string;
  low24h: string;
  volume24h: string;
  change24h: string;
  fundingRate: string;
  nextFunding: number;
  timestamp: number;
}

export interface PriceLevel {
  price: string;
  quantity: string;
}

export interface Orderbook {
  marketId: string;
  bids: PriceLevel[];
  asks: PriceLevel[];
  timestamp: number;
}

export interface Trade {
  tradeId: string;
  marketId: string;
  price: string;
  quantity: string;
  side: 'buy' | 'sell';
  timestamp: number;
}

type MessageHandler = (data: any) => void;

export class WSClient {
  private ws: WebSocket | null = null;
  private url: string;
  private reconnectAttempts = 0;
  private maxReconnects = 10;
  private reconnectDelay = 1000;
  private maxReconnectDelay = 30000; // CRITICAL FIX: Maximum 30 second delay cap
  private subscriptions = new Map<string, Set<MessageHandler>>();
  private pendingSubscriptions: string[] = [];
  private heartbeatInterval: ReturnType<typeof setInterval> | null = null;
  private heartbeatTimeout: ReturnType<typeof setTimeout> | null = null;
  private isConnecting = false;
  private onConnectCallbacks: (() => void)[] = [];
  private onDisconnectCallbacks: (() => void)[] = [];
  private onErrorCallbacks: ((error: Event) => void)[] = [];

  constructor(url: string) {
    this.url = url;
  }

  /**
   * Connect to WebSocket server
   */
  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN || this.isConnecting) {
      return;
    }

    this.isConnecting = true;

    try {
      this.ws = new WebSocket(this.url);

      this.ws.onopen = this.handleOpen.bind(this);
      this.ws.onmessage = this.handleMessage.bind(this);
      this.ws.onclose = this.handleClose.bind(this);
      this.ws.onerror = this.handleError.bind(this);
    } catch (error) {
      console.error('WebSocket connection failed:', error);
      this.isConnecting = false;
      this.scheduleReconnect();
    }
  }

  /**
   * Disconnect from WebSocket server
   */
  disconnect(): void {
    this.stopHeartbeat();
    this.reconnectAttempts = this.maxReconnects; // Prevent reconnection

    if (this.ws) {
      this.ws.close(1000, 'Client disconnect');
      this.ws = null;
    }

    this.onDisconnectCallbacks.forEach((cb) => cb());
  }

  /**
   * Subscribe to a channel
   */
  subscribe(channel: string, handler: MessageHandler): void {
    if (!this.subscriptions.has(channel)) {
      this.subscriptions.set(channel, new Set());

      // Send subscribe message if connected
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.send({ type: 'subscribe', channel });
      } else {
        // Queue for when connected
        this.pendingSubscriptions.push(channel);
      }
    }

    this.subscriptions.get(channel)!.add(handler);
  }

  /**
   * Unsubscribe from a channel
   */
  unsubscribe(channel: string, handler?: MessageHandler): void {
    const handlers = this.subscriptions.get(channel);

    if (handlers) {
      if (handler) {
        handlers.delete(handler);
      } else {
        handlers.clear();
      }

      if (handlers.size === 0) {
        this.subscriptions.delete(channel);
        if (this.ws?.readyState === WebSocket.OPEN) {
          this.send({ type: 'unsubscribe', channel });
        }
      }
    }
  }

  /**
   * Check if connected
   */
  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  /**
   * Add connection callback
   */
  onConnect(callback: () => void): void {
    this.onConnectCallbacks.push(callback);
  }

  /**
   * Add disconnection callback
   */
  onDisconnect(callback: () => void): void {
    this.onDisconnectCallbacks.push(callback);
  }

  /**
   * Add error callback
   */
  onError(callback: (error: Event) => void): void {
    this.onErrorCallbacks.push(callback);
  }

  /**
   * Send a message to the server
   */
  private send(message: object): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  /**
   * Handle WebSocket open
   */
  private handleOpen(): void {
    console.log('WebSocket connected');
    this.isConnecting = false;
    this.reconnectAttempts = 0;

    // Start heartbeat
    this.startHeartbeat();

    // Resubscribe to channels
    this.subscriptions.forEach((_, channel) => {
      this.send({ type: 'subscribe', channel });
    });

    // Subscribe to pending channels
    this.pendingSubscriptions.forEach((channel) => {
      if (!this.subscriptions.has(channel)) {
        this.subscriptions.set(channel, new Set());
      }
      this.send({ type: 'subscribe', channel });
    });
    this.pendingSubscriptions = [];

    // Notify callbacks
    this.onConnectCallbacks.forEach((cb) => cb());
  }

  /**
   * Handle incoming messages
   */
  private handleMessage(event: MessageEvent): void {
    try {
      const message: WSMessage = JSON.parse(event.data);

      // Handle pong for heartbeat
      if (message.type === 'pong') {
        this.resetHeartbeatTimeout();
        return;
      }

      // Handle subscription confirmations
      if (message.type === 'subscribed' || message.type === 'unsubscribed') {
        console.log(`WebSocket ${message.type}: ${message.channel}`);
        return;
      }

      // Dispatch to handlers
      const handlers = this.subscriptions.get(message.channel);
      if (handlers) {
        handlers.forEach((handler) => {
          try {
            handler(message.data);
          } catch (error) {
            console.error('Handler error:', error);
          }
        });
      }
    } catch (error) {
      console.error('Failed to parse WebSocket message:', error);
    }
  }

  /**
   * Handle WebSocket close
   */
  private handleClose(event: CloseEvent): void {
    console.log('WebSocket disconnected:', event.code, event.reason);
    this.isConnecting = false;
    this.stopHeartbeat();

    this.onDisconnectCallbacks.forEach((cb) => cb());

    // Attempt to reconnect if not a clean close
    if (event.code !== 1000) {
      this.scheduleReconnect();
    }
  }

  /**
   * Handle WebSocket error
   */
  private handleError(error: Event): void {
    console.error('WebSocket error:', error);
    this.isConnecting = false;

    this.onErrorCallbacks.forEach((cb) => cb(error));
  }

  /**
   * Schedule a reconnection attempt
   * CRITICAL FIX: Added maximum delay cap to prevent excessive wait times
   */
  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnects) {
      console.error('Max reconnection attempts reached');
      return;
    }

    // Calculate exponential backoff with cap
    const exponentialDelay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts);
    const delay = Math.min(exponentialDelay, this.maxReconnectDelay); // Cap at maxReconnectDelay
    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts + 1}/${this.maxReconnects})`);

    setTimeout(() => {
      this.reconnectAttempts++;
      this.connect();
    }, delay);
  }

  /**
   * Start heartbeat
   */
  private startHeartbeat(): void {
    this.stopHeartbeat();

    // Send ping every 30 seconds
    this.heartbeatInterval = setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.send({ type: 'ping' });
        this.startHeartbeatTimeout();
      }
    }, 30000);
  }

  /**
   * Stop heartbeat
   */
  private stopHeartbeat(): void {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
    if (this.heartbeatTimeout) {
      clearTimeout(this.heartbeatTimeout);
      this.heartbeatTimeout = null;
    }
  }

  /**
   * Start heartbeat timeout
   */
  private startHeartbeatTimeout(): void {
    if (this.heartbeatTimeout) {
      clearTimeout(this.heartbeatTimeout);
    }

    // If no pong received within 10 seconds, reconnect
    this.heartbeatTimeout = setTimeout(() => {
      console.warn('Heartbeat timeout, reconnecting...');
      this.ws?.close();
      this.scheduleReconnect();
    }, 10000);
  }

  /**
   * Reset heartbeat timeout
   */
  private resetHeartbeatTimeout(): void {
    if (this.heartbeatTimeout) {
      clearTimeout(this.heartbeatTimeout);
      this.heartbeatTimeout = null;
    }
  }
}

export default WSClient;
