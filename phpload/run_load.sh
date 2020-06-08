kubectl run --generator=run-pod/v1 -it --rm load-generator --image=busybox /bin/sh

#while true; do wget -q -O- http://php-apache.default.svc.cluster.local; done