import { useState } from 'react'
import { mockAccount, mockPositions, mockPriceInfo } from '@/stores/tradingStore'
import BigNumber from 'bignumber.js'

export default function AccountPage() {
  const [depositAmount, setDepositAmount] = useState('')
  const [withdrawAmount, setWithdrawAmount] = useState('')
  const [activeTab, setActiveTab] = useState<'deposit' | 'withdraw'>('deposit')

  const account = mockAccount

  // Calculate total unrealized PnL
  const totalUnrealizedPnL = mockPositions.reduce((sum, pos) => {
    const markPrice = new BigNumber(mockPriceInfo.markPrice)
    const entryPrice = new BigNumber(pos.entryPrice)
    const size = new BigNumber(pos.size)
    let priceDiff = markPrice.minus(entryPrice)
    if (pos.side === 'short') {
      priceDiff = priceDiff.negated()
    }
    return sum.plus(size.times(priceDiff))
  }, new BigNumber(0))

  const totalEquity = new BigNumber(account.balance).plus(totalUnrealizedPnL)
  const availableBalance = new BigNumber(account.balance).minus(account.lockedMargin)

  const handleDeposit = (e: React.FormEvent) => {
    e.preventDefault()
    console.log('Deposit:', depositAmount)
    setDepositAmount('')
  }

  const handleWithdraw = (e: React.FormEvent) => {
    e.preventDefault()
    console.log('Withdraw:', withdrawAmount)
    setWithdrawAmount('')
  }

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
      <h1 className="text-2xl font-bold text-white mb-6">Account</h1>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Account Overview */}
        <div className="lg:col-span-2 space-y-6">
          {/* Balance Cards */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="bg-dark-900 rounded-lg border border-dark-700 p-6">
              <div className="flex items-center justify-between mb-4">
                <span className="text-dark-400">Total Equity</span>
                <div className="w-8 h-8 bg-primary-500/20 rounded-lg flex items-center justify-center">
                  <svg className="w-4 h-4 text-primary-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                </div>
              </div>
              <div className="text-3xl font-bold text-white font-mono">
                ${totalEquity.toFixed(2)}
              </div>
              <div className={`text-sm mt-2 ${totalUnrealizedPnL.isPositive() ? 'text-primary-400' : 'text-danger-400'}`}>
                {totalUnrealizedPnL.isPositive() ? '+' : ''}{totalUnrealizedPnL.toFixed(2)} unrealized PnL
              </div>
            </div>

            <div className="bg-dark-900 rounded-lg border border-dark-700 p-6">
              <div className="flex items-center justify-between mb-4">
                <span className="text-dark-400">Available Balance</span>
                <div className="w-8 h-8 bg-dark-700 rounded-lg flex items-center justify-center">
                  <svg className="w-4 h-4 text-dark-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
                  </svg>
                </div>
              </div>
              <div className="text-3xl font-bold text-white font-mono">
                ${availableBalance.toFixed(2)}
              </div>
              <div className="text-sm text-dark-400 mt-2">
                ${account.lockedMargin} locked in positions
              </div>
            </div>
          </div>

          {/* Account Details */}
          <div className="bg-dark-900 rounded-lg border border-dark-700 p-6">
            <h3 className="text-lg font-medium text-white mb-4">Account Details</h3>
            <div className="space-y-4">
              <div className="flex justify-between py-2 border-b border-dark-700">
                <span className="text-dark-400">Account Address</span>
                <span className="text-white font-mono text-sm">{account.trader}</span>
              </div>
              <div className="flex justify-between py-2 border-b border-dark-700">
                <span className="text-dark-400">Wallet Balance</span>
                <span className="text-white font-mono">${parseFloat(account.balance).toLocaleString()}</span>
              </div>
              <div className="flex justify-between py-2 border-b border-dark-700">
                <span className="text-dark-400">Locked Margin</span>
                <span className="text-white font-mono">${parseFloat(account.lockedMargin).toLocaleString()}</span>
              </div>
              <div className="flex justify-between py-2 border-b border-dark-700">
                <span className="text-dark-400">Unrealized PnL</span>
                <span className={`font-mono ${totalUnrealizedPnL.isPositive() ? 'text-primary-400' : 'text-danger-400'}`}>
                  {totalUnrealizedPnL.isPositive() ? '+' : ''}${totalUnrealizedPnL.toFixed(2)}
                </span>
              </div>
              <div className="flex justify-between py-2">
                <span className="text-dark-400">Open Positions</span>
                <span className="text-white">{mockPositions.length}</span>
              </div>
            </div>
          </div>

          {/* Transaction History */}
          <div className="bg-dark-900 rounded-lg border border-dark-700 p-6">
            <h3 className="text-lg font-medium text-white mb-4">Transaction History</h3>
            <table className="w-full">
              <thead>
                <tr className="text-xs text-dark-400 border-b border-dark-700">
                  <th className="text-left py-3">Time</th>
                  <th className="text-left py-3">Type</th>
                  <th className="text-right py-3">Amount</th>
                  <th className="text-right py-3">Status</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td colSpan={4} className="text-center py-8 text-dark-400 text-sm">
                    No transaction history
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        {/* Deposit/Withdraw Panel */}
        <div className="lg:col-span-1">
          <div className="bg-dark-900 rounded-lg border border-dark-700 sticky top-6">
            {/* Tabs */}
            <div className="flex border-b border-dark-700">
              <button
                onClick={() => setActiveTab('deposit')}
                className={`flex-1 py-3 text-sm font-medium transition-colors ${
                  activeTab === 'deposit'
                    ? 'text-primary-400 border-b-2 border-primary-400'
                    : 'text-dark-400 hover:text-white'
                }`}
              >
                Deposit
              </button>
              <button
                onClick={() => setActiveTab('withdraw')}
                className={`flex-1 py-3 text-sm font-medium transition-colors ${
                  activeTab === 'withdraw'
                    ? 'text-primary-400 border-b-2 border-primary-400'
                    : 'text-dark-400 hover:text-white'
                }`}
              >
                Withdraw
              </button>
            </div>

            {/* Form */}
            <div className="p-4">
              {activeTab === 'deposit' ? (
                <form onSubmit={handleDeposit} className="space-y-4">
                  <div>
                    <label className="block text-xs text-dark-400 mb-2">Amount (USDC)</label>
                    <input
                      type="number"
                      value={depositAmount}
                      onChange={(e) => setDepositAmount(e.target.value)}
                      placeholder="0.00"
                      className="w-full bg-dark-800 border border-dark-600 rounded-lg px-4 py-3 text-white font-mono focus:border-primary-500"
                    />
                  </div>
                  <div className="bg-dark-800 rounded-lg p-3">
                    <div className="text-xs text-dark-400 mb-1">You will receive</div>
                    <div className="text-white font-mono">
                      {depositAmount || '0.00'} USDC (margin)
                    </div>
                  </div>
                  <button
                    type="submit"
                    className="w-full bg-primary-600 hover:bg-primary-500 text-white py-3 rounded-lg font-medium transition-colors"
                  >
                    Deposit
                  </button>
                </form>
              ) : (
                <form onSubmit={handleWithdraw} className="space-y-4">
                  <div>
                    <label className="block text-xs text-dark-400 mb-2">Amount (USDC)</label>
                    <input
                      type="number"
                      value={withdrawAmount}
                      onChange={(e) => setWithdrawAmount(e.target.value)}
                      placeholder="0.00"
                      className="w-full bg-dark-800 border border-dark-600 rounded-lg px-4 py-3 text-white font-mono focus:border-primary-500"
                    />
                  </div>
                  <div className="flex justify-between text-xs mb-2">
                    <span className="text-dark-400">Available</span>
                    <button
                      type="button"
                      onClick={() => setWithdrawAmount(availableBalance.toString())}
                      className="text-primary-400 hover:text-primary-300"
                    >
                      Max: ${availableBalance.toFixed(2)}
                    </button>
                  </div>
                  <button
                    type="submit"
                    className="w-full bg-dark-700 hover:bg-dark-600 text-white py-3 rounded-lg font-medium transition-colors"
                  >
                    Withdraw
                  </button>
                </form>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
