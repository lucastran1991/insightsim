'use client';

import {
  Select,
  FormControl,
  FormLabel,
  Text,
  Box,
} from '@chakra-ui/react';

interface TagSelectorProps {
  tags: string[];
  selectedTags: string[];
  onChange: (tags: string[]) => void;
  isMulti?: boolean;
  placeholder?: string;
}

export default function TagSelector({
  tags,
  selectedTags,
  onChange,
  isMulti = true,
  placeholder = 'Select tags...',
}: TagSelectorProps) {
  if (isMulti) {
    return (
      <FormControl>
        <FormLabel>Tags</FormLabel>
        <Box>
          <Box
            as="select"
            multiple
            value={selectedTags}
            onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
              const selected = Array.from(
                e.target.selectedOptions,
                (option) => option.value
              );
              onChange(selected);
            }}
            width="100%"
            minHeight="200px"
            padding={2}
            border="1px"
            borderColor="gray.300"
            borderRadius="md"
            fontSize="md"
            _focus={{
              borderColor: 'blue.500',
              boxShadow: '0 0 0 1px blue.500',
            }}
          >
            {tags.map((tag) => (
              <option key={tag} value={tag}>
                {tag}
              </option>
            ))}
          </Box>
          {selectedTags.length > 0 && (
            <Text mt={2} fontSize="sm" color="gray.600">
              {selectedTags.length} tag(s) selected
            </Text>
          )}
        </Box>
      </FormControl>
    );
  }

  return (
    <FormControl>
      <FormLabel>Tag</FormLabel>
      <Select
        value={selectedTags[0] || ''}
        onChange={(e) => onChange(e.target.value ? [e.target.value] : [])}
        placeholder={placeholder}
      >
        {tags.map((tag) => (
          <option key={tag} value={tag}>
            {tag}
          </option>
        ))}
      </Select>
    </FormControl>
  );
}
