# Insightsim Frontend

Frontend application built with Next.js 13 and Chakra UI for visualizing and managing timeseries data.

## Features

- **Chart Page**: Visualize timeseries data with interactive charts
  - Time mode selector: 1 day, 1 week, 1 month
  - Multi-tag selection
  - Real-time data fetching from backend API

- **Setup Page**: Generate dummy data
  - Generate for all tags or single tag
  - Date range configuration (uses backend config)
  - Progress indicators and error handling

## Getting Started

### Prerequisites

- Node.js 18+ and npm
- Backend API running (default: http://localhost:8088)

### Installation

```bash
cd frontend
npm install
```

### Development

```bash
npm run dev
```

The application will be available at http://localhost:3000

### Build

```bash
npm run build
npm start
```

## Configuration

The backend API URL can be configured via environment variable:

```bash
NEXT_PUBLIC_API_URL=http://localhost:8088 npm run dev
```

Default backend URL: `http://localhost:8088`

## Project Structure

```
frontend/
├── src/
│   ├── app/              # Next.js 13 app directory
│   │   ├── layout.tsx    # Root layout with ChakraProvider
│   │   ├── page.tsx      # Home page (redirects to /chart)
│   │   ├── chart/        # Chart page
│   │   └── setup/        # Setup page
│   ├── components/       # Reusable components
│   │   ├── Navigation.tsx
│   │   ├── TimeModeSelector.tsx
│   │   ├── TagSelector.tsx
│   │   ├── TimeseriesChart.tsx
│   │   ├── DateRangePicker.tsx
│   │   └── Providers.tsx
│   ├── lib/              # Utilities
│   │   ├── api.ts        # API client functions
│   │   └── config.ts     # Configuration
│   └── types/            # TypeScript types
│       └── api.ts
└── public/               # Static assets
    └── tag_list.json     # Tag list file
```

## Technologies

- **Next.js 13**: React framework with App Router
- **Chakra UI 2.10.9**: Component library
- **Recharts**: Chart visualization library
- **date-fns**: Date utility library
- **TypeScript**: Type safety

## API Integration

The frontend communicates with the Golang backend API:

- `GET /health` - Health check
- `GET /api/timeseriesdata/{start}/{end}?tags=...` - Query timeseries data
- `POST /api/generate-dummy` - Generate dummy data

See `API.md` in the project root for detailed API documentation.
