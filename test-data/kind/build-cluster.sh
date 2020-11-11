export KUBECONFIG=~/.kube/config
kind delete cluster
kind create cluster --config kind-cluster-1.14.10.yaml
kind get clusters 
kubectl get po -A 
docker pull bitnami/metrics-server:0.3.7
kind load docker-image bitnami/metrics-server:0.3.7
kubectl apply -f metrics-server.yaml 
