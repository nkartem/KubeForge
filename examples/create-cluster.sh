#!/bin/bash
# Example script to create a Kubernetes cluster using KubeForge

KUBEFORGE_API=${KUBEFORGE_API:-http://localhost:8080}

echo "Creating cluster on KubeForge API: $KUBEFORGE_API"

curl -X POST "$KUBEFORGE_API/api/clusters" \
  -H "Content-Type: application/json" \
  -d @cluster-example.json

echo ""
echo "Cluster creation initiated!"
echo "Check status with: curl $KUBEFORGE_API/api/clusters"
