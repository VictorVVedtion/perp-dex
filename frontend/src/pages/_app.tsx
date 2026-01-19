import '@/styles/globals.css'
import type { AppProps } from 'next/app'
import Head from 'next/head'
import { useEffect } from 'react'
import { Layout } from '@/components/Layout'
import { ToastProvider } from '@/contexts/ToastContext'
import { useTradingStore } from '@/stores/tradingStore'
import { config } from '@/lib/config'

function DataInitializer() {
  const { initHyperliquid, initWebSocket, closeHyperliquid, closeWebSocket } = useTradingStore()

  useEffect(() => {
    // Initialize based on configuration
    if (config.features.useHyperliquid && !config.features.mockMode) {
      console.log('Initializing Hyperliquid data connection...')
      initHyperliquid()
    } else if (!config.features.mockMode) {
      console.log('Initializing local WebSocket connection...')
      initWebSocket()
    } else {
      console.log('Running in mock mode, no data connection initialized')
    }

    // Cleanup on unmount
    return () => {
      closeHyperliquid()
      closeWebSocket()
    }
  }, [initHyperliquid, initWebSocket, closeHyperliquid, closeWebSocket])

  return null
}

export default function App({ Component, pageProps }: AppProps) {
  return (
    <>
      <Head>
        <title>PerpDEX - Perpetual DEX</title>
        <meta name="description" content="Decentralized Perpetual Exchange built on Cosmos SDK" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <link rel="icon" href="/favicon.ico" />
      </Head>
      <ToastProvider>
        <DataInitializer />
        <Layout>
          <Component {...pageProps} />
        </Layout>
      </ToastProvider>
    </>
  )
}
