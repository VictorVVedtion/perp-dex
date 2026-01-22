/**
 * PerpDEX WebSocket Client
 * Real-time data streaming
 */

type MessageHandler = (data: any) => void;

export interface WebSocketConfig {
  reconnectInterval: number;
  maxReconnectAttempts: number;
  pingInterval: number;
}

const DEFAULT_CONFIG: WebSocketConfig = {
  reconnectInterval: 5000,
  maxReconnectAttempts: 10,
  pingInterval: 30000,
};

/**
 * WebSocket Client for real-time data
 */
export class WebSocketClient {
  private _url: string;
  private _config: WebSocketConfig;
  private _ws: WebSocket | null = null;
  private _reconnectAttempts: number = 0;
  private _pingTimer: NodeJS.Timer | null = null;
  private _reconnectTimer: NodeJS.Timer | null = null;
  private _subscriptions: Map<string, Set<MessageHandler>> = new Map();
  private _connected: boolean = false;
  private _connecting: boolean = false;

  // Event handlers
  private _onConnect: (() => void) | null = null;
  private _onDisconnect: ((code: number, reason: string) => void) | null = null;
  private _onError: ((error: Error) => void) | null = null;

  constructor(url: string, config: Partial<WebSocketConfig> = {}) {
    this._url = url;
    this._config = { ...DEFAULT_CONFIG, ...config };
  }

  /**
   * Check if connected
   */
  get connected(): boolean {
    return this._connected;
  }

  /**
   * Connect to WebSocket server
   */
  connect(): void {
    if (this._connected || this._connecting) {
      return;
    }

    this._connecting = true;

    try {
      this._ws = new WebSocket(this._url);

      this._ws.onopen = () => {
        this._connected = true;
        this._connecting = false;
        this._reconnectAttempts = 0;

        // Resubscribe to all channels
        this._resubscribe();

        // Start ping timer
        this._startPingTimer();

        if (this._onConnect) {
          this._onConnect();
        }
      };

      this._ws.onclose = (event) => {
        this._connected = false;
        this._connecting = false;
        this._stopPingTimer();

        if (this._onDisconnect) {
          this._onDisconnect(event.code, event.reason);
        }

        // Attempt reconnect
        this._scheduleReconnect();
      };

      this._ws.onerror = (event) => {
        if (this._onError) {
          this._onError(new Error('WebSocket error'));
        }
      };

      this._ws.onmessage = (event) => {
        this._handleMessage(event.data);
      };
    } catch (error: any) {
      this._connecting = false;
      if (this._onError) {
        this._onError(error);
      }
      this._scheduleReconnect();
    }
  }

  /**
   * Disconnect from WebSocket server
   */
  disconnect(): void {
    this._stopPingTimer();
    this._stopReconnectTimer();
    this._reconnectAttempts = this._config.maxReconnectAttempts; // Prevent auto-reconnect

    if (this._ws) {
      this._ws.close();
      this._ws = null;
    }

    this._connected = false;
    this._connecting = false;
  }

  /**
   * Subscribe to a channel
   */
  subscribe(channel: string, handler: MessageHandler): void {
    if (!this._subscriptions.has(channel)) {
      this._subscriptions.set(channel, new Set());
    }
    this._subscriptions.get(channel)!.add(handler);

    // Send subscribe message if connected
    if (this._connected && this._ws) {
      this._send({
        action: 'subscribe',
        channel,
      });
    }
  }

  /**
   * Unsubscribe from a channel
   */
  unsubscribe(channel: string, handler?: MessageHandler): void {
    const handlers = this._subscriptions.get(channel);
    if (!handlers) return;

    if (handler) {
      handlers.delete(handler);
      if (handlers.size === 0) {
        this._subscriptions.delete(channel);
      }
    } else {
      this._subscriptions.delete(channel);
    }

    // Send unsubscribe message if connected
    if (this._connected && this._ws) {
      this._send({
        action: 'unsubscribe',
        channel,
      });
    }
  }

  // ============ Convenience Subscription Methods ============

  /**
   * Subscribe to ticker updates
   */
  subscribeTicker(marketId: string, handler: MessageHandler): void {
    this.subscribe(`ticker:${marketId}`, handler);
  }

  /**
   * Subscribe to orderbook depth updates
   */
  subscribeDepth(marketId: string, handler: MessageHandler): void {
    this.subscribe(`depth:${marketId}`, handler);
  }

  /**
   * Subscribe to trade updates
   */
  subscribeTrades(marketId: string, handler: MessageHandler): void {
    this.subscribe(`trades:${marketId}`, handler);
  }

  /**
   * Subscribe to position updates (requires auth)
   */
  subscribePositions(userId: string, handler: MessageHandler): void {
    this.subscribe(`positions:${userId}`, handler);
  }

  /**
   * Subscribe to order updates (requires auth)
   */
  subscribeOrders(userId: string, handler: MessageHandler): void {
    this.subscribe(`orders:${userId}`, handler);
  }

  /**
   * Subscribe to all tickers
   */
  subscribeAllTickers(handler: MessageHandler): void {
    this.subscribe('tickers', handler);
  }

  // ============ RiverPool Subscriptions ============

  /**
   * Subscribe to pool updates for a specific pool
   */
  subscribePool(poolId: string, handler: MessageHandler): void {
    this.subscribe(`riverpool:pool:${poolId}`, handler);
  }

  /**
   * Subscribe to all pool updates
   */
  subscribeAllPools(handler: MessageHandler): void {
    this.subscribe('riverpool:pools', handler);
  }

  /**
   * Subscribe to NAV updates for a pool
   */
  subscribeNAV(poolId: string, handler: MessageHandler): void {
    this.subscribe(`riverpool:nav:${poolId}`, handler);
  }

  /**
   * Subscribe to DDGuard updates for a pool
   */
  subscribeDDGuard(poolId: string, handler: MessageHandler): void {
    this.subscribe(`riverpool:ddguard:${poolId}`, handler);
  }

  /**
   * Subscribe to user's withdrawal updates
   */
  subscribeWithdrawals(userId: string, handler: MessageHandler): void {
    this.subscribe(`riverpool:withdrawals:${userId}`, handler);
  }

  /**
   * Subscribe to user's deposit confirmations
   */
  subscribeDeposits(userId: string, handler: MessageHandler): void {
    this.subscribe(`riverpool:deposits:${userId}`, handler);
  }

  /**
   * Subscribe to revenue events for a pool
   */
  subscribeRevenue(poolId: string, handler: MessageHandler): void {
    this.subscribe(`riverpool:revenue:${poolId}`, handler);
  }

  // ============ Event Handlers ============

  /**
   * Set connection handler
   */
  onConnect(handler: () => void): void {
    this._onConnect = handler;
  }

  /**
   * Set disconnection handler
   */
  onDisconnect(handler: (code: number, reason: string) => void): void {
    this._onDisconnect = handler;
  }

  /**
   * Set error handler
   */
  onError(handler: (error: Error) => void): void {
    this._onError = handler;
  }

  // ============ Private Methods ============

  private _send(message: object): void {
    if (this._ws && this._connected) {
      this._ws.send(JSON.stringify(message));
    }
  }

  private _handleMessage(data: string): void {
    try {
      const message = JSON.parse(data);

      // Handle pong
      if (message.type === 'pong') {
        return;
      }

      // Handle subscription confirmation
      if (message.type === 'subscribed' || message.type === 'unsubscribed') {
        return;
      }

      // Handle error
      if (message.type === 'error') {
        if (this._onError) {
          this._onError(new Error(message.data?.message || 'Unknown error'));
        }
        return;
      }

      // Dispatch to handlers
      const channel = message.channel;
      if (channel) {
        const handlers = this._subscriptions.get(channel);
        if (handlers) {
          handlers.forEach((handler) => {
            try {
              handler(message.data);
            } catch (error) {
              console.error('Error in message handler:', error);
            }
          });
        }
      }
    } catch (error) {
      console.error('Failed to parse WebSocket message:', error);
    }
  }

  private _resubscribe(): void {
    this._subscriptions.forEach((_, channel) => {
      this._send({
        action: 'subscribe',
        channel,
      });
    });
  }

  private _startPingTimer(): void {
    this._stopPingTimer();
    this._pingTimer = setInterval(() => {
      if (this._connected && this._ws) {
        this._send({ action: 'ping' });
      }
    }, this._config.pingInterval);
  }

  private _stopPingTimer(): void {
    if (this._pingTimer) {
      clearInterval(this._pingTimer);
      this._pingTimer = null;
    }
  }

  private _scheduleReconnect(): void {
    if (this._reconnectAttempts >= this._config.maxReconnectAttempts) {
      console.error('Max reconnect attempts reached');
      return;
    }

    this._stopReconnectTimer();
    this._reconnectTimer = setTimeout(() => {
      this._reconnectAttempts++;
      console.log(
        `Reconnecting... (attempt ${this._reconnectAttempts}/${this._config.maxReconnectAttempts})`
      );
      this.connect();
    }, this._config.reconnectInterval);
  }

  private _stopReconnectTimer(): void {
    if (this._reconnectTimer) {
      clearTimeout(this._reconnectTimer);
      this._reconnectTimer = null;
    }
  }
}

export default WebSocketClient;
