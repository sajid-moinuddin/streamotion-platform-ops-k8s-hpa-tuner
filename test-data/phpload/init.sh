docker build -t streamotion/phpload:1.1.1 .
kind load --name hpa-tuner-controller  docker-image  streamotion/phpload:1.1.1

kubectl delete -f php-apache-application.yaml

kubectl apply -f php-apache-application.yaml
