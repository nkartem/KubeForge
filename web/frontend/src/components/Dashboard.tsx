import { useQuery } from '@tanstack/react-query';
import { clustersApi, Cluster } from '../api/client';
import { Link } from 'react-router-dom';
import { useState } from 'react';

export const Dashboard = () => {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['clusters'],
    queryFn: async () => {
      const response = await clustersApi.list();
      return response.data.data;
    },
    refetchInterval: 5000, // Refresh every 5 seconds
  });

  const handleDelete = async (id: number) => {
    if (!confirm('Are you sure you want to delete this cluster?')) return;
    try {
      await clustersApi.delete(id);
      refetch();
    } catch (error) {
      console.error('Failed to delete cluster:', error);
      alert('Failed to delete cluster');
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'ready':
        return 'bg-green-100 text-green-800';
      case 'provisioning':
        return 'bg-yellow-100 text-yellow-800';
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

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-4">
        <p className="text-red-800">Failed to load clusters: {String(error)}</p>
      </div>
    );
  }

  const clusters = data || [];

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold text-gray-900">Kubernetes Clusters</h1>
        <Link
          to="/create"
          className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg font-medium transition-colors"
        >
          + Create Cluster
        </Link>
      </div>

      {clusters.length === 0 ? (
        <div className="bg-gray-50 border-2 border-dashed border-gray-300 rounded-lg p-12 text-center">
          <svg
            className="mx-auto h-12 w-12 text-gray-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
            />
          </svg>
          <h3 className="mt-2 text-sm font-medium text-gray-900">No clusters</h3>
          <p className="mt-1 text-sm text-gray-500">Get started by creating a new cluster.</p>
          <div className="mt-6">
            <Link
              to="/create"
              className="inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700"
            >
              + New Cluster
            </Link>
          </div>
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {clusters.map((cluster) => (
            <div
              key={cluster.id}
              className="bg-white border border-gray-200 rounded-lg p-6 hover:shadow-lg transition-shadow"
            >
              <div className="flex justify-between items-start mb-4">
                <h3 className="text-lg font-semibold text-gray-900">{cluster.name}</h3>
                <span
                  className={`px-2 py-1 text-xs font-medium rounded-full ${getStatusColor(
                    cluster.status
                  )}`}
                >
                  {cluster.status}
                </span>
              </div>

              <div className="space-y-2 text-sm text-gray-600 mb-4">
                <div className="flex justify-between">
                  <span>Version:</span>
                  <span className="font-medium">{cluster.k8s_version}</span>
                </div>
                <div className="flex justify-between">
                  <span>CNI:</span>
                  <span className="font-medium">{cluster.cni}</span>
                </div>
                <div className="flex justify-between">
                  <span>Runtime:</span>
                  <span className="font-medium">{cluster.container_runtime}</span>
                </div>
                <div className="flex justify-between">
                  <span>Nodes:</span>
                  <span className="font-medium">{cluster.nodes?.length || 0}</span>
                </div>
              </div>

              <div className="flex gap-2">
                <Link
                  to={`/clusters/${cluster.id}`}
                  className="flex-1 text-center bg-blue-50 hover:bg-blue-100 text-blue-700 px-3 py-2 rounded-md text-sm font-medium transition-colors"
                >
                  View Details
                </Link>
                <button
                  onClick={() => handleDelete(cluster.id)}
                  className="flex-1 bg-red-50 hover:bg-red-100 text-red-700 px-3 py-2 rounded-md text-sm font-medium transition-colors"
                >
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
