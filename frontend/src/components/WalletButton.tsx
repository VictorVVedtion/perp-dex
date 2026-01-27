/**
 * Wallet Connection Button Component
 * Handles wallet connection and displays connected address
 */

import { useState, useRef, useEffect } from 'react';
import { useWallet, shortenAddress } from '@/hooks/useWallet';

export function WalletButton() {
  const { connected, connecting, address, error, isMockMode, connect, disconnect } = useWallet();
  const [showDropdown, setShowDropdown] = useState(false);
  const [disconnectConfirm, setDisconnectConfirm] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setShowDropdown(false);
        setDisconnectConfirm(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

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
      <div className="relative" ref={dropdownRef}>
        <button
          onClick={() => setShowDropdown(!showDropdown)}
          className="bg-dark-700/80 backdrop-blur-sm border border-dark-600/50 hover:bg-dark-600/80 hover:border-primary-500/30 text-white px-4 py-2 rounded-lg text-sm font-medium flex items-center space-x-2 transition-all duration-300 shadow-lg hover:shadow-primary-500/10 group"
        >
          <div className="w-2 h-2 rounded-full bg-primary-400 animate-pulse group-hover:shadow-[0_0_8px_rgba(var(--primary-400),0.6)]" />
          <span className="font-mono text-gray-200 group-hover:text-white transition-colors">{shortenAddress(address)}</span>
          {/* Mock mode badge */}
          {isMockMode && (
            <span className="px-1.5 py-0.5 bg-warning-900/50 text-warning-400 text-[10px] rounded font-medium border border-warning-500/20">
              DEMO
            </span>
          )}
          <svg
            className={`w-4 h-4 text-dark-400 transition-transform duration-300 ${showDropdown ? 'rotate-180 text-white' : 'group-hover:text-white'}`}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        {showDropdown && (
            <div className="absolute right-0 mt-2 w-64 bg-dark-800/95 backdrop-blur-xl border border-dark-600/50 rounded-xl shadow-2xl z-50 overflow-hidden animate-in fade-in slide-in-from-top-2 duration-200 ring-1 ring-white/5">
              <div className="p-4 border-b border-dark-600/50 bg-gradient-to-r from-dark-800/50 to-dark-700/30">
                <p className="text-xs text-dark-400 uppercase tracking-wider font-semibold mb-1">已连接地址</p>
                <div className="flex items-center justify-between">
                  <p className="text-sm text-white font-mono truncate">{address}</p>
                  <div className="w-2 h-2 rounded-full bg-emerald-500 shadow-[0_0_5px_rgba(16,185,129,0.5)]"></div>
                </div>
              </div>
              <div className="p-2 space-y-1">
                <button
                  onClick={() => {
                    navigator.clipboard.writeText(address);
                    setShowDropdown(false);
                  }}
                  className="w-full text-left px-3 py-2.5 text-sm text-dark-300 hover:text-white hover:bg-dark-700/50 rounded-lg transition-colors flex items-center space-x-3 group"
                >
                  <div className="p-1.5 rounded-md bg-dark-700/50 group-hover:bg-dark-600 transition-colors">
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                    </svg>
                  </div>
                  <span>复制地址</span>
                </button>
                <button
                  onClick={() => {
                    if (disconnectConfirm) {
                      disconnect();
                      setShowDropdown(false);
                      setDisconnectConfirm(false);
                    } else {
                      setDisconnectConfirm(true);
                      // Auto reset after 3s
                      setTimeout(() => setDisconnectConfirm(false), 3000);
                    }
                  }}
                  className={`w-full text-left px-3 py-2.5 text-sm rounded-lg transition-all duration-200 flex items-center space-x-3 group ${
                    disconnectConfirm
                      ? 'bg-danger-500/10 text-danger-400 hover:bg-danger-500/20'
                      : 'text-dark-300 hover:text-danger-400 hover:bg-danger-500/10'
                  }`}
                >
                  <div className={`p-1.5 rounded-md transition-colors ${
                    disconnectConfirm ? 'bg-danger-500/20 text-danger-400' : 'bg-dark-700/50 group-hover:bg-danger-500/20 group-hover:text-danger-400'
                  }`}>
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
                    </svg>
                  </div>
                  <span className="font-medium">{disconnectConfirm ? '确认断开?' : '断开连接'}</span>
                </button>
              </div>
            </div>
        )}
      </div>
    );
  }

  return (
    <div className="flex flex-col items-end">
      <button
        onClick={() => connect()}
        className="relative overflow-hidden bg-gradient-to-r from-primary-600 to-primary-500 hover:from-primary-500 hover:to-primary-400 text-white px-5 py-2.5 rounded-lg text-sm font-bold shadow-lg shadow-primary-500/25 hover:shadow-primary-500/40 transition-all duration-300 flex items-center space-x-2 group transform hover:-translate-y-0.5 active:translate-y-0"
      >
        <div className="absolute inset-0 w-full h-full bg-white/20 transform -skew-x-12 -translate-x-full group-hover:animate-shine" />
        <svg className="w-4 h-4 transition-transform group-hover:scale-110" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
        </svg>
        <span>Connect Wallet</span>
      </button>
      {error && (
        <p className="text-xs text-danger-400 mt-2 animate-in fade-in slide-in-from-top-1 bg-danger-500/10 px-2 py-1 rounded border border-danger-500/20">{error}</p>
      )}
    </div>
  );
}

export default WalletButton;
