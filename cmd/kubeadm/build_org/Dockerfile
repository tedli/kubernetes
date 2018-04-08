FROM index.tenxcloud.com/google_containers/alpine-with-bash:1.0

MAINTAINER weiwei   weiwei@tenxcloud.com

ADD ./kubeadm /kubeadm

COPY run.sh /
RUN chmod -R 755 run.sh
VOLUME ["/tmp","/tmp"]

ENTRYPOINT ["/bin/bash", "/run.sh"]