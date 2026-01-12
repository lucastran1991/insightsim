'use client';

import { useState, useEffect } from 'react';
import {
  Container,
  Box,
  Heading,
  VStack,
  HStack,
  Spinner,
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
  useToast,
} from '@chakra-ui/react';
import { format, subDays, startOfToday, endOfToday } from 'date-fns';
import TimeModeSelector, { TimeMode } from '@/components/TimeModeSelector';
import TagSelector from '@/components/TagSelector';
import TimeseriesChart from '@/components/TimeseriesChart';
import { getTimeseriesData, getTagList } from '@/lib/api';
import { TimeseriesResponse } from '@/types/api';

export default function ChartPage() {
  const [timeMode, setTimeMode] = useState<TimeMode>('1day');
  const [tags, setTags] = useState<string[]>([]);
  const [selectedTags, setSelectedTags] = useState<string[]>([]);
  const [data, setData] = useState<TimeseriesResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const toast = useToast();

  // Load available tags on mount
  useEffect(() => {
    async function loadTags() {
      try {
        const tagList = await getTagList();
        setTags(tagList);
        // Auto-select first tag if available
        if (tagList.length > 0 && selectedTags.length === 0) {
          setSelectedTags([tagList[0]]);
        }
      } catch (err) {
        console.error('Failed to load tags:', err);
        toast({
          title: 'Error',
          description: 'Failed to load tag list',
          status: 'error',
          duration: 5000,
          isClosable: true,
        });
      }
    }
    loadTags();
  }, []);

  // Calculate time range based on mode
  const getTimeRange = (mode: TimeMode): { start: string; end: string } => {
    const now = new Date();
    let start: Date;
    let end: Date = endOfToday();

    switch (mode) {
      case '1day':
        start = startOfToday();
        break;
      case '1week':
        start = subDays(now, 7);
        break;
      case '1month':
        start = subDays(now, 30);
        break;
      default:
        start = startOfToday();
    }

    return {
      start: format(start, "yyyy-MM-dd'T'HH:mm:ss"),
      end: format(end, "yyyy-MM-dd'T'HH:mm:ss"),
    };
  };

  // Fetch data when time mode or selected tags change
  useEffect(() => {
    if (selectedTags.length === 0) {
      setData(null);
      return;
    }

    async function fetchData() {
      setLoading(true);
      setError(null);

      try {
        const { start, end } = getTimeRange(timeMode);
        const result = await getTimeseriesData(start, end, selectedTags);
        setData(result);
      } catch (err) {
        const errorMessage =
          err instanceof Error ? err.message : 'Failed to fetch data';
        setError(errorMessage);
        toast({
          title: 'Error',
          description: errorMessage,
          status: 'error',
          duration: 5000,
          isClosable: true,
        });
      } finally {
        setLoading(false);
      }
    }

    fetchData();
  }, [timeMode, selectedTags, toast]);

  return (
    <Container maxW="container.xl" py={8}>
      <VStack spacing={6} align="stretch">
        <Heading size="lg">Timeseries Data Chart</Heading>

        <HStack spacing={4} align="start">
          <Box>
            <TimeModeSelector value={timeMode} onChange={setTimeMode} />
          </Box>
          <Box flex={1}>
            <TagSelector
              tags={tags}
              selectedTags={selectedTags}
              onChange={setSelectedTags}
              isMulti={true}
            />
          </Box>
        </HStack>

        {loading && (
          <Box textAlign="center" py={8}>
            <Spinner size="xl" />
          </Box>
        )}

        {error && (
          <Alert status="error">
            <AlertIcon />
            <Box>
              <AlertTitle>Error loading data</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Box>
          </Alert>
        )}

        {!loading && !error && data && (
          <Box>
            <TimeseriesChart data={data} />
          </Box>
        )}

        {!loading && !error && !data && selectedTags.length > 0 && (
          <Alert status="info">
            <AlertIcon />
            <AlertDescription>No data available for selected tags</AlertDescription>
          </Alert>
        )}

        {!loading && !error && selectedTags.length === 0 && (
          <Alert status="warning">
            <AlertIcon />
            <AlertDescription>Please select at least one tag to view data</AlertDescription>
          </Alert>
        )}
      </VStack>
    </Container>
  );
}
