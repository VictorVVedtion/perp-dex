/**
 * RiverPool Store - Manages liquidity pool state
 * Supports Foundation LP, Main LP, and Community Pools
 */

import { create } from 'zustand';
import BigNumber from 'bignumber.js';
import { config } from '@/lib/config';

// Types
export interface Pool {
  poolId: string;
  poolType: 'foundation' | 'main' | 'community';
  name: string;
  description: string;
  status: 'active' | 'paused' | 'closed';
  totalDeposits: string;
  totalShares: string;
  nav: string;
  highWaterMark: string;
  currentDrawdown: string;
  ddGuardLevel: 'normal' | 'warning' | 'reduce' | 'halt';
  minDeposit: string;
  maxDeposit: string;
  lockPeriodDays: number;
  redemptionDelayDays: number;
  dailyRedemptionLimit: string;
  seatsAvailable?: number; // Foundation LP only
  createdAt: number;
  updatedAt: number;
  // Community Pool specific fields
  owner?: string;
  managementFee?: string;
  performanceFee?: string;
  ownerMinStake?: string;
  ownerCurrentStake?: string;
  isPrivate?: boolean;
  requiresInviteCode?: boolean;
  totalHolders?: number;
  allowedMarkets?: string[];
  maxLeverage?: string;
  tags?: string[];
}

// Community Pool creation config
export interface CreateCommunityPoolConfig {
  name: string;
  description: string;
  minDeposit: string;
  maxDeposit: string;
  managementFee: string;
  performanceFee: string;
  ownerMinStake: string;
  lockPeriodDays: number;
  redemptionDelayDays: number;
  isPrivate: boolean;
  maxLeverage: string;
  allowedMarkets: string[];
  tags: string[];
}

// Pool holder info
export interface PoolHolder {
  address: string;
  shares: string;
  value: string;
  depositedAt: number;
  isOwner: boolean;
}

// Pool position (for owner trading)
export interface PoolPosition {
  positionId: string;
  marketId: string;
  side: 'long' | 'short';
  size: string;
  entryPrice: string;
  markPrice: string;
  pnl: string;
  pnlPercent: string;
  leverage: string;
  liquidationPrice: string;
  margin: string;
}

// Pool trade history
export interface PoolTrade {
  tradeId: string;
  marketId: string;
  side: 'buy' | 'sell';
  price: string;
  size: string;
  fee: string;
  pnl: string;
  timestamp: number;
}

// Invite code
export interface InviteCode {
  code: string;
  maxUses: number;
  usedCount: number;
  expiresAt: number;
  createdAt: number;
  isActive: boolean;
}

export interface Deposit {
  depositId: string;
  poolId: string;
  depositor: string;
  amount: string;
  shares: string;
  navAtDeposit: string;
  depositedAt: number;
  unlockAt: number;
  pointsEarned: string;
  isLocked: boolean;
}

export interface Withdrawal {
  withdrawalId: string;
  poolId: string;
  withdrawer: string;
  sharesRequested: string;
  sharesRedeemed: string;
  amountReceived: string;
  navAtRequest: string;
  status: 'pending' | 'processing' | 'completed' | 'cancelled';
  requestedAt: number;
  availableAt: number;
  completedAt: number;
  isReady: boolean;
}

export interface PoolStats {
  poolId: string;
  totalValueLocked: string;
  totalDepositors: number;
  totalPendingWithdrawals: string;
  realizedPnl: string;
  unrealizedPnl: string;
  totalFeesCollected: string;
  return1d: string;
  return7d: string;
  return30d: string;
  returnAllTime: string;
  updatedAt: number;
}

export interface DDGuardState {
  poolId: string;
  level: string;
  peakNav: string;
  currentNav: string;
  drawdownPercent: string;
  maxExposureLimit: string;
  triggeredAt: number;
  lastCheckedAt: number;
}

export interface NAVHistory {
  poolId: string;
  nav: string;
  totalValue: string;
  timestamp: number;
}

export interface UserPoolBalance {
  shares: string;
  value: string;
  costBasis: string;
  unrealizedPnl: string;
  pnlPercent: string;
  unlockAt: number;
  canWithdraw: boolean;
}

interface RiverpoolState {
  // Pool data
  pools: Pool[];
  selectedPool: Pool | null;
  poolStats: PoolStats | null;
  ddGuardState: DDGuardState | null;
  navHistory: NAVHistory[];

  // User data
  userDeposits: Deposit[];
  userWithdrawals: Withdrawal[];
  userPoolBalance: UserPoolBalance | null;

  // Community Pool specific data
  poolHolders: PoolHolder[];
  poolPositions: PoolPosition[];
  poolTrades: PoolTrade[];
  inviteCodes: InviteCode[];
  userOwnedPools: Pool[];

  // UI state
  activeTab: 'foundation' | 'main' | 'community';
  isLoading: boolean;
  error: string | null;

  // Deposit form
  depositAmount: string;
  depositPoolId: string;

  // Withdrawal form
  withdrawShares: string;
  withdrawPoolId: string;

  // Community Pool filters
  communityPoolFilter: {
    sortBy: 'tvl' | 'nav' | 'return' | 'holders' | 'created';
    sortOrder: 'asc' | 'desc';
    showPrivate: boolean;
    minTvl: string;
    tags: string[];
  };

  // Actions
  setActiveTab: (tab: 'foundation' | 'main' | 'community') => void;
  setSelectedPool: (pool: Pool | null) => void;
  setDepositAmount: (amount: string) => void;
  setDepositPoolId: (poolId: string) => void;
  setWithdrawShares: (shares: string) => void;
  setWithdrawPoolId: (poolId: string) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  setCommunityPoolFilter: (filter: Partial<RiverpoolState['communityPoolFilter']>) => void;

  // API actions
  fetchPools: () => Promise<void>;
  fetchPool: (poolId: string) => Promise<void>;
  fetchPoolStats: (poolId: string) => Promise<void>;
  fetchDDGuardState: (poolId: string) => Promise<void>;
  fetchNAVHistory: (poolId: string, from?: number, to?: number) => Promise<void>;
  fetchUserDeposits: (user: string) => Promise<void>;
  fetchUserWithdrawals: (user: string) => Promise<void>;
  fetchUserPoolBalance: (poolId: string, user: string) => Promise<void>;

  // Community Pool API actions
  fetchPoolHolders: (poolId: string) => Promise<void>;
  fetchPoolPositions: (poolId: string) => Promise<void>;
  fetchPoolTrades: (poolId: string, limit?: number) => Promise<void>;
  fetchInviteCodes: (poolId: string) => Promise<void>;
  fetchUserOwnedPools: (owner: string) => Promise<void>;

  // Transaction actions
  deposit: (depositor: string, poolId: string, amount: string, inviteCode?: string) => Promise<any>;
  requestWithdrawal: (withdrawer: string, poolId: string, shares: string) => Promise<any>;
  claimWithdrawal: (withdrawer: string, withdrawalId: string) => Promise<any>;
  cancelWithdrawal: (withdrawer: string, withdrawalId: string) => Promise<any>;

  // Community Pool transaction actions
  createCommunityPool: (owner: string, config: CreateCommunityPoolConfig) => Promise<any>;
  depositOwnerStake: (owner: string, poolId: string, amount: string) => Promise<any>;
  generateInviteCode: (owner: string, poolId: string, maxUses: number, expiresInDays: number) => Promise<any>;
  pausePool: (owner: string, poolId: string) => Promise<any>;
  resumePool: (owner: string, poolId: string) => Promise<any>;
  closePool: (owner: string, poolId: string) => Promise<any>;

  // Estimation
  estimateDeposit: (poolId: string, amount: string) => Promise<{ shares: string; nav: string; sharePrice: string }>;
  estimateWithdrawal: (poolId: string, shares: string) => Promise<{
    amount: string;
    nav: string;
    availableAt: number;
    queuePosition: string;
    mayBeProrated: boolean;
  }>;

  // Computed
  calculateUserTotalValue: () => string;
  getPoolByType: (type: 'foundation' | 'main' | 'community') => Pool[];
  getFilteredCommunityPools: () => Pool[];
}

const API_BASE = config.api.baseUrl;

// Mock data for development and E2E testing
const MOCK_POOLS: Pool[] = [
  {
    poolId: 'foundation-1',
    poolType: 'foundation',
    name: 'Foundation LP',
    description: '100 seats × $100K, 180-day lock, 5M Points/seat. Early liquidity providers earn maximum rewards.',
    status: 'active',
    totalDeposits: '7500000',
    totalShares: '7500000',
    nav: '1.0000',
    highWaterMark: '1.0000',
    currentDrawdown: '0.00',
    ddGuardLevel: 'normal',
    minDeposit: '100000',
    maxDeposit: '100000',
    lockPeriodDays: 180,
    redemptionDelayDays: 0,
    dailyRedemptionLimit: '0',
    seatsAvailable: 25,
    createdAt: Date.now() / 1000 - 86400 * 30,
    updatedAt: Date.now() / 1000,
  },
  {
    poolId: 'main-1',
    poolType: 'main',
    name: 'Main LP',
    description: '$100 minimum deposit, no lock period, T+4 redemption with 15% daily limit.',
    status: 'active',
    totalDeposits: '2500000',
    totalShares: '2500000',
    nav: '1.0012',
    highWaterMark: '1.0015',
    currentDrawdown: '0.0003',
    ddGuardLevel: 'normal',
    minDeposit: '100',
    maxDeposit: '0',
    lockPeriodDays: 0,
    redemptionDelayDays: 4,
    dailyRedemptionLimit: '0.15',
    createdAt: Date.now() / 1000 - 86400 * 30,
    updatedAt: Date.now() / 1000,
  },
  {
    poolId: 'community-1',
    poolType: 'community',
    name: 'Alpha Trend Strategy',
    description: 'Trend-following strategy focused on BTC and ETH with moderate leverage.',
    status: 'active',
    totalDeposits: '150000',
    totalShares: '150000',
    nav: '1.0235',
    highWaterMark: '1.0300',
    currentDrawdown: '0.0063',
    ddGuardLevel: 'normal',
    minDeposit: '100',
    maxDeposit: '50000',
    lockPeriodDays: 7,
    redemptionDelayDays: 3,
    dailyRedemptionLimit: '0.20',
    owner: 'perpdex1abc123def456...',
    managementFee: '0.02',
    performanceFee: '0.20',
    ownerMinStake: '0.05',
    ownerCurrentStake: '0.08',
    isPrivate: false,
    requiresInviteCode: false,
    totalHolders: 42,
    allowedMarkets: ['BTC-USDC', 'ETH-USDC'],
    maxLeverage: '10',
    tags: ['BTC', 'ETH', 'Trend'],
    createdAt: Date.now() / 1000 - 86400 * 14,
    updatedAt: Date.now() / 1000,
  },
];

export const useRiverpoolStore = create<RiverpoolState>((set, get) => ({
  // Initial state
  pools: [],
  selectedPool: null,
  poolStats: null,
  ddGuardState: null,
  navHistory: [],
  userDeposits: [],
  userWithdrawals: [],
  userPoolBalance: null,
  // Community Pool specific
  poolHolders: [],
  poolPositions: [],
  poolTrades: [],
  inviteCodes: [],
  userOwnedPools: [],
  // UI state
  activeTab: 'foundation',
  isLoading: true, // Start with loading=true to show loading state until first fetch completes
  error: null,
  depositAmount: '',
  depositPoolId: '',
  withdrawShares: '',
  withdrawPoolId: '',
  // Community Pool filters
  communityPoolFilter: {
    sortBy: 'tvl',
    sortOrder: 'desc',
    showPrivate: false,
    minTvl: '0',
    tags: [],
  },

  // Actions
  setActiveTab: (tab) => set({ activeTab: tab }),
  setSelectedPool: (pool) => set({ selectedPool: pool }),
  setDepositAmount: (amount) => set({ depositAmount: amount }),
  setDepositPoolId: (poolId) => set({ depositPoolId: poolId }),
  setWithdrawShares: (shares) => set({ withdrawShares: shares }),
  setWithdrawPoolId: (poolId) => set({ withdrawPoolId: poolId }),
  setLoading: (loading) => set({ isLoading: loading }),
  setError: (error) => set({ error }),
  setCommunityPoolFilter: (filter) =>
    set((state) => ({
      communityPoolFilter: { ...state.communityPoolFilter, ...filter },
    })),

  // API actions
  fetchPools: async () => {
    set({ isLoading: true, error: null });
    try {
      const response = await fetch(`${API_BASE}/v1/riverpool/pools`);
      if (!response.ok) throw new Error('Failed to fetch pools');
      const data = await response.json();
      
      const mappedPools: Pool[] = (data.pools || []).map((p: any) => ({
        poolId: p.pool_id,
        poolType: p.pool_type,
        name: p.name,
        description: p.description,
        status: p.status,
        totalDeposits: p.total_deposits,
        totalShares: p.total_shares,
        nav: p.nav,
        highWaterMark: p.high_water_mark,
        currentDrawdown: p.current_drawdown,
        ddGuardLevel: p.dd_guard_level,
        minDeposit: p.min_deposit,
        maxDeposit: p.max_deposit,
        lockPeriodDays: p.lock_period_days,
        redemptionDelayDays: p.redemption_delay_days,
        dailyRedemptionLimit: p.daily_redemption_limit,
        seatsAvailable: p.seats_available,
        createdAt: p.created_at,
        updatedAt: p.updated_at,
        // Community pool fields
        owner: p.owner,
        managementFee: p.management_fee,
        performanceFee: p.performance_fee,
        ownerMinStake: p.owner_min_stake,
        ownerCurrentStake: p.owner_current_stake,
        isPrivate: p.is_private,
        requiresInviteCode: p.requires_invite_code,
        totalHolders: p.total_holders,
        allowedMarkets: p.allowed_markets,
        maxLeverage: p.max_leverage,
        tags: p.tags,
      }));

      set({ pools: mappedPools, isLoading: false });
    } catch (error) {
      // Use mock data in development/testing when API is unavailable
      console.warn('API unavailable, using mock data:', (error as Error).message);
      set({ pools: MOCK_POOLS, isLoading: false, error: null });
    }
  },

  fetchPool: async (poolId: string) => {
    set({ isLoading: true, error: null });
    try {
      const response = await fetch(`${API_BASE}/v1/riverpool/pools/${poolId}`);
      if (!response.ok) throw new Error('Failed to fetch pool');
      const data = await response.json();
      
      const mappedPool: Pool = {
        poolId: data.pool_id,
        poolType: data.pool_type,
        name: data.name,
        description: data.description,
        status: data.status,
        totalDeposits: data.total_deposits,
        totalShares: data.total_shares,
        nav: data.nav,
        highWaterMark: data.high_water_mark,
        currentDrawdown: data.current_drawdown,
        ddGuardLevel: data.dd_guard_level,
        minDeposit: data.min_deposit,
        maxDeposit: data.max_deposit,
        lockPeriodDays: data.lock_period_days,
        redemptionDelayDays: data.redemption_delay_days,
        dailyRedemptionLimit: data.daily_redemption_limit,
        seatsAvailable: data.seats_available,
        createdAt: data.created_at,
        updatedAt: data.updated_at,
        // Community pool fields
        owner: data.owner,
        managementFee: data.management_fee,
        performanceFee: data.performance_fee,
        ownerMinStake: data.owner_min_stake,
        ownerCurrentStake: data.owner_current_stake,
        isPrivate: data.is_private,
        requiresInviteCode: data.requires_invite_code,
        totalHolders: data.total_holders,
        allowedMarkets: data.allowed_markets,
        maxLeverage: data.max_leverage,
        tags: data.tags,
      };

      set({ selectedPool: mappedPool, isLoading: false });
    } catch (error) {
      set({ error: (error as Error).message, isLoading: false });
    }
  },

  fetchPoolStats: async (poolId: string) => {
    try {
      const response = await fetch(`${API_BASE}/v1/riverpool/pools/${poolId}/stats`);
      if (!response.ok) throw new Error('Failed to fetch pool stats');
      const data = await response.json();
      
      const mappedStats: PoolStats = {
        poolId: data.pool_id,
        totalValueLocked: data.total_value_locked,
        totalDepositors: data.total_depositors,
        totalPendingWithdrawals: data.total_pending_withdrawals,
        realizedPnl: data.realized_pnl,
        unrealizedPnl: data.unrealized_pnl,
        totalFeesCollected: data.total_fees_collected,
        return1d: data.return_1d,
        return7d: data.return_7d,
        return30d: data.return_30d,
        returnAllTime: data.return_all_time,
        updatedAt: data.updated_at,
      };

      set({ poolStats: mappedStats });
    } catch (error) {
      console.error('Failed to fetch pool stats:', error);
    }
  },

  fetchDDGuardState: async (poolId: string) => {
    try {
      const response = await fetch(`${API_BASE}/v1/riverpool/pools/${poolId}/ddguard`);
      if (!response.ok) throw new Error('Failed to fetch DDGuard state');
      const data = await response.json();
      
      const mappedState: DDGuardState = {
        poolId: data.pool_id,
        level: data.level,
        peakNav: data.peak_nav,
        currentNav: data.current_nav,
        drawdownPercent: data.drawdown_percent,
        maxExposureLimit: data.max_exposure_limit,
        triggeredAt: data.triggered_at,
        lastCheckedAt: data.last_checked_at,
      };

      set({ ddGuardState: mappedState });
    } catch (error) {
      console.error('Failed to fetch DDGuard state:', error);
    }
  },

  fetchNAVHistory: async (poolId: string, from?: number, to?: number) => {
    try {
      let url = `${API_BASE}/v1/riverpool/pools/${poolId}/nav/history`;
      const params = new URLSearchParams();
      if (from) params.append('from', from.toString());
      if (to) params.append('to', to.toString());
      if (params.toString()) url += `?${params.toString()}`;

      const response = await fetch(url);
      if (!response.ok) throw new Error('Failed to fetch NAV history');
      const data = await response.json();
      
      const mappedHistory: NAVHistory[] = (data.history || []).map((h: any) => ({
        poolId: h.pool_id,
        nav: h.nav,
        totalValue: h.total_value,
        timestamp: h.timestamp,
      }));

      set({ navHistory: mappedHistory });
    } catch (error) {
      console.error('Failed to fetch NAV history:', error);
    }
  },

  fetchUserDeposits: async (user: string) => {
    try {
      const response = await fetch(`${API_BASE}/v1/riverpool/user/${user}/deposits`);
      if (!response.ok) throw new Error('Failed to fetch user deposits');
      const data = await response.json();
      
      const mappedDeposits: Deposit[] = (data.deposits || []).map((d: any) => ({
        depositId: d.deposit_id,
        poolId: d.pool_id,
        depositor: d.depositor,
        amount: d.amount,
        shares: d.shares,
        navAtDeposit: d.nav_at_deposit,
        depositedAt: d.deposited_at,
        unlockAt: d.unlock_at,
        pointsEarned: d.points_earned,
        isLocked: d.is_locked,
      }));

      set({ userDeposits: mappedDeposits });
    } catch (error) {
      console.error('Failed to fetch user deposits:', error);
    }
  },

  fetchUserWithdrawals: async (user: string) => {
    try {
      const response = await fetch(`${API_BASE}/v1/riverpool/user/${user}/withdrawals`);
      if (!response.ok) throw new Error('Failed to fetch user withdrawals');
      const data = await response.json();
      
      const mappedWithdrawals: Withdrawal[] = (data.withdrawals || []).map((w: any) => ({
        withdrawalId: w.withdrawal_id,
        poolId: w.pool_id,
        withdrawer: w.withdrawer,
        sharesRequested: w.shares_requested,
        sharesRedeemed: w.shares_redeemed,
        amountReceived: w.amount_received,
        navAtRequest: w.nav_at_request,
        status: w.status,
        requestedAt: w.requested_at,
        availableAt: w.available_at,
        completedAt: w.completed_at,
        isReady: w.is_ready,
      }));

      set({ userWithdrawals: mappedWithdrawals });
    } catch (error) {
      console.error('Failed to fetch user withdrawals:', error);
    }
  },

  fetchUserPoolBalance: async (poolId: string, user: string) => {
    try {
      const response = await fetch(`${API_BASE}/v1/riverpool/pools/${poolId}/user/${user}/balance`);
      if (!response.ok) throw new Error('Failed to fetch user pool balance');
      const data = await response.json();
      
      const mappedBalance: UserPoolBalance = {
        shares: data.shares,
        value: data.value,
        costBasis: data.cost_basis,
        unrealizedPnl: data.unrealized_pnl,
        pnlPercent: data.pnl_percent,
        unlockAt: data.unlock_at,
        canWithdraw: data.can_withdraw,
      };

      set({ userPoolBalance: mappedBalance });
    } catch (error) {
      console.error('Failed to fetch user pool balance:', error);
    }
  },

  // Transaction actions
  deposit: async (depositor: string, poolId: string, amount: string, inviteCode?: string) => {
    set({ isLoading: true, error: null });
    try {
      const response = await fetch(`${API_BASE}/v1/riverpool/deposit`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          depositor,
          pool_id: poolId,
          amount,
          invite_code: inviteCode,
        }),
      });
      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }
      const data = await response.json();
      set({ isLoading: false });
      // Refresh data
      get().fetchPools();
      get().fetchUserDeposits(depositor);
      return data;
    } catch (error) {
      set({ error: (error as Error).message, isLoading: false });
      throw error;
    }
  },

  requestWithdrawal: async (withdrawer: string, poolId: string, shares: string) => {
    set({ isLoading: true, error: null });
    try {
      const response = await fetch(`${API_BASE}/v1/riverpool/withdrawal/request`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          withdrawer,
          pool_id: poolId,
          shares,
        }),
      });
      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }
      const data = await response.json();
      set({ isLoading: false });
      // Refresh data
      get().fetchUserWithdrawals(withdrawer);
      return data;
    } catch (error) {
      set({ error: (error as Error).message, isLoading: false });
      throw error;
    }
  },

  claimWithdrawal: async (withdrawer: string, withdrawalId: string) => {
    set({ isLoading: true, error: null });
    try {
      const response = await fetch(`${API_BASE}/v1/riverpool/withdrawal/claim`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          withdrawer,
          withdrawal_id: withdrawalId,
        }),
      });
      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }
      const data = await response.json();
      set({ isLoading: false });
      // Refresh data
      get().fetchPools();
      get().fetchUserWithdrawals(withdrawer);
      return data;
    } catch (error) {
      set({ error: (error as Error).message, isLoading: false });
      throw error;
    }
  },

  cancelWithdrawal: async (withdrawer: string, withdrawalId: string) => {
    set({ isLoading: true, error: null });
    try {
      const response = await fetch(`${API_BASE}/v1/riverpool/withdrawal/cancel`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          withdrawer,
          withdrawal_id: withdrawalId,
        }),
      });
      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }
      const data = await response.json();
      set({ isLoading: false });
      // Refresh data
      get().fetchUserWithdrawals(withdrawer);
      return data;
    } catch (error) {
      set({ error: (error as Error).message, isLoading: false });
      throw error;
    }
  },

  // Estimation
  estimateDeposit: async (poolId: string, amount: string) => {
    try {
      const response = await fetch(
        `${API_BASE}/v1/riverpool/pools/${poolId}/estimate/deposit?amount=${amount}`
      );
      if (!response.ok) throw new Error('Failed to estimate deposit');
      return await response.json();
    } catch (error) {
      console.error('Failed to estimate deposit:', error);
      return { shares: '0', nav: '1', sharePrice: '1' };
    }
  },

  estimateWithdrawal: async (poolId: string, shares: string) => {
    try {
      const response = await fetch(
        `${API_BASE}/v1/riverpool/pools/${poolId}/estimate/withdrawal?shares=${shares}`
      );
      if (!response.ok) throw new Error('Failed to estimate withdrawal');
      return await response.json();
    } catch (error) {
      console.error('Failed to estimate withdrawal:', error);
      return {
        amount: '0',
        nav: '1',
        availableAt: 0,
        queuePosition: '0',
        mayBeProrated: false,
      };
    }
  },

  // Computed
  calculateUserTotalValue: () => {
    const { userDeposits, pools } = get();
    let total = new BigNumber(0);

    userDeposits.forEach((deposit) => {
      const pool = pools.find((p) => p.poolId === deposit.poolId);
      if (pool) {
        const value = new BigNumber(deposit.shares).times(pool.nav);
        total = total.plus(value);
      }
    });

    return total.toFixed(2);
  },

  getPoolByType: (type: 'foundation' | 'main' | 'community') => {
    const { pools } = get();
    return pools.filter((p) => p.poolType === type);
  },

  getFilteredCommunityPools: () => {
    const { pools, communityPoolFilter } = get();
    let filtered = pools.filter((p) => p.poolType === 'community');

    // Filter by privacy
    if (!communityPoolFilter.showPrivate) {
      filtered = filtered.filter((p) => !p.isPrivate);
    }

    // Filter by min TVL
    const minTvl = new BigNumber(communityPoolFilter.minTvl);
    if (minTvl.gt(0)) {
      filtered = filtered.filter((p) => new BigNumber(p.totalDeposits).gte(minTvl));
    }

    // Filter by tags
    if (communityPoolFilter.tags.length > 0) {
      filtered = filtered.filter((p) =>
        communityPoolFilter.tags.some((tag) => p.tags?.includes(tag))
      );
    }

    // Sort
    filtered.sort((a, b) => {
      let aVal: number, bVal: number;
      switch (communityPoolFilter.sortBy) {
        case 'tvl':
          aVal = parseFloat(a.totalDeposits);
          bVal = parseFloat(b.totalDeposits);
          break;
        case 'nav':
          aVal = parseFloat(a.nav);
          bVal = parseFloat(b.nav);
          break;
        case 'holders':
          aVal = a.totalHolders || 0;
          bVal = b.totalHolders || 0;
          break;
        case 'created':
          aVal = a.createdAt;
          bVal = b.createdAt;
          break;
        default:
          aVal = 0;
          bVal = 0;
      }
      return communityPoolFilter.sortOrder === 'asc' ? aVal - bVal : bVal - aVal;
    });

    return filtered;
  },

  // Community Pool API actions
  fetchPoolHolders: async (poolId: string) => {
    try {
      const response = await fetch(`${API_BASE}/v1/riverpool/community/${poolId}/holders`);
      if (!response.ok) throw new Error('Failed to fetch pool holders');
      const data = await response.json();
      
      const mappedHolders: PoolHolder[] = (data.holders || []).map((h: any) => ({
        address: h.address,
        shares: h.shares,
        value: h.value,
        depositedAt: h.deposited_at,
        isOwner: h.is_owner,
      }));

      set({ poolHolders: mappedHolders });
    } catch (error) {
      console.error('Failed to fetch pool holders:', error);
    }
  },

  fetchPoolPositions: async (poolId: string) => {
    try {
      const response = await fetch(`${API_BASE}/v1/riverpool/community/${poolId}/positions`);
      if (!response.ok) throw new Error('Failed to fetch pool positions');
      const data = await response.json();
      
      const mappedPositions: PoolPosition[] = (data.positions || []).map((p: any) => ({
        positionId: p.position_id,
        marketId: p.market_id,
        side: p.side,
        size: p.size,
        entryPrice: p.entry_price,
        markPrice: p.mark_price,
        pnl: p.pnl,
        pnlPercent: p.pnl_percent,
        leverage: p.leverage,
        liquidationPrice: p.liquidation_price,
        margin: p.margin,
      }));

      set({ poolPositions: mappedPositions });
    } catch (error) {
      console.error('Failed to fetch pool positions:', error);
    }
  },

  fetchPoolTrades: async (poolId: string, limit = 50) => {
    try {
      const response = await fetch(
        `${API_BASE}/v1/riverpool/community/${poolId}/trades?limit=${limit}`
      );
      if (!response.ok) throw new Error('Failed to fetch pool trades');
      const data = await response.json();
      
      const mappedTrades: PoolTrade[] = (data.trades || []).map((t: any) => ({
        tradeId: t.trade_id,
        marketId: t.market_id,
        side: t.side,
        price: t.price,
        size: t.size,
        fee: t.fee,
        pnl: t.pnl,
        timestamp: t.timestamp,
      }));

      set({ poolTrades: mappedTrades });
    } catch (error) {
      console.error('Failed to fetch pool trades:', error);
    }
  },

  fetchInviteCodes: async (poolId: string) => {
    try {
      const response = await fetch(`${API_BASE}/v1/riverpool/community/${poolId}/invites`);
      if (!response.ok) throw new Error('Failed to fetch invite codes');
      const data = await response.json();
      
      const mappedCodes: InviteCode[] = (data.codes || []).map((c: any) => ({
        code: c.code,
        maxUses: c.max_uses,
        usedCount: c.used_count,
        expiresAt: c.expires_at,
        createdAt: c.created_at,
        isActive: c.is_active,
      }));

      set({ inviteCodes: mappedCodes });
    } catch (error) {
      console.error('Failed to fetch invite codes:', error);
    }
  },

  fetchUserOwnedPools: async (owner: string) => {
    try {
      const response = await fetch(`${API_BASE}/v1/riverpool/user/${owner}/owned-pools`);
      if (!response.ok) throw new Error('Failed to fetch owned pools');
      const data = await response.json();
      
      const mappedPools: Pool[] = (data.pools || []).map((p: any) => ({
        poolId: p.pool_id,
        poolType: p.pool_type,
        name: p.name,
        description: p.description,
        status: p.status,
        totalDeposits: p.total_deposits,
        totalShares: p.total_shares,
        nav: p.nav,
        highWaterMark: p.high_water_mark,
        currentDrawdown: p.current_drawdown,
        ddGuardLevel: p.dd_guard_level,
        minDeposit: p.min_deposit,
        maxDeposit: p.max_deposit,
        lockPeriodDays: p.lock_period_days,
        redemptionDelayDays: p.redemption_delay_days,
        dailyRedemptionLimit: p.daily_redemption_limit,
        seatsAvailable: p.seats_available,
        createdAt: p.created_at,
        updatedAt: p.updated_at,
        owner: p.owner,
        managementFee: p.management_fee,
        performanceFee: p.performance_fee,
        ownerMinStake: p.owner_min_stake,
        ownerCurrentStake: p.owner_current_stake,
        isPrivate: p.is_private,
        requiresInviteCode: p.requires_invite_code,
        totalHolders: p.total_holders,
        allowedMarkets: p.allowed_markets,
        maxLeverage: p.max_leverage,
        tags: p.tags,
      }));

      set({ userOwnedPools: mappedPools });
    } catch (error) {
      console.error('Failed to fetch owned pools:', error);
    }
  },

  // Community Pool transaction actions
  createCommunityPool: async (owner: string, config: CreateCommunityPoolConfig) => {
    set({ isLoading: true, error: null });
    try {
      const response = await fetch(`${API_BASE}/v1/riverpool/community/create`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ owner, ...config }),
      });
      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }
      const data = await response.json();
      set({ isLoading: false });
      get().fetchPools();
      get().fetchUserOwnedPools(owner);
      return data;
    } catch (error) {
      set({ error: (error as Error).message, isLoading: false });
      throw error;
    }
  },

  depositOwnerStake: async (owner: string, poolId: string, amount: string) => {
    set({ isLoading: true, error: null });
    try {
      const response = await fetch(`${API_BASE}/v1/riverpool/community/${poolId}/stake`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ owner, amount }),
      });
      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }
      const data = await response.json();
      set({ isLoading: false });
      get().fetchPool(poolId);
      return data;
    } catch (error) {
      set({ error: (error as Error).message, isLoading: false });
      throw error;
    }
  },

  generateInviteCode: async (
    owner: string,
    poolId: string,
    maxUses: number,
    expiresInDays: number
  ) => {
    set({ isLoading: true, error: null });
    try {
      const response = await fetch(`${API_BASE}/v1/riverpool/community/${poolId}/invites`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ owner, max_uses: maxUses, expires_in_days: expiresInDays }),
      });
      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }
      const data = await response.json();
      set({ isLoading: false });
      get().fetchInviteCodes(poolId);
      return data;
    } catch (error) {
      set({ error: (error as Error).message, isLoading: false });
      throw error;
    }
  },

  pausePool: async (owner: string, poolId: string) => {
    set({ isLoading: true, error: null });
    try {
      const response = await fetch(`${API_BASE}/v1/riverpool/community/${poolId}/pause`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ owner }),
      });
      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }
      const data = await response.json();
      set({ isLoading: false });
      get().fetchPool(poolId);
      return data;
    } catch (error) {
      set({ error: (error as Error).message, isLoading: false });
      throw error;
    }
  },

  resumePool: async (owner: string, poolId: string) => {
    set({ isLoading: true, error: null });
    try {
      const response = await fetch(`${API_BASE}/v1/riverpool/community/${poolId}/resume`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ owner }),
      });
      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }
      const data = await response.json();
      set({ isLoading: false });
      get().fetchPool(poolId);
      return data;
    } catch (error) {
      set({ error: (error as Error).message, isLoading: false });
      throw error;
    }
  },

  closePool: async (owner: string, poolId: string) => {
    set({ isLoading: true, error: null });
    try {
      const response = await fetch(`${API_BASE}/v1/riverpool/community/${poolId}/close`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ owner }),
      });
      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }
      const data = await response.json();
      set({ isLoading: false });
      get().fetchPool(poolId);
      get().fetchPools();
      return data;
    } catch (error) {
      set({ error: (error as Error).message, isLoading: false });
      throw error;
    }
  },
}));
