#!/bin/bash

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

echo "Force deleting any stuck pods..."
# Delete all pods created by the KEDA scaled jobs and nuclei scanner
PODS=$(kubectl get pods -n default | grep "nuclei-scanner-job-" | awk '{print $1}') || true
if [ ! -z "$PODS" ]; then
    echo "$PODS" | xargs -I {} kubectl delete pod {} --force --grace-period=0 --ignore-not-found
fi

echo "Deleting all nuclei-scanner jobs..."
kubectl delete jobs -l app=nuclei-scanner-job --ignore-not-found --force

echo "Deleting nuclei-scanner job..."
kubectl delete -f "${SCRIPT_DIR}/nuclei-scanner-job.yaml" --ignore-not-found --force

echo "Deleting KEDA ScaledJob..."
kubectl delete -f "${SCRIPT_DIR}/keda-rabbitmq-scaled-job.yaml" --ignore-not-found --force

echo "Waiting for cleanup..."
sleep 5

echo "Reapplying KEDA ScaledJob..."
kubectl apply -f "${SCRIPT_DIR}/keda-rabbitmq-scaled-job.yaml"

echo "Reapplying nuclei-scanner job..."
kubectl apply -f "${SCRIPT_DIR}/nuclei-scanner-job.yaml"

echo "Done!" 