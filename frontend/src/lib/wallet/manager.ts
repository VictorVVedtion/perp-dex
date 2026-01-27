/**
 * PerpDEX Wallet Manager
 * Unified interface for multiple wallet providers
 */

// Note: Window.ethereum type is declared in metamask.ts

import type {
  IWallet,
  WalletProvider,
  WalletAccount,
  WalletState,
  WalletEvent,
  WalletEventHandler,
  ChainConfig,
} from './types';
import { KeplrWallet, PERPDEX_CHAIN_CONFIG } from './keplr';
import { MetaMaskWallet } from './metamask';
import { MockWallet } from './mock';

// Storage key for persisting wallet connection
const STORAGE_KEY = 'perpdex_wallet';

// Wallet registry - MetaMask is now the primary wallet (like Hyperliquid)
const walletRegistry: Record<WalletProvider, new (config?: ChainConfig) => IWallet> = {
  metamask: MetaMaskWallet,  // Primary wallet - EIP-712 signing
  keplr: KeplrWallet,        // Cosmos native wallet
  walletconnect: KeplrWallet, // Placeholder - would have separate implementation
  mock: MockWallet,          // Development mode only
};

/**
 * Wallet Manager - handles wallet connections and state
 */
export class WalletManager {
  private _wallet: IWallet | null = null;
  private _state: WalletState;
  private _chainConfig: ChainConfig;
  private _eventHandlers: Map<WalletEvent, Set<WalletEventHandler>> = new Map();
  private _autoConnectAttempted: boolean = false;

  constructor(chainConfig: ChainConfig = PERPDEX_CHAIN_CONFIG) {
    this._chainConfig = chainConfig;
    this._state = {
      connected: false,
      connecting: false,
      provider: null,
      account: null,
      chainId: null,
      error: null,
    };
  }

  /**
   * Get current wallet state
   */
  get state(): WalletState {
    return { ...this._state };
  }

  /**
   * Get current wallet instance
   */
  get wallet(): IWallet | null {
    return this._wallet;
  }

  /**
   * Check if a wallet provider is available
   */
  isProviderAvailable(provider: WalletProvider): boolean {
    switch (provider) {
      case 'keplr':
        return KeplrWallet.isInstalled();
      case 'metamask':
        return typeof window !== 'undefined' && !!window.ethereum;
      case 'walletconnect':
        return true; // WalletConnect is always available
      case 'mock':
        return true; // Mock is always available
      default:
        return false;
    }
  }

  /**
   * Get list of available wallet providers
   */
  getAvailableProviders(): WalletProvider[] {
    const providers: WalletProvider[] = ['keplr', 'metamask', 'walletconnect', 'mock'];
    return providers.filter((p) => this.isProviderAvailable(p));
  }

  /**
   * Connect to a wallet provider
   */
  async connect(provider: WalletProvider): Promise<WalletAccount> {
    if (this._state.connecting) {
      throw new Error('Connection already in progress');
    }

    this._updateState({ connecting: true, error: null });

    try {
      // Create wallet instance
      const WalletClass = walletRegistry[provider];
      if (!WalletClass) {
        throw new Error(`Unknown wallet provider: ${provider}`);
      }

      this._wallet = new WalletClass(this._chainConfig);

      // Setup event forwarding
      this._setupWalletEvents();

      // Connect
      const account = await this._wallet.connect();

      // Update state
      this._updateState({
        connected: true,
        connecting: false,
        provider,
        account,
        chainId: this._chainConfig.chainId,
      });

      // Persist connection
      this._saveConnection(provider);

      this._emit('connect', account);

      return account;
    } catch (error: any) {
      this._updateState({
        connected: false,
        connecting: false,
        provider: null,
        account: null,
        error: error.message,
      });

      throw error;
    }
  }

  /**
   * Disconnect from wallet
   */
  async disconnect(): Promise<void> {
    if (this._wallet) {
      await this._wallet.disconnect();
    }

    this._wallet = null;
    this._updateState({
      connected: false,
      connecting: false,
      provider: null,
      account: null,
      chainId: null,
      error: null,
    });

    this._clearConnection();
    this._emit('disconnect', null);
  }

  /**
   * Attempt to auto-connect to previously connected wallet
   */
  async autoConnect(): Promise<WalletAccount | null> {
    if (this._autoConnectAttempted) {
      return this._state.account;
    }

    this._autoConnectAttempted = true;

    const savedConnection = this._loadConnection();
    if (!savedConnection) {
      return null;
    }

    try {
      return await this.connect(savedConnection.provider);
    } catch (error) {
      console.warn('Auto-connect failed:', error);
      this._clearConnection();
      return null;
    }
  }

  /**
   * Sign a message
   */
  async signMessage(message: string): Promise<string> {
    if (!this._wallet || !this._state.account) {
      throw new Error('Wallet not connected');
    }

    if (this._wallet instanceof MetaMaskWallet) {
      return (this._wallet as MetaMaskWallet).signMessage(message);
    }

    if (this._wallet instanceof KeplrWallet) {
      const result = await (this._wallet as KeplrWallet).signArbitrary(
        this._state.account.address,
        message
      );
      return result.signature;
    }

    throw new Error('Sign message not supported for this wallet');
  }

  /**
   * Sign typed data using EIP-712 (MetaMask only)
   */
  async signTypedData<T extends Record<string, unknown>>(
    primaryType: string,
    message: T
  ): Promise<string> {
    if (!this._wallet || !this._state.account) {
      throw new Error('Wallet not connected');
    }

    if (this._wallet instanceof MetaMaskWallet) {
      return (this._wallet as MetaMaskWallet).signTypedData(primaryType, message);
    }

    throw new Error('EIP-712 signing only supported with MetaMask');
  }

  /**
   * Sign an order (MetaMask EIP-712)
   */
  async signOrder(order: {
    marketId: string;
    side: string;
    type: string;
    price: string;
    size: string;
    leverage: string;
  }): Promise<{ signature: string; nonce: number; expiry: number }> {
    if (!this._wallet || !this._state.account) {
      throw new Error('Wallet not connected');
    }

    if (this._wallet instanceof MetaMaskWallet) {
      return (this._wallet as MetaMaskWallet).signOrder(order);
    }

    throw new Error('Order signing only supported with MetaMask');
  }

  /**
   * Send a transaction
   */
  async sendTx(tx: Uint8Array): Promise<Uint8Array> {
    if (!this._wallet) {
      throw new Error('Wallet not connected');
    }

    return this._wallet.sendTx(tx);
  }

  /**
   * Get offline signer for direct signing
   */
  getOfflineSigner(): any {
    if (!this._wallet) {
      throw new Error('Wallet not connected');
    }

    if (this._wallet instanceof KeplrWallet) {
      return (this._wallet as KeplrWallet).getOfflineSigner();
    }

    throw new Error('Offline signer not available for this wallet');
  }

  /**
   * Subscribe to wallet events
   */
  on(event: WalletEvent, handler: WalletEventHandler): void {
    if (!this._eventHandlers.has(event)) {
      this._eventHandlers.set(event, new Set());
    }
    this._eventHandlers.get(event)!.add(handler);
  }

  /**
   * Unsubscribe from wallet events
   */
  off(event: WalletEvent, handler: WalletEventHandler): void {
    const handlers = this._eventHandlers.get(event);
    if (handlers) {
      handlers.delete(handler);
    }
  }

  /**
   * Update internal state
   */
  private _updateState(updates: Partial<WalletState>): void {
    this._state = { ...this._state, ...updates };
  }

  /**
   * Setup event forwarding from wallet
   */
  private _setupWalletEvents(): void {
    if (!this._wallet) return;

    this._wallet.on('accountChange', (account: WalletAccount) => {
      this._updateState({ account });
      this._emit('accountChange', account);
    });

    this._wallet.on('networkChange', (chainId: string) => {
      this._updateState({ chainId });
      this._emit('networkChange', chainId);
    });

    this._wallet.on('disconnect', () => {
      this.disconnect();
    });
  }

  /**
   * Emit event to handlers
   */
  private _emit(event: WalletEvent, data: any): void {
    const handlers = this._eventHandlers.get(event);
    if (handlers) {
      handlers.forEach((handler) => {
        try {
          handler(data);
        } catch (error) {
          console.error(`Error in ${event} handler:`, error);
        }
      });
    }
  }

  /**
   * Save connection to storage
   */
  private _saveConnection(provider: WalletProvider): void {
    if (typeof window === 'undefined') return;

    try {
      localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({
          provider,
          chainId: this._chainConfig.chainId,
          timestamp: Date.now(),
        })
      );
    } catch (error) {
      console.warn('Failed to save wallet connection:', error);
    }
  }

  /**
   * Load connection from storage
   */
  private _loadConnection(): { provider: WalletProvider; chainId: string } | null {
    if (typeof window === 'undefined') return null;

    try {
      const saved = localStorage.getItem(STORAGE_KEY);
      if (!saved) return null;

      const data = JSON.parse(saved);

      // Check if connection is recent (within 24 hours)
      if (Date.now() - data.timestamp > 24 * 60 * 60 * 1000) {
        this._clearConnection();
        return null;
      }

      return {
        provider: data.provider,
        chainId: data.chainId,
      };
    } catch (error) {
      return null;
    }
  }

  /**
   * Clear saved connection
   */
  private _clearConnection(): void {
    if (typeof window === 'undefined') return;

    try {
      localStorage.removeItem(STORAGE_KEY);
    } catch (error) {
      console.warn('Failed to clear wallet connection:', error);
    }
  }
}

// Singleton instance
let _instance: WalletManager | null = null;

/**
 * Get the wallet manager singleton
 */
export function getWalletManager(
  chainConfig?: ChainConfig
): WalletManager {
  if (!_instance) {
    _instance = new WalletManager(chainConfig);
  }
  return _instance;
}

export default WalletManager;
