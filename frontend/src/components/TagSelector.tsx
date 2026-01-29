'use client';

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
} from '@chakra-ui/react';
import { ChevronDownIcon } from '@chakra-ui/icons';

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
        <Menu closeOnSelect={false}>
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
            <MenuOptionGroup
              type="checkbox"
              value={selectedTags}
              onChange={(values) =>
              onChange(
                Array.isArray(values) ? values : values ? [values] : []
              )}
            >
              {tags.map((tag) => (
                <MenuItemOption key={tag} value={tag}>
                  {tag}
                </MenuItemOption>
              ))}
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
        {tags.map((tag) => (
          <option key={tag} value={tag}>
            {tag}
          </option>
        ))}
      </Select>
    </FormControl>
  );
}
