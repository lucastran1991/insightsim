'use client';

import { ChakraProvider } from '@chakra-ui/react';
import Navigation from './Navigation';

export default function Providers({ children }: { children: React.ReactNode }) {
  return (
    <ChakraProvider>
      <Navigation />
      {children}
    </ChakraProvider>
  );
}
