/**
 * DDGuardIndicator Component
 * Displays the current DDGuard risk management level for a pool
 */

import { DDGuardState } from '@/stores/riverpoolStore';
import BigNumber from 'bignumber.js';

interface DDGuardIndicatorProps {
  state: DDGuardState | null;
  showDetails?: boolean;
  size?: 'sm' | 'md' | 'lg';
}

const levelConfig = {
  normal: {
    label: 'Normal',
    color: 'green',
    bgColor: 'bg-green-500/10',
    borderColor: 'border-green-500/30',
    textColor: 'text-green-400',
    icon: '✓',
    description: 'Pool operating normally',
  },
  warning: {
    label: 'Warning',
    color: 'yellow',
    bgColor: 'bg-yellow-500/10',
    borderColor: 'border-yellow-500/30',
    textColor: 'text-yellow-400',
    icon: '⚠',
    description: 'Drawdown at 10%+ - monitoring closely',
  },
  reduce: {
    label: 'Reduce Exposure',
    color: 'orange',
    bgColor: 'bg-orange-500/10',
    borderColor: 'border-orange-500/30',
    textColor: 'text-orange-400',
    icon: '⚡',
    description: 'Drawdown at 15%+ - exposure limited to 50%',
  },
  halt: {
    label: 'Halted',
    color: 'red',
    bgColor: 'bg-red-500/10',
    borderColor: 'border-red-500/30',
    textColor: 'text-red-400',
    icon: '⛔',
    description: 'Drawdown at 30%+ - new positions blocked',
  },
};

export default function DDGuardIndicator({
  state,
  showDetails = false,
  size = 'md',
}: DDGuardIndicatorProps) {
  if (!state) {
    return null;
  }

  const level = state.level as keyof typeof levelConfig;
  const config = levelConfig[level] || levelConfig.normal;

  const sizeClasses = {
    sm: 'text-xs px-2 py-0.5',
    md: 'text-sm px-3 py-1',
    lg: 'text-base px-4 py-2',
  };

  // Calculate drawdown percentage for display
  const drawdownPercent = new BigNumber(state.drawdownPercent).times(100);
  const maxExposure = new BigNumber(state.maxExposureLimit).times(100);

  // Badge only view
  if (!showDetails) {
    return (
      <div
        data-testid="ddguard-indicator"
        className={`inline-flex items-center gap-1.5 rounded-full font-medium ${config.bgColor} ${config.borderColor} border ${config.textColor} ${sizeClasses[size]}`}
      >
        <span>{config.icon}</span>
        <span>{config.label}</span>
      </div>
    );
  }

  // Detailed view with all information
  return (
    <div data-testid="ddguard-indicator" className={`${config.bgColor} ${config.borderColor} border rounded-xl p-4`}>
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <span className={`text-xl ${config.textColor}`}>{config.icon}</span>
          <div>
            <div className={`font-semibold ${config.textColor}`}>
              DDGuard: {config.label}
            </div>
            <div className="text-xs text-gray-400">{config.description}</div>
          </div>
        </div>
      </div>

      {/* Drawdown Progress Bar */}
      <div className="mb-3">
        <div className="flex justify-between text-xs mb-1">
          <span className="text-gray-400">Current Drawdown</span>
          <span className={config.textColor}>{drawdownPercent.toFixed(2)}%</span>
        </div>
        <div className="h-2 bg-gray-700 rounded-full overflow-hidden">
          <div
            className={`h-full transition-all duration-500 ${
              level === 'normal'
                ? 'bg-green-500'
                : level === 'warning'
                ? 'bg-yellow-500'
                : level === 'reduce'
                ? 'bg-orange-500'
                : 'bg-red-500'
            }`}
            style={{ width: `${Math.min(drawdownPercent.toNumber(), 100)}%` }}
          />
          {/* Threshold markers */}
          <div className="relative -mt-2">
            <div
              className="absolute w-0.5 h-4 bg-yellow-500/50"
              style={{ left: '10%', top: '-8px' }}
            />
            <div
              className="absolute w-0.5 h-4 bg-orange-500/50"
              style={{ left: '15%', top: '-8px' }}
            />
            <div
              className="absolute w-0.5 h-4 bg-red-500/50"
              style={{ left: '30%', top: '-8px' }}
            />
          </div>
        </div>
        <div className="flex justify-between text-[10px] text-gray-500 mt-1">
          <span>0%</span>
          <span>10%</span>
          <span>15%</span>
          <span>30%</span>
        </div>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-2 gap-3 text-sm">
        <div className="bg-gray-800/50 rounded-lg p-2">
          <div className="text-xs text-gray-400">Peak NAV</div>
          <div className="text-white font-medium">
            ${new BigNumber(state.peakNav).toFixed(4)}
          </div>
        </div>
        <div className="bg-gray-800/50 rounded-lg p-2">
          <div className="text-xs text-gray-400">Current NAV</div>
          <div className="text-white font-medium">
            ${new BigNumber(state.currentNav).toFixed(4)}
          </div>
        </div>
        <div className="bg-gray-800/50 rounded-lg p-2">
          <div className="text-xs text-gray-400">Max Exposure</div>
          <div className={`font-medium ${config.textColor}`}>
            {maxExposure.toFixed(0)}%
          </div>
        </div>
        <div className="bg-gray-800/50 rounded-lg p-2">
          <div className="text-xs text-gray-400">Last Check</div>
          <div className="text-white font-medium text-xs">
            {new Date(state.lastCheckedAt * 1000).toLocaleTimeString()}
          </div>
        </div>
      </div>

      {/* Level Thresholds Legend */}
      <div className="mt-3 pt-3 border-t border-gray-700">
        <div className="text-xs text-gray-400 mb-2">Risk Levels:</div>
        <div className="flex flex-wrap gap-2 text-xs">
          <div className="flex items-center gap-1">
            <span className="w-2 h-2 rounded-full bg-green-500" />
            <span className="text-gray-400">&lt;10%</span>
          </div>
          <div className="flex items-center gap-1">
            <span className="w-2 h-2 rounded-full bg-yellow-500" />
            <span className="text-gray-400">10-15%</span>
          </div>
          <div className="flex items-center gap-1">
            <span className="w-2 h-2 rounded-full bg-orange-500" />
            <span className="text-gray-400">15-30%</span>
          </div>
          <div className="flex items-center gap-1">
            <span className="w-2 h-2 rounded-full bg-red-500" />
            <span className="text-gray-400">&gt;30%</span>
          </div>
        </div>
      </div>
    </div>
  );
}

/**
 * Compact DDGuard badge for use in lists/tables
 */
export function DDGuardBadge({
  level,
  size = 'sm',
}: {
  level: string;
  size?: 'sm' | 'md';
}) {
  const config = levelConfig[level as keyof typeof levelConfig] || levelConfig.normal;

  const sizeClasses = {
    sm: 'text-[10px] px-1.5 py-0.5',
    md: 'text-xs px-2 py-1',
  };

  return (
    <span
      className={`inline-flex items-center gap-1 rounded font-medium ${config.bgColor} ${config.textColor} ${sizeClasses[size]}`}
    >
      <span>{config.icon}</span>
      <span className="uppercase">{level}</span>
    </span>
  );
}
