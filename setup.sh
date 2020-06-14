make install 
kubectl get customresourcedefinition.apiextensions.k8s.io/hpatuners.webapp.streamotion.com.au  -o yaml
make run 
# k apply -f config/samples/webapp_v1_hpatuner.yaml 
