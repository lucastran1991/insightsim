'use client';

import { useState, useMemo } from 'react';
import {
  Select,
  FormControl,
  FormLabel,
  Box,
  Menu,
  MenuButton,
  MenuList,
  MenuItemOption,
  MenuOptionGroup,
  Button,
  Input,
  InputGroup,
  InputLeftElement,
  Icon,
} from '@chakra-ui/react';
import { ChevronDownIcon } from '@chakra-ui/icons';
import { FiSearch } from 'react-icons/fi';

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
  const [searchKeyword, setSearchKeyword] = useState('');

  // Filter tags locally based on search keyword (case-insensitive)
  const filteredTags = useMemo(() => {
    if (!searchKeyword.trim()) {
      return tags;
    }
    const keyword = searchKeyword.toLowerCase();
    return tags.filter((tag) => tag.toLowerCase().includes(keyword));
  }, [tags, searchKeyword]);

  if (isMulti) {
    return (
      <FormControl>
        <FormLabel>Tags</FormLabel>
        <Menu closeOnSelect={false} onClose={() => setSearchKeyword('')}>
          <MenuButton
            as={Button}
            rightIcon={<ChevronDownIcon />}
            width="100%"
            textAlign="left"
            fontWeight="normal"
          >
            {selectedTags.length === 0
              ? placeholder
              : `${selectedTags.length} tag(s) selected`}
          </MenuButton>
          <MenuList maxH="300px" overflowY="auto">
            <Box px={3} pt={2} pb={2} borderBottomWidth="1px" borderColor="gray.200">
              <InputGroup size="sm">
                <InputLeftElement pointerEvents="none">
                  <Icon as={FiSearch} color="gray.400" boxSize={4} />
                </InputLeftElement>
                <Input
                  placeholder="Search tags..."
                  value={searchKeyword}
                  onChange={(e) => setSearchKeyword(e.target.value)}
                  onClick={(e) => e.stopPropagation()}
                  onKeyDown={(e) => e.stopPropagation()}
                  bg="white"
                  borderColor="gray.200"
                  _placeholder={{ color: 'gray.500' }}
                />
              </InputGroup>
            </Box>
            <MenuOptionGroup
              type="checkbox"
              value={selectedTags}
              onChange={(values) =>
              onChange(
                Array.isArray(values) ? values : values ? [values] : []
              )}
            >
              {filteredTags.length > 0 ? (
                filteredTags.map((tag) => (
                  <MenuItemOption key={tag} value={tag}>
                    {tag}
                  </MenuItemOption>
                ))
              ) : (
                <Box px={3} py={2} color="gray.500" fontSize="sm">
                  No tags found
                </Box>
              )}
            </MenuOptionGroup>
          </MenuList>
        </Menu>
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
        {filteredTags.map((tag) => (
          <option key={tag} value={tag}>
            {tag}
          </option>
        ))}
      </Select>
    </FormControl>
  );
}
