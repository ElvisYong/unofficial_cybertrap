# Setting up EKS Cluster for Cybertrap Backend

This README provides instructions on how to set up an Amazon EKS (Elastic Kubernetes Service) cluster using eksctl for the Cybertrap backend.

## Prerequisites

1. Install eksctl: https://eksctl.io/installation/
2. Install AWS CLI and configure it with your credentials
3. Install kubectl: https://kubernetes.io/docs/tasks/tools/

## Uploading image to ECR

1. Login to ECR

```
aws ecr get-login-password --region ap-southeast-1 | docker login --username AWS --password-stdin 897729130899.dkr.ecr.ap-southeast-1.amazonaws.com
```

2. Build the images
```
docker-compose build
```

3. Tag the domains api image

```
docker tag backend-domains_api 897729130899.dkr.ecr.ap-southeast-1.amazonaws.com/cybertrap-backend:backend-api-v0.1.0 
```

4. Push the image to ECR

```
docker push 897729130899.dkr.ecr.ap-southeast-1.amazonaws.com/cybertrap-backend:backend-api-v0.1.0
```


## Creating the EKS Cluster

1. Create a basic EKS cluster:
```
eksctl create cluster \
--name cybertrap-cluster \
--region ap-southeast-1 \
```

2. update the aws eks update-kubeconfig

```
aws eks update-kubeconfig --name cybertrap-cluster --region ap-southeast-1
```

3. Apply the node group

```
eksctl apply -f k8s/nodes/arm64-nodegroup.yaml 
```

4. Check the nodes in the cluster

```
kubectl get nodes
```

## EKS Autoscaling group
https://harsh05.medium.com/mastering-autoscaling-in-amazon-eks-scaling-your-kubernetes-workloads-dynamically-098c8e5f9902

## Install Keda
https://keda.sh/docs/2.15/deploy/

## Sealing the secrets

Guide to checkout:
https://dev.to/stack-labs/store-your-kubernetes-secrets-in-git-thanks-to-kubeseal-hello-sealedsecret-2i6h


4. Apply the deployment from k8s

```
kubectl apply -f domains-api-deployment.yaml
```

5. Check the pods in the cluster

```
kubectl get pods
```

## K8s
Apply in the following order:
1. asg
2. nodes
4. sealed-secrets
3. deployments