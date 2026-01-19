/**
 * PerpDEX Wallet Integration Types
 */

// Wallet provider type
export type WalletProvider = 'keplr' | 'metamask' | 'walletconnect';

// Chain configuration
export interface ChainConfig {
  chainId: string;
  chainName: string;
  rpcUrl: string;
  restUrl: string;
  wsUrl: string;
  stakeCurrency: Currency;
  currencies: Currency[];
  feeCurrencies: Currency[];
  bip44: {
    coinType: number;
  };
  bech32Config: {
    bech32PrefixAccAddr: string;
    bech32PrefixAccPub: string;
    bech32PrefixValAddr: string;
    bech32PrefixValPub: string;
    bech32PrefixConsAddr: string;
    bech32PrefixConsPub: string;
  };
  features: string[];
}

// Currency definition
export interface Currency {
  coinDenom: string;
  coinMinimalDenom: string;
  coinDecimals: number;
  coinGeckoId?: string;
  coinImageUrl?: string;
}

// Wallet account
export interface WalletAccount {
  address: string;
  pubKey: Uint8Array;
  algo: string;
  name?: string;
}

// Wallet state
export interface WalletState {
  connected: boolean;
  connecting: boolean;
  provider: WalletProvider | null;
  account: WalletAccount | null;
  chainId: string | null;
  error: string | null;
}

// Transaction result
export interface TxResult {
  transactionHash: string;
  code: number;
  height: number;
  rawLog: string;
  gasUsed: number;
  gasWanted: number;
}

// Sign options
export interface SignOptions {
  preferNoSetFee?: boolean;
  preferNoSetMemo?: boolean;
  disableBalanceCheck?: boolean;
}

// Wallet events
export type WalletEvent =
  | 'connect'
  | 'disconnect'
  | 'accountChange'
  | 'networkChange';

export type WalletEventHandler = (data: any) => void;

// Wallet interface
export interface IWallet {
  readonly provider: WalletProvider;
  readonly connected: boolean;
  readonly account: WalletAccount | null;

  connect(): Promise<WalletAccount>;
  disconnect(): Promise<void>;
  getAccount(): Promise<WalletAccount>;
  signDirect(
    signerAddress: string,
    signDoc: any
  ): Promise<{ signed: any; signature: any }>;
  signAmino(
    signerAddress: string,
    signDoc: any
  ): Promise<{ signed: any; signature: any }>;
  sendTx(tx: Uint8Array, mode?: any): Promise<Uint8Array>;
  on(event: WalletEvent, handler: WalletEventHandler): void;
  off(event: WalletEvent, handler: WalletEventHandler): void;
}

// Order types for trading
export interface OrderRequest {
  marketId: string;
  side: 'buy' | 'sell';
  type: 'limit' | 'market' | 'stop_limit' | 'stop_market';
  price?: string;
  size: string;
  leverage?: string;
  reduceOnly?: boolean;
  postOnly?: boolean;
  timeInForce?: 'gtc' | 'ioc' | 'fok';
  triggerPrice?: string;
}

export interface Order {
  orderId: string;
  marketId: string;
  trader: string;
  side: 'buy' | 'sell';
  type: string;
  price: string;
  size: string;
  filledSize: string;
  status: 'open' | 'partial' | 'filled' | 'cancelled';
  createdAt: number;
  updatedAt: number;
}

// Position types
export interface Position {
  trader: string;
  marketId: string;
  side: 'long' | 'short';
  size: string;
  entryPrice: string;
  markPrice: string;
  unrealizedPnl: string;
  realizedPnl: string;
  margin: string;
  leverage: string;
  liquidationPrice: string;
  marginMode: 'isolated' | 'cross';
}

// Account types
export interface Account {
  address: string;
  balance: string;
  availableBalance: string;
  lockedMargin: string;
  unrealizedPnl: string;
  marginMode: 'isolated' | 'cross';
}
