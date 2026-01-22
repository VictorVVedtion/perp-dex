/**
 * RiverPool SDK Client
 * TypeScript client for interacting with RiverPool liquidity pools
 */

import axios, { AxiosInstance } from 'axios';

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
  seatsAvailable?: number;
  createdAt: number;
  updatedAt: number;
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

export interface DepositEstimate {
  shares: string;
  nav: string;
  sharePrice: string;
}

export interface WithdrawalEstimate {
  amount: string;
  nav: string;
  availableAt: number;
  queuePosition: string;
  mayBeProrated: boolean;
}

export interface DepositResponse {
  depositId: string;
  sharesReceived: string;
  navAtDeposit: string;
  unlockAt: number;
}

export interface WithdrawalRequestResponse {
  withdrawalId: string;
  sharesRequested: string;
  estimatedAmount: string;
  availableAt: number;
  queuePosition: string;
}

export interface WithdrawalClaimResponse {
  amountReceived: string;
  sharesRedeemed: string;
  remainingShares: string;
}

export interface WithdrawalCancelResponse {
  sharesReturned: string;
}

export interface RevenueStats {
  poolId: string;
  totalRevenue: string;
  spreadRevenue: string;
  fundingRevenue: string;
  liquidationProfit: string;
  tradingPnl: string;
  feeRebates: string;
  return1d: string;
  return7d: string;
  return30d: string;
  lastUpdated: number;
}

export interface RevenueRecord {
  recordId: string;
  poolId: string;
  source: 'spread' | 'funding' | 'liquidation' | 'trading' | 'fees';
  amount: string;
  navImpact: string;
  timestamp: number;
  blockHeight: number;
  marketId?: string;
  details?: string;
}

export interface RevenueBreakdown {
  poolId: string;
  period: string;
  totalAmount: string;
  breakdown: {
    spread: string;
    funding: string;
    liquidation: string;
    trading: string;
    fees: string;
  };
}

export interface RiverpoolClientConfig {
  baseUrl: string;
  timeout?: number;
}

/**
 * RiverPool Client
 * Provides methods for interacting with RiverPool liquidity pools
 */
export class RiverpoolClient {
  private client: AxiosInstance;

  constructor(config: RiverpoolClientConfig) {
    this.client = axios.create({
      baseURL: config.baseUrl,
      timeout: config.timeout || 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    });
  }

  // ============ Pool Queries ============

  /**
   * Get all pools
   */
  async getPools(offset = 0, limit = 20): Promise<{ pools: Pool[]; total: number }> {
    const response = await this.client.get('/v1/riverpool/pools', {
      params: { offset, limit },
    });
    return response.data;
  }

  /**
   * Get a single pool by ID
   */
  async getPool(poolId: string): Promise<Pool> {
    const response = await this.client.get(`/v1/riverpool/pools/${poolId}`);
    return response.data;
  }

  /**
   * Get pools by type
   */
  async getPoolsByType(
    poolType: 'foundation' | 'main' | 'community'
  ): Promise<{ pools: Pool[] }> {
    const response = await this.client.get(`/v1/riverpool/pools/type/${poolType}`);
    return response.data;
  }

  /**
   * Get pool statistics
   */
  async getPoolStats(poolId: string): Promise<PoolStats> {
    const response = await this.client.get(`/v1/riverpool/pools/${poolId}/stats`);
    return response.data;
  }

  /**
   * Get DDGuard state for a pool
   */
  async getDDGuardState(poolId: string): Promise<DDGuardState> {
    const response = await this.client.get(`/v1/riverpool/pools/${poolId}/ddguard`);
    return response.data;
  }

  /**
   * Get NAV history for a pool
   */
  async getNAVHistory(
    poolId: string,
    from?: number,
    to?: number
  ): Promise<{ history: NAVHistory[] }> {
    const params: Record<string, string> = {};
    if (from) params.from = from.toString();
    if (to) params.to = to.toString();

    const response = await this.client.get(`/v1/riverpool/pools/${poolId}/nav/history`, {
      params,
    });
    return response.data;
  }

  // ============ User Queries ============

  /**
   * Get user deposits
   */
  async getUserDeposits(
    user: string
  ): Promise<{ deposits: Deposit[]; totalValue: string }> {
    const response = await this.client.get(`/v1/riverpool/user/${user}/deposits`);
    return response.data;
  }

  /**
   * Get user withdrawals
   */
  async getUserWithdrawals(user: string): Promise<{ withdrawals: Withdrawal[] }> {
    const response = await this.client.get(`/v1/riverpool/user/${user}/withdrawals`);
    return response.data;
  }

  /**
   * Get user's balance in a pool
   */
  async getUserPoolBalance(poolId: string, user: string): Promise<UserPoolBalance> {
    const response = await this.client.get(
      `/v1/riverpool/pools/${poolId}/user/${user}/balance`
    );
    return response.data;
  }

  // ============ Pool Queries ============

  /**
   * Get deposits in a pool
   */
  async getPoolDeposits(
    poolId: string,
    offset = 0,
    limit = 20
  ): Promise<{ deposits: Deposit[]; total: number }> {
    const response = await this.client.get(`/v1/riverpool/pools/${poolId}/deposits`, {
      params: { offset, limit },
    });
    return response.data;
  }

  /**
   * Get pending withdrawals for a pool
   */
  async getPendingWithdrawals(poolId: string): Promise<{
    withdrawals: Withdrawal[];
    totalPendingShares: string;
    totalPendingValue: string;
    dailyLimitRemaining: string;
  }> {
    const response = await this.client.get(
      `/v1/riverpool/pools/${poolId}/withdrawals/pending`
    );
    return response.data;
  }

  // ============ Estimation ============

  /**
   * Estimate shares for a deposit
   */
  async estimateDeposit(poolId: string, amount: string): Promise<DepositEstimate> {
    const response = await this.client.get(
      `/v1/riverpool/pools/${poolId}/estimate/deposit`,
      {
        params: { amount },
      }
    );
    return response.data;
  }

  /**
   * Estimate amount for a withdrawal
   */
  async estimateWithdrawal(poolId: string, shares: string): Promise<WithdrawalEstimate> {
    const response = await this.client.get(
      `/v1/riverpool/pools/${poolId}/estimate/withdrawal`,
      {
        params: { shares },
      }
    );
    return response.data;
  }

  // ============ Transactions ============

  /**
   * Deposit into a pool
   */
  async deposit(
    depositor: string,
    poolId: string,
    amount: string,
    inviteCode?: string
  ): Promise<DepositResponse> {
    const response = await this.client.post('/v1/riverpool/deposit', {
      depositor,
      pool_id: poolId,
      amount,
      invite_code: inviteCode,
    });
    return response.data;
  }

  /**
   * Request a withdrawal
   */
  async requestWithdrawal(
    withdrawer: string,
    poolId: string,
    shares: string
  ): Promise<WithdrawalRequestResponse> {
    const response = await this.client.post('/v1/riverpool/withdrawal/request', {
      withdrawer,
      pool_id: poolId,
      shares,
    });
    return response.data;
  }

  /**
   * Claim a withdrawal
   */
  async claimWithdrawal(
    withdrawer: string,
    withdrawalId: string
  ): Promise<WithdrawalClaimResponse> {
    const response = await this.client.post('/v1/riverpool/withdrawal/claim', {
      withdrawer,
      withdrawal_id: withdrawalId,
    });
    return response.data;
  }

  /**
   * Cancel a withdrawal
   */
  async cancelWithdrawal(
    withdrawer: string,
    withdrawalId: string
  ): Promise<WithdrawalCancelResponse> {
    const response = await this.client.post('/v1/riverpool/withdrawal/cancel', {
      withdrawer,
      withdrawal_id: withdrawalId,
    });
    return response.data;
  }

  // ============ Revenue Queries ============

  /**
   * Get pool revenue statistics
   */
  async getPoolRevenue(poolId: string): Promise<RevenueStats> {
    const response = await this.client.get(`/v1/riverpool/pools/${poolId}/revenue`);
    return response.data;
  }

  /**
   * Get revenue records for a pool
   */
  async getRevenueRecords(
    poolId: string,
    from?: number,
    to?: number,
    limit?: number
  ): Promise<{ poolId: string; records: RevenueRecord[]; total: number }> {
    const params: Record<string, string> = {};
    if (from) params.from = from.toString();
    if (to) params.to = to.toString();
    if (limit) params.limit = limit.toString();

    const response = await this.client.get(`/v1/riverpool/pools/${poolId}/revenue/records`, {
      params,
    });
    return response.data;
  }

  /**
   * Get revenue breakdown by source
   */
  async getRevenueBreakdown(
    poolId: string,
    period: '1d' | '7d' | '30d' | 'all' = '7d'
  ): Promise<RevenueBreakdown> {
    const response = await this.client.get(`/v1/riverpool/pools/${poolId}/revenue/breakdown`, {
      params: { period },
    });
    return response.data;
  }

  // ============ Utility Methods ============

  /**
   * Get Foundation LP pool
   */
  async getFoundationPool(): Promise<Pool | null> {
    try {
      return await this.getPool('foundation-lp');
    } catch {
      return null;
    }
  }

  /**
   * Get Main LP pool
   */
  async getMainPool(): Promise<Pool | null> {
    try {
      return await this.getPool('main-lp');
    } catch {
      return null;
    }
  }

  /**
   * Check if Foundation LP has available seats
   */
  async hasFoundationSeats(): Promise<boolean> {
    const pool = await this.getFoundationPool();
    return pool !== null && pool.seatsAvailable !== undefined && pool.seatsAvailable > 0;
  }

  /**
   * Calculate total user value across all pools
   */
  async getUserTotalValue(user: string): Promise<string> {
    const { deposits, totalValue } = await this.getUserDeposits(user);
    return totalValue;
  }
}

/**
 * Create a RiverPool client instance
 */
export function createRiverpoolClient(baseUrl: string, timeout?: number): RiverpoolClient {
  return new RiverpoolClient({ baseUrl, timeout });
}

export default RiverpoolClient;
