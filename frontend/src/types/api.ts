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
