'use client';

import { Select } from '@chakra-ui/react';

export type TimeMode = '1day' | '1week' | '1month';

interface TimeModeSelectorProps {
  value: TimeMode;
  onChange: (mode: TimeMode) => void;
}

export default function TimeModeSelector({
  value,
  onChange,
}: TimeModeSelectorProps) {
  return (
    <Select
      value={value}
      onChange={(e) => onChange(e.target.value as TimeMode)}
      maxW="200px"
    >
      <option value="1day">1 Day</option>
      <option value="1week">1 Week</option>
      <option value="1month">1 Month</option>
    </Select>
  );
}
