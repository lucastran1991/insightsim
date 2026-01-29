'use client';

import { useState, useEffect, useRef } from 'react';
import {
  Container,
  Box,
  Heading,
  Text,
  Button,
  useToast,
  Card,
  CardBody,
  SimpleGrid,
  Flex,
  IconButton,
  Input,
  InputGroup,
  InputLeftElement,
  AlertDialog,
  AlertDialogBody,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogContent,
  AlertDialogOverlay,
  useDisclosure,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
  FormControl,
  FormLabel,
  Spinner,
  HStack,
  Icon,
} from '@chakra-ui/react';
import { FiTrash2, FiPlus, FiTag, FiCalendar, FiSearch } from 'react-icons/fi';
import { getTagsWithStats, deleteTag, createTag } from '@/lib/api';
import type { TagWithStats } from '@/types/api';
import { format } from 'date-fns';

function formatTime(iso: string | null): string {
  if (!iso) return 'No data';
  try {
    const d = new Date(iso);
    return isNaN(d.getTime()) ? iso : format(d, 'yyyy-MM-dd HH:mm');
  } catch {
    return iso;
  }
}

const PAGE_SIZE = 9;

export default function TagsPage() {
  const [tags, setTags] = useState<TagWithStats[]>([]);
  const [totalTags, setTotalTags] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [tagToDelete, setTagToDelete] = useState<string | null>(null);
  const [deleteLoading, setDeleteLoading] = useState(false);
  const [newTagName, setNewTagName] = useState('');
  const [createLoading, setCreateLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const toast = useToast();
  const deleteDialog = useDisclosure();
  const createModal = useDisclosure();
  const cancelDeleteRef = useRef<HTMLButtonElement>(null);

  // Debounce search keyword (300ms) before sending to API
  useEffect(() => {
    const t = setTimeout(() => {
      setDebouncedSearch(searchKeyword);
      setPage(1);
    }, 300);
    return () => clearTimeout(t);
  }, [searchKeyword]);

  const fetchTags = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await getTagsWithStats(page, PAGE_SIZE, debouncedSearch.trim() || undefined);
      const items = Array.isArray(res?.items) ? res.items : [];
      const total = typeof res?.total === 'number' ? res.total : 0;
      setTags(items);
      setTotalTags(total);
      // If we got empty items but there are still tags (e.g. deleted last item on page), go to previous page
      if (items.length === 0 && total > 0 && page > 1) {
        setPage(1);
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to load tags';
      setError(msg);
      toast({ title: 'Error', description: msg, status: 'error', duration: 5000, isClosable: true });
    } finally {
      setLoading(false);
    }
  };

  const fetchTagsRef = useRef(fetchTags);
  useEffect(() => {
    fetchTagsRef.current = fetchTags;
  });

  useEffect(() => {
    fetchTags();
  }, [page, debouncedSearch]);

  const totalPages = Math.max(1, Math.ceil(totalTags / PAGE_SIZE));

  const openDeleteDialog = (tag: string) => {
    setTagToDelete(tag);
    deleteDialog.onOpen();
  };

  const handleConfirmDelete = async () => {
    if (!tagToDelete) return;
    setDeleteLoading(true);
    try {
      await deleteTag(tagToDelete);
      toast({ title: 'Tag deleted', description: `"${tagToDelete}" and its data removed.`, status: 'success', duration: 4000, isClosable: true });
      deleteDialog.onClose();
      setTagToDelete(null);
      await fetchTagsRef.current();
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to delete tag';
      toast({ title: 'Error', description: msg, status: 'error', duration: 5000, isClosable: true });
    } finally {
      setDeleteLoading(false);
    }
  };

  const handleCreate = async () => {
    const name = newTagName.trim();
    if (!name) {
      toast({ title: 'Validation', description: 'Enter a tag name.', status: 'warning', duration: 3000, isClosable: true });
      return;
    }
    setCreateLoading(true);
    try {
      await createTag(name);
      createModal.onClose();
      setNewTagName('');
      await fetchTags();
      toast({ title: 'Tag created', description: `"${name}" added. Generate or upload data from Setup/Upload.`, status: 'success', duration: 5000, isClosable: true });
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to create tag';
      toast({ title: 'Error', description: msg, status: 'error', duration: 5000, isClosable: true });
    } finally {
      setCreateLoading(false);
    }
  };

  return (
    <Container maxW="container.xl" py={8} px={{ base: 4, md: 6 }}>
      <Flex justify="space-between" align="center" mb={6}>
        <Box>
          <Heading size="lg" fontWeight="semibold" color="gray.800">
            Tags
          </Heading>
          <Text mt={1} fontSize="sm" color="gray.600">
            Tags in the system with record count and time range
          </Text>
        </Box>
        <Button leftIcon={<FiPlus />} colorScheme="blue" size="sm" onClick={createModal.onOpen}>
          Create
        </Button>
      </Flex>

      <Flex justify="center" mb={6}>
        <InputGroup maxW="400px" size="md">
          <InputLeftElement pointerEvents="none">
            <Icon as={FiSearch} color="gray.400" />
          </InputLeftElement>
          <Input
            placeholder="Search tags by name..."
            value={searchKeyword}
            onChange={(e) => setSearchKeyword(e.target.value)}
            bg="white"
            borderColor="gray.200"
            _placeholder={{ color: 'gray.500' }}
          />
        </InputGroup>
      </Flex>

      {loading ? (
        <Flex justify="center" py={12}>
          <Spinner size="lg" />
        </Flex>
      ) : error ? (
        <Text color="red.600">{error}</Text>
      ) : totalTags === 0 ? (
        <Text color="gray.600">No tags yet. Create one or generate/upload data from Setup/Upload.</Text>
      ) : (
        <Box>
          <SimpleGrid columns={{ base: 1, sm: 2, md: 3 }} spacing={5}>
            {tags.map((t) => (
              <Card
                key={t.tag}
                borderRadius="xl"
                borderWidth="1px"
                borderColor="gray.200"
                overflow="hidden"
                shadow="md"
                _hover={{ shadow: 'lg', borderColor: 'blue.200' }}
                transition="all 0.2s"
                bg="white"
              >
                <Box h="1" bgGradient="linear(to-r, blue.400, blue.600)" />
                <CardBody pt={4} pb={3}>
                  <Flex justify="space-between" align="flex-start" gap={2}>
                    <HStack spacing={2} flex={1} minW={0}>
                      <Flex
                        align="center"
                        justify="center"
                        w={9}
                        h={9}
                        borderRadius="lg"
                        bg="blue.50"
                        color="blue.600"
                        flexShrink={0}
                      >
                        <Icon as={FiTag} boxSize={4} />
                      </Flex>
                      <Heading size="sm" fontWeight="semibold" color="gray.800" noOfLines={2} wordBreak="break-all">
                        {t.tag}
                      </Heading>
                    </HStack>
                    <IconButton
                      aria-label="Delete tag"
                      icon={<FiTrash2 />}
                      variant="ghost"
                      size="sm"
                      colorScheme="red"
                      flexShrink={0}
                      onClick={() => openDeleteDialog(t.tag)}
                    />
                  </Flex>
                  <HStack mt={2} spacing={2} align="baseline" fontSize="xs">
                    <Icon as={FiCalendar} boxSize={3} flexShrink={0} color="gray.500" />
                    <Text as="span" color="gray.500" fontWeight="medium">
                      Created
                    </Text>
                    <Text as="span" color="gray.600" noOfLines={1} minW={0}>
                      {t.created_at?.trim() ? formatTime(t.created_at) : '—'}
                    </Text>
                  </HStack>
                </CardBody>
              </Card>
            ))}
          </SimpleGrid>
          {totalPages > 1 && (
            <Flex justify="center" align="center" gap={2} mt={6} pb={4}>
              <Button
                size="sm"
                variant="outline"
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                isDisabled={page <= 1}
              >
                Previous
              </Button>
              <Text fontSize="sm" color="gray.600" minW="100px" textAlign="center">
                Page {page} of {totalPages}
              </Text>
              <Button
                size="sm"
                variant="outline"
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                isDisabled={page >= totalPages}
              >
                Next
              </Button>
            </Flex>
          )}
        </Box>
      )}

      <AlertDialog isOpen={deleteDialog.isOpen} onClose={deleteDialog.onClose} leastDestructiveRef={cancelDeleteRef}>
        <AlertDialogOverlay>
          <AlertDialogContent>
            <AlertDialogHeader>Delete tag</AlertDialogHeader>
            <AlertDialogBody>Bye bye this tag and it&apos;s data?</AlertDialogBody>
            <AlertDialogFooter>
              <Button ref={cancelDeleteRef} onClick={deleteDialog.onClose}>
                No
              </Button>
              <Button colorScheme="red" ml={3} onClick={handleConfirmDelete} isLoading={deleteLoading}>
                Yes
              </Button>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialogOverlay>
      </AlertDialog>

      <Modal isOpen={createModal.isOpen} onClose={createModal.onClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Create tag</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <FormControl>
              <FormLabel>Tag name</FormLabel>
              <Input
                placeholder="e.g. MY.TAG.001"
                value={newTagName}
                onChange={(e) => setNewTagName(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
              />
            </FormControl>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={createModal.onClose}>
              Cancel
            </Button>
            <Button colorScheme="blue" onClick={handleCreate} isLoading={createLoading}>
              Create
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Container>
  );
}
