import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useMutation } from '@tanstack/react-query';
import { clustersApi, CreateClusterRequest, HostSpec } from '../api/client';

export const ClusterForm = () => {
  const navigate = useNavigate();
  const [formData, setFormData] = useState<CreateClusterRequest>({
    name: '',
    k8s_version: '1.28.0',
    cni: 'calico',
    container_runtime: 'containerd',
    pod_network_cidr: '10.244.0.0/16',
    service_cidr: '10.96.0.0/12',
    control_planes: [{ hostname: '', address: '', user: 'ubuntu', ssh_key_path: '', port: 22 }],
    workers: [],
  });

  const createMutation = useMutation({
    mutationFn: (data: CreateClusterRequest) => clustersApi.create(data),
    onSuccess: (response) => {
      navigate(`/clusters/${response.data.data.id}`);
    },
    onError: (error) => {
      console.error('Failed to create cluster:', error);
      alert('Failed to create cluster. Check console for details.');
    },
  });

  const addControlPlane = () => {
    setFormData({
      ...formData,
      control_planes: [
        ...formData.control_planes,
        { hostname: '', address: '', user: 'ubuntu', ssh_key_path: '', port: 22 },
      ],
    });
  };

  const removeControlPlane = (index: number) => {
    setFormData({
      ...formData,
      control_planes: formData.control_planes.filter((_, i) => i !== index),
    });
  };

  const updateControlPlane = (index: number, field: keyof HostSpec, value: string | number) => {
    const updated = [...formData.control_planes];
    updated[index] = { ...updated[index], [field]: value };
    setFormData({ ...formData, control_planes: updated });
  };

  const addWorker = () => {
    setFormData({
      ...formData,
      workers: [
        ...formData.workers,
        { hostname: '', address: '', user: 'ubuntu', ssh_key_path: '', port: 22 },
      ],
    });
  };

  const removeWorker = (index: number) => {
    setFormData({
      ...formData,
      workers: formData.workers.filter((_, i) => i !== index),
    });
  };

  const updateWorker = (index: number, field: keyof HostSpec, value: string | number) => {
    const updated = [...formData.workers];
    updated[index] = { ...updated[index], [field]: value };
    setFormData({ ...formData, workers: updated });
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    createMutation.mutate(formData);
  };

  return (
    <div className="max-w-4xl mx-auto">
      <div className="mb-6">
        <h1 className="text-3xl font-bold text-gray-900">Create New Cluster</h1>
        <p className="text-gray-600 mt-2">Configure and deploy a new Kubernetes cluster</p>
      </div>

      <form onSubmit={handleSubmit} className="bg-white rounded-lg border border-gray-200 p-6">
        {/* Basic Configuration */}
        <div className="mb-8">
          <h2 className="text-xl font-semibold text-gray-900 mb-4">Basic Configuration</h2>
          <div className="grid gap-4 md:grid-cols-2">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Cluster Name *
              </label>
              <input
                type="text"
                required
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="my-cluster"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Kubernetes Version *
              </label>
              <input
                type="text"
                required
                value={formData.k8s_version}
                onChange={(e) => setFormData({ ...formData, k8s_version: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="1.28.0"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">CNI Plugin *</label>
              <select
                value={formData.cni}
                onChange={(e) => setFormData({ ...formData, cni: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="calico">Calico</option>
                <option value="flannel">Flannel</option>
                <option value="weave">Weave</option>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Container Runtime *
              </label>
              <select
                value={formData.container_runtime}
                onChange={(e) => setFormData({ ...formData, container_runtime: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="containerd">containerd</option>
                <option value="cri-o">CRI-O</option>
              </select>
            </div>
          </div>
        </div>

        {/* Control Plane Nodes */}
        <div className="mb-8">
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-xl font-semibold text-gray-900">Control Plane Nodes</h2>
            <button
              type="button"
              onClick={addControlPlane}
              className="text-blue-600 hover:text-blue-700 text-sm font-medium"
            >
              + Add Control Plane
            </button>
          </div>

          {formData.control_planes.map((cp, index) => (
            <div key={index} className="border border-gray-200 rounded-lg p-4 mb-4">
              <div className="flex justify-between items-start mb-3">
                <h3 className="font-medium text-gray-900">Control Plane {index + 1}</h3>
                {formData.control_planes.length > 1 && (
                  <button
                    type="button"
                    onClick={() => removeControlPlane(index)}
                    className="text-red-600 hover:text-red-700 text-sm"
                  >
                    Remove
                  </button>
                )}
              </div>

              <div className="grid gap-3 md:grid-cols-2">
                <input
                  type="text"
                  required
                  placeholder="Hostname"
                  value={cp.hostname}
                  onChange={(e) => updateControlPlane(index, 'hostname', e.target.value)}
                  className="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <input
                  type="text"
                  required
                  placeholder="IP Address"
                  value={cp.address}
                  onChange={(e) => updateControlPlane(index, 'address', e.target.value)}
                  className="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <input
                  type="text"
                  required
                  placeholder="SSH User"
                  value={cp.user}
                  onChange={(e) => updateControlPlane(index, 'user', e.target.value)}
                  className="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <input
                  type="text"
                  required
                  placeholder="SSH Key Path"
                  value={cp.ssh_key_path}
                  onChange={(e) => updateControlPlane(index, 'ssh_key_path', e.target.value)}
                  className="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
            </div>
          ))}
        </div>

        {/* Worker Nodes */}
        <div className="mb-8">
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-xl font-semibold text-gray-900">Worker Nodes</h2>
            <button
              type="button"
              onClick={addWorker}
              className="text-blue-600 hover:text-blue-700 text-sm font-medium"
            >
              + Add Worker
            </button>
          </div>

          {formData.workers.length === 0 ? (
            <p className="text-gray-500 text-sm italic">No worker nodes yet. Click "Add Worker" to add one.</p>
          ) : (
            formData.workers.map((worker, index) => (
              <div key={index} className="border border-gray-200 rounded-lg p-4 mb-4">
                <div className="flex justify-between items-start mb-3">
                  <h3 className="font-medium text-gray-900">Worker {index + 1}</h3>
                  <button
                    type="button"
                    onClick={() => removeWorker(index)}
                    className="text-red-600 hover:text-red-700 text-sm"
                  >
                    Remove
                  </button>
                </div>

                <div className="grid gap-3 md:grid-cols-2">
                  <input
                    type="text"
                    required
                    placeholder="Hostname"
                    value={worker.hostname}
                    onChange={(e) => updateWorker(index, 'hostname', e.target.value)}
                    className="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                  <input
                    type="text"
                    required
                    placeholder="IP Address"
                    value={worker.address}
                    onChange={(e) => updateWorker(index, 'address', e.target.value)}
                    className="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                  <input
                    type="text"
                    required
                    placeholder="SSH User"
                    value={worker.user}
                    onChange={(e) => updateWorker(index, 'user', e.target.value)}
                    className="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                  <input
                    type="text"
                    required
                    placeholder="SSH Key Path"
                    value={worker.ssh_key_path}
                    onChange={(e) => updateWorker(index, 'ssh_key_path', e.target.value)}
                    className="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
              </div>
            ))
          )}
        </div>

        {/* Submit */}
        <div className="flex gap-3 justify-end">
          <button
            type="button"
            onClick={() => navigate('/')}
            className="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50 font-medium"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={createMutation.isPending}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md font-medium disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {createMutation.isPending ? 'Creating...' : 'Create Cluster'}
          </button>
        </div>
      </form>
    </div>
  );
};
