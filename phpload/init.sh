docker build -t phpload:1.1.1 . 
kind load docker-image  phpload:1.1.1

kubectl delete -f php-apache-application.yaml

kubectl apply -f php-apache-application.yaml

kubectl apply -f hpa.yaml
