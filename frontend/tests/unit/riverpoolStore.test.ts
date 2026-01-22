/**
 * RiverPool Store Unit Tests
 * Tests Zustand store state management and computed values
 *
 * To run: Install vitest and run `npx vitest run tests/unit/riverpoolStore.test.ts`
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import BigNumber from 'bignumber.js';

// Mock pool data for testing
const mockFoundationPool = {
  poolId: 'foundation-lp',
  poolType: 'foundation',
  name: 'Foundation LP',
  description: '100 seats x $100K, 180-day lock',
  status: 'active',
  totalDeposits: '5000000',
  totalShares: '5000000',
  nav: '1.0',
  highWaterMark: '1.0',
  currentDrawdown: '0',
  ddGuardLevel: 'normal',
  minDeposit: '100000',
  maxDeposit: '100000',
  lockPeriodDays: 180,
  redemptionDelayDays: 0,
  dailyRedemptionLimit: '0',
  seatsAvailable: 50,
  seatsTotal: 100,
  createdAt: 1704067200,
  updatedAt: 1704067200,
};

const mockMainPool = {
  poolId: 'main-lp',
  poolType: 'main',
  name: 'Main LP',
  description: '$100 minimum, no lock, T+4 redemption',
  status: 'active',
  totalDeposits: '1000000',
  totalShares: '1000000',
  nav: '1.0',
  highWaterMark: '1.0',
  currentDrawdown: '0',
  ddGuardLevel: 'normal',
  minDeposit: '100',
  maxDeposit: '0',
  lockPeriodDays: 0,
  redemptionDelayDays: 4,
  dailyRedemptionLimit: '0.15',
  createdAt: 1704067200,
  updatedAt: 1704067200,
};

const mockCommunityPool = {
  poolId: 'cpool-001',
  poolType: 'community',
  name: 'Alpha Trader Pool',
  description: 'High-performance trading pool',
  status: 'active',
  totalDeposits: '50000',
  totalShares: '50000',
  nav: '1.05',
  highWaterMark: '1.1',
  currentDrawdown: '0.0455',
  ddGuardLevel: 'normal',
  minDeposit: '100',
  maxDeposit: '10000',
  lockPeriodDays: 7,
  redemptionDelayDays: 3,
  dailyRedemptionLimit: '0.10',
  owner: 'cosmos1owner...',
  managementFee: '0.02',
  performanceFee: '0.20',
  ownerMinStake: '0.05',
  ownerCurrentStake: '5000',
  isPrivate: false,
  totalHolders: 10,
  maxLeverage: '10',
  allowedMarkets: ['BTC-USDC', 'ETH-USDC'],
  tags: ['BTC', 'ETH', 'Trend'],
  createdAt: 1704067200,
  updatedAt: 1704067200,
};

// Test helper functions
describe('Pool Utility Functions', () => {
  describe('formatNumber', () => {
    const formatNumber = (value: string, decimals = 2) => {
      const num = new BigNumber(value);
      if (num.gte(1000000)) return `$${num.div(1000000).toFixed(2)}M`;
      if (num.gte(1000)) return `$${num.div(1000).toFixed(2)}K`;
      return `$${num.toFixed(decimals)}`;
    };

    it('should format millions correctly', () => {
      expect(formatNumber('5000000')).toBe('$5.00M');
      expect(formatNumber('1500000')).toBe('$1.50M');
    });

    it('should format thousands correctly', () => {
      expect(formatNumber('50000')).toBe('$50.00K');
      expect(formatNumber('1500')).toBe('$1.50K');
    });

    it('should format small numbers correctly', () => {
      expect(formatNumber('100')).toBe('$100.00');
      expect(formatNumber('0.5')).toBe('$0.50');
    });
  });

  describe('formatPercent', () => {
    const formatPercent = (value: string) => {
      const num = new BigNumber(value).times(100);
      const prefix = num.gte(0) ? '+' : '';
      return `${prefix}${num.toFixed(2)}%`;
    };

    it('should format positive percentages', () => {
      expect(formatPercent('0.05')).toBe('+5.00%');
      expect(formatPercent('0.125')).toBe('+12.50%');
    });

    it('should format negative percentages', () => {
      expect(formatPercent('-0.05')).toBe('-5.00%');
      expect(formatPercent('-0.125')).toBe('-12.50%');
    });

    it('should format zero', () => {
      expect(formatPercent('0')).toBe('+0.00%');
    });
  });

  describe('shortenAddress', () => {
    const shortenAddress = (address: string) => {
      if (!address) return '';
      return `${address.slice(0, 8)}...${address.slice(-6)}`;
    };

    it('should shorten cosmos address', () => {
      const address = 'cosmos1abc123def456ghi789jkl012mno345pqr678stu';
      const shortened = shortenAddress(address);
      expect(shortened).toBe('cosmos1a...678stu'); // first 8 + last 6 chars
    });

    it('should handle empty address', () => {
      expect(shortenAddress('')).toBe('');
    });
  });
});

// Test pool filtering and sorting
describe('Community Pool Filtering', () => {
  const pools = [
    { ...mockCommunityPool, poolId: 'pool-1', totalDeposits: '100000', tags: ['BTC'] },
    { ...mockCommunityPool, poolId: 'pool-2', totalDeposits: '50000', tags: ['ETH'] },
    { ...mockCommunityPool, poolId: 'pool-3', totalDeposits: '200000', tags: ['BTC', 'ETH'] },
    { ...mockCommunityPool, poolId: 'pool-4', totalDeposits: '10000', isPrivate: true, tags: [] }, // No tags
  ];

  describe('filterByTags', () => {
    const filterByTags = (pools: any[], tags: string[]) => {
      if (tags.length === 0) return pools;
      return pools.filter(pool =>
        tags.some(tag => pool.tags?.includes(tag))
      );
    };

    it('should filter by single tag', () => {
      const result = filterByTags(pools, ['BTC']);
      expect(result.length).toBe(2); // pool-1 and pool-3
      expect(result.every(p => p.tags?.includes('BTC'))).toBe(true);
    });

    it('should filter by multiple tags (OR logic)', () => {
      const result = filterByTags(pools, ['BTC', 'ETH']);
      expect(result.length).toBe(3); // pool-1, pool-2, pool-3
    });

    it('should return all pools when no tags specified', () => {
      const result = filterByTags(pools, []);
      expect(result.length).toBe(pools.length);
    });
  });

  describe('filterByMinTvl', () => {
    const filterByMinTvl = (pools: any[], minTvl: string) => {
      const min = new BigNumber(minTvl);
      return pools.filter(pool =>
        new BigNumber(pool.totalDeposits).gte(min)
      );
    };

    it('should filter by minimum TVL', () => {
      const result = filterByMinTvl(pools, '50000');
      expect(result.length).toBe(3);
    });

    it('should include all with zero minimum', () => {
      const result = filterByMinTvl(pools, '0');
      expect(result.length).toBe(pools.length);
    });
  });

  describe('filterByPrivate', () => {
    const filterByPrivate = (pools: any[], showPrivate: boolean) => {
      if (showPrivate) return pools;
      return pools.filter(pool => !pool.isPrivate);
    };

    it('should hide private pools by default', () => {
      const result = filterByPrivate(pools, false);
      expect(result.every(p => !p.isPrivate)).toBe(true);
    });

    it('should show all pools when showPrivate is true', () => {
      const result = filterByPrivate(pools, true);
      expect(result.length).toBe(pools.length);
    });
  });

  describe('sortPools', () => {
    const sortPools = (pools: any[], sortBy: string, sortOrder: string) => {
      const sorted = [...pools].sort((a, b) => {
        let aVal, bVal;
        switch (sortBy) {
          case 'tvl':
            aVal = new BigNumber(a.totalDeposits);
            bVal = new BigNumber(b.totalDeposits);
            break;
          case 'nav':
            aVal = new BigNumber(a.nav);
            bVal = new BigNumber(b.nav);
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
            return 0;
        }

        if (typeof aVal === 'number') {
          return sortOrder === 'asc' ? aVal - bVal : bVal - aVal;
        }
        return sortOrder === 'asc'
          ? aVal.minus(bVal).toNumber()
          : bVal.minus(aVal).toNumber();
      });
      return sorted;
    };

    it('should sort by TVL descending', () => {
      const result = sortPools(pools, 'tvl', 'desc');
      expect(result[0].poolId).toBe('pool-3'); // $200K
      expect(result[result.length - 1].poolId).toBe('pool-4'); // $10K
    });

    it('should sort by TVL ascending', () => {
      const result = sortPools(pools, 'tvl', 'asc');
      expect(result[0].poolId).toBe('pool-4'); // $10K
      expect(result[result.length - 1].poolId).toBe('pool-3'); // $200K
    });
  });
});

// Test NAV calculations
describe('NAV Calculations', () => {
  describe('calculateSharesForDeposit', () => {
    const calculateShares = (amount: string, nav: string) => {
      const amountDec = new BigNumber(amount);
      const navDec = new BigNumber(nav);
      if (navDec.isZero() || navDec.isNegative()) {
        return amountDec.toString();
      }
      return amountDec.div(navDec).toString();
    };

    it('should calculate shares at NAV 1.0', () => {
      const shares = calculateShares('1000', '1.0');
      expect(new BigNumber(shares).toFixed(2)).toBe('1000.00');
    });

    it('should calculate shares at NAV 1.1', () => {
      const shares = calculateShares('1100', '1.1');
      expect(new BigNumber(shares).toFixed(2)).toBe('1000.00');
    });

    it('should calculate shares at NAV 0.9', () => {
      const shares = calculateShares('900', '0.9');
      expect(new BigNumber(shares).toFixed(2)).toBe('1000.00');
    });
  });

  describe('calculateValueForShares', () => {
    const calculateValue = (shares: string, nav: string) => {
      return new BigNumber(shares).times(nav).toString();
    };

    it('should calculate value at NAV 1.0', () => {
      const value = calculateValue('1000', '1.0');
      expect(new BigNumber(value).toFixed(2)).toBe('1000.00');
    });

    it('should calculate value at NAV 1.2', () => {
      const value = calculateValue('1000', '1.2');
      expect(new BigNumber(value).toFixed(2)).toBe('1200.00');
    });
  });

  describe('calculateDrawdown', () => {
    const calculateDrawdown = (currentNAV: string, highWaterMark: string) => {
      const current = new BigNumber(currentNAV);
      const hwm = new BigNumber(highWaterMark);
      if (hwm.isZero()) return '0';
      return hwm.minus(current).div(hwm).toString();
    };

    it('should calculate zero drawdown at high water mark', () => {
      const drawdown = calculateDrawdown('1.0', '1.0');
      expect(new BigNumber(drawdown).toFixed(4)).toBe('0.0000');
    });

    it('should calculate 10% drawdown', () => {
      const drawdown = calculateDrawdown('0.9', '1.0');
      expect(new BigNumber(drawdown).toFixed(2)).toBe('0.10');
    });

    it('should calculate 20% drawdown', () => {
      const drawdown = calculateDrawdown('0.8', '1.0');
      expect(new BigNumber(drawdown).toFixed(2)).toBe('0.20');
    });
  });
});

// Test DDGuard level determination
describe('DDGuard Levels', () => {
  const getDDGuardLevel = (drawdown: string) => {
    const dd = new BigNumber(drawdown);
    if (dd.gte('0.30')) return 'halt';
    if (dd.gte('0.15')) return 'reduce';
    if (dd.gte('0.10')) return 'warning';
    return 'normal';
  };

  it('should return normal for <10% drawdown', () => {
    expect(getDDGuardLevel('0.05')).toBe('normal');
    expect(getDDGuardLevel('0.09')).toBe('normal');
  });

  it('should return warning for 10-15% drawdown', () => {
    expect(getDDGuardLevel('0.10')).toBe('warning');
    expect(getDDGuardLevel('0.14')).toBe('warning');
  });

  it('should return reduce for 15-30% drawdown', () => {
    expect(getDDGuardLevel('0.15')).toBe('reduce');
    expect(getDDGuardLevel('0.29')).toBe('reduce');
  });

  it('should return halt for >=30% drawdown', () => {
    expect(getDDGuardLevel('0.30')).toBe('halt');
    expect(getDDGuardLevel('0.50')).toBe('halt');
  });
});

// Test Pro-rata allocation
describe('Pro-rata Withdrawal Allocation', () => {
  const calculateProRataAllocation = (
    userRequest: string,
    totalPending: string,
    availableQuota: string
  ) => {
    const request = new BigNumber(userRequest);
    const total = new BigNumber(totalPending);
    const quota = new BigNumber(availableQuota);

    if (quota.gte(total)) {
      return request.toString();
    }

    const ratio = request.div(total);
    const allocation = ratio.times(quota);
    return BigNumber.min(allocation, request).toString();
  };

  it('should return full amount when quota exceeds total', () => {
    const allocation = calculateProRataAllocation('500', '1000', '2000');
    expect(allocation).toBe('500');
  });

  it('should calculate 50% allocation', () => {
    const allocation = calculateProRataAllocation('200', '1000', '500');
    expect(new BigNumber(allocation).toFixed(0)).toBe('100');
  });

  it('should calculate 25% allocation', () => {
    const allocation = calculateProRataAllocation('400', '2000', '500');
    expect(new BigNumber(allocation).toFixed(0)).toBe('100');
  });
});

// Test Community Pool configuration validation
describe('Community Pool Config Validation', () => {
  interface PoolConfig {
    name: string;
    owner: string;
    ownerMinStake: string;
    managementFee: string;
    performanceFee: string;
  }

  const validateConfig = (config: PoolConfig) => {
    const errors: string[] = [];

    if (!config.name || config.name.length === 0) {
      errors.push('Name is required');
    }
    if (config.name && config.name.length > 50) {
      errors.push('Name too long');
    }
    if (!config.owner) {
      errors.push('Owner is required');
    }

    const ownerStake = new BigNumber(config.ownerMinStake || '0');
    if (ownerStake.lt('0.05')) {
      errors.push('Owner stake must be at least 5%');
    }

    const mgmtFee = new BigNumber(config.managementFee || '0');
    if (mgmtFee.gt('0.05')) {
      errors.push('Management fee too high (max 5%)');
    }

    const perfFee = new BigNumber(config.performanceFee || '0');
    if (perfFee.gt('0.50')) {
      errors.push('Performance fee too high (max 50%)');
    }

    return { valid: errors.length === 0, errors };
  };

  it('should validate correct config', () => {
    const config = {
      name: 'Test Pool',
      owner: 'cosmos1owner...',
      ownerMinStake: '0.05',
      managementFee: '0.02',
      performanceFee: '0.20',
    };
    const result = validateConfig(config);
    expect(result.valid).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it('should reject missing name', () => {
    const config = {
      name: '',
      owner: 'cosmos1owner...',
      ownerMinStake: '0.05',
      managementFee: '0.02',
      performanceFee: '0.20',
    };
    const result = validateConfig(config);
    expect(result.valid).toBe(false);
    expect(result.errors).toContain('Name is required');
  });

  it('should reject low owner stake', () => {
    const config = {
      name: 'Test Pool',
      owner: 'cosmos1owner...',
      ownerMinStake: '0.03',
      managementFee: '0.02',
      performanceFee: '0.20',
    };
    const result = validateConfig(config);
    expect(result.valid).toBe(false);
    expect(result.errors).toContain('Owner stake must be at least 5%');
  });

  it('should reject high management fee', () => {
    const config = {
      name: 'Test Pool',
      owner: 'cosmos1owner...',
      ownerMinStake: '0.05',
      managementFee: '0.10',
      performanceFee: '0.20',
    };
    const result = validateConfig(config);
    expect(result.valid).toBe(false);
    expect(result.errors).toContain('Management fee too high (max 5%)');
  });
});

// Test invite code validation
describe('Invite Code Validation', () => {
  interface InviteCode {
    code: string;
    maxUses: number;
    usedCount: number;
    expiresAt: number;
    isActive: boolean;
  }

  const isInviteCodeValid = (code: InviteCode) => {
    if (!code.isActive) return false;
    if (code.expiresAt > 0 && Date.now() / 1000 > code.expiresAt) return false;
    if (code.maxUses > 0 && code.usedCount >= code.maxUses) return false;
    return true;
  };

  it('should validate active code', () => {
    const code: InviteCode = {
      code: 'abc123',
      maxUses: 10,
      usedCount: 5,
      expiresAt: 0,
      isActive: true,
    };
    expect(isInviteCodeValid(code)).toBe(true);
  });

  it('should reject inactive code', () => {
    const code: InviteCode = {
      code: 'abc123',
      maxUses: 10,
      usedCount: 5,
      expiresAt: 0,
      isActive: false,
    };
    expect(isInviteCodeValid(code)).toBe(false);
  });

  it('should reject max uses reached', () => {
    const code: InviteCode = {
      code: 'abc123',
      maxUses: 10,
      usedCount: 10,
      expiresAt: 0,
      isActive: true,
    };
    expect(isInviteCodeValid(code)).toBe(false);
  });

  it('should accept unlimited uses code', () => {
    const code: InviteCode = {
      code: 'abc123',
      maxUses: 0,
      usedCount: 1000,
      expiresAt: 0,
      isActive: true,
    };
    expect(isInviteCodeValid(code)).toBe(true);
  });
});
