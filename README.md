Experimental project to learn Kubernetes Controller Framework with the endgoal to tune the HPA 
based on more customized parameters like `time of day` (evenings are more aggressively scaled than days) etc. 

Just a POC for now.

The project scaffold is generated from : https://book.kubebuilder.io/ 

Also ideas borrowed from:
https://github.com/postmates/configurable-hpa 

(instead of updating the deployment, the plan is to re-use the logic of the HPA but tune the HPA for min Pods as we go)



#NOTES as I go:

start cluster:

```
> cd kind 
> build-cluster.sh
```

deploy the k8s deployment/services/hpa etc:
```
> cd phpload
> ./init.sh
> ./run_load.sh
```

```
./setup.sh

#new terminal
kubectl apply -f config/samples/
```

