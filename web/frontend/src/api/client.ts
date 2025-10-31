import axios from 'axios';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export const apiClient = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

export interface Cluster {
  id: number;
  name: string;
  k8s_version: string;
  pod_network_cidr: string;
  service_cidr: string;
  cni: string;
  container_runtime: string;
  api_server_endpoint: string;
  status: string;
  created_at: string;
  updated_at: string;
  nodes?: Node[];
}

export interface Node {
  id: number;
  cluster_id: number;
  hostname: string;
  address: string;
  role: string;
  status: string;
  k8s_version: string;
  created_at: string;
}

export interface ProvisionEvent {
  id: number;
  cluster_id: number;
  timestamp: string;
  level: string;
  host: string;
  step: string;
  message: string;
  output?: string;
}

export interface CreateClusterRequest {
  name: string;
  k8s_version: string;
  pod_network_cidr?: string;
  service_cidr?: string;
  cni: string;
  container_runtime: string;
  api_server_endpoint?: string;
  control_planes: HostSpec[];
  workers: HostSpec[];
}

export interface HostSpec {
  hostname: string;
  address: string;
  user: string;
  ssh_key_path: string;
  port: number;
}

// API functions
export const clustersApi = {
  list: () => apiClient.get<{ success: boolean; data: Cluster[] }>('/api/clusters'),

  get: (id: number) => apiClient.get<{ success: boolean; data: Cluster }>(`/api/clusters/${id}`),

  create: (data: CreateClusterRequest) =>
    apiClient.post<{ success: boolean; data: Cluster }>('/api/clusters', data),

  delete: (id: number) => apiClient.delete(`/api/clusters/${id}`),

  getKubeconfig: (id: number) =>
    apiClient.get(`/api/clusters/${id}/kubeconfig`, { responseType: 'blob' }),

  getEvents: (id: number) =>
    apiClient.get<{ success: boolean; data: ProvisionEvent[] }>(`/api/clusters/${id}/events`),
};
