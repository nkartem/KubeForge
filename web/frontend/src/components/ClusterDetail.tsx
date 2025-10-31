import { useParams, useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { clustersApi } from '../api/client';
import { useWebSocket } from '../hooks/useWebSocket';
import { useEffect, useRef } from 'react';

export const ClusterDetail = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const logsEndRef = useRef<HTMLDivElement>(null);
  const clusterId = id ? parseInt(id) : null;

  const { data: cluster, isLoading } = useQuery({
    queryKey: ['cluster', clusterId],
    queryFn: async () => {
      const response = await clustersApi.get(clusterId!);
      return response.data.data;
    },
    enabled: !!clusterId,
    refetchInterval: 3000,
  });

  const { data: eventsData } = useQuery({
    queryKey: ['events', clusterId],
    queryFn: async () => {
      const response = await clustersApi.getEvents(clusterId!);
      return response.data.data;
    },
    enabled: !!clusterId,
    refetchInterval: 2000,
  });

  const { events: wsEvents, isConnected } = useWebSocket(clusterId);

  // Auto-scroll to bottom when new events arrive
  useEffect(() => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [wsEvents, eventsData]);

  const allEvents = [...(eventsData || []), ...wsEvents];

  const handleDownloadKubeconfig = async () => {
    try {
      const response = await clustersApi.getKubeconfig(clusterId!);
      const blob = new Blob([response.data], { type: 'application/x-yaml' });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${cluster?.name}-kubeconfig.yaml`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (error) {
      console.error('Failed to download kubeconfig:', error);
      alert('Failed to download kubeconfig');
    }
  };

  const getEventColor = (level: string) => {
    switch (level) {
      case 'info':
        return 'text-blue-600';
      case 'warn':
        return 'text-yellow-600';
      case 'error':
        return 'text-red-600';
      default:
        return 'text-gray-600';
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'ready':
        return 'bg-green-100 text-green-800';
      case 'provisioning':
        return 'bg-yellow-100 text-yellow-800 animate-pulse';
      case 'failed':
        return 'bg-red-100 text-red-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  if (isLoading) {
    return (
      <div className="flex justify-center items-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  if (!cluster) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-4">
        <p className="text-red-800">Cluster not found</p>
      </div>
    );
  }

  return (
    <div>
      {/* Header */}
      <div className="mb-6">
        <button
          onClick={() => navigate('/')}
          className="text-blue-600 hover:text-blue-700 mb-4 flex items-center gap-2"
        >
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
          Back to Dashboard
        </button>

        <div className="flex justify-between items-start">
          <div>
            <h1 className="text-3xl font-bold text-gray-900">{cluster.name}</h1>
            <div className="flex items-center gap-3 mt-2">
              <span
                className={`px-3 py-1 text-sm font-medium rounded-full ${getStatusColor(
                  cluster.status
                )}`}
              >
                {cluster.status}
              </span>
              {isConnected && (
                <span className="flex items-center gap-2 text-sm text-green-600">
                  <span className="w-2 h-2 bg-green-600 rounded-full animate-pulse"></span>
                  Live
                </span>
              )}
            </div>
          </div>

          <button
            onClick={handleDownloadKubeconfig}
            disabled={cluster.status !== 'ready'}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Download Kubeconfig
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Cluster Info */}
        <div className="lg:col-span-1">
          <div className="bg-white border border-gray-200 rounded-lg p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Cluster Information</h2>
            <dl className="space-y-3">
              <div>
                <dt className="text-sm text-gray-600">Kubernetes Version</dt>
                <dd className="text-sm font-medium text-gray-900">{cluster.k8s_version}</dd>
              </div>
              <div>
                <dt className="text-sm text-gray-600">CNI Plugin</dt>
                <dd className="text-sm font-medium text-gray-900">{cluster.cni}</dd>
              </div>
              <div>
                <dt className="text-sm text-gray-600">Container Runtime</dt>
                <dd className="text-sm font-medium text-gray-900">{cluster.container_runtime}</dd>
              </div>
              <div>
                <dt className="text-sm text-gray-600">Pod Network CIDR</dt>
                <dd className="text-sm font-medium text-gray-900">{cluster.pod_network_cidr}</dd>
              </div>
              <div>
                <dt className="text-sm text-gray-600">Service CIDR</dt>
                <dd className="text-sm font-medium text-gray-900">{cluster.service_cidr}</dd>
              </div>
              <div>
                <dt className="text-sm text-gray-600">Created</dt>
                <dd className="text-sm font-medium text-gray-900">
                  {new Date(cluster.created_at).toLocaleString()}
                </dd>
              </div>
            </dl>
          </div>

          {/* Nodes */}
          <div className="bg-white border border-gray-200 rounded-lg p-6 mt-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Nodes</h2>
            {cluster.nodes && cluster.nodes.length > 0 ? (
              <div className="space-y-3">
                {cluster.nodes.map((node) => (
                  <div key={node.id} className="border border-gray-200 rounded-md p-3">
                    <div className="flex justify-between items-start mb-2">
                      <span className="font-medium text-gray-900">{node.hostname}</span>
                      <span className="text-xs px-2 py-1 bg-blue-100 text-blue-800 rounded">
                        {node.role}
                      </span>
                    </div>
                    <div className="text-sm text-gray-600">{node.address}</div>
                    <div className="text-xs text-gray-500 mt-1">Status: {node.status}</div>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-gray-500">No nodes yet</p>
            )}
          </div>
        </div>

        {/* Provision Logs */}
        <div className="lg:col-span-2">
          <div className="bg-white border border-gray-200 rounded-lg p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Provision Logs</h2>
            <div className="bg-gray-900 rounded-lg p-4 h-[600px] overflow-y-auto font-mono text-sm">
              {allEvents.length === 0 ? (
                <div className="text-gray-500">No events yet...</div>
              ) : (
                allEvents.map((event, index) => (
                  <div key={index} className="mb-2">
                    <span className="text-gray-500">
                      {new Date(event.timestamp).toLocaleTimeString()}
                    </span>
                    <span className={`ml-2 font-semibold ${getEventColor(event.level)}`}>
                      [{event.level.toUpperCase()}]
                    </span>
                    <span className="text-blue-400 ml-2">[{event.host}]</span>
                    <span className="text-yellow-400 ml-2">{event.step}:</span>
                    <span className="text-gray-300 ml-2">{event.message}</span>
                    {event.output && (
                      <div className="text-gray-400 ml-8 mt-1 text-xs whitespace-pre-wrap">
                        {event.output}
                      </div>
                    )}
                  </div>
                ))
              )}
              <div ref={logsEndRef} />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
