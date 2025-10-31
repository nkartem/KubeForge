# KubeForge Frontend

React-based web UI for KubeForge Kubernetes cluster management.

## Tech Stack

- **React 18** with TypeScript
- **Vite** - Lightning fast build tool
- **Tailwind CSS** - Utility-first CSS
- **React Router** - Client-side routing
- **TanStack Query** - Data fetching and caching
- **Axios** - HTTP client
- **WebSocket** - Real-time event streaming

## Getting Started

### Prerequisites

- Node.js 18+ and npm
- Running KubeForge backend on `http://localhost:8080`

### Installation

```bash
npm install
```

### Development

Start the dev server with hot reload:

```bash
npm run dev
```

The app will be available at `http://localhost:3000`

### Building for Production

```bash
npm run build
```

Built files will be in the `dist/` directory.

### Environment Variables

Create a `.env` file:

```env
VITE_API_URL=http://localhost:8080
VITE_WS_URL=ws://localhost:8080
```

## Features

### Dashboard
- View all Kubernetes clusters
- Quick cluster creation
- Cluster status monitoring
- Real-time status updates

### Cluster Creation
- Multi-step form
- Support for multiple control planes (HA)
- Support for worker nodes
- CNI plugin selection (Calico, Flannel, Weave)
- Container runtime selection (containerd, CRI-O)

### Cluster Detail View
- Real-time provision logs via WebSocket
- Cluster information display
- Node list with status
- Kubeconfig download
- Live connection indicator

### Real-time Features
- WebSocket connection for live event streaming
- Auto-refreshing cluster list
- Live provision progress tracking
- Color-coded log levels (info, warn, error)

## Project Structure

```
src/
├── api/
│   └── client.ts          # API client and types
├── components/
│   ├── Dashboard.tsx      # Main dashboard
│   ├── ClusterForm.tsx    # Cluster creation form
│   └── ClusterDetail.tsx  # Cluster detail with logs
├── hooks/
│   └── useWebSocket.ts    # WebSocket hook
├── App.tsx                # Main app with routing
├── main.tsx               # Entry point
└── index.css              # Tailwind CSS imports
```

## API Integration

The frontend communicates with the backend via:

1. **REST API** (`/api/*`) - CRUD operations
2. **WebSocket** (`/ws/clusters/:id/events`) - Real-time logs

### API Endpoints Used

- `GET /api/clusters` - List all clusters
- `POST /api/clusters` - Create cluster
- `GET /api/clusters/:id` - Get cluster details
- `DELETE /api/clusters/:id` - Delete cluster
- `GET /api/clusters/:id/kubeconfig` - Download kubeconfig
- `GET /api/clusters/:id/events` - Get provision events
- `WS /ws/clusters/:id/events` - Real-time event stream

## License

MIT
