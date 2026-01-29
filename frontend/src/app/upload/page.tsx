'use client';

import { useState, useRef } from 'react';
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
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
  useToast,
  Text,
  Flex,
  Card,
  CardBody,
  CardHeader,
  Icon,
  IconButton,
} from '@chakra-ui/react';
import { FiUploadCloud, FiFileText, FiX } from 'react-icons/fi';
import { uploadCsv } from '@/lib/api';

type UploadMode = 'override' | 'replace';

const ACCEPT_CSV = '.csv,text/csv';

function isCsvFile(file: File): boolean {
  const name = file.name.toLowerCase();
  const type = file.type.toLowerCase();
  return name.endsWith('.csv') || type === 'text/csv' || type === 'application/csv';
}

export default function UploadPage() {
  const [mode, setMode] = useState<UploadMode>('override');
  const [file, setFile] = useState<File | null>(null);
  const [dragActive, setDragActive] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const toast = useToast();

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(true);
  };

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);
    const dropped = e.dataTransfer.files?.[0];
    if (!dropped) return;
    if (!isCsvFile(dropped)) {
      toast({
        title: 'Invalid file',
        description: 'Please drop a CSV file.',
        status: 'warning',
        duration: 5000,
        isClosable: true,
      });
      return;
    }
    setFile(dropped);
    setError(null);
    setSuccess(null);
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const selected = e.target.files?.[0];
    if (!selected) return;
    if (!isCsvFile(selected)) {
      toast({
        title: 'Invalid file',
        description: 'Please choose a CSV file.',
        status: 'warning',
        duration: 5000,
        isClosable: true,
      });
      return;
    }
    setFile(selected);
    setError(null);
    setSuccess(null);
    e.target.value = '';
  };

  const handleClearFile = () => {
    setFile(null);
    setError(null);
    setSuccess(null);
    if (fileInputRef.current) fileInputRef.current.value = '';
  };

  const handleUpload = async () => {
    if (!file) {
      toast({
        title: 'No file selected',
        description: 'Please select or drop a CSV file.',
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
      const result = await uploadCsv(file, mode);
      if (result.success) {
        const msg = `Imported ${result.count ?? 0} rows for ${result.tags_affected ?? 0} tag(s).`;
        setSuccess(msg);
        toast({
          title: 'Upload successful',
          description: msg,
          status: 'success',
          duration: 5000,
          isClosable: true,
        });
      } else {
        throw new Error(result.message || 'Upload failed');
      }
    } catch (err) {
      const errorMessage =
        err instanceof Error ? err.message : 'Failed to upload CSV';
      setError(errorMessage);
      toast({
        title: 'Upload failed',
        description: errorMessage,
        status: 'error',
        duration: 5000,
        isClosable: true,
      });
    } finally {
      setLoading(false);
    }
  };

  const fileSizeKb = file ? (file.size / 1024).toFixed(1) : '';

  return (
    <Container maxW="container.xl" py={8} px={{ base: 4, md: 6 }}>
      <VStack spacing={8} align="stretch">
        <Card shadow="md" borderRadius="xl" borderWidth="1px" borderColor="gray.200" overflow="hidden">
          <CardHeader bg="gray.50" borderBottomWidth="1px" borderColor="gray.100" py={5}>
            <Flex align="center" gap={3}>
              <Flex
                align="center"
                justify="center"
                w={10}
                h={10}
                borderRadius="lg"
                bg="blue.50"
                color="blue.500"
              >
                <Icon as={FiUploadCloud} boxSize={5} />
              </Flex>
              <Box>
                <Heading size="md" fontWeight="semibold" color="gray.800">
                  Upload files
                </Heading>
                <Text mt={0.5} fontSize="sm" color="gray.600">
                  Select and upload a CSV to import tag data.
                </Text>
              </Box>
            </Flex>
          </CardHeader>
          <CardBody py={6}>
            <VStack spacing={6} align="stretch">
              <FormControl>
                <FormLabel fontSize="sm" color="gray.600" mb={3}>
                  Import mode
                </FormLabel>
                <RadioGroup
                  value={mode}
                  onChange={(value) => setMode(value as UploadMode)}
                >
                  <Stack spacing={2}>
                    <Radio value="override">
                      Override within CSV range — upsert only rows in the file
                    </Radio>
                    <Radio value="replace">
                      Replace all for tags — delete tag data, then insert CSV rows
                    </Radio>
                  </Stack>
                </RadioGroup>
              </FormControl>

              <FormControl>
                <FormLabel fontSize="sm" color="gray.600" mb={3}>
                  File
                </FormLabel>
                <Box
                  borderWidth="2px"
                  borderStyle="dashed"
                  borderColor={dragActive ? 'blue.400' : 'gray.300'}
                  borderRadius="xl"
                  p={8}
                  textAlign="center"
                  bg={dragActive ? 'blue.50' : 'gray.50'}
                  cursor="pointer"
                  onClick={() => fileInputRef.current?.click()}
                  onDragOver={handleDragOver}
                  onDragLeave={handleDragLeave}
                  onDrop={handleDrop}
                  _hover={{ borderColor: 'gray.400', bg: dragActive ? 'blue.50' : 'gray.100' }}
                  transition="all 0.15s"
                >
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept={ACCEPT_CSV}
                    onChange={handleFileSelect}
                    style={{ display: 'none' }}
                    aria-label="Choose CSV file"
                  />
                  <Flex justify="center" mb={3}>
                    <Icon as={FiUploadCloud} boxSize={10} color="gray.400" />
                  </Flex>
                  <Text color="gray.700" fontWeight="medium">
                    Choose a file or drag & drop it here
                  </Text>
                  <Text fontSize="sm" color="gray.500" mt={2}>
                    CSV format, up to 50MB. Header: timestamp, tag1, tag2, ...
                  </Text>
                  <Button
                    mt={4}
                    size="sm"
                    colorScheme="blue"
                    variant="outline"
                    onClick={(e) => {
                      e.stopPropagation();
                      fileInputRef.current?.click();
                    }}
                  >
                    Browse File
                  </Button>
                </Box>

                {file && (
                  <Flex
                    mt={4}
                    align="center"
                    gap={3}
                    p={3}
                    borderRadius="lg"
                    borderWidth="1px"
                    borderColor="gray.200"
                    bg="white"
                  >
                    <Flex
                      align="center"
                      justify="center"
                      w={10}
                      h={10}
                      borderRadius="md"
                      bg="green.50"
                      color="green.600"
                    >
                      <Icon as={FiFileText} boxSize={5} />
                    </Flex>
                    <Box flex={1} minW={0}>
                      <Text fontSize="sm" fontWeight="medium" color="gray.800" isTruncated>
                        {file.name}
                      </Text>
                      <Text fontSize="xs" color="gray.500">
                        {fileSizeKb} KB
                      </Text>
                    </Box>
                    <IconButton
                      aria-label="Remove file"
                      size="sm"
                      variant="ghost"
                      colorScheme="red"
                      icon={<FiX />}
                      onClick={(e) => {
                        e.stopPropagation();
                        handleClearFile();
                      }}
                    />
                  </Flex>
                )}
              </FormControl>

              <Button
                colorScheme="blue"
                size="md"
                w="full"
                onClick={handleUpload}
                isLoading={loading}
                loadingText="Importing..."
                isDisabled={!file}
                leftIcon={<Icon as={FiUploadCloud} />}
              >
                Import CSV
              </Button>
            </VStack>
          </CardBody>
        </Card>

        {error && (
          <Alert status="error" borderRadius="lg" variant="left-accent">
            <AlertIcon />
            <Box>
              <AlertTitle>Upload failed</AlertTitle>
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
      </VStack>
    </Container>
  );
}
