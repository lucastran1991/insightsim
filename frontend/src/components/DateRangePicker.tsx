'use client';

import { FormControl, FormLabel, Input, HStack } from '@chakra-ui/react';
import { format } from 'date-fns';

interface DateRangePickerProps {
  fromDate: string;
  toDate: string;
  onFromDateChange: (date: string) => void;
  onToDateChange: (date: string) => void;
}

export default function DateRangePicker({
  fromDate,
  toDate,
  onFromDateChange,
  onToDateChange,
}: DateRangePickerProps) {
  // Convert ISO string to datetime-local format (YYYY-MM-DDTHH:mm)
  const formatForInput = (isoString: string): string => {
    try {
      const date = new Date(isoString);
      return format(date, "yyyy-MM-dd'T'HH:mm");
    } catch {
      return '';
    }
  };

  // Convert datetime-local format to ISO string
  const parseFromInput = (inputValue: string): string => {
    if (!inputValue) return '';
    try {
      const date = new Date(inputValue);
      return format(date, "yyyy-MM-dd'T'HH:mm:ss");
    } catch {
      return inputValue;
    }
  };

  return (
    <HStack spacing={4} align="end">
      <FormControl>
        <FormLabel>From</FormLabel>
        <Input
          type="datetime-local"
          value={formatForInput(fromDate)}
          onChange={(e) => onFromDateChange(parseFromInput(e.target.value))}
        />
      </FormControl>
      <FormControl>
        <FormLabel>To</FormLabel>
        <Input
          type="datetime-local"
          value={formatForInput(toDate)}
          onChange={(e) => onToDateChange(parseFromInput(e.target.value))}
        />
      </FormControl>
    </HStack>
  );
}
