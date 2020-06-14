#streamotion-platform-ops-scaling-decision-service
# Context

Experimental project to learn Kubernetes Controller Framework with the endgoal to tune the HPA 
based on more customized parameters like `time of day` (evenings are more aggressively scaled than days) etc. 

Just a POC for now.

The project scaffold is generated from : https://book.kubebuilder.io/ 

Also ideas borrowed from:
https://github.com/postmates/configurable-hpa 

This will leverage https://github.com/fsa-streamotion/streamotion-platform-ops-scaling-decision-service to actually make the decision to scale (writing code in GO is too much pain!)



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
External system dependencies (e.g. DB, Kafka) or service providers that it requires at runtime.

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
Bedrock: https://foxsportsau.atlassian.net/wiki/spaces/DEV/pages/734004070/Java+Bedrocks


(instead of updating the deployment, the plan is to re-use the logic of the HPA but tune the HPA for min Pods as we go)
