#!/bin/bash
mkdir -p /root/ibp/trident-deploy/kuiperd
helm template kuiperd --name=kuiperd --namespace=ivideo --output-dir=/root/ibp/trident-deploy
kubectl delete -f /root/ibp/trident-deploy/kuiperd/templates
