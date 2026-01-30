import {
  TimeseriesResponse,
  GenerateResponse,
  LoadResponse,
  HealthResponse,
  UploadCsvResponse,
  TagWithStats,
  TagsListResponse,
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
 * Fetch value_range (min, max) from backend config for use as defaults (e.g. chart value filter).
 */
export async function getValueRangeConfig(): Promise<{ min: number; max: number }> {
  try {
    const response = await fetch(`${API_BASE_URL}${API_ENDPOINTS.config}`);
    if (!response.ok) return { min: 1, max: 10000 };
    const data = await response.json();
    const vr = data?.value_range;
    if (vr != null && typeof vr.min === 'number' && typeof vr.max === 'number') {
      return { min: vr.min, max: vr.max };
    }
  } catch {
    // ignore
  }
  return { min: 1, max: 10000 };
}

export type AggregateMode = 'raw' | 'daily' | 'monthly' | 'quarterly' | 'yearly';

/**
 * Query timeseries data
 * @param start Start timestamp in ISO 8601 format (e.g., "2025-12-01T00:00:00")
 * @param end End timestamp in ISO 8601 format (e.g., "2025-12-01T23:59:59")
 * @param tags Optional array of tag names to filter
 * @param aggregate Optional aggregation: raw, daily, monthly, quarterly, yearly
 */
export async function getTimeseriesData(
  start: string,
  end: string,
  tags?: string[],
  aggregate?: AggregateMode
): Promise<TimeseriesResponse> {
  const url = new URL(
    `${API_BASE_URL}${API_ENDPOINTS.timeseriesData}/${start}/${end}`
  );

  if (tags && tags.length > 0) {
    url.searchParams.set('tags', tags.join(','));
  }
  if (aggregate && aggregate !== 'raw') {
    url.searchParams.set('aggregate', aggregate);
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

/** Stream event from backend generate-dummy NDJSON response */
export interface GenerateStreamEvent {
  event: 'tag_complete' | 'done' | 'error';
  tag?: string;
  records?: number;
  count?: number;
  tags_count?: number;
  message?: string;
}

export type GenerateFrequency = '1min' | '5min' | '15min' | '30min' | '1hour';

/**
 * Generate dummy data (streaming). Backend sends NDJSON: tag_complete per tag, then done or error.
 * @param tag Optional tag name. If provided, only generates data for that tag.
 * @param start Optional start time (ISO 8601). If provided with end, overrides backend config time range.
 * @param end Optional end time (ISO 8601). If provided with start, overrides backend config time range.
 * @param frequencyOrOnTagComplete Optional frequency (1 record per N minutes) or callback for backward compatibility.
 * @param onTagComplete Called when the backend finishes generating data for each tag.
 */
export async function generateDummyData(
  tag?: string,
  start?: string,
  end?: string,
  frequencyOrOnTagComplete?: GenerateFrequency | ((tag: string, records: number) => void),
  onTagComplete?: (tag: string, records: number) => void
): Promise<GenerateResponse> {
  const frequency =
    typeof frequencyOrOnTagComplete === 'string' ? frequencyOrOnTagComplete : undefined;
  const cb = typeof frequencyOrOnTagComplete === 'function' ? frequencyOrOnTagComplete : onTagComplete;

  const payload: { tag?: string; start?: string; end?: string; frequency?: string } = {};
  if (tag) payload.tag = tag;
  if (start?.trim()) payload.start = start.trim();
  if (end?.trim()) payload.end = end.trim();
  if (frequency) payload.frequency = frequency;
  const body = Object.keys(payload).length > 0 ? JSON.stringify(payload) : undefined;

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

  const reader = response.body?.getReader();
  if (!reader) {
    throw new Error('No response body');
  }

  const decoder = new TextDecoder();
  let buffer = '';
  let result: GenerateResponse | null = null;
  let streamError: Error | null = null;

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop() ?? '';
    for (const line of lines) {
      const trimmed = line.trim();
      if (!trimmed) continue;
      try {
        const ev = JSON.parse(trimmed) as GenerateStreamEvent;
        if (ev.event === 'tag_complete' && ev.tag != null && ev.records != null) {
          cb?.(ev.tag, ev.records);
        } else if (ev.event === 'done') {
          result = {
            success: true,
            message: 'Dummy data generated successfully',
            count: ev.count,
            tags_count: ev.tags_count,
          };
        } else if (ev.event === 'error') {
          streamError = new Error(ev.message ?? 'Generation failed');
        }
      } catch {
        // ignore non-JSON lines
      }
    }
  }
  if (buffer.trim()) {
    try {
      const ev = JSON.parse(buffer.trim()) as GenerateStreamEvent;
      if (ev.event === 'tag_complete' && ev.tag != null && ev.records != null) {
        cb?.(ev.tag, ev.records);
      } else if (ev.event === 'done') {
        result = {
          success: true,
          message: 'Dummy data generated successfully',
          count: ev.count,
          tags_count: ev.tags_count,
        };
      } else if (ev.event === 'error') {
        streamError = new Error(ev.message ?? 'Generation failed');
      }
    } catch {
      // ignore
    }
  }

  if (streamError) throw streamError;
  if (result) return result;
  throw new Error('No done or error event in stream');
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
 * Upload CSV to override or replace tag data.
 * @param file CSV file (header: timestamp, tag1, tag2, ...)
 * @param mode override = upsert within CSV timestamps; replace = delete all data for tags in CSV then insert CSV rows
 */
export async function uploadCsv(
  file: File,
  mode: 'override' | 'replace'
): Promise<UploadCsvResponse> {
  const formData = new FormData();
  formData.append('file', file);
  formData.append('mode', mode);

  const response = await fetch(`${API_BASE_URL}${API_ENDPOINTS.uploadCsv}`, {
    method: 'POST',
    body: formData,
  });

  const data: UploadCsvResponse = await response.json().catch(() => ({
    success: false,
    message: `HTTP ${response.status}: ${response.statusText}`,
  }));

  if (!response.ok) {
    throw new Error(data.message || 'Failed to upload CSV');
  }

  return data;
}

/**
 * Get tag list from backend API (tags table)
 */
export async function getTagList(): Promise<string[]> {
  try {
    const response = await fetch(`${API_BASE_URL}${API_ENDPOINTS.tags}/names`);
    if (response.ok) {
      const data = await response.json();
      const tags = data?.tags;
      if (Array.isArray(tags)) {
        return tags.map((t: unknown) => String(t).trim()).filter(Boolean);
      }
    }
  } catch (error) {
    console.error('Failed to fetch tag list:', error);
  }

  return [];
}

/**
 * Get a page of tags with stats from backend (paginated)
 * @param page 1-based page (default 1)
 * @param limit page size (default 9)
 * @param search optional keyword to filter tags by name (case-insensitive)
 */
export async function getTagsWithStats(
  page: number = 1,
  limit: number = 9,
  search?: string
): Promise<TagsListResponse> {
  const params = new URLSearchParams({ page: String(page), limit: String(limit) });
  if (search?.trim()) params.set('q', search.trim());
  const url = `${API_BASE_URL}${API_ENDPOINTS.tags}?${params.toString()}`;
  const response = await fetch(url);
  if (!response.ok) {
    const err = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error((err as { error?: string }).error || 'Failed to fetch tags');
  }
  const data = await response.json();
  const rawItems = Array.isArray(data?.items) ? data.items : [];
  const items = rawItems.map((row: Record<string, unknown>) => ({
    tag: typeof row?.tag === 'string' ? row.tag : '',
    created_at: row?.created_at != null ? String(row.created_at) : null,
    updated_at: row?.updated_at != null ? String(row.updated_at) : null,
    source: typeof row?.source === 'string' ? row.source : 'custom',
  }));
  return {
    items,
    total: typeof data?.total === 'number' ? data.total : 0,
  };
}

/**
 * Delete all data for a tag
 */
export async function deleteTag(tag: string): Promise<void> {
  const url = `${API_BASE_URL}${API_ENDPOINTS.tags}?tag=${encodeURIComponent(tag)}`;
  const response = await fetch(url, { method: 'DELETE' });
  if (!response.ok) {
    const err = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error((err as { error?: string }).error || 'Failed to delete tag');
  }
}

/**
 * Create a new tag (add name to tag list)
 */
export async function createTag(tag: string): Promise<void> {
  const response = await fetch(`${API_BASE_URL}${API_ENDPOINTS.tags}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ tag: tag.trim() }),
  });
  if (!response.ok) {
    const err = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error((err as { error?: string }).error || 'Failed to create tag');
  }
}
