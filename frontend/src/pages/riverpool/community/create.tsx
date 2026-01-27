/**
 * Create Community Pool Page
 * Multi-step wizard for creating a new community pool
 */

import { useState } from 'react';
import { useRouter } from 'next/router';
import Head from 'next/head';
import { useRiverpoolStore, CreateCommunityPoolConfig } from '@/stores/riverpoolStore';
import { useWallet } from '@/hooks/useWallet';

type Step = 1 | 2 | 3 | 4 | 5;

const STEPS = [
  { id: 1, title: 'Basic Info', description: 'Name and description' },
  { id: 2, title: 'Deposit Settings', description: 'Min/max deposits' },
  { id: 3, title: 'Fee Structure', description: 'Management and performance fees' },
  { id: 4, title: 'Trading Rules', description: 'Leverage and markets' },
  { id: 5, title: 'Review', description: 'Confirm and create' },
];

const AVAILABLE_MARKETS = ['BTC-USDC', 'ETH-USDC', 'SOL-USDC', 'ARB-USDC', 'OP-USDC'];
const AVAILABLE_TAGS = ['BTC', 'ETH', 'Trend', 'Grid', 'Arbitrage', 'DeFi', 'High-Risk', 'Conservative', 'Scalping', 'Swing'];

export default function CreatePoolPage() {
  const router = useRouter();
  const { createCommunityPool, isLoading, error } = useRiverpoolStore();
  const { connected, address } = useWallet();

  const [currentStep, setCurrentStep] = useState<Step>(1);
  const [formData, setFormData] = useState<CreateCommunityPoolConfig>({
    name: '',
    description: '',
    minDeposit: '100',
    maxDeposit: '1000000',
    managementFee: '0.02',
    performanceFee: '0.20',
    ownerMinStake: '0.05',
    lockPeriodDays: 0,
    redemptionDelayDays: 3,
    isPrivate: false,
    maxLeverage: '10',
    allowedMarkets: ['BTC-USDC', 'ETH-USDC'],
    tags: [],
  });
  const [ownerStakeAmount, setOwnerStakeAmount] = useState('1000');

  const updateFormData = (data: Partial<CreateCommunityPoolConfig>) => {
    setFormData((prev) => ({ ...prev, ...data }));
  };

  const handleNext = () => {
    if (currentStep < 5) {
      setCurrentStep((currentStep + 1) as Step);
    }
  };

  const handleBack = () => {
    if (currentStep > 1) {
      setCurrentStep((currentStep - 1) as Step);
    }
  };

  const handleCreate = async () => {
    // Check wallet connection
    if (!connected || !address) {
      console.error('Wallet not connected');
      return;
    }

    try {
      // Use connected wallet address as pool owner
      await createCommunityPool(address, formData);
      router.push('/riverpool?tab=community');
    } catch (err) {
      console.error('Failed to create pool:', err);
    }
  };

  const toggleMarket = (market: string) => {
    const current = formData.allowedMarkets;
    const newMarkets = current.includes(market)
      ? current.filter((m) => m !== market)
      : [...current, market];
    updateFormData({ allowedMarkets: newMarkets });
  };

  const toggleTag = (tag: string) => {
    const current = formData.tags;
    const newTags = current.includes(tag)
      ? current.filter((t) => t !== tag)
      : current.length < 5
      ? [...current, tag]
      : current;
    updateFormData({ tags: newTags });
  };

  const isStepValid = () => {
    switch (currentStep) {
      case 1:
        return formData.name.trim().length >= 3 && formData.description.trim().length >= 10;
      case 2:
        return (
          parseFloat(formData.minDeposit) > 0 &&
          parseFloat(formData.maxDeposit) >= parseFloat(formData.minDeposit)
        );
      case 3:
        return (
          parseFloat(formData.managementFee) >= 0 &&
          parseFloat(formData.managementFee) <= 0.1 &&
          parseFloat(formData.performanceFee) >= 0 &&
          parseFloat(formData.performanceFee) <= 0.5 &&
          parseFloat(formData.ownerMinStake) >= 0.05
        );
      case 4:
        return (
          parseFloat(formData.maxLeverage) >= 1 &&
          parseFloat(formData.maxLeverage) <= 50 &&
          formData.allowedMarkets.length > 0
        );
      case 5:
        return true;
      default:
        return false;
    }
  };

  return (
    <>
      <Head>
        <title>Create Community Pool | RiverPool</title>
        <meta name="description" content="Create your own community trading pool" />
      </Head>

      <div className="min-h-screen bg-gray-900 text-white">
        <div className="max-w-4xl mx-auto px-4 py-8">
          {/* Header */}
          <div className="mb-8">
            <button
              onClick={() => router.back()}
              className="flex items-center gap-2 text-gray-400 hover:text-white mb-4"
            >
              <span>←</span>
              <span>Back</span>
            </button>
            <h1 className="text-3xl font-bold">Create Community Pool</h1>
            <p className="text-gray-400 mt-2">
              Set up your own trading pool and earn fees from successful strategies
            </p>
          </div>

          {/* Progress Steps */}
          <div className="mb-8">
            <div className="flex items-center justify-between">
              {STEPS.map((step, index) => (
                <div key={step.id} className="flex items-center flex-1">
                  <div className="flex flex-col items-center">
                    <div
                      className={`w-10 h-10 rounded-full flex items-center justify-center font-semibold transition-colors ${
                        currentStep >= step.id
                          ? 'bg-blue-600 text-white'
                          : 'bg-gray-700 text-gray-400'
                      }`}
                    >
                      {currentStep > step.id ? '✓' : step.id}
                    </div>
                    <div className="mt-2 text-center">
                      <div
                        className={`text-sm font-medium ${
                          currentStep >= step.id ? 'text-white' : 'text-gray-500'
                        }`}
                      >
                        {step.title}
                      </div>
                      <div className="text-xs text-gray-500 hidden sm:block">
                        {step.description}
                      </div>
                    </div>
                  </div>
                  {index < STEPS.length - 1 && (
                    <div
                      className={`flex-1 h-0.5 mx-2 ${
                        currentStep > step.id ? 'bg-blue-600' : 'bg-gray-700'
                      }`}
                    />
                  )}
                </div>
              ))}
            </div>
          </div>

          {/* Form Content */}
          <div className="bg-gray-800/50 rounded-xl border border-gray-700/50 p-6">
            {/* Step 1: Basic Info */}
            {currentStep === 1 && (
              <div className="space-y-6">
                <h2 className="text-xl font-semibold">Basic Information</h2>

                <div>
                  <label htmlFor="pool-name" className="block text-sm font-medium text-gray-300 mb-2">
                    Pool Name *
                  </label>
                  <input
                    id="pool-name"
                    type="text"
                    value={formData.name}
                    onChange={(e) => updateFormData({ name: e.target.value })}
                    placeholder="e.g., Alpha Trend Strategy"
                    maxLength={50}
                    aria-label="Pool Name"
                    className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-4 py-3 text-white placeholder-gray-400 focus:outline-none focus:border-blue-500"
                  />
                  <p className="text-xs text-gray-400 mt-1">
                    {formData.name.length}/50 characters (min 3)
                  </p>
                </div>

                <div>
                  <label htmlFor="pool-description" className="block text-sm font-medium text-gray-300 mb-2">
                    Description *
                  </label>
                  <textarea
                    id="pool-description"
                    value={formData.description}
                    onChange={(e) => updateFormData({ description: e.target.value })}
                    placeholder="Describe your trading strategy and goals..."
                    maxLength={500}
                    rows={4}
                    aria-label="Description"
                    className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-4 py-3 text-white placeholder-gray-400 focus:outline-none focus:border-blue-500 resize-none"
                  />
                  <p className="text-xs text-gray-400 mt-1">
                    {formData.description.length}/500 characters (min 10)
                  </p>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    Tags (up to 5)
                  </label>
                  <div className="flex flex-wrap gap-2">
                    {AVAILABLE_TAGS.map((tag) => (
                      <button
                        key={tag}
                        onClick={() => toggleTag(tag)}
                        className={`px-3 py-1 rounded-full text-sm font-medium transition-colors ${
                          formData.tags.includes(tag)
                            ? 'bg-blue-600 text-white'
                            : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                        }`}
                      >
                        {tag}
                      </button>
                    ))}
                  </div>
                </div>

                <div className="flex items-center gap-3">
                  <input
                    type="checkbox"
                    id="isPrivate"
                    checked={formData.isPrivate}
                    onChange={(e) => updateFormData({ isPrivate: e.target.checked })}
                    className="w-5 h-5 rounded bg-gray-700 border-gray-600 text-blue-600 focus:ring-blue-500"
                  />
                  <label htmlFor="isPrivate" className="text-gray-300">
                    Make this pool private (invite-only)
                  </label>
                </div>
              </div>
            )}

            {/* Step 2: Deposit Settings */}
            {currentStep === 2 && (
              <div className="space-y-6">
                <h2 className="text-xl font-semibold">Deposit Settings</h2>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">
                      Minimum Deposit (USDC)
                    </label>
                    <input
                      type="number"
                      value={formData.minDeposit}
                      onChange={(e) => updateFormData({ minDeposit: e.target.value })}
                      min="1"
                      step="1"
                      className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-4 py-3 text-white focus:outline-none focus:border-blue-500"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">
                      Maximum Deposit (USDC)
                    </label>
                    <input
                      type="number"
                      value={formData.maxDeposit}
                      onChange={(e) => updateFormData({ maxDeposit: e.target.value })}
                      min="1"
                      step="1"
                      className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-4 py-3 text-white focus:outline-none focus:border-blue-500"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">
                      Lock Period (days)
                    </label>
                    <input
                      type="number"
                      value={formData.lockPeriodDays}
                      onChange={(e) =>
                        updateFormData({ lockPeriodDays: parseInt(e.target.value) || 0 })
                      }
                      min="0"
                      max="365"
                      className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-4 py-3 text-white focus:outline-none focus:border-blue-500"
                    />
                    <p className="text-xs text-gray-400 mt-1">0 = no lock period</p>
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">
                      Redemption Delay (days)
                    </label>
                    <input
                      type="number"
                      value={formData.redemptionDelayDays}
                      onChange={(e) =>
                        updateFormData({ redemptionDelayDays: parseInt(e.target.value) || 0 })
                      }
                      min="0"
                      max="30"
                      className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-4 py-3 text-white focus:outline-none focus:border-blue-500"
                    />
                    <p className="text-xs text-gray-400 mt-1">Recommended: 3-7 days</p>
                  </div>
                </div>
              </div>
            )}

            {/* Step 3: Fee Structure */}
            {currentStep === 3 && (
              <div className="space-y-6">
                <h2 className="text-xl font-semibold">Fee Structure</h2>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">
                      Management Fee (% per year)
                    </label>
                    <input
                      type="number"
                      value={(parseFloat(formData.managementFee) * 100).toString()}
                      onChange={(e) =>
                        updateFormData({
                          managementFee: (parseFloat(e.target.value) / 100).toString(),
                        })
                      }
                      min="0"
                      max="10"
                      step="0.1"
                      className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-4 py-3 text-white focus:outline-none focus:border-blue-500"
                    />
                    <p className="text-xs text-gray-400 mt-1">Max: 10% per year</p>
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">
                      Performance Fee (% of profits)
                    </label>
                    <input
                      type="number"
                      value={(parseFloat(formData.performanceFee) * 100).toString()}
                      onChange={(e) =>
                        updateFormData({
                          performanceFee: (parseFloat(e.target.value) / 100).toString(),
                        })
                      }
                      min="0"
                      max="50"
                      step="1"
                      className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-4 py-3 text-white focus:outline-none focus:border-blue-500"
                    />
                    <p className="text-xs text-gray-400 mt-1">
                      Max: 50% (only charged on new profits)
                    </p>
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    Your Minimum Stake (% of pool)
                  </label>
                  <input
                    type="number"
                    value={(parseFloat(formData.ownerMinStake) * 100).toString()}
                    onChange={(e) =>
                      updateFormData({
                        ownerMinStake: (parseFloat(e.target.value) / 100).toString(),
                      })
                    }
                    min="5"
                    max="100"
                    step="1"
                    className="w-full bg-gray-700/50 border border-gray-600 rounded-lg px-4 py-3 text-white focus:outline-none focus:border-blue-500"
                  />
                  <p className="text-xs text-gray-400 mt-1">
                    Minimum 5% - ensures you have skin in the game
                  </p>
                </div>

                <div className="bg-blue-500/10 border border-blue-500/30 rounded-lg p-4">
                  <h4 className="text-blue-400 font-medium mb-2">Fee Calculation Example</h4>
                  <p className="text-sm text-gray-300">
                    For a $100K pool with {(parseFloat(formData.managementFee) * 100).toFixed(1)}%
                    management fee and {(parseFloat(formData.performanceFee) * 100).toFixed(0)}%
                    performance fee:
                  </p>
                  <ul className="text-sm text-gray-400 mt-2 space-y-1">
                    <li>
                      • Annual management fee: $
                      {(100000 * parseFloat(formData.managementFee)).toFixed(0)}
                    </li>
                    <li>
                      • If profits are 20% ($20K): Performance fee = $
                      {(20000 * parseFloat(formData.performanceFee)).toFixed(0)}
                    </li>
                  </ul>
                </div>
              </div>
            )}

            {/* Step 4: Trading Rules */}
            {currentStep === 4 && (
              <div className="space-y-6">
                <h2 className="text-xl font-semibold">Trading Rules</h2>

                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    Maximum Leverage
                  </label>
                  <div className="flex items-center gap-4">
                    <input
                      type="range"
                      value={formData.maxLeverage}
                      onChange={(e) => updateFormData({ maxLeverage: e.target.value })}
                      min="1"
                      max="50"
                      step="1"
                      className="flex-1"
                    />
                    <span className="text-xl font-bold text-white w-16 text-center">
                      {formData.maxLeverage}x
                    </span>
                  </div>
                  <p className="text-xs text-gray-400 mt-1">
                    Higher leverage = higher risk. Choose wisely.
                  </p>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    Allowed Markets
                  </label>
                  <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
                    {AVAILABLE_MARKETS.map((market) => (
                      <button
                        key={market}
                        onClick={() => toggleMarket(market)}
                        className={`p-3 rounded-lg border text-sm font-medium transition-all ${
                          formData.allowedMarkets.includes(market)
                            ? 'bg-blue-600/20 border-blue-500 text-blue-400'
                            : 'bg-gray-700/50 border-gray-600 text-gray-300 hover:border-gray-500'
                        }`}
                      >
                        {market}
                      </button>
                    ))}
                  </div>
                  <p className="text-xs text-gray-400 mt-2">
                    Select at least one market to trade
                  </p>
                </div>
              </div>
            )}

            {/* Step 5: Review */}
            {currentStep === 5 && (
              <div className="space-y-6">
                <h2 className="text-xl font-semibold">Review Your Pool</h2>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  {/* Basic Info */}
                  <div className="bg-gray-700/30 rounded-lg p-4 space-y-3">
                    <h3 className="font-medium text-gray-300">Basic Info</h3>
                    <div className="space-y-2 text-sm">
                      <div className="flex justify-between">
                        <span className="text-gray-400">Name</span>
                        <span className="text-white">{formData.name}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-gray-400">Privacy</span>
                        <span className="text-white">
                          {formData.isPrivate ? 'Private' : 'Public'}
                        </span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-gray-400">Tags</span>
                        <span className="text-white">{formData.tags.join(', ') || 'None'}</span>
                      </div>
                    </div>
                  </div>

                  {/* Deposit Settings */}
                  <div className="bg-gray-700/30 rounded-lg p-4 space-y-3">
                    <h3 className="font-medium text-gray-300">Deposits</h3>
                    <div className="space-y-2 text-sm">
                      <div className="flex justify-between">
                        <span className="text-gray-400">Min Deposit</span>
                        <span className="text-white">${formData.minDeposit}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-gray-400">Max Deposit</span>
                        <span className="text-white">${formData.maxDeposit}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-gray-400">Lock Period</span>
                        <span className="text-white">{formData.lockPeriodDays} days</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-gray-400">Redemption Delay</span>
                        <span className="text-white">T+{formData.redemptionDelayDays}</span>
                      </div>
                    </div>
                  </div>

                  {/* Fees */}
                  <div className="bg-gray-700/30 rounded-lg p-4 space-y-3">
                    <h3 className="font-medium text-gray-300">Fees</h3>
                    <div className="space-y-2 text-sm">
                      <div className="flex justify-between">
                        <span className="text-gray-400">Management Fee</span>
                        <span className="text-white">
                          {(parseFloat(formData.managementFee) * 100).toFixed(1)}%/year
                        </span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-gray-400">Performance Fee</span>
                        <span className="text-white">
                          {(parseFloat(formData.performanceFee) * 100).toFixed(0)}%
                        </span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-gray-400">Your Min Stake</span>
                        <span className="text-white">
                          {(parseFloat(formData.ownerMinStake) * 100).toFixed(0)}%
                        </span>
                      </div>
                    </div>
                  </div>

                  {/* Trading */}
                  <div className="bg-gray-700/30 rounded-lg p-4 space-y-3">
                    <h3 className="font-medium text-gray-300">Trading</h3>
                    <div className="space-y-2 text-sm">
                      <div className="flex justify-between">
                        <span className="text-gray-400">Max Leverage</span>
                        <span className="text-white">{formData.maxLeverage}x</span>
                      </div>
                      <div>
                        <span className="text-gray-400">Markets:</span>
                        <div className="flex flex-wrap gap-1 mt-1">
                          {formData.allowedMarkets.map((m) => (
                            <span key={m} className="px-2 py-0.5 bg-gray-600 rounded text-xs">
                              {m}
                            </span>
                          ))}
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                {/* Initial Stake */}
                <div className="bg-yellow-500/10 border border-yellow-500/30 rounded-lg p-4">
                  <h4 className="text-yellow-400 font-medium mb-2">Initial Owner Stake</h4>
                  <p className="text-sm text-gray-300 mb-3">
                    You must deposit at least{' '}
                    {(parseFloat(formData.ownerMinStake) * 100).toFixed(0)}% of the pool value as
                    your stake. Enter your initial deposit:
                  </p>
                  <div className="flex items-center gap-3">
                    <input
                      type="number"
                      value={ownerStakeAmount}
                      onChange={(e) => setOwnerStakeAmount(e.target.value)}
                      min="100"
                      className="flex-1 bg-gray-700/50 border border-gray-600 rounded-lg px-4 py-2 text-white focus:outline-none focus:border-blue-500"
                    />
                    <span className="text-gray-300">USDC</span>
                  </div>
                </div>

                {error && (
                  <div className="bg-red-500/10 border border-red-500/30 rounded-lg p-4 text-red-400">
                    {error}
                  </div>
                )}
              </div>
            )}

            {/* Navigation Buttons */}
            <div className="flex justify-between mt-8 pt-6 border-t border-gray-700">
              <button
                onClick={handleBack}
                disabled={currentStep === 1}
                className="px-6 py-3 bg-gray-700 hover:bg-gray-600 disabled:bg-gray-800 disabled:text-gray-500 text-white font-medium rounded-lg transition-colors"
              >
                Back
              </button>

              {currentStep < 5 ? (
                <button
                  onClick={handleNext}
                  disabled={!isStepValid()}
                  className="px-6 py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white font-medium rounded-lg transition-colors"
                >
                  Next
                </button>
              ) : (
                <button
                  onClick={handleCreate}
                  disabled={isLoading || !isStepValid()}
                  className="px-8 py-3 bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 disabled:from-gray-600 disabled:to-gray-600 disabled:cursor-not-allowed text-white font-medium rounded-lg transition-all flex items-center gap-2"
                >
                  {isLoading ? (
                    <>
                      <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white" />
                      <span>Creating...</span>
                    </>
                  ) : (
                    <>
                      <span>Create Pool</span>
                      <span>→</span>
                    </>
                  )}
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
    </>
  );
}