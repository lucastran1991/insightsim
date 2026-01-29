'use client';

import { useMemo } from 'react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  TooltipProps,
} from 'recharts';
import { Box, Text, useToken, Alert, AlertIcon } from '@chakra-ui/react';
import { TimeseriesResponse } from '@/types/api';
import { format } from 'date-fns';

// Maximum number of data points to render (for performance)
const MAX_DATA_POINTS = 2000;

interface TimeseriesChartProps {
  data: TimeseriesResponse;
  /** When true, disables chart animations (e.g. for Raw aggregate mode with large datasets) */
  disableAnimation?: boolean;
  /** When 'raw', x-axis shows HH:mm; otherwise shows dd/MM */
  aggregateMode?: 'raw' | string;
}

/**
 * Sample data points to reduce the number of points while preserving visual shape.
 * Uses uniform sampling (every Nth point) when data exceeds maxPoints.
 */
function sampleDataPoints<T>(data: T[], maxPoints: number): T[] {
  if (data.length <= maxPoints) {
    return data;
  }
  const step = Math.ceil(data.length / maxPoints);
  const sampled: T[] = [];
  sampled.push(data[0]);
  for (let i = step; i < data.length - 1; i += step) {
    sampled.push(data[i]);
  }
  if (data.length > 1) {
    sampled.push(data[data.length - 1]);
  }
  return sampled;
}

// Format large numbers for Y-axis: 1000 -> 1K, 1300000 -> 1.3M, etc.
function formatYAxisValue(value: number): string {
  const abs = Math.abs(value);
  if (abs >= 1e9) return (value / 1e9).toFixed(1).replace(/\.0$/, '') + 'B';
  if (abs >= 1e6) return (value / 1e6).toFixed(1).replace(/\.0$/, '') + 'M';
  if (abs >= 1e3) return (value / 1e3).toFixed(1).replace(/\.0$/, '') + 'K';
  if (Number.isInteger(value)) return String(value);
  return value.toFixed(1);
}

// Chakra UI palette (hex) for consistent chart colors
const CHART_COLORS = [
  '#3182CE', // blue.500
  '#38A169', // green.500
  '#DD6B20', // orange.400
  '#E53E3E', // red.500
  '#805AD5', // purple.500
  '#319795', // teal.500
  '#D53F8C', // pink.500
  '#00B5D8', // cyan.500
];

function formatTooltipLabel(label: string): string {
  if (!label) return '';
  const d = new Date(label);
  if (isNaN(d.getTime())) return label;
  return format(d, 'yyyy-MM-dd HH:mm:ss');
}

function CustomTooltip({ active, payload, label }: TooltipProps<number, string>) {
  if (!active || !payload?.length || !label) return null;
  return (
    <Box
      bg="white"
      borderWidth="1px"
      borderColor="gray.200"
      borderRadius="md"
      shadow="md"
      px={3}
      py={2}
      fontSize="sm"
    >
      <Text fontWeight="semibold" color="gray.700" mb={1}>
        {formatTooltipLabel(String(label))}
      </Text>
      {payload.map((entry) => (
        <Text key={entry.dataKey} color={entry.color}>
          {entry.name}: {typeof entry.value === 'number' ? entry.value.toFixed(2) : 'N/A'}
        </Text>
      ))}
    </Box>
  );
}

export default function TimeseriesChart({ data, disableAnimation = false, aggregateMode }: TimeseriesChartProps) {
  const gridColor = useToken('colors', 'gray.200');
  const textColor = useToken('colors', 'gray.600');
  const colors = CHART_COLORS;

  // Memoize data transformation for performance
  const { chartData, isSampled, originalCount } = useMemo(() => {
    const tags = Object.keys(data.result);

    if (tags.length === 0) {
      return { chartData: [], isSampled: false, originalCount: 0 };
    }

    const allTimestamps = new Set<string>();
    tags.forEach((tag) => {
      data.result[tag].forEach((point) => {
        allTimestamps.add(point.timestamp);
      });
    });

    const sortedTimestamps = Array.from(allTimestamps).sort();
    const originalCount = sortedTimestamps.length;

    const sampledTimestamps = sampleDataPoints(sortedTimestamps, MAX_DATA_POINTS);
    const isSampled = sampledTimestamps.length < sortedTimestamps.length;

    const chartDataArray: Array<Record<string, any>> = [];
    sampledTimestamps.forEach((timestamp) => {
      const point: Record<string, any> = {
        timestamp,
      };
      tags.forEach((tag) => {
        const tagData = data.result[tag].find((p) => p.timestamp === timestamp);
        point[tag] = tagData ? tagData.value : null;
      });
      chartDataArray.push(point);
    });

    return { chartData: chartDataArray, isSampled, originalCount };
  }, [data]);

  const tags = Object.keys(data.result);

  if (tags.length === 0) {
    return (
      <Box p={12} textAlign="center">
        <Text color="gray.500">No data available</Text>
      </Box>
    );
  }

  // X-axis format: Raw mode → HH:mm; other aggregate modes → dd/MM
  const formatXTick = (timestamp: string) => {
    const d = new Date(timestamp);
    if (isNaN(d.getTime())) return timestamp;
    if (aggregateMode === 'raw') return format(d, 'HH:mm');
    return format(d, 'dd/MM');
  };

  return (
    <Box
      width="100%"
      height={isSampled ? '600px' : '520px'}
      px={6}
      py={4}
      bg="gray.50"
      display="flex"
      flexDirection="column"
    >
      {isSampled && (
        <Alert status="info" mb={4} borderRadius="md" fontSize="sm" flexShrink={0}>
          <AlertIcon />
          <Box>
            <Text fontWeight="medium">
              Large dataset detected ({originalCount.toLocaleString()} points)
            </Text>
            <Text fontSize="xs" mt={1}>
              Displaying {chartData.length.toLocaleString()} sampled points for better performance.
              Use aggregation (Daily/Monthly) for smoother visualization.
            </Text>
          </Box>
        </Alert>
      )}
      <Box flex={1} minHeight={0}>
        <ResponsiveContainer width="100%" height="100%">
        <LineChart
          data={chartData}
          margin={{ top: 16, right: 24, left: 8, bottom: 16 }}
        >
          <CartesianGrid strokeDasharray="3 3" stroke={gridColor} vertical={false} />
          <XAxis
            dataKey="timestamp"
            tick={{ fontSize: 12, fill: textColor }}
            axisLine={{ stroke: gridColor }}
            tickLine={{ stroke: gridColor }}
            interval="preserveStartEnd"
            tickFormatter={formatXTick}
          />
          <YAxis
            tick={{ fontSize: 12, fill: textColor }}
            axisLine={{ stroke: gridColor }}
            tickLine={{ stroke: gridColor }}
            tickFormatter={(v) => formatYAxisValue(Number(v))}
          />
          <Tooltip content={({ active, payload, label }) => (
            <CustomTooltip
              active={active}
              payload={payload as TooltipProps<number, string>['payload']}
              label={label}
            />
          )} />
          <Legend
            wrapperStyle={{ paddingTop: 8 }}
            iconType="line"
            iconSize={10}
            formatter={(value) => <Text as="span" fontSize="sm" color="gray.700">{value}</Text>}
          />
          {tags.map((tag, index) => (
            <Line
              key={tag}
              type="monotone"
              dataKey={tag}
              stroke={colors[index % colors.length]}
              strokeWidth={2.5}
              dot={false}
              activeDot={{ r: 4, strokeWidth: 2 }}
              name={tag}
              connectNulls
              isAnimationActive={!disableAnimation}
            />
          ))}
        </LineChart>
        </ResponsiveContainer>
      </Box>
    </Box>
  );
}
