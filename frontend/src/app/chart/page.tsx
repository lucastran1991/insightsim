'use client';

import { useState, useEffect, type ChangeEvent } from 'react';
import {
  Container,
  Box,
  Heading,
  Text,
  VStack,
  Flex,
  Spinner,
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
  useToast,
  Card,
  CardBody,
  CardHeader,
  SimpleGrid,
  FormControl,
  FormLabel,
  Select,
  Button,
  Input,
  Tag,
  TagLabel,
  TagCloseButton,
  Wrap,
} from '@chakra-ui/react';
import {
  format,
  startOfToday,
  startOfWeek,
  startOfMonth,
  startOfYear,
  subYears,
} from 'date-fns';
import TimeModeSelector, { TimeMode } from '@/components/TimeModeSelector';
import TagSelector from '@/components/TagSelector';
import TimeseriesChart from '@/components/TimeseriesChart';
import { getTimeseriesData, getTagList, getValueRangeConfig, type AggregateMode } from '@/lib/api';
import { TimeseriesResponse } from '@/types/api';

function escapeCsvField(field: string): string {
  if (/[",\n\r]/.test(field)) {
    return `"${field.replace(/"/g, '""')}"`;
  }
  return field;
}

function dataToCsv(data: TimeseriesResponse): string {
  const tags = Object.keys(data.result);
  if (tags.length === 0) return 'timestamp\n';

  const allTimestamps = new Set<string>();
  tags.forEach((tag) => {
    data.result[tag].forEach((point) => allTimestamps.add(point.timestamp));
  });
  const sortedTimestamps = Array.from(allTimestamps).sort();

  const header = ['timestamp', ...tags].map(escapeCsvField).join(',');
  const rows = sortedTimestamps.map((timestamp) => {
    const values = [
      escapeCsvField(timestamp),
      ...tags.map((tag) => {
        const point = data.result[tag].find((p) => p.timestamp === timestamp);
        return point != null ? String(point.value) : '';
      }),
    ];
    return values.join(',');
  });
  return [header, ...rows].join('\n');
}

function downloadCsv(data: TimeseriesResponse) {
  const csv = dataToCsv(data);
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = `timeseries-${format(new Date(), 'yyyy-MM-dd-HHmm')}.csv`;
  link.click();
  URL.revokeObjectURL(url);
}

export type FrequencyOption = '1min' | '5min' | '15min' | '30min' | '1hour';

const FREQUENCY_MINUTES: Record<FrequencyOption, number> = {
  '1min': 1,
  '5min': 5,
  '15min': 15,
  '30min': 30,
  '1hour': 60,
};

function parseTimestamp(ts: string): number {
  return /^\d+$/.test(ts) ? Number(ts) : new Date(ts).getTime();
}

function filterByFrequency(data: TimeseriesResponse, frequencyMinutes: number): TimeseriesResponse {
  if (frequencyMinutes <= 1) return data;
  const intervalMs = frequencyMinutes * 60 * 1000;
  const result: TimeseriesResponse['result'] = {};
  for (const tag of Object.keys(data.result)) {
    const points = data.result[tag];
    if (points.length === 0) {
      result[tag] = [];
      continue;
    }
    const sorted = [...points].sort((a, b) => parseTimestamp(a.timestamp) - parseTimestamp(b.timestamp));
    const baseTs = parseTimestamp(sorted[0].timestamp);
    result[tag] = sorted.filter((p) => {
      const t = parseTimestamp(p.timestamp);
      const offset = t - baseTs;
      return offset % intervalMs === 0;
    });
  }
  return { result };
}

function filterByValueRange(data: TimeseriesResponse, minVal: number, maxVal: number): TimeseriesResponse {
  const result: TimeseriesResponse['result'] = {};
  for (const tag of Object.keys(data.result)) {
    result[tag] = data.result[tag].filter(
      (p) => p.value >= minVal && p.value <= maxVal
    );
  }
  return { result };
}

export default function ChartPage() {
  const [timeMode, setTimeMode] = useState<TimeMode>('monthToDate');
  const [aggregateMode, setAggregateMode] = useState<AggregateMode>('daily');
  const [frequency, setFrequency] = useState<FrequencyOption>('1min');
  const [minValue, setMinValue] = useState<number>(0);
  const [maxValue, setMaxValue] = useState<number>(10000);
  const [tags, setTags] = useState<string[]>([]);
  const [selectedTags, setSelectedTags] = useState<string[]>([]);
  const [data, setData] = useState<TimeseriesResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const toast = useToast();

  // Load value range defaults from backend config
  useEffect(() => {
    getValueRangeConfig().then(({ min, max }) => {
      setMinValue(min);
      setMaxValue(max);
    });
  }, []);

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

  // Calculate time range based on datetime filter (start of period to now)
  const getTimeRange = (mode: TimeMode): { start: string; end: string } => {
    const now = new Date();
    let start: Date;

    switch (mode) {
      case 'today':
        start = startOfToday();
        break;
      case 'weekToDate':
        start = startOfWeek(now, { weekStartsOn: 1 }); // Monday
        break;
      case 'monthToDate':
        start = startOfMonth(now);
        break;
      case 'yearToDate':
        start = startOfYear(now);
        break;
      case 'previous1Year':
        start = subYears(now, 1);
        break;
      case 'previous2Year':
        start = subYears(now, 2);
        break;
      default:
        start = startOfToday();
    }

    return {
      start: format(start, "yyyy-MM-dd'T'HH:mm:ss"),
      end: format(now, "yyyy-MM-dd'T'HH:mm:ss"),
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
        const result = await getTimeseriesData(start, end, selectedTags, aggregateMode);
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
  }, [timeMode, selectedTags, aggregateMode, toast]);

  return (
    <Container maxW="container.xl" py={8} px={{ base: 4, md: 6 }}>
      <VStack spacing={8} align="stretch">
        <Box>
          <Heading size="lg" fontWeight="semibold" color="gray.800">
            Chart
          </Heading>
          <Text mt={1} fontSize="sm" color="gray.600">
            View and download timeseries data by tags and time range.
          </Text>
        </Box>

        <Card shadow="sm" borderRadius="lg" borderWidth="1px" borderColor="gray.100">
          <CardHeader pb={2}>
            <Heading size="sm" fontWeight="medium" color="gray.600">
              Filters
            </Heading>
          </CardHeader>
          <CardBody pt={0}>
            <SimpleGrid columns={{ base: 1, md: 4 }} spacing={6} alignItems="flex-end">
              <FormControl minW={0}>
                <TagSelector
                  tags={tags}
                  selectedTags={selectedTags}
                  onChange={setSelectedTags}
                  isMulti={true}
                  label="Specific tags"
                />
                <Box mt={2}>
                  <Text fontSize="xs" color="gray.500" mb={1}>
                    Selected:
                  </Text>
                  {selectedTags.length === 0 ? (
                    <Text fontSize="sm" color="gray.400">
                      No tags selected
                    </Text>
                  ) : (
                    <Wrap spacing={2}>
                      {selectedTags.map((tag) => (
                        <Tag key={tag} size="sm" colorScheme="blue">
                          <TagLabel>{tag}</TagLabel>
                          <TagCloseButton
                            onClick={() => setSelectedTags(selectedTags.filter((t) => t !== tag))}
                          />
                        </Tag>
                      ))}
                    </Wrap>
                  )}
                </Box>
              </FormControl>
              <FormControl minW={0}>
                <FormLabel fontSize="sm" color="gray.600">
                  Frequency
                </FormLabel>
                <Box
                  as="select"
                  value={frequency}
                  onChange={(e: ChangeEvent<HTMLSelectElement>) =>
                    setFrequency(e.target.value as FrequencyOption)
                  }
                  width="100%"
                  px={3}
                  py={2}
                  borderWidth="1px"
                  borderColor="gray.200"
                  borderRadius="md"
                  bg="white"
                  fontSize="md"
                  _hover={{ borderColor: 'gray.300' }}
                  _focus={{
                    outline: 'none',
                    borderColor: 'blue.500',
                    boxShadow: '0 0 0 1px var(--chakra-colors-blue-500)',
                  }}
                  cursor="pointer"
                >
                  <option value="1min">1 record / 1 min</option>
                  <option value="5min">1 record / 5 mins</option>
                  <option value="15min">1 record / 15 mins</option>
                  <option value="30min">1 record / 30 mins</option>
                  <option value="1hour">1 record / 1 hour</option>
                </Box>
              </FormControl>
              <FormControl minW={0}>
                <FormLabel fontSize="sm" color="gray.600">
                  Aggregate
                </FormLabel>
                <Select
                  value={aggregateMode}
                  onChange={(e) => setAggregateMode(e.target.value as AggregateMode)}
                  width="100%"
                >
                  <option value="raw">Raw</option>
                  <option value="daily">Daily</option>
                  <option value="monthly">Monthly</option>
                  <option value="quarterly">Quarterly</option>
                  <option value="yearly">Yearly</option>
                </Select>
              </FormControl>
              <FormControl minW={0}>
                <FormLabel fontSize="sm" color="gray.600">
                  Time range
                </FormLabel>
                <TimeModeSelector value={timeMode} onChange={setTimeMode} />
              </FormControl>
            </SimpleGrid>
            <SimpleGrid columns={{ base: 1, md: 2 }} spacing={6} alignItems="flex-end" mt={4}>
              <FormControl minW={0} maxW="200px">
                <FormLabel fontSize="sm" color="gray.600">
                  Min value
                </FormLabel>
                <Input
                  type="number"
                  value={minValue}
                  onChange={(e) => {
                    const v = parseFloat(e.target.value);
                    if (!Number.isNaN(v)) setMinValue(v);
                  }}
                  step="any"
                />
              </FormControl>
              <FormControl minW={0} maxW="200px">
                <FormLabel fontSize="sm" color="gray.600">
                  Max value
                </FormLabel>
                <Input
                  type="number"
                  value={maxValue}
                  onChange={(e) => {
                    const v = parseFloat(e.target.value);
                    if (!Number.isNaN(v)) setMaxValue(v);
                  }}
                  step="any"
                />
              </FormControl>
            </SimpleGrid>
          </CardBody>
        </Card>

        {loading && (
          <Card shadow="sm" borderRadius="lg" overflow="hidden">
            <CardBody py={16}>
              <Flex justify="center" align="center" direction="column" gap={4}>
                <Spinner size="xl" color="blue.500" thickness="3px" />
                <Box as="span" fontSize="sm" color="gray.500">
                  Loading chart data...
                </Box>
              </Flex>
            </CardBody>
          </Card>
        )}

        {error && (
          <Alert status="error" borderRadius="lg" variant="left-accent">
            <AlertIcon />
            <Box flex={1}>
              <AlertTitle>Error loading data</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Box>
          </Alert>
        )}

        {!loading && !error && data && (() => {
          let chartData =
            aggregateMode === 'raw'
              ? filterByFrequency(data, FREQUENCY_MINUTES[frequency])
              : data;
          chartData = filterByValueRange(chartData, minValue, maxValue);
          return (
            <Card shadow="sm" borderRadius="lg" borderWidth="1px" borderColor="gray.100" overflow="hidden">
              <CardHeader py={4} borderBottomWidth="1px" borderColor="gray.100">
                <Flex justify="space-between" align="center" wrap="wrap" gap={2}>
                  <Heading size="sm" fontWeight="medium" color="gray.700">
                    Chart
                  </Heading>
                  {aggregateMode === 'raw' && (
                    <Button
                      size="sm"
                      colorScheme="blue"
                      variant="outline"
                      onClick={() => downloadCsv(chartData)}
                    >
                      Download CSV
                    </Button>
                  )}
                </Flex>
              </CardHeader>
              <CardBody p={0}>
                <TimeseriesChart
                  data={chartData}
                  disableAnimation={aggregateMode === 'raw'}
                  aggregateMode={aggregateMode}
                />
              </CardBody>
            </Card>
          );
        })()}

        {!loading && !error && !data && selectedTags.length > 0 && (
          <Alert status="info" borderRadius="lg" variant="left-accent">
            <AlertIcon />
            <AlertDescription>No data available for selected tags</AlertDescription>
          </Alert>
        )}

        {!loading && !error && selectedTags.length === 0 && (
          <Alert status="warning" borderRadius="lg" variant="left-accent">
            <AlertIcon />
            <AlertDescription>Please select at least one tag to view data</AlertDescription>
          </Alert>
        )}
      </VStack>
    </Container>
  );
}
