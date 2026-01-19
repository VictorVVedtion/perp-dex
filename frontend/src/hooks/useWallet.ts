/**
 * Wallet Hook for PerpDEX
 * Provides wallet connection state and actions
 * Supports both Keplr and Mock wallet modes
 */

import { useCallback, useEffect, useState } from 'react';
import { KeplrWallet, PERPDEX_CHAIN_CONFIG } from '@/lib/wallet/keplr';
import { MockWallet, mockSignAndBroadcast } from '@/lib/wallet/mock';
import type { IWallet, WalletAccount } from '@/lib/wallet/types';
import config from '@/lib/config';

interface UseWalletReturn {
  connected: boolean;
  connecting: boolean;
  account: WalletAccount | null;
  address: string;
  error: string | null;
  isMockMode: boolean;
  connect: () => Promise<void>;
  disconnect: () => Promise<void>;
  signAndBroadcast: (messages: any[], memo?: string) => Promise<any>;
}

// Check if mock mode is enabled
const isMockMode = config.features.mockMode;

// Singleton wallet instance
let walletInstance: IWallet | null = null;

function getWalletInstance(): IWallet {
  if (!walletInstance) {
    if (isMockMode) {
      console.log('[useWallet] Using Mock Wallet (NEXT_PUBLIC_MOCK_MODE=true)');
      walletInstance = new MockWallet();
    } else {
      walletInstance = new KeplrWallet(PERPDEX_CHAIN_CONFIG);
    }
  }
  return walletInstance;
}

export function useWallet(): UseWalletReturn {
  const [connected, setConnected] = useState(false);
  const [connecting, setConnecting] = useState(false);
  const [account, setAccount] = useState<WalletAccount | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Check for existing connection on mount
  useEffect(() => {
    const wallet = getWalletInstance();
    if (wallet.connected && wallet.account) {
      setConnected(true);
      setAccount(wallet.account);
    }

    // Listen for account changes
    wallet.on('accountChange', (newAccount) => {
      setAccount(newAccount);
    });

    wallet.on('disconnect', () => {
      setConnected(false);
      setAccount(null);
    });
  }, []);

  const connect = useCallback(async () => {
    const wallet = getWalletInstance();

    // In mock mode, skip Keplr installation check
    if (!isMockMode && !KeplrWallet.isInstalled()) {
      setError('请先安装 Keplr 钱包扩展');
      window.open('https://www.keplr.app/download', '_blank');
      return;
    }

    try {
      setConnecting(true);
      setError(null);

      const connectedAccount = await wallet.connect();
      setAccount(connectedAccount);
      setConnected(true);

      // Store connection state
      if (typeof window !== 'undefined') {
        localStorage.setItem('wallet_connected', 'true');
      }
    } catch (err: any) {
      // Classify errors for better UX
      let errorMessage = err.message || '连接钱包失败';
      if (err.message?.includes('rejected')) {
        errorMessage = '已取消钱包连接';
      } else if (err.message?.includes('not installed')) {
        errorMessage = '请先安装 Keplr 钱包扩展';
      }
      setError(errorMessage);
      setConnected(false);
      setAccount(null);
    } finally {
      setConnecting(false);
    }
  }, []);

  const disconnect = useCallback(async () => {
    const wallet = getWalletInstance();

    try {
      await wallet.disconnect();
      setConnected(false);
      setAccount(null);
      setError(null);

      // Clear stored connection state
      if (typeof window !== 'undefined') {
        localStorage.removeItem('wallet_connected');
      }
    } catch (err: any) {
      setError(err.message || '断开连接失败');
    }
  }, []);

  const signAndBroadcast = useCallback(
    async (messages: any[], memo?: string) => {
      const wallet = getWalletInstance();

      if (!wallet.connected || !account) {
        throw new Error('钱包未连接');
      }

      try {
        // Use mock sign and broadcast in mock mode
        if (isMockMode) {
          return await mockSignAndBroadcast(messages, memo);
        }

        // Get the offline signer
        const signer = wallet.getOfflineSigner();

        // Import SigningStargateClient dynamically to avoid SSR issues
        const { SigningStargateClient } = await import('@cosmjs/stargate');

        const client = await SigningStargateClient.connectWithSigner(
          PERPDEX_CHAIN_CONFIG.rpcUrl,
          signer
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

        const result = await client.signAndBroadcast(
          account.address,
          messages,
          fee,
          memo || ''
        );

        return result;
      } catch (err: any) {
        // Classify errors for better UX
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
    [account]
  );

  // Auto-reconnect on page load if previously connected
  useEffect(() => {
    if (typeof window !== 'undefined') {
      const wasConnected = localStorage.getItem('wallet_connected') === 'true';
      if (wasConnected && !connected && !connecting) {
        connect();
      }
    }
  }, [connect, connected, connecting]);

  return {
    connected,
    connecting,
    account,
    address: account?.address || '',
    error,
    isMockMode,
    connect,
    disconnect,
    signAndBroadcast,
  };
}

// Helper function to shorten address
export function shortenAddress(address: string, chars = 6): string {
  if (!address) return '';
  return `${address.slice(0, chars)}...${address.slice(-chars)}`;
}

export default useWallet;
