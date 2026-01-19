/**
 * Mock Wallet for Development/Demo Mode
 * Simulates wallet connection and transaction signing
 */

import type { IWallet, WalletAccount, WalletEvent, WalletEventHandler } from './types';

// Generate a deterministic mock address
function generateMockAddress(): string {
  return 'perpdex1mock7demo8wallet9address0xyz';
}

// Generate a mock transaction hash
function generateMockTxHash(): string {
  const chars = '0123456789abcdef';
  let hash = '';
  for (let i = 0; i < 64; i++) {
    hash += chars[Math.floor(Math.random() * chars.length)];
  }
  return hash.toUpperCase();
}

// Sleep helper
function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export class MockWallet implements IWallet {
  readonly provider = 'mock' as const;
  private _connected: boolean = false;
  private _account: WalletAccount | null = null;
  private _eventHandlers: Map<WalletEvent, Set<WalletEventHandler>> = new Map();

  get connected(): boolean {
    return this._connected;
  }

  get account(): WalletAccount | null {
    return this._account;
  }

  get chainId(): string {
    return 'perpdex-local-1';
  }

  /**
   * Always returns true for mock wallet
   */
  static isInstalled(): boolean {
    return true;
  }

  /**
   * Simulate wallet connection
   */
  async connect(): Promise<WalletAccount> {
    // Simulate connection delay
    await sleep(500);

    this._account = {
      address: generateMockAddress(),
      pubKey: new Uint8Array(33).fill(1),
      algo: 'secp256k1',
      name: 'Demo Account',
    };

    this._connected = true;
    this._emit('connect', this._account);

    console.log('[MockWallet] Connected:', this._account.address);

    return this._account;
  }

  /**
   * Simulate wallet disconnection
   */
  async disconnect(): Promise<void> {
    this._connected = false;
    this._account = null;
    this._emit('disconnect', null);
    console.log('[MockWallet] Disconnected');
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
   * Simulate signing (just returns mock data)
   */
  async signDirect(
    signerAddress: string,
    signDoc: any
  ): Promise<{ signed: any; signature: any }> {
    if (!this._connected) {
      throw new Error('Wallet not connected');
    }

    await sleep(300);

    return {
      signed: signDoc,
      signature: {
        pub_key: {
          type: 'tendermint/PubKeySecp256k1',
          value: 'mock-pub-key',
        },
        signature: 'mock-signature-base64',
      },
    };
  }

  /**
   * Simulate amino signing
   */
  async signAmino(
    signerAddress: string,
    signDoc: any
  ): Promise<{ signed: any; signature: any }> {
    return this.signDirect(signerAddress, signDoc);
  }

  /**
   * Simulate sending transaction
   */
  async sendTx(tx: Uint8Array, mode?: any): Promise<Uint8Array> {
    if (!this._connected) {
      throw new Error('Wallet not connected');
    }

    await sleep(1000);

    // Return mock tx hash as Uint8Array
    const hash = generateMockTxHash();
    return new TextEncoder().encode(hash);
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

    await sleep(200);

    return {
      signature: 'mock-arbitrary-signature',
      pub_key: {
        type: 'tendermint/PubKeySecp256k1',
        value: 'mock-pub-key-value',
      },
    };
  }

  /**
   * Verify arbitrary (always returns true in mock)
   */
  async verifyArbitrary(
    signerAddress: string,
    data: string | Uint8Array,
    signature: any
  ): Promise<boolean> {
    return true;
  }

  /**
   * Subscribe to events
   */
  on(event: WalletEvent, handler: WalletEventHandler): void {
    if (!this._eventHandlers.has(event)) {
      this._eventHandlers.set(event, new Set());
    }
    this._eventHandlers.get(event)!.add(handler);
  }

  /**
   * Unsubscribe from events
   */
  off(event: WalletEvent, handler: WalletEventHandler): void {
    const handlers = this._eventHandlers.get(event);
    if (handlers) {
      handlers.delete(handler);
    }
  }

  /**
   * Get mock offline signer
   */
  getOfflineSigner(): any {
    if (!this._connected) {
      throw new Error('Wallet not connected');
    }

    return {
      getAccounts: async () => [
        {
          address: this._account!.address,
          pubkey: this._account!.pubKey,
          algo: this._account!.algo,
        },
      ],
      signDirect: this.signDirect.bind(this),
      signAmino: this.signAmino.bind(this),
    };
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

/**
 * Mock sign and broadcast function
 * Simulates the full transaction flow
 */
export async function mockSignAndBroadcast(
  messages: any[],
  memo?: string
): Promise<{
  code: number;
  transactionHash: string;
  rawLog: string;
  gasUsed: number;
  gasWanted: number;
}> {
  console.log('[MockWallet] Signing messages:', messages);
  console.log('[MockWallet] Memo:', memo);

  // Simulate signing delay
  await sleep(1000);

  // Simulate broadcast delay
  await sleep(1500);

  const txHash = generateMockTxHash();

  console.log('[MockWallet] Transaction broadcast:', txHash);

  return {
    code: 0,
    transactionHash: txHash,
    rawLog: 'Mock transaction executed successfully',
    gasUsed: 150000,
    gasWanted: 200000,
  };
}

export default MockWallet;
