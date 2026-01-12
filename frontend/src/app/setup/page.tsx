'use client';

import { useState, useEffect } from 'react';
import {
  Container,
  Box,
  Heading,
  VStack,
  FormControl,
  FormLabel,
  RadioGroup,
  Radio,
  Stack,
  Button,
  Spinner,
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
  useToast,
  Divider,
} from '@chakra-ui/react';
import { format } from 'date-fns';
import TagSelector from '@/components/TagSelector';
import DateRangePicker from '@/components/DateRangePicker';
import { generateDummyData, getTagList } from '@/lib/api';

type GenerationMode = 'all' | 'single';

export default function SetupPage() {
  const [mode, setMode] = useState<GenerationMode>('all');
  const [tags, setTags] = useState<string[]>([]);
  const [selectedTag, setSelectedTag] = useState<string[]>([]);
  const [fromDate, setFromDate] = useState<string>(
    format(new Date(), "yyyy-MM-dd'T'00:00:00")
  );
  const [toDate, setToDate] = useState<string>(
    format(new Date(), "yyyy-MM-dd'T'23:59:59")
  );
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
      const result = await generateDummyData(tag);

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
    <Container maxW="container.md" py={8}>
      <VStack spacing={6} align="stretch">
        <Heading size="lg">Generate Dummy Data</Heading>

        <FormControl>
          <FormLabel>Generation Mode</FormLabel>
          <RadioGroup value={mode} onChange={(value) => setMode(value as GenerationMode)}>
            <Stack direction="row" spacing={4}>
              <Radio value="all">All Tags</Radio>
              <Radio value="single">Single Tag</Radio>
            </Stack>
          </RadioGroup>
        </FormControl>

        {mode === 'single' && (
          <TagSelector
            tags={tags}
            selectedTags={selectedTag}
            onChange={setSelectedTag}
            isMulti={false}
            placeholder="Select a tag..."
          />
        )}

        <Divider />

        <Box>
          <Heading size="sm" mb={4}>
            Time Range
          </Heading>
          <DateRangePicker
            fromDate={fromDate}
            toDate={toDate}
            onFromDateChange={setFromDate}
            onToDateChange={setToDate}
          />
          <Box mt={4} fontSize="sm" color="gray.600">
            <p>
              <strong>Note:</strong> The time range is configured in the backend config.json file.
              The dates above are for reference only. The actual generation will use the time range
              from the backend configuration.
            </p>
          </Box>
        </Box>

        <Divider />

        <Button
          colorScheme="blue"
          size="lg"
          onClick={handleGenerate}
          isLoading={loading}
          loadingText="Generating..."
          isDisabled={mode === 'single' && selectedTag.length === 0}
        >
          Generate Data
        </Button>

        {loading && (
          <Box textAlign="center" py={4}>
            <Spinner size="lg" />
            <Box mt={4} fontSize="sm" color="gray.600">
              This may take a few minutes depending on the number of tags and time range...
            </Box>
          </Box>
        )}

        {error && (
          <Alert status="error">
            <AlertIcon />
            <Box>
              <AlertTitle>Generation Failed</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Box>
          </Alert>
        )}

        {success && (
          <Alert status="success">
            <AlertIcon />
            <AlertDescription>{success}</AlertDescription>
          </Alert>
        )}

        <Alert status="warning">
          <AlertIcon />
          <Box>
            <AlertTitle>Warning</AlertTitle>
            <AlertDescription>
              {mode === 'all'
                ? 'Generating for all tags will delete ALL existing data in the database before generating new data.'
                : 'Generating for a single tag will delete only that tag\'s data before generating new data.'}
            </AlertDescription>
          </Box>
        </Alert>
      </VStack>
    </Container>
  );
}
