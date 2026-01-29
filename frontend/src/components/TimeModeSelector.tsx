'use client';

import { Select } from '@chakra-ui/react';

export type TimeMode =
  | 'today'
  | 'weekToDate'
  | 'monthToDate'
  | 'yearToDate'
  | 'previous1Year'
  | 'previous2Year';

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
      width="100%"
    >
      <option value="today">Today</option>
      <option value="weekToDate">Week to date</option>
      <option value="monthToDate">Month to date</option>
      <option value="yearToDate">Year to date</option>
      <option value="previous1Year">Previous 1 year</option>
      <option value="previous2Year">Previous 2 years</option>
    </Select>
  );
}
