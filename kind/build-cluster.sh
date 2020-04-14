kind delete cluster
kind create cluster --config kind-cluster-1.14.10.yaml
kind get clusters 
kubectl get po -A 
docker pull k8s.gcr.io/metrics-server-amd64:v0.3.6
kind load docker-image k8s.gcr.io/metrics-server-amd64:v0.3.6
kubectl apply -f metrics-server.yaml 
