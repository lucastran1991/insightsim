'use client';

import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import { Box, Text } from '@chakra-ui/react';
import { TimeseriesResponse } from '@/types/api';
import { format } from 'date-fns';

interface TimeseriesChartProps {
  data: TimeseriesResponse;
}

export default function TimeseriesChart({ data }: TimeseriesChartProps) {
  // Transform data for Recharts
  // Recharts expects array of objects with time and value fields for each series
  const chartData: Array<Record<string, any>> = [];
  const tags = Object.keys(data.result);

  if (tags.length === 0) {
    return (
      <Box p={8} textAlign="center">
        <Text color="gray.500">No data available</Text>
      </Box>
    );
  }

  // Get all unique timestamps
  const allTimestamps = new Set<string>();
  tags.forEach((tag) => {
    data.result[tag].forEach((point) => {
      allTimestamps.add(point.timestamp);
    });
  });

  const sortedTimestamps = Array.from(allTimestamps).sort();

  // Create chart data points
  sortedTimestamps.forEach((timestamp) => {
    const point: Record<string, any> = {
      timestamp,
      time: format(new Date(timestamp), 'HH:mm:ss'),
    };

    tags.forEach((tag) => {
      const tagData = data.result[tag].find((p) => p.timestamp === timestamp);
      point[tag] = tagData ? tagData.value : null;
    });

    chartData.push(point);
  });

  // Generate colors for each tag
  const colors = [
    '#3182CE',
    '#38A169',
    '#D69E2E',
    '#E53E3E',
    '#805AD5',
    '#DD6B20',
    '#319795',
    '#9F7AEA',
  ];

  return (
    <Box width="100%" height="500px" p={4}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={chartData} margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis
            dataKey="time"
            tick={{ fontSize: 12 }}
            interval="preserveStartEnd"
          />
          <YAxis tick={{ fontSize: 12 }} />
          <Tooltip
            formatter={(value: any) => [value?.toFixed(2) || 'N/A', 'Value']}
            labelFormatter={(label) => `Time: ${label}`}
          />
          <Legend />
          {tags.map((tag, index) => (
            <Line
              key={tag}
              type="monotone"
              dataKey={tag}
              stroke={colors[index % colors.length]}
              strokeWidth={2}
              dot={false}
              name={tag}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </Box>
  );
}
