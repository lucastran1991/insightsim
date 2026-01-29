'use client';

import { useState, useEffect, type ChangeEvent } from 'react';
import {
  Container,
  Box,
  Heading,
  Text,
  VStack,
  FormControl,
  FormLabel,
  RadioGroup,
  Radio,
  Stack,
  Button,
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
  useToast,
  Card,
  CardBody,
  CardHeader,
  Divider,
} from '@chakra-ui/react';
import {
  format,
  startOfToday,
  startOfWeek,
  startOfMonth,
  startOfYear,
  subYears,
} from 'date-fns';
import TagSelector from '@/components/TagSelector';
import DateRangePicker from '@/components/DateRangePicker';
import { generateDummyData, getTagList } from '@/lib/api';

type GenerationMode = 'all' | 'single';

type TimeRangePreset =
  | 'today'
  | 'weekToDate'
  | 'monthToDate'
  | 'yearToDate'
  | 'previous1Year'
  | 'previous2Year'
  | 'custom';

function getTimeRangeForPreset(preset: Exclude<TimeRangePreset, 'custom'>): {
  start: string;
  end: string;
} {
  const now = new Date();
  let start: Date;
  switch (preset) {
    case 'today':
      start = startOfToday();
      break;
    case 'weekToDate':
      start = startOfWeek(now, { weekStartsOn: 1 });
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
}

const defaultPreset: TimeRangePreset = 'today';
const defaultRange = getTimeRangeForPreset(defaultPreset);

export default function SetupPage() {
  const [mode, setMode] = useState<GenerationMode>('all');
  const [tags, setTags] = useState<string[]>([]);
  const [selectedTag, setSelectedTag] = useState<string[]>([]);
  const [timeRangePreset, setTimeRangePreset] = useState<TimeRangePreset>(defaultPreset);
  const [fromDate, setFromDate] = useState<string>(defaultRange.start);
  const [toDate, setToDate] = useState<string>(defaultRange.end);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const toast = useToast();

  // Load available tags on mount
  useEffect(() => {
    async function loadTags() {
      try {
        const tagList = await getTagList();
        setTags(tagList);
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
  }, [toast]);

  const handleTimeRangePresetChange = (value: string) => {
    const preset = value as TimeRangePreset;
    setTimeRangePreset(preset);
    if (preset !== 'custom') {
      const { start, end } = getTimeRangeForPreset(preset);
      setFromDate(start);
      setToDate(end);
    }
  };

  const handleGenerate = async () => {
    if (mode === 'single' && selectedTag.length === 0) {
      toast({
        title: 'Validation Error',
        description: 'Please select a tag when using single tag mode',
        status: 'warning',
        duration: 5000,
        isClosable: true,
      });
      return;
    }

    setLoading(true);
    setError(null);
    setSuccess(null);

    try {
      const tag = mode === 'single' ? selectedTag[0] : undefined;
      // Always send start/end that match the dropdown so backend never falls back to config
      const effectiveStart =
        timeRangePreset === 'custom'
          ? (fromDate?.trim() || defaultRange.start)
          : getTimeRangeForPreset(timeRangePreset).start;
      const effectiveEnd =
        timeRangePreset === 'custom'
          ? (toDate?.trim() || defaultRange.end)
          : getTimeRangeForPreset(timeRangePreset).end;
      const result = await generateDummyData(tag, effectiveStart, effectiveEnd, (tagName, records) => {
        toast({
          title: 'Tag complete',
          description: `${tagName}: ${records} records generated`,
          status: 'info',
          duration: 4000,
          isClosable: true,
        });
      });

      if (result.success) {
        const message = `Successfully generated ${result.count || 0} records for ${result.tags_count || 0} tag(s)`;
        setSuccess(message);
        toast({
          title: 'Success',
          description: message,
          status: 'success',
          duration: 5000,
          isClosable: true,
        });
      } else {
        throw new Error(result.message || 'Generation failed');
      }
    } catch (err) {
      const errorMessage =
        err instanceof Error ? err.message : 'Failed to generate data';
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
  };

  return (
    <Container maxW="container.xl" py={8} px={{ base: 4, md: 6 }}>
      <VStack spacing={8} align="stretch">
        <Box>
          <Heading size="lg" fontWeight="semibold" color="gray.800">
            Setup
          </Heading>
          <Text mt={1} fontSize="sm" color="gray.600">
            Generate dummy timeseries data. Time range is from the dropdown (e.g. Today = from 00:00 to now).
          </Text>
        </Box>

        <Card shadow="sm" borderRadius="lg" borderWidth="1px" borderColor="gray.100">
          <CardHeader pb={2}>
            <Heading size="sm" fontWeight="medium" color="gray.600">
              Generation
            </Heading>
          </CardHeader>
          <CardBody pt={0}>
            <VStack spacing={5} align="stretch">
              <FormControl>
                <FormLabel fontSize="sm" color="gray.600">
                  Mode
                </FormLabel>
                <RadioGroup value={mode} onChange={(value) => setMode(value as GenerationMode)}>
                  <Stack direction="row" spacing={4}>
                    <Radio value="all">All tags</Radio>
                    <Radio value="single">Single tag</Radio>
                  </Stack>
                </RadioGroup>
              </FormControl>

              {mode === 'single' && (
                <FormControl minW={0}>
                  <TagSelector
                    tags={tags}
                    selectedTags={selectedTag}
                    onChange={setSelectedTag}
                    isMulti={false}
                    placeholder="Select a tag..."
                  />
                </FormControl>
              )}

              <Divider />

              <FormControl>
                <FormLabel fontSize="sm" color="gray.600">
                  Time range
                </FormLabel>
                <Box
                  as="select"
                  value={timeRangePreset}
                  onChange={(e: ChangeEvent<HTMLSelectElement>) => handleTimeRangePresetChange(e.target.value)}
                  maxW="280px"
                  mb={timeRangePreset === 'custom' ? 3 : 0}
                  px={3}
                  py={2}
                  borderWidth="1px"
                  borderColor="gray.200"
                  borderRadius="md"
                  bg="white"
                  fontSize="md"
                  _hover={{ borderColor: 'gray.300' }}
                  _focus={{ outline: 'none', borderColor: 'blue.500', boxShadow: '0 0 0 1px var(--chakra-colors-blue-500)' }}
                  cursor="pointer"
                >
                  <option value="today">Today</option>
                  <option value="weekToDate">Week to date</option>
                  <option value="monthToDate">Month to date</option>
                  <option value="yearToDate">Year to date</option>
                  <option value="previous1Year">Previous 1 year</option>
                  <option value="previous2Year">Previous 2 years</option>
                  <option value="custom">Custom</option>
                </Box>
                {timeRangePreset === 'custom' && (
                  <DateRangePicker
                    fromDate={fromDate}
                    toDate={toDate}
                    onFromDateChange={setFromDate}
                    onToDateChange={setToDate}
                  />
                )}
                {timeRangePreset !== 'custom' && (
                  <Text mt={2} fontSize="xs" color="gray.500">
                    {format(new Date(fromDate), 'yyyy-MM-dd HH:mm')} →{' '}
                    {format(new Date(toDate), 'yyyy-MM-dd HH:mm')}
                  </Text>
                )}
                <Text mt={2} fontSize="xs" color="gray.500">
                  Backend config controls actual generation range.
                </Text>
              </FormControl>

              <Button
                colorScheme="blue"
                size="md"
                onClick={handleGenerate}
                isLoading={loading}
                loadingText="Generating..."
                isDisabled={mode === 'single' && selectedTag.length === 0}
              >
                Generate Data
              </Button>
            </VStack>
          </CardBody>
        </Card>

        {error && (
          <Alert status="error" borderRadius="lg" variant="left-accent">
            <AlertIcon />
            <Box>
              <AlertTitle>Generation failed</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Box>
          </Alert>
        )}

        {success && (
          <Alert status="success" borderRadius="lg" variant="left-accent">
            <AlertIcon />
            <AlertDescription>{success}</AlertDescription>
          </Alert>
        )}

        <Alert status="warning" borderRadius="lg" variant="left-accent">
          <AlertIcon />
          <AlertDescription>
            {mode === 'all'
              ? 'All tags: deletes all data, then generates.'
              : 'Single tag: deletes that tag\'s data, then generates.'}
          </AlertDescription>
        </Alert>
      </VStack>
    </Container>
  );
}
