/**
 * Main Layout Component
 * Includes navigation, header with wallet connection, and footer
 * Optimized with glass-morphism, animations, and modern DEX styling
 */

import Link from 'next/link';
import { useRouter } from 'next/router';
import { ReactNode, useState, useEffect } from 'react';
import { WalletButton } from './WalletButton';
import { useTradingStore } from '@/stores/tradingStore';

interface LayoutProps {
  children: ReactNode;
}

// Navigation icons as inline SVGs
const TradeIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
  </svg>
);

const PositionsIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
  </svg>
);

const AccountIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
  </svg>
);

const RiverPoolIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 21a6 6 0 01-6-6c0-4 6-11 6-11s6 7 6 11a6 6 0 01-6 6z" />
  </svg>
);

export function Layout({ children }: LayoutProps) {
  const router = useRouter();
  const { ticker, wsConnected } = useTradingStore();

  // Price change animation state
  const [priceClass, setPriceClass] = useState('');
  const [lastPrice, setLastPrice] = useState<string | null>(null);

  // Animate price changes
  useEffect(() => {
    if (ticker?.lastPrice) {
      const currentPrice = ticker.lastPrice;
      if (lastPrice !== null) {
        if (parseFloat(currentPrice) > parseFloat(lastPrice)) {
          setPriceClass('text-primary-500 scale-105');
        } else if (parseFloat(currentPrice) < parseFloat(lastPrice)) {
          setPriceClass('text-danger-500 scale-105');
        }
        // Reset animation after 500ms
        const timer = setTimeout(() => setPriceClass(''), 500);
        return () => clearTimeout(timer);
      }
      setLastPrice(currentPrice);
    }
  }, [ticker?.lastPrice, lastPrice]);

  const navigation = [
    { name: 'Trade', href: '/', icon: TradeIcon },
    { name: 'Positions', href: '/positions', icon: PositionsIcon },
    { name: 'RiverPool', href: '/riverpool', icon: RiverPoolIcon },
    { name: 'Account', href: '/account', icon: AccountIcon },
  ];

  // Format price with thousands separator
  const formatPrice = (price: string | undefined): string => {
    if (!price) return '$50,000.00';
    const num = parseFloat(price);
    return `$${num.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
  };

  // Format 24h change
  const formatChange = (change: string | undefined): { text: string; positive: boolean } => {
    if (!change) return { text: '+0.00%', positive: true };
    const num = parseFloat(change);
    const positive = num >= 0;
    return {
      text: `${positive ? '+' : ''}${num.toFixed(2)}%`,
      positive,
    };
  };

  const change24h = ticker?.change24h ? formatChange(ticker.change24h) : formatChange(undefined);

  return (
    <div className="min-h-screen flex flex-col bg-dark-950 animate-fade-in">
      {/* Header with glass-morphism effect */}
      <header className="bg-dark-900/80 backdrop-blur-xl border-b border-dark-700/50 sticky top-0 z-50 transition-all duration-300">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            {/* Logo with hover animation */}
            <div className="flex items-center">
              <Link href="/" className="flex items-center space-x-2 group">
                <div className="w-8 h-8 bg-gradient-to-br from-primary-500 to-primary-600 rounded-lg flex items-center justify-center shadow-lg shadow-primary-500/20 transform transition-all duration-500 group-hover:rotate-12 group-hover:shadow-glow-primary">
                  <span className="text-white font-bold text-sm">P</span>
                </div>
                <span className="text-xl font-bold text-white tracking-tight transition-colors duration-300 group-hover:text-primary-500">PerpDEX</span>
              </Link>
            </div>

            {/* Desktop Navigation with underline animation */}
            <nav className="hidden md:flex space-x-8">
              {navigation.map((item) => {
                const isActive = router.pathname === item.href;
                return (
                  <Link
                    key={item.name}
                    href={item.href}
                    className="relative group px-3 py-2 text-sm font-medium"
                  >
                    <span className={`transition-colors duration-200 ${
                      isActive ? 'text-primary-500' : 'text-dark-300 group-hover:text-white'
                    }`}>
                      {item.name}
                    </span>
                    <span className={`absolute bottom-0 left-0 h-0.5 bg-primary-500 transition-all duration-300 ease-out ${
                      isActive ? 'w-full' : 'w-0 group-hover:w-full'
                    }`} />
                  </Link>
                );
              })}
            </nav>

            {/* Right Section: Price + Wallet */}
            <div className="flex items-center space-x-4">
              {/* Market Info */}
              <div className="hidden sm:flex items-center space-x-3">
                {/* Connection Status with enhanced pulse */}
                <div className="flex items-center space-x-2 bg-dark-800/50 px-3 py-1.5 rounded-full border border-dark-700/50">
                  <div className="relative flex h-2.5 w-2.5">
                    {wsConnected && (
                      <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary-500 opacity-75"></span>
                    )}
                    <span className={`relative inline-flex rounded-full h-2.5 w-2.5 ${
                      wsConnected ? 'bg-primary-500' : 'bg-dark-500'
                    }`}></span>
                  </div>
                  <span className="text-dark-300 text-xs font-medium">BTC-USDC</span>
                </div>

                {/* Price Display with animation */}
                <div className="flex items-center space-x-2 bg-dark-800/50 px-3 py-1.5 rounded-lg border border-dark-700/50">
                  <span
                    className={`text-white font-mono text-sm font-medium transition-all duration-200 inline-block origin-center ${priceClass}`}
                  >
                    {formatPrice(ticker?.lastPrice)}
                  </span>
                  <span
                    className={`text-xs font-semibold px-1.5 py-0.5 rounded ${
                      change24h.positive
                        ? 'text-primary-500 bg-primary-500/10'
                        : 'text-danger-500 bg-danger-500/10'
                    }`}
                  >
                    {change24h.text}
                  </span>
                </div>

                {/* 24h Volume (if available) */}
                {ticker?.volume24h && (
                  <div className="text-xs text-dark-400 hidden lg:block">
                    <span className="text-dark-500">Vol:</span> ${parseFloat(ticker.volume24h).toLocaleString()}
                  </div>
                )}
              </div>

              {/* Wallet Button */}
              <WalletButton />
            </div>
          </div>
        </div>

        {/* Mobile Navigation with icons */}
        <div className="md:hidden border-t border-dark-700/50 bg-dark-900/90 backdrop-blur-md">
          <div className="flex justify-around py-3">
            {navigation.map((item) => {
              const isActive = router.pathname === item.href;
              const Icon = item.icon;
              return (
                <Link
                  key={item.name}
                  href={item.href}
                  className={`flex flex-col items-center space-y-1 px-4 py-1 transition-colors duration-200 ${
                    isActive ? 'text-primary-500' : 'text-dark-400 hover:text-white'
                  }`}
                >
                  <Icon className="w-5 h-5" />
                  <span className="text-xs font-medium">{item.name}</span>
                </Link>
              );
            })}
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1">{children}</main>

      {/* Footer with subtle styling */}
      <footer className="bg-dark-900/50 backdrop-blur-sm border-t border-dark-700/50 py-4">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex flex-col sm:flex-row items-center justify-between text-sm text-dark-400 space-y-2 sm:space-y-0">
            <div className="flex items-center space-x-2">
              <span>© 2024 PerpDEX</span>
              <span className="text-dark-600">|</span>
              <span>Built on Cosmos SDK</span>
              {wsConnected && (
                <>
                  <span className="text-dark-600">|</span>
                  <span className="flex items-center space-x-1.5">
                    <span className="relative flex h-2 w-2">
                      <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary-500 opacity-75"></span>
                      <span className="relative inline-flex rounded-full h-2 w-2 bg-primary-500"></span>
                    </span>
                    <span className="text-primary-500 font-medium text-xs uppercase tracking-wider">Live</span>
                  </span>
                </>
              )}
            </div>
            <div className="flex space-x-4">
              <a href="#" className="hover:text-white transition-colors duration-200">
                Docs
              </a>
              <a href="#" className="hover:text-white transition-colors duration-200">
                GitHub
              </a>
              <a href="#" className="hover:text-white transition-colors duration-200">
                Discord
              </a>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}

export default Layout;
