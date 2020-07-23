docker build -t sajid2045/phpload:1.1.1 . 
kind load --name hpa-tuner-controller  docker-image  sajid2045/phpload:1.1.1

kubectl delete -f php-apache-application.yaml

kubectl apply -f php-apache-application.yaml
