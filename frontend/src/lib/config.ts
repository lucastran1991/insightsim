// Backend API configuration
// Can be overridden by environment variable NEXT_PUBLIC_API_URL

export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8888';

export const API_ENDPOINTS = {
  health: '/health',
  load: '/api/load',
  generateDummy: '/api/generate-dummy',
  timeseriesData: '/api/timeseriesdata',
  uploadCsv: '/api/upload-csv',
} as const;
