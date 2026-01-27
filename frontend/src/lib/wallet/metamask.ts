/**
 * MetaMask Wallet Integration for PerpDEX
 * Supports EVM-compatible signing via EIP-712
 * Similar to Hyperliquid's approach
 */

import type {
  IWallet,
  WalletAccount,
  WalletEvent,
  WalletEventHandler,
  ChainConfig,
} from './types';

// EIP-712 typed data types
interface TypedDataDomain {
  name: string;
  version: string;
  chainId: number;
  verifyingContract?: string;
}

interface TypedDataTypes {
  [key: string]: { name: string; type: string }[];
}

// Ethereum provider interface
interface EthereumProvider {
  isMetaMask?: boolean;
  request: (args: { method: string; params?: unknown[] }) => Promise<unknown>;
  on: (event: string, handler: (...args: unknown[]) => void) => void;
  removeListener: (event: string, handler: (...args: unknown[]) => void) => void;
  selectedAddress: string | null;
}

// Extend Window interface
declare global {
  interface Window {
    ethereum?: EthereumProvider;
  }
}

// PerpDEX EIP-712 Domain
const PERPDEX_DOMAIN: TypedDataDomain = {
  name: 'PerpDEX',
  version: '1',
  chainId: 1, // Will be updated on connect
};

// Common EIP-712 types for trading
const PERPDEX_TYPES: TypedDataTypes = {
  // Order signing type
  Order: [
    { name: 'marketId', type: 'string' },
    { name: 'side', type: 'string' },
    { name: 'type', type: 'string' },
    { name: 'price', type: 'string' },
    { name: 'size', type: 'string' },
    { name: 'leverage', type: 'string' },
    { name: 'nonce', type: 'uint256' },
    { name: 'expiry', type: 'uint256' },
  ],
  // Cancel order type
  CancelOrder: [
    { name: 'orderId', type: 'string' },
    { name: 'nonce', type: 'uint256' },
  ],
  // Withdrawal request type
  Withdrawal: [
    { name: 'amount', type: 'string' },
    { name: 'destination', type: 'address' },
    { name: 'nonce', type: 'uint256' },
  ],
  // Generic action type
  Action: [
    { name: 'action', type: 'string' },
    { name: 'payload', type: 'string' },
    { name: 'nonce', type: 'uint256' },
    { name: 'timestamp', type: 'uint256' },
  ],
};

export class MetaMaskWallet implements IWallet {
  readonly provider = 'metamask' as const;
  private _connected: boolean = false;
  private _account: WalletAccount | null = null;
  private _chainId: number = 1;
  private _chainConfig: ChainConfig;
  private _eventHandlers: Map<WalletEvent, Set<WalletEventHandler>> = new Map();
  private _accountsChangedHandler: ((accounts: string[]) => void) | null = null;
  private _chainChangedHandler: ((chainId: string) => void) | null = null;
  private _nonce: number = 0;

  constructor(chainConfig?: ChainConfig) {
    this._chainConfig = chainConfig || {} as ChainConfig;

    // Setup MetaMask event listeners
    if (typeof window !== 'undefined' && window.ethereum) {
      this._accountsChangedHandler = (accounts: string[]) => {
        this._handleAccountsChanged(accounts);
      };
      this._chainChangedHandler = (chainId: string) => {
        this._handleChainChanged(chainId);
      };

      window.ethereum.on('accountsChanged', this._accountsChangedHandler as (...args: unknown[]) => void);
      window.ethereum.on('chainChanged', this._chainChangedHandler as (...args: unknown[]) => void);
    }
  }

  /**
   * Cleanup resources
   */
  destroy(): void {
    if (typeof window !== 'undefined' && window.ethereum) {
      if (this._accountsChangedHandler) {
        window.ethereum.removeListener('accountsChanged', this._accountsChangedHandler as (...args: unknown[]) => void);
      }
      if (this._chainChangedHandler) {
        window.ethereum.removeListener('chainChanged', this._chainChangedHandler as (...args: unknown[]) => void);
      }
    }
    this._eventHandlers.clear();
    this._connected = false;
    this._account = null;
  }

  get connected(): boolean {
    return this._connected;
  }

  get account(): WalletAccount | null {
    return this._account;
  }

  /**
   * Check if MetaMask is installed
   */
  static isInstalled(): boolean {
    return typeof window !== 'undefined' && !!window.ethereum?.isMetaMask;
  }

  /**
   * Connect to MetaMask wallet
   */
  async connect(): Promise<WalletAccount> {
    if (!MetaMaskWallet.isInstalled() || !window.ethereum) {
      throw new Error('MetaMask wallet is not installed');
    }

    try {
      // Request account access
      const accounts = (await window.ethereum.request({
        method: 'eth_requestAccounts',
      })) as string[];

      if (!accounts || accounts.length === 0) {
        throw new Error('No accounts found in MetaMask');
      }

      // Get chain ID
      const chainIdHex = (await window.ethereum.request({
        method: 'eth_chainId',
      })) as string;
      this._chainId = parseInt(chainIdHex, 16);

      // Create account object
      const address = accounts[0].toLowerCase();

      // Generate a pseudo pubKey from address (for compatibility)
      const pubKeyBytes = new Uint8Array(33);
      const addressBytes = this._hexToBytes(address.slice(2));
      pubKeyBytes.set(addressBytes.slice(0, Math.min(20, 33)));

      this._account = {
        address: address, // Keep 0x format
        pubKey: pubKeyBytes,
        algo: 'secp256k1',
        name: `MetaMask (${address.slice(0, 6)}...${address.slice(-4)})`,
      };

      this._connected = true;
      this._emit('connect', this._account);

      return this._account;
    } catch (error: unknown) {
      this._connected = false;
      this._account = null;
      const message = error instanceof Error ? error.message : 'Unknown error';
      throw new Error(`Failed to connect to MetaMask: ${message}`);
    }
  }

  /**
   * Disconnect from MetaMask wallet
   */
  async disconnect(): Promise<void> {
    this._connected = false;
    this._account = null;
    this._emit('disconnect', null);
  }

  /**
   * Get current account
   */
  async getAccount(): Promise<WalletAccount> {
    if (!this._connected || !this._account) {
      return this.connect();
    }
    return this._account;
  }

  /**
   * Sign EIP-712 typed data (primary signing method for MetaMask)
   */
  async signTypedData<T extends Record<string, unknown>>(
    primaryType: string,
    message: T,
    customTypes?: TypedDataTypes
  ): Promise<string> {
    if (!this._connected || !this._account || !window.ethereum) {
      throw new Error('Wallet not connected');
    }

    const domain = {
      ...PERPDEX_DOMAIN,
      chainId: this._chainId,
    };

    const types = customTypes || PERPDEX_TYPES;

    // EIP-712 signature request
    const signature = (await window.ethereum.request({
      method: 'eth_signTypedData_v4',
      params: [
        this._account.address,
        JSON.stringify({
          types: {
            EIP712Domain: [
              { name: 'name', type: 'string' },
              { name: 'version', type: 'string' },
              { name: 'chainId', type: 'uint256' },
            ],
            ...types,
          },
          primaryType,
          domain,
          message,
        }),
      ],
    })) as string;

    return signature;
  }

  /**
   * Sign an order using EIP-712
   */
  async signOrder(order: {
    marketId: string;
    side: string;
    type: string;
    price: string;
    size: string;
    leverage: string;
  }): Promise<{ signature: string; nonce: number; expiry: number }> {
    const nonce = this._getNextNonce();
    const expiry = Math.floor(Date.now() / 1000) + 300; // 5 minutes

    const message = {
      ...order,
      nonce,
      expiry,
    };

    const signature = await this.signTypedData('Order', message);

    return { signature, nonce, expiry };
  }

  /**
   * Sign a generic action using EIP-712
   */
  async signAction(
    action: string,
    payload: Record<string, unknown>
  ): Promise<{ signature: string; nonce: number; timestamp: number }> {
    const nonce = this._getNextNonce();
    const timestamp = Math.floor(Date.now() / 1000);

    const message = {
      action,
      payload: JSON.stringify(payload),
      nonce,
      timestamp,
    };

    const signature = await this.signTypedData('Action', message);

    return { signature, nonce, timestamp };
  }

  /**
   * Sign arbitrary message (personal_sign)
   */
  async signMessage(message: string): Promise<string> {
    if (!this._connected || !this._account || !window.ethereum) {
      throw new Error('Wallet not connected');
    }

    const signature = (await window.ethereum.request({
      method: 'personal_sign',
      params: [message, this._account.address],
    })) as string;

    return signature;
  }

  /**
   * Sign direct - adapts to EIP-712 for Cosmos compatibility
   * This is used for backward compatibility with Cosmos signing interfaces
   */
  async signDirect(
    signerAddress: string,
    signDoc: unknown
  ): Promise<{ signed: unknown; signature: unknown }> {
    // Convert Cosmos signDoc to EIP-712 action
    const { signature, nonce, timestamp } = await this.signAction(
      'cosmos_sign_direct',
      { signDoc }
    );

    return {
      signed: signDoc,
      signature: {
        signature,
        pub_key: {
          type: 'ethereum/secp256k1',
          value: this._account?.address || '',
        },
        nonce,
        timestamp,
      },
    };
  }

  /**
   * Sign amino - adapts to EIP-712 for Cosmos compatibility
   */
  async signAmino(
    signerAddress: string,
    signDoc: unknown
  ): Promise<{ signed: unknown; signature: unknown }> {
    const { signature, nonce, timestamp } = await this.signAction(
      'cosmos_sign_amino',
      { signDoc }
    );

    return {
      signed: signDoc,
      signature: {
        signature,
        pub_key: {
          type: 'ethereum/secp256k1',
          value: this._account?.address || '',
        },
        nonce,
        timestamp,
      },
    };
  }

  /**
   * Send transaction - not directly supported, use API instead
   */
  async sendTx(tx: Uint8Array, mode?: unknown): Promise<Uint8Array> {
    // For MetaMask, transactions are submitted via API with signature
    // This method exists for interface compatibility
    throw new Error(
      'Direct transaction sending not supported with MetaMask. ' +
      'Use signTypedData and submit via API.'
    );
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
   * Get offline signer - returns a compatible signer object
   */
  getOfflineSigner(): {
    getAccounts: () => Promise<{ address: string; pubkey: Uint8Array; algo: string }[]>;
    signDirect: (signerAddress: string, signDoc: unknown) => Promise<{ signed: unknown; signature: unknown }>;
  } {
    if (!this._connected || !this._account) {
      throw new Error('Wallet not connected');
    }

    return {
      getAccounts: async () => {
        if (!this._account) return [];
        return [{
          address: this._account.address,
          pubkey: this._account.pubKey,
          algo: this._account.algo,
        }];
      },
      signDirect: this.signDirect.bind(this),
    };
  }

  /**
   * Get current chain ID
   */
  getChainId(): number {
    return this._chainId;
  }

  /**
   * Switch to a different chain
   */
  async switchChain(chainId: number): Promise<void> {
    if (!window.ethereum) {
      throw new Error('MetaMask not available');
    }

    try {
      await window.ethereum.request({
        method: 'wallet_switchEthereumChain',
        params: [{ chainId: `0x${chainId.toString(16)}` }],
      });
      this._chainId = chainId;
      this._emit('networkChange', chainId.toString());
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : 'Unknown error';
      throw new Error(`Failed to switch chain: ${message}`);
    }
  }

  /**
   * Add a new chain to MetaMask
   */
  async addChain(chainConfig: {
    chainId: number;
    chainName: string;
    rpcUrls: string[];
    nativeCurrency: {
      name: string;
      symbol: string;
      decimals: number;
    };
    blockExplorerUrls?: string[];
  }): Promise<void> {
    if (!window.ethereum) {
      throw new Error('MetaMask not available');
    }

    await window.ethereum.request({
      method: 'wallet_addEthereumChain',
      params: [{
        chainId: `0x${chainConfig.chainId.toString(16)}`,
        chainName: chainConfig.chainName,
        rpcUrls: chainConfig.rpcUrls,
        nativeCurrency: chainConfig.nativeCurrency,
        blockExplorerUrls: chainConfig.blockExplorerUrls,
      }],
    });
  }

  // Private methods

  private _getNextNonce(): number {
    return ++this._nonce;
  }

  private _hexToBytes(hex: string): Uint8Array {
    const bytes = new Uint8Array(hex.length / 2);
    for (let i = 0; i < bytes.length; i++) {
      bytes[i] = parseInt(hex.substr(i * 2, 2), 16);
    }
    return bytes;
  }

  private _handleAccountsChanged(accounts: string[]): void {
    if (accounts.length === 0) {
      this.disconnect();
    } else if (this._connected && this._account) {
      const newAddress = accounts[0].toLowerCase();
      if (newAddress !== this._account.address) {
        const pubKeyBytes = new Uint8Array(33);
        const addressBytes = this._hexToBytes(newAddress.slice(2));
        pubKeyBytes.set(addressBytes.slice(0, Math.min(20, 33)));

        this._account = {
          address: newAddress,
          pubKey: pubKeyBytes,
          algo: 'secp256k1',
          name: `MetaMask (${newAddress.slice(0, 6)}...${newAddress.slice(-4)})`,
        };
        this._emit('accountChange', this._account);
      }
    }
  }

  private _handleChainChanged(chainIdHex: string): void {
    this._chainId = parseInt(chainIdHex, 16);
    this._emit('networkChange', this._chainId.toString());
  }

  private _emit(event: WalletEvent, data: unknown): void {
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
}

export default MetaMaskWallet;
