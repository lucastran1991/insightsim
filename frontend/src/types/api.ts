export interface TimeseriesDataPoint {
  timestamp: string;
  value: number;
  quality: number;
}

export interface TimeseriesResponse {
  result: {
    [tag: string]: TimeseriesDataPoint[];
  };
}

export interface GenerateResponse {
  success: boolean;
  message: string;
  count?: number;
  tags_count?: number;
}

export interface LoadResponse {
  success: boolean;
  message: string;
  count?: number;
  files_count?: number;
}

export interface HealthResponse {
  status: string;
}

export interface UploadCsvResponse {
  success: boolean;
  message: string;
  count?: number;
  tags_affected?: number;
}

export interface TagWithStats {
  tag: string;
  created_at: string | null;
  updated_at: string | null;
  source: string;
}

export interface TagsListResponse {
  items: TagWithStats[];
  total: number;
}
