/**
 * Wallet Hook for PerpDEX
 * Provides wallet connection state and actions
 * Supports MetaMask (primary), Keplr, and Mock wallet modes
 */

import { useCallback, useEffect, useState } from 'react';
import { MetaMaskWallet } from '@/lib/wallet/metamask';
import { KeplrWallet, PERPDEX_CHAIN_CONFIG } from '@/lib/wallet/keplr';
import { MockWallet, mockSignAndBroadcast } from '@/lib/wallet/mock';
import type { IWallet, WalletAccount, WalletProvider } from '@/lib/wallet/types';
import config from '@/lib/config';

interface UseWalletReturn {
  connected: boolean;
  connecting: boolean;
  account: WalletAccount | null;
  address: string;
  error: string | null;
  provider: WalletProvider | null;
  isMockMode: boolean;
  isMetaMask: boolean;
  connect: (provider?: WalletProvider) => Promise<void>;
  disconnect: () => Promise<void>;
  signAndBroadcast: (messages: any[], memo?: string) => Promise<any>;
  signOrder: (order: {
    marketId: string;
    side: string;
    type: string;
    price: string;
    size: string;
    leverage: string;
  }) => Promise<{ signature: string; nonce: number; expiry: number }>;
  signAction: (action: string, payload: Record<string, unknown>) => Promise<{ signature: string; nonce: number; timestamp: number }>;
}

// Check if mock mode is enabled
const isMockMode = config.features.mockMode;

// Wallet instances cache
const walletInstances: Map<WalletProvider, IWallet> = new Map();

function getWalletInstance(provider: WalletProvider): IWallet {
  if (!walletInstances.has(provider)) {
    switch (provider) {
      case 'metamask':
        walletInstances.set(provider, new MetaMaskWallet());
        break;
      case 'keplr':
        walletInstances.set(provider, new KeplrWallet(PERPDEX_CHAIN_CONFIG));
        break;
      case 'mock':
        walletInstances.set(provider, new MockWallet());
        break;
      default:
        throw new Error(`Unsupported wallet provider: ${provider}`);
    }
  }
  return walletInstances.get(provider)!;
}

function detectPreferredProvider(): WalletProvider {
  if (isMockMode) {
    return 'mock';
  }
  // MetaMask is preferred (like Hyperliquid)
  if (MetaMaskWallet.isInstalled()) {
    return 'metamask';
  }
  // Fallback to Keplr for Cosmos users
  if (KeplrWallet.isInstalled()) {
    return 'keplr';
  }
  // Default to MetaMask (user will be prompted to install)
  return 'metamask';
}

export function useWallet(): UseWalletReturn {
  const [connected, setConnected] = useState(false);
  const [connecting, setConnecting] = useState(false);
  const [account, setAccount] = useState<WalletAccount | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [currentProvider, setCurrentProvider] = useState<WalletProvider | null>(null);
  const [walletInstance, setWalletInstance] = useState<IWallet | null>(null);

  // Check for existing connection on mount
  useEffect(() => {
    const savedProvider = localStorage.getItem('wallet_provider') as WalletProvider | null;
    if (savedProvider && walletInstances.has(savedProvider)) {
      const wallet = walletInstances.get(savedProvider)!;
      if (wallet.connected && wallet.account) {
        setConnected(true);
        setAccount(wallet.account);
        setCurrentProvider(savedProvider);
        setWalletInstance(wallet);
      }
    }
  }, []);

  // Setup event listeners when wallet changes
  useEffect(() => {
    if (!walletInstance) return;

    const handleAccountChange = (newAccount: WalletAccount) => {
      setAccount(newAccount);
    };

    const handleDisconnect = () => {
      setConnected(false);
      setAccount(null);
      setCurrentProvider(null);
    };

    walletInstance.on('accountChange', handleAccountChange);
    walletInstance.on('disconnect', handleDisconnect);

    return () => {
      walletInstance.off('accountChange', handleAccountChange);
      walletInstance.off('disconnect', handleDisconnect);
    };
  }, [walletInstance]);

  const connect = useCallback(async (provider?: WalletProvider) => {
    const selectedProvider = provider || detectPreferredProvider();

    // Check wallet availability
    if (selectedProvider === 'metamask' && !MetaMaskWallet.isInstalled()) {
      setError('请先安装 MetaMask 钱包扩展');
      window.open('https://metamask.io/download/', '_blank');
      return;
    }

    if (selectedProvider === 'keplr' && !KeplrWallet.isInstalled()) {
      setError('请先安装 Keplr 钱包扩展');
      window.open('https://www.keplr.app/download', '_blank');
      return;
    }

    try {
      setConnecting(true);
      setError(null);

      const wallet = getWalletInstance(selectedProvider);
      const connectedAccount = await wallet.connect();

      setAccount(connectedAccount);
      setConnected(true);
      setCurrentProvider(selectedProvider);
      setWalletInstance(wallet);

      // Store connection state
      if (typeof window !== 'undefined') {
        localStorage.setItem('wallet_connected', 'true');
        localStorage.setItem('wallet_provider', selectedProvider);
      }
    } catch (err: any) {
      let errorMessage = err.message || '连接钱包失败';
      if (err.message?.includes('rejected') || err.message?.includes('denied')) {
        errorMessage = '已取消钱包连接';
      } else if (err.message?.includes('not installed')) {
        errorMessage = selectedProvider === 'metamask'
          ? '请先安装 MetaMask 钱包扩展'
          : '请先安装 Keplr 钱包扩展';
      }
      setError(errorMessage);
      setConnected(false);
      setAccount(null);
    } finally {
      setConnecting(false);
    }
  }, []);

  const disconnect = useCallback(async () => {
    if (walletInstance) {
      try {
        await walletInstance.disconnect();
      } catch (err: any) {
        console.error('Disconnect error:', err);
      }
    }

    setConnected(false);
    setAccount(null);
    setCurrentProvider(null);
    setWalletInstance(null);
    setError(null);

    if (typeof window !== 'undefined') {
      localStorage.removeItem('wallet_connected');
      localStorage.removeItem('wallet_provider');
    }
  }, [walletInstance]);

  const signOrder = useCallback(
    async (order: {
      marketId: string;
      side: string;
      type: string;
      price: string;
      size: string;
      leverage: string;
    }) => {
      if (!walletInstance || !account) {
        throw new Error('钱包未连接');
      }

      if (walletInstance instanceof MetaMaskWallet) {
        return walletInstance.signOrder(order);
      }

      // Fallback for non-MetaMask: encode as action
      throw new Error('订单签名仅支持 MetaMask');
    },
    [walletInstance, account]
  );

  const signAction = useCallback(
    async (action: string, payload: Record<string, unknown>) => {
      if (!walletInstance || !account) {
        throw new Error('钱包未连接');
      }

      if (walletInstance instanceof MetaMaskWallet) {
        return walletInstance.signAction(action, payload);
      }

      throw new Error('操作签名仅支持 MetaMask');
    },
    [walletInstance, account]
  );

  const signAndBroadcast = useCallback(
    async (messages: any[], memo?: string) => {
      if (!walletInstance || !account) {
        throw new Error('钱包未连接');
      }

      try {
        // Use mock sign and broadcast in mock mode
        if (currentProvider === 'mock') {
          return await mockSignAndBroadcast(messages, memo);
        }

        // For MetaMask, use EIP-712 signing and submit via API
        if (walletInstance instanceof MetaMaskWallet) {
          // Sign the transaction using EIP-712
          const { signature, nonce, timestamp } = await walletInstance.signAction(
            'cosmos_tx',
            { messages, memo }
          );

          // Submit to backend API
          const response = await fetch(`${config.api.baseUrl}/v1/tx/submit`, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'X-Signature': signature,
              'X-Address': account.address,
              'X-Nonce': nonce.toString(),
              'X-Timestamp': timestamp.toString(),
            },
            body: JSON.stringify({ messages, memo }),
          });

          if (!response.ok) {
            const error = await response.json();
            throw new Error(error.message || '交易提交失败');
          }

          return await response.json();
        }

        // For Keplr, use CosmJS
        if (walletInstance instanceof KeplrWallet) {
          const signer = walletInstance.getOfflineSigner();
          const { SigningStargateClient } = await import('@cosmjs/stargate');

          // Keplr's OfflineSigner is compatible with CosmJS at runtime
          const client = await SigningStargateClient.connectWithSigner(
            PERPDEX_CHAIN_CONFIG.rpcUrl,
            signer as any
          );

          const fee = {
            amount: [
              {
                denom: PERPDEX_CHAIN_CONFIG.feeCurrencies[0].coinMinimalDenom,
                amount: '5000',
              },
            ],
            gas: '200000',
          };

          return await client.signAndBroadcast(
            account.address,
            messages,
            fee,
            memo || ''
          );
        }

        throw new Error('不支持的钱包类型');
      } catch (err: any) {
        let errorMessage = err.message || '交易签名失败';
        if (err.message?.includes('rejected') || err.message?.includes('denied')) {
          errorMessage = '交易已取消';
        } else if (err.message?.includes('timeout')) {
          errorMessage = '交易超时，请检查网络';
        } else if (err.message?.includes('insufficient')) {
          errorMessage = '余额不足';
        } else if (err.message?.includes('connect')) {
          errorMessage = '网络连接失败，请重试';
        }
        throw new Error(errorMessage);
      }
    },
    [walletInstance, account, currentProvider]
  );

  // Auto-reconnect on page load if previously connected
  useEffect(() => {
    if (typeof window !== 'undefined') {
      const wasConnected = localStorage.getItem('wallet_connected') === 'true';
      const savedProvider = localStorage.getItem('wallet_provider') as WalletProvider | null;
      if (wasConnected && savedProvider && !connected && !connecting) {
        connect(savedProvider);
      }
    }
  }, [connect, connected, connecting]);

  return {
    connected,
    connecting,
    account,
    address: account?.address || '',
    error,
    provider: currentProvider,
    isMockMode: currentProvider === 'mock',
    isMetaMask: currentProvider === 'metamask',
    connect,
    disconnect,
    signAndBroadcast,
    signOrder,
    signAction,
  };
}

// Helper function to shorten address
export function shortenAddress(address: string, chars = 6): string {
  if (!address) return '';
  // Handle both 0x and bech32 addresses
  if (address.startsWith('0x')) {
    return `${address.slice(0, chars + 2)}...${address.slice(-chars)}`;
  }
  return `${address.slice(0, chars)}...${address.slice(-chars)}`;
}

export default useWallet;
