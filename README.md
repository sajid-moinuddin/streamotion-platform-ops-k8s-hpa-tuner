#streamotion-platform-ops-scaling-decision-service
# Context

Experimental project to learn Kubernetes Controller Framework with the endgoal to tune the HPA 
based on more customized parameters like `time of day` (evenings are more aggressively scaled than days) etc. 

Just a POC for now.

The project scaffold is generated from : https://book.kubebuilder.io/ 

NOTE: if you are working on this project, watch this video a few times: https://www.youtube.com/watch?v=wMqzAOp15wo&t=1227s 

Also ideas borrowed from:
https://github.com/postmates/configurable-hpa 

This will leverage https://github.com/fsa-streamotion/streamotion-platform-ops-scaling-decision-service to actually make the decision to scale (writing code in GO is too much pain!)

Jenkins Build: http://jenkins.web.dev.cluster.foxsports-gitops-prod.com.au/job/fsa-streamotion/job/streamotion-platform-ops-k8s-hpa-tuner/job/master/ 

# How to start
How to start in command line and in IDE.

1. Start Your local k8s server (via kind)

```
    cd kind
    ./build-cluster.sh
```

2. Deploy some sample k8s resource


```
    cd phpload
    ./init.sh #this builds a sample docker image and registers it with your kind cluster along with a deployment and a horizontal-pod-autoscaler that you can now test
```

3. Run your controller
```
    cd $PROJECT_DIR
    ./setup.sh
```

4. Deploy a sample HpaTuner Resource

```
    kubectl apply -f config/samples/webapp_v1_hpatuner.yaml
```

5. Generate some load:
```
    cd phpload
    ./run_load.sh 
```

TBD: GO setup in visual-studio-code details / links: What IDE configurations required (e.g. AWS credentials, Maven settings.xml, etc.)?

# External dependencies

kind
make 

```
https://github.com/fsa-streamotion/streamotion-platform-ops-scaling-decision-service.git
```


# Inbound interfaces
What input interfaces it exposes (event or API). What is their business importance?
What do they do? What are specific business logics they handle and how?
Explain or point to confluence.

# Outbound calls
Important downstream/3rd party systems that it interfaces with. How? Why?
Explain or point to confluence.

# Scenarios
Important "business" scenarios that this application handles? how? Explain or point to confluence.

# Business logic
What important business logic, workflows or calculations this application does? Explain or point to confluence.

# References

    best reference project (clean + functional tests): https://github.com/microsoft/k8s-cronjob-prescaler.git
    https://itnext.io/testing-kubernetes-operators-with-ginkgo-gomega-and-the-operator-runtime-6ad4c2492379
    GO111MODULE=off go get -u github.com/golangci/golangci-lint/cmd/golangci-lint

(instead of updating the deployment, the plan is to re-use the logic of the HPA but tune the HPA for min Pods as we go)
