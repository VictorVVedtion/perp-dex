/**
 * Main Layout Component
 * Includes navigation, header with wallet connection, and footer
 */

import Link from 'next/link';
import { useRouter } from 'next/router';
import { ReactNode, useState, useEffect } from 'react';
import { WalletButton } from './WalletButton';
import { useTradingStore } from '@/stores/tradingStore';

interface LayoutProps {
  children: ReactNode;
}

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
          setPriceClass('text-primary-400');
        } else if (parseFloat(currentPrice) < parseFloat(lastPrice)) {
          setPriceClass('text-danger-400');
        }
        // Reset animation after 500ms
        const timer = setTimeout(() => setPriceClass(''), 500);
        return () => clearTimeout(timer);
      }
      setLastPrice(currentPrice);
    }
  }, [ticker?.lastPrice, lastPrice]);

  const navigation = [
    { name: 'Trade', href: '/' },
    { name: 'Positions', href: '/positions' },
    { name: 'Account', href: '/account' },
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
    <div className="min-h-screen flex flex-col bg-dark-950">
      {/* Header */}
      <header className="bg-dark-900 border-b border-dark-700 sticky top-0 z-40">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            {/* Logo */}
            <div className="flex items-center">
              <Link href="/" className="flex items-center space-x-2">
                <div className="w-8 h-8 bg-gradient-to-br from-primary-400 to-primary-600 rounded-lg flex items-center justify-center">
                  <span className="text-white font-bold text-sm">P</span>
                </div>
                <span className="text-xl font-bold text-white">PerpDEX</span>
              </Link>
            </div>

            {/* Navigation */}
            <nav className="hidden md:flex space-x-8">
              {navigation.map((item) => {
                const isActive = router.pathname === item.href;
                return (
                  <Link
                    key={item.name}
                    href={item.href}
                    className={`px-3 py-2 text-sm font-medium transition-colors ${
                      isActive
                        ? 'text-primary-400 border-b-2 border-primary-400'
                        : 'text-dark-300 hover:text-white'
                    }`}
                  >
                    {item.name}
                  </Link>
                );
              })}
            </nav>

            {/* Right Section: Price + Wallet */}
            <div className="flex items-center space-x-6">
              {/* Market Info */}
              <div className="hidden sm:flex items-center space-x-4">
                {/* Connection Status */}
                <div className="flex items-center space-x-2">
                  <div
                    className={`w-2 h-2 rounded-full ${
                      wsConnected ? 'bg-primary-400 animate-pulse' : 'bg-dark-500'
                    }`}
                  />
                  <span className="text-dark-300 text-sm">BTC-USDC</span>
                </div>

                {/* Price */}
                <div className="flex items-center space-x-2">
                  <span
                    className={`text-white font-mono text-sm font-medium transition-colors duration-300 ${priceClass}`}
                  >
                    {formatPrice(ticker?.lastPrice)}
                  </span>
                  <span
                    className={`text-xs font-medium ${
                      change24h.positive ? 'text-primary-400' : 'text-danger-400'
                    }`}
                  >
                    {change24h.text}
                  </span>
                </div>

                {/* 24h Volume (if available) */}
                {ticker?.volume24h && (
                  <div className="text-xs text-dark-400">
                    Vol: ${parseFloat(ticker.volume24h).toLocaleString()}
                  </div>
                )}
              </div>

              {/* Wallet Button */}
              <WalletButton />
            </div>
          </div>
        </div>

        {/* Mobile Navigation */}
        <div className="md:hidden border-t border-dark-700">
          <div className="flex justify-around py-2">
            {navigation.map((item) => {
              const isActive = router.pathname === item.href;
              return (
                <Link
                  key={item.name}
                  href={item.href}
                  className={`px-4 py-2 text-sm font-medium transition-colors ${
                    isActive ? 'text-primary-400' : 'text-dark-300'
                  }`}
                >
                  {item.name}
                </Link>
              );
            })}
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1">{children}</main>

      {/* Footer */}
      <footer className="bg-dark-900 border-t border-dark-700 py-4">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex flex-col sm:flex-row items-center justify-between text-sm text-dark-400 space-y-2 sm:space-y-0">
            <div className="flex items-center space-x-2">
              <span>© 2024 PerpDEX</span>
              <span className="text-dark-600">|</span>
              <span>Built on Cosmos SDK</span>
              {wsConnected && (
                <>
                  <span className="text-dark-600">|</span>
                  <span className="flex items-center space-x-1">
                    <div className="w-1.5 h-1.5 rounded-full bg-primary-400" />
                    <span className="text-primary-400">Live</span>
                  </span>
                </>
              )}
            </div>
            <div className="flex space-x-4">
              <a href="#" className="hover:text-white transition-colors">
                Docs
              </a>
              <a href="#" className="hover:text-white transition-colors">
                GitHub
              </a>
              <a href="#" className="hover:text-white transition-colors">
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
