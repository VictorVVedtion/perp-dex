/**
 * PerpDEX Wallet Integration
 *
 * Usage:
 * ```typescript
 * import { getWalletManager } from '@perpdex/wallet';
 *
 * const wallet = getWalletManager();
 *
 * // Auto-connect to previously connected wallet
 * await wallet.autoConnect();
 *
 * // Or manually connect
 * const account = await wallet.connect('keplr');
 * console.log('Connected:', account.address);
 *
 * // Subscribe to events
 * wallet.on('accountChange', (account) => {
 *   console.log('Account changed:', account.address);
 * });
 * ```
 */

export * from './types';
export { KeplrWallet, PERPDEX_CHAIN_CONFIG } from './keplr';
export { WalletManager, getWalletManager } from './manager';

// Re-export commonly used types
export type {
  WalletProvider,
  WalletAccount,
  WalletState,
  WalletEvent,
  ChainConfig,
  Order,
  OrderRequest,
  Position,
  Account,
} from './types';
