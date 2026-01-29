'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { Container, Spinner, VStack, Text } from '@chakra-ui/react';

export default function HomePage() {
  const router = useRouter();

  useEffect(() => {
    // Redirect to chart page by default
    router.push('/chart');
  }, [router]);

  return (
    <Container maxW="container.md" py={20}>
      <VStack spacing={4}>
        <Spinner size="xl" />
        <Text color="gray.600">Redirecting...</Text>
      </VStack>
    </Container>
  );
}
