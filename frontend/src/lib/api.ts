import {
  TimeseriesResponse,
  GenerateResponse,
  LoadResponse,
  HealthResponse,
} from '@/types/api';
import { API_BASE_URL, API_ENDPOINTS } from './config';

/**
 * Check backend health status
 */
export async function getHealthCheck(): Promise<HealthResponse> {
  const response = await fetch(`${API_BASE_URL}${API_ENDPOINTS.health}`);
  if (!response.ok) {
    throw new Error('Health check failed');
  }
  const text = await response.text();
  return { status: text };
}

/**
 * Query timeseries data
 * @param start Start timestamp in ISO 8601 format (e.g., "2025-12-01T00:00:00")
 * @param end End timestamp in ISO 8601 format (e.g., "2025-12-01T23:59:59")
 * @param tags Optional array of tag names to filter
 */
export async function getTimeseriesData(
  start: string,
  end: string,
  tags?: string[]
): Promise<TimeseriesResponse> {
  const url = new URL(
    `${API_BASE_URL}${API_ENDPOINTS.timeseriesData}/${start}/${end}`
  );

  if (tags && tags.length > 0) {
    url.searchParams.set('tags', tags.join(','));
  }

  const response = await fetch(url.toString());

  if (!response.ok) {
    const error = await response.json().catch(() => ({
      error: `HTTP ${response.status}: ${response.statusText}`,
    }));
    throw new Error(error.error || 'Failed to fetch timeseries data');
  }

  return response.json();
}

/**
 * Generate dummy data
 * @param tag Optional tag name. If provided, only generates data for that tag.
 */
export async function generateDummyData(
  tag?: string
): Promise<GenerateResponse> {
  const body = tag ? JSON.stringify({ tag }) : undefined;

  const response = await fetch(
    `${API_BASE_URL}${API_ENDPOINTS.generateDummy}`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body,
    }
  );

  if (!response.ok) {
    const error = await response.json().catch(() => ({
      message: `HTTP ${response.status}: ${response.statusText}`,
    }));
    throw new Error(error.message || 'Failed to generate dummy data');
  }

  return response.json();
}

/**
 * Load data from raw_data folder
 */
export async function loadData(): Promise<LoadResponse> {
  const response = await fetch(`${API_BASE_URL}${API_ENDPOINTS.load}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({
      message: `HTTP ${response.status}: ${response.statusText}`,
    }));
    throw new Error(error.message || 'Failed to load data');
  }

  return response.json();
}

/**
 * Get tag list from public tag_list.json file
 */
export async function getTagList(): Promise<string[]> {
  try {
    const response = await fetch('/tag_list.json');
    if (response.ok) {
      const data = await response.json();
      // Support both "tag" and "tag_list" keys
      const tagListStr = data.tag || data.tag_list || '';
      if (tagListStr) {
        return tagListStr.split(',').map((tag: string) => tag.trim()).filter(Boolean);
      }
    }
  } catch (error) {
    console.error('Failed to fetch tag list:', error);
  }

  return [];
}
