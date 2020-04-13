# kind delete cluster
# kind create cluster
# kind create cluster --config kind-cluster-1.14.10.yaml
# kind get clusters 
# kubectl get po -A 
# kind load docker-image k8s.gcr.io/metrics-server-amd64:v0.3.6
# kubectl apply -f phpload/metrics-server.yaml 

make install 
kubectl get customresourcedefinition.apiextensions.k8s.io/hpatuners.webapp.streamotion.com.au  -o yaml
make run 