/**
 * NAVChart Component
 * Displays historical NAV data for a pool
 */

import { useEffect, useRef, useState } from 'react';
import { useRiverpoolStore } from '@/stores/riverpoolStore';
import BigNumber from 'bignumber.js';

interface NAVChartProps {
  poolId: string;
}

type TimeRange = '24h' | '7d' | '30d' | 'all';

export default function NAVChart({ poolId }: NAVChartProps) {
  const { navHistory, fetchNAVHistory } = useRiverpoolStore();
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [timeRange, setTimeRange] = useState<TimeRange>('7d');
  const [hoveredPoint, setHoveredPoint] = useState<{
    nav: string;
    timestamp: number;
    x: number;
    y: number;
  } | null>(null);

  useEffect(() => {
    const now = Math.floor(Date.now() / 1000);
    let fromTime = 0;

    switch (timeRange) {
      case '24h':
        fromTime = now - 24 * 60 * 60;
        break;
      case '7d':
        fromTime = now - 7 * 24 * 60 * 60;
        break;
      case '30d':
        fromTime = now - 30 * 24 * 60 * 60;
        break;
      case 'all':
        fromTime = 0;
        break;
    }

    fetchNAVHistory(poolId, fromTime, now);
  }, [poolId, timeRange, fetchNAVHistory]);

  useEffect(() => {
    if (!canvasRef.current || navHistory.length === 0) return;

    const canvas = canvasRef.current;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    // Get dimensions
    const rect = canvas.getBoundingClientRect();
    const dpr = window.devicePixelRatio || 1;
    canvas.width = rect.width * dpr;
    canvas.height = rect.height * dpr;
    ctx.scale(dpr, dpr);

    const width = rect.width;
    const height = rect.height;
    const padding = { top: 20, right: 20, bottom: 30, left: 60 };

    // Clear canvas
    ctx.fillStyle = '#1f2937'; // gray-800
    ctx.fillRect(0, 0, width, height);

    // Prepare data
    const data = navHistory.map((h) => ({
      nav: parseFloat(h.nav),
      timestamp: h.timestamp,
    }));

    if (data.length < 2) {
      ctx.fillStyle = '#9ca3af';
      ctx.font = '14px sans-serif';
      ctx.textAlign = 'center';
      ctx.fillText('Not enough data', width / 2, height / 2);
      return;
    }

    // Calculate bounds
    const navValues = data.map((d) => d.nav);
    const minNav = Math.min(...navValues) * 0.999;
    const maxNav = Math.max(...navValues) * 1.001;
    const minTime = data[0].timestamp;
    const maxTime = data[data.length - 1].timestamp;

    // Scale functions
    const scaleX = (timestamp: number) =>
      padding.left +
      ((timestamp - minTime) / (maxTime - minTime)) *
        (width - padding.left - padding.right);

    const scaleY = (nav: number) =>
      height -
      padding.bottom -
      ((nav - minNav) / (maxNav - minNav)) *
        (height - padding.top - padding.bottom);

    // Draw grid lines
    ctx.strokeStyle = '#374151'; // gray-700
    ctx.lineWidth = 1;

    // Horizontal grid lines
    const numHLines = 5;
    for (let i = 0; i <= numHLines; i++) {
      const y = padding.top + (i * (height - padding.top - padding.bottom)) / numHLines;
      ctx.beginPath();
      ctx.moveTo(padding.left, y);
      ctx.lineTo(width - padding.right, y);
      ctx.stroke();

      // Y-axis labels
      const nav = maxNav - (i * (maxNav - minNav)) / numHLines;
      ctx.fillStyle = '#9ca3af';
      ctx.font = '12px sans-serif';
      ctx.textAlign = 'right';
      ctx.fillText(`$${nav.toFixed(4)}`, padding.left - 8, y + 4);
    }

    // Draw line
    ctx.beginPath();
    ctx.strokeStyle = '#3b82f6'; // blue-500
    ctx.lineWidth = 2;

    data.forEach((point, i) => {
      const x = scaleX(point.timestamp);
      const y = scaleY(point.nav);

      if (i === 0) {
        ctx.moveTo(x, y);
      } else {
        ctx.lineTo(x, y);
      }
    });
    ctx.stroke();

    // Draw gradient fill
    const gradient = ctx.createLinearGradient(0, padding.top, 0, height - padding.bottom);
    gradient.addColorStop(0, 'rgba(59, 130, 246, 0.3)');
    gradient.addColorStop(1, 'rgba(59, 130, 246, 0)');

    ctx.beginPath();
    ctx.moveTo(scaleX(data[0].timestamp), height - padding.bottom);
    data.forEach((point) => {
      ctx.lineTo(scaleX(point.timestamp), scaleY(point.nav));
    });
    ctx.lineTo(scaleX(data[data.length - 1].timestamp), height - padding.bottom);
    ctx.closePath();
    ctx.fillStyle = gradient;
    ctx.fill();

    // Draw X-axis labels
    const numLabels = 5;
    ctx.fillStyle = '#9ca3af';
    ctx.font = '12px sans-serif';
    ctx.textAlign = 'center';

    for (let i = 0; i <= numLabels; i++) {
      const timestamp = minTime + (i * (maxTime - minTime)) / numLabels;
      const x = scaleX(timestamp);
      const date = new Date(timestamp * 1000);
      const label =
        timeRange === '24h'
          ? date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' })
          : date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
      ctx.fillText(label, x, height - 10);
    }

  }, [navHistory, timeRange]);

  const handleMouseMove = (e: React.MouseEvent<HTMLCanvasElement>) => {
    if (!canvasRef.current || navHistory.length === 0) return;

    const rect = canvasRef.current.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const padding = { top: 20, right: 20, bottom: 30, left: 60 };
    const width = rect.width;
    const height = rect.height;

    const data = navHistory.map((h) => ({
      nav: h.nav,
      timestamp: h.timestamp,
    }));

    const minTime = data[0].timestamp;
    const maxTime = data[data.length - 1].timestamp;

    // Find closest point
    const timestamp =
      minTime + ((x - padding.left) / (width - padding.left - padding.right)) * (maxTime - minTime);

    let closest = data[0];
    let minDiff = Infinity;

    data.forEach((point) => {
      const diff = Math.abs(point.timestamp - timestamp);
      if (diff < minDiff) {
        minDiff = diff;
        closest = point;
      }
    });

    const navValues = data.map((d) => parseFloat(d.nav));
    const minNav = Math.min(...navValues) * 0.999;
    const maxNav = Math.max(...navValues) * 1.001;

    const scaleX = (ts: number) =>
      padding.left +
      ((ts - minTime) / (maxTime - minTime)) * (width - padding.left - padding.right);

    const scaleY = (nav: number) =>
      height -
      padding.bottom -
      ((nav - minNav) / (maxNav - minNav)) * (height - padding.top - padding.bottom);

    setHoveredPoint({
      nav: closest.nav,
      timestamp: closest.timestamp,
      x: scaleX(closest.timestamp),
      y: scaleY(parseFloat(closest.nav)),
    });
  };

  const handleMouseLeave = () => {
    setHoveredPoint(null);
  };

  const timeRanges: TimeRange[] = ['24h', '7d', '30d', 'all'];

  return (
    <div data-testid="nav-chart" className="bg-gray-800/50 rounded-xl border border-gray-700/50 p-5">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-white">NAV History</h3>
        <div className="flex gap-1">
          {timeRanges.map((range) => (
            <button
              key={range}
              onClick={() => setTimeRange(range)}
              className={`px-3 py-1 text-sm font-medium rounded transition-colors ${
                timeRange === range
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-700 text-gray-400 hover:bg-gray-600 hover:text-white'
              }`}
            >
              {range}
            </button>
          ))}
        </div>
      </div>

      {/* Chart */}
      <div className="relative">
        <canvas
          ref={canvasRef}
          className="w-full h-64 rounded-lg"
          onMouseMove={handleMouseMove}
          onMouseLeave={handleMouseLeave}
        />

        {/* Tooltip */}
        {hoveredPoint && (
          <div
            className="absolute pointer-events-none bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-sm shadow-lg"
            style={{
              left: hoveredPoint.x,
              top: hoveredPoint.y - 60,
              transform: 'translateX(-50%)',
            }}
          >
            <div className="text-white font-semibold">
              ${new BigNumber(hoveredPoint.nav).toFixed(4)}
            </div>
            <div className="text-gray-400 text-xs">
              {new Date(hoveredPoint.timestamp * 1000).toLocaleString()}
            </div>
          </div>
        )}
      </div>

      {/* Current Stats */}
      {navHistory.length > 0 && (
        <div className="grid grid-cols-3 gap-4 mt-4 pt-4 border-t border-gray-700">
          <div>
            <div className="text-sm text-gray-400">Current NAV</div>
            <div className="text-lg font-semibold text-white">
              ${new BigNumber(navHistory[navHistory.length - 1].nav).toFixed(4)}
            </div>
          </div>
          <div>
            <div className="text-sm text-gray-400">Period High</div>
            <div className="text-lg font-semibold text-green-400">
              $
              {new BigNumber(Math.max(...navHistory.map((h) => parseFloat(h.nav)))).toFixed(
                4
              )}
            </div>
          </div>
          <div>
            <div className="text-sm text-gray-400">Period Low</div>
            <div className="text-lg font-semibold text-red-400">
              $
              {new BigNumber(Math.min(...navHistory.map((h) => parseFloat(h.nav)))).toFixed(
                4
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
