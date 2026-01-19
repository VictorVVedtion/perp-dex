/**
 * Wallet Connection Button Component
 * Handles wallet connection and displays connected address
 */

import { useState } from 'react';
import { useWallet, shortenAddress } from '@/hooks/useWallet';

export function WalletButton() {
  const { connected, connecting, address, error, connect, disconnect } = useWallet();
  const [showDropdown, setShowDropdown] = useState(false);

  if (connecting) {
    return (
      <button
        disabled
        className="bg-dark-700 text-dark-300 px-4 py-2 rounded-lg text-sm font-medium flex items-center space-x-2"
      >
        <svg
          className="animate-spin h-4 w-4"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
        >
          <circle
            className="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            strokeWidth="4"
          />
          <path
            className="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
          />
        </svg>
        <span>连接中...</span>
      </button>
    );
  }

  if (connected && address) {
    return (
      <div className="relative">
        <button
          onClick={() => setShowDropdown(!showDropdown)}
          className="bg-dark-700 hover:bg-dark-600 text-white px-4 py-2 rounded-lg text-sm font-medium flex items-center space-x-2 transition-colors"
        >
          <div className="w-2 h-2 rounded-full bg-primary-400" />
          <span className="font-mono">{shortenAddress(address)}</span>
          <svg
            className={`w-4 h-4 transition-transform ${showDropdown ? 'rotate-180' : ''}`}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        {showDropdown && (
          <>
            {/* Backdrop */}
            <div
              className="fixed inset-0 z-10"
              onClick={() => setShowDropdown(false)}
            />
            {/* Dropdown */}
            <div className="absolute right-0 mt-2 w-48 bg-dark-800 border border-dark-600 rounded-lg shadow-lg z-20">
              <div className="p-3 border-b border-dark-600">
                <p className="text-xs text-dark-400">已连接地址</p>
                <p className="text-sm text-white font-mono truncate">{address}</p>
              </div>
              <div className="p-2">
                <button
                  onClick={() => {
                    navigator.clipboard.writeText(address);
                    setShowDropdown(false);
                  }}
                  className="w-full text-left px-3 py-2 text-sm text-dark-300 hover:text-white hover:bg-dark-700 rounded transition-colors flex items-center space-x-2"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                  </svg>
                  <span>复制地址</span>
                </button>
                <button
                  onClick={() => {
                    disconnect();
                    setShowDropdown(false);
                  }}
                  className="w-full text-left px-3 py-2 text-sm text-danger-400 hover:text-danger-300 hover:bg-dark-700 rounded transition-colors flex items-center space-x-2"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
                  </svg>
                  <span>断开连接</span>
                </button>
              </div>
            </div>
          </>
        )}
      </div>
    );
  }

  return (
    <div className="flex flex-col items-end">
      <button
        onClick={connect}
        className="bg-primary-600 hover:bg-primary-500 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors flex items-center space-x-2"
      >
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
        </svg>
        <span>Connect Wallet</span>
      </button>
      {error && (
        <p className="text-xs text-danger-400 mt-1">{error}</p>
      )}
    </div>
  );
}

export default WalletButton;
