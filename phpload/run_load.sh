
echo 'Run In Container: while true; do wget -q -O- http://php-apache.default.svc.cluster.local; done'
kubectl run --generator=run-pod/v1 -it --rm load-generator --image=busybox /bin/sh
