/**
 * Keplr Wallet Integration for PerpDEX
 * Supports Cosmos SDK based chains
 */

import type {
  IWallet,
  WalletAccount,
  WalletEvent,
  WalletEventHandler,
  ChainConfig,
} from './types';

// Keplr window interface
declare global {
  interface Window {
    keplr?: any;
    getOfflineSigner?: (chainId: string) => any;
    getOfflineSignerOnlyAmino?: (chainId: string) => any;
  }
}

// Import config for environment-based URLs
import config from '@/lib/config';

// Default PerpDEX chain configuration
export const PERPDEX_CHAIN_CONFIG: ChainConfig = {
  chainId: config.chain.chainId,
  chainName: 'PerpDEX',
  rpcUrl: config.chain.rpcUrl,
  restUrl: config.chain.restUrl,
  wsUrl: config.api.wsUrl,
  stakeCurrency: {
    coinDenom: 'PERP',
    coinMinimalDenom: 'uperp',
    coinDecimals: 6,
  },
  currencies: [
    {
      coinDenom: 'PERP',
      coinMinimalDenom: 'uperp',
      coinDecimals: 6,
    },
    {
      coinDenom: 'USDC',
      coinMinimalDenom: 'uusdc',
      coinDecimals: 6,
      coinGeckoId: 'usd-coin',
    },
  ],
  feeCurrencies: [
    {
      coinDenom: 'PERP',
      coinMinimalDenom: 'uperp',
      coinDecimals: 6,
    },
  ],
  bip44: {
    coinType: 118,
  },
  bech32Config: {
    bech32PrefixAccAddr: 'perpdex',
    bech32PrefixAccPub: 'perpdexpub',
    bech32PrefixValAddr: 'perpdexvaloper',
    bech32PrefixValPub: 'perpdexvaloperpub',
    bech32PrefixConsAddr: 'perpdexvalcons',
    bech32PrefixConsPub: 'perpdexvalconspub',
  },
  features: ['ibc-transfer', 'ibc-go'],
};

export class KeplrWallet implements IWallet {
  readonly provider = 'keplr' as const;
  private _connected: boolean = false;
  private _account: WalletAccount | null = null;
  private _chainId: string;
  private _chainConfig: ChainConfig;
  private _eventHandlers: Map<WalletEvent, Set<WalletEventHandler>> = new Map();

  constructor(chainConfig: ChainConfig = PERPDEX_CHAIN_CONFIG) {
    this._chainConfig = chainConfig;
    this._chainId = chainConfig.chainId;

    // Setup Keplr event listeners
    if (typeof window !== 'undefined') {
      window.addEventListener('keplr_keystorechange', () => {
        this._handleAccountChange();
      });
    }
  }

  get connected(): boolean {
    return this._connected;
  }

  get account(): WalletAccount | null {
    return this._account;
  }

  get chainId(): string {
    return this._chainId;
  }

  /**
   * Check if Keplr is installed
   */
  static isInstalled(): boolean {
    return typeof window !== 'undefined' && !!window.keplr;
  }

  /**
   * Connect to Keplr wallet
   */
  async connect(): Promise<WalletAccount> {
    if (!KeplrWallet.isInstalled()) {
      throw new Error('Keplr wallet is not installed');
    }

    try {
      // Suggest chain if not already added
      await this._suggestChain();

      // Enable the chain
      await window.keplr.enable(this._chainId);

      // Get account
      const offlineSigner = window.getOfflineSigner!(this._chainId);
      const accounts = await offlineSigner.getAccounts();

      if (accounts.length === 0) {
        throw new Error('No accounts found in Keplr');
      }

      const key = await window.keplr.getKey(this._chainId);

      this._account = {
        address: accounts[0].address,
        pubKey: accounts[0].pubkey,
        algo: accounts[0].algo,
        name: key.name,
      };

      this._connected = true;
      this._emit('connect', this._account);

      return this._account;
    } catch (error: any) {
      this._connected = false;
      this._account = null;
      throw new Error(`Failed to connect to Keplr: ${error.message}`);
    }
  }

  /**
   * Disconnect from Keplr wallet
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
   * Sign a transaction using direct signing
   */
  async signDirect(
    signerAddress: string,
    signDoc: any
  ): Promise<{ signed: any; signature: any }> {
    if (!this._connected) {
      throw new Error('Wallet not connected');
    }

    const offlineSigner = window.getOfflineSigner!(this._chainId);
    return offlineSigner.signDirect(signerAddress, signDoc);
  }

  /**
   * Sign a transaction using amino signing
   */
  async signAmino(
    signerAddress: string,
    signDoc: any
  ): Promise<{ signed: any; signature: any }> {
    if (!this._connected) {
      throw new Error('Wallet not connected');
    }

    const offlineSigner = window.getOfflineSignerOnlyAmino!(this._chainId);
    return offlineSigner.signAmino(signerAddress, signDoc);
  }

  /**
   * Send a signed transaction
   */
  async sendTx(tx: Uint8Array, mode: any = 'sync'): Promise<Uint8Array> {
    if (!this._connected) {
      throw new Error('Wallet not connected');
    }

    return window.keplr.sendTx(this._chainId, tx, mode);
  }

  /**
   * Sign arbitrary data
   */
  async signArbitrary(
    signerAddress: string,
    data: string | Uint8Array
  ): Promise<{ signature: string; pub_key: { type: string; value: string } }> {
    if (!this._connected) {
      throw new Error('Wallet not connected');
    }

    return window.keplr.signArbitrary(this._chainId, signerAddress, data);
  }

  /**
   * Verify arbitrary signature
   */
  async verifyArbitrary(
    signerAddress: string,
    data: string | Uint8Array,
    signature: any
  ): Promise<boolean> {
    return window.keplr.verifyArbitrary(
      this._chainId,
      signerAddress,
      data,
      signature
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
   * Get offline signer
   */
  getOfflineSigner(): any {
    if (!this._connected) {
      throw new Error('Wallet not connected');
    }
    return window.getOfflineSigner!(this._chainId);
  }

  /**
   * Suggest the chain to Keplr
   */
  private async _suggestChain(): Promise<void> {
    try {
      await window.keplr.experimentalSuggestChain({
        chainId: this._chainConfig.chainId,
        chainName: this._chainConfig.chainName,
        rpc: this._chainConfig.rpcUrl,
        rest: this._chainConfig.restUrl,
        stakeCurrency: this._chainConfig.stakeCurrency,
        bip44: this._chainConfig.bip44,
        bech32Config: this._chainConfig.bech32Config,
        currencies: this._chainConfig.currencies,
        feeCurrencies: this._chainConfig.feeCurrencies,
        features: this._chainConfig.features,
      });
    } catch (error: any) {
      // Chain might already be added, continue
      console.warn('Failed to suggest chain:', error.message);
    }
  }

  /**
   * Handle account change
   */
  private async _handleAccountChange(): Promise<void> {
    if (!this._connected) return;

    try {
      const key = await window.keplr.getKey(this._chainId);
      const offlineSigner = window.getOfflineSigner!(this._chainId);
      const accounts = await offlineSigner.getAccounts();

      if (accounts.length > 0) {
        this._account = {
          address: accounts[0].address,
          pubKey: accounts[0].pubkey,
          algo: accounts[0].algo,
          name: key.name,
        };
        this._emit('accountChange', this._account);
      }
    } catch (error) {
      console.error('Failed to handle account change:', error);
    }
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
}

export default KeplrWallet;
