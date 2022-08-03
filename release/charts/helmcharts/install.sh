#!/bin/bash
mkdir -p /root/ibp/trident-deploy/kuiperd
rm -rf /root/ibp/trident-deploy/kuiperd/templates
helm dependency update kuiperd
helm template kuiperd --name=kuiperd --namespace=ivideo --output-dir=/root/ibp/trident-deploy --values=$1
kubectl apply -f /root/ibp/trident-deploy/kuiperd/templates
