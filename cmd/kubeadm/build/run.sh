#!/bin/bash
REGISTRY_SERVER="index.tenxcloud.com"
REGISTRY_USER="tenx_containers"
AGENT_VERSION="v3.0.0"
K8S_VERSION="v1.9.2"
ETCD_VERSION="3.1.11"
CALICO_VERSION="v1.6.3"
ROLE="node"
welcome() {
message="$(cat <<"EOF"
*******************************************************
||    _____                   _                 _    ||
||   |_   _|__ _ __ __  _____| | ___  _   _  __| |   ||
||     | |/ _ \ '_ \\ \/ / __| |/ _ \| | | |/ _` |   ||
||     | |  __/ | | |>  < (__| | (_) | |_| | (_| |   ||
||     |_|\___|_| |_/_/\_\___|_|\___/ \__,_|\__,_|   ||
||                                                   ||
***********************************TenxCloud Enterprise
Welcome to TenxCloud Enterprise Cloud Deployment Engine
EOF
)"
echo "welcome() {"
echo "  cat <<\"EOF\""
echo "$message"
echo "EOF"
echo "}"
}

Usage() {
cat <<EOF
TenxCloud Enterprise Cloud Deployment Engine\n
\n
Command: \n
    [Option] Join [Master] \n
    [OPtion] Init \n
    [OPtion] Uninstall \n
\n
Option:\n
    --registry       \t Registry server, default is index.tenxcloud.com \n
    --address        \t Advertised address of the current machine, if not set, it will get a one automatically\n
    --version        \t Cluster version that will be deployed\n
    --token          \t kubernetes token \n
    --credential     \t credential token to access server\n
    --ha-peer        \t Peer master in HA mode\n
    --role           \t Role of current machine: master, node, loadbalancer
EOF
}



Clean=$(cat <<EOF
  Clean() {
    cp /kubeadm  /tmp/  1>/dev/null 2>/dev/null
    #remove agent
    echo "Cleaning previous agent if existing"
    docker stop agent   1>/dev/null 2>/dev/null
    docker rm -f agent  1>/dev/null 2>/dev/null
    /tmp/kubeadm reset
  }
EOF
)
eval "${Clean}"


PullImage=$(cat <<EOF
  PullImage() {
  echo "Pulling Necessary Images from \${1}"
  if [ \${3} == "master" ]; then
      docker pull \${1}/\${2}/hyperkube-amd64:${K8S_VERSION}
      docker pull \${1}/\${2}/agent:${AGENT_VERSION}
      docker pull  \${1}/\${2}/kubectl-amd64:${K8S_VERSION}
      docker pull  \${1}/\${2}/etcd-amd64:${ETCD_VERSION}
      docker pull  \${1}/\${2}/ctl:${CALICO_VERSION}
  else
      docker pull \${1}/\${2}/hyperkube-amd64:${K8S_VERSION}
      docker pull \${1}/\${2}/agent:${AGENT_VERSION}
      docker pull  \${1}/\${2}/kubectl-amd64:${K8S_VERSION}
  fi
  }
EOF
)


uninstall(){
#copy kubeadm from containers to /tmp
cp /kubeadm  /tmp/  1>/dev/null 2>/dev/null

cat <<EOF
#!/bin/bash
${Clean}
Clean
rm /tmp/kubeadm 2>/dev/null
echo "Uninstall Node Successfully"
EOF
}



#Deploy kubernetes master
Master() {
  #copy kubeadm from containers to /tmp
  cp /kubeadm  /tmp/  1>/dev/null 2>/dev/null
  ADVERTISE_ADDRESSES=""
  ADVERTISE_ADDRESSES_AGENT=""
  if [ -n "${ADDRESS}" ]; then
    ADVERTISE_ADDRESSES="--apiserver-advertise-address ${ADDRESS}"
    ADVERTISE_ADDRESSES_AGENT="--advertise-address=${ADDRESS}"
  fi

  local pod_cidr_parameter=""
  if [ -n "${POD_CIDR}" ]; then
    pod_cidr_parameter="--pod-network-cidr ${POD_CIDR}"
  fi

  local service_cidr_parameter=""
  if [ -n "${SERVICE_CIDR}" ]; then
    service_cidr_parameter="--service-cidr ${SERVICE_CIDR}"
  fi

  local service_dns_domain_parameter=""
  if [ -n "${SERVICE_DNS_DOMAIN}" ]; then
    service_dns_domain_parameter="--service-dns-domain ${SERVICE_DNS_DOMAIN}"
  fi

  local apiserver_cert_extra_sans_parameter=""
  if [ -n "${CERT_EXTRA_SANS}" ]; then
    apiserver_cert_extra_sans_parameter="--apiserver-cert-extra-sans ${CERT_EXTRA_SANS}"
  fi


  #master ha peer
  if [ -n "${HA_PEER}" ]; then
    if [ -z "${K8S_TOKEN}" ]; then
      cat <<EOF
#!/bin/bash
echo "For HA mode, Please set kubernetes token with parameter --token <tokenstring>"
EOF
      exit 1
    fi

    if [ -z "${CA_CERT_HASH}" ]; then
    cat <<EOF
#!/bin/bash
echo "Please set kubernetes root ca cert hash with parameter --ca-cert-hash sha256:<hash>"
EOF
    exit 1
  fi

    cat <<EOF
#!/bin/bash
$(welcome)
welcome
${Clean}
Clean

${PullImage}
PullImage ${REGISTRY_SERVER} ${REGISTRY_USER}  "master"

result=0
/tmp/kubeadm init ${pod_cidr_parameter}  ${service_cidr_parameter} ${service_dns_domain_parameter}  ${apiserver_cert_extra_sans_parameter}  --token ${K8S_TOKEN}  --discovery-token-ca-cert-hash ${CA_CERT_HASH}   ${ADVERTISE_ADDRESSES} --ha-peer ${HA_PEER}:6443   --kubernetes-version ${K8S_VERSION} --image-repository ${REGISTRY_SERVER}/${REGISTRY_USER}
result=\$?
rm -rf $(which kubeadm)
mv /tmp/kubeadm /usr/bin/  >/dev/null

docker run --rm -v /tmp:/tmp --entrypoint cp  ${REGISTRY_SERVER}/${REGISTRY_USER}/kubectl-amd64:${K8S_VERSION} /bin/kubectl /tmp
rm -rf $(which kubectl)
mv /tmp/kubectl /usr/bin/  >/dev/null

docker run --rm -v /tmp:/tmp --entrypoint cp  ${REGISTRY_SERVER}/${REGISTRY_USER}/etcd-amd64:${ETCD_VERSION}  /usr/local/bin/etcdctl /tmp
rm -rf $(which etcdctl)
mv /tmp/etcdctl /usr/bin/  >/dev/null

docker run --rm -v /tmp:/tmp --entrypoint cp  ${REGISTRY_SERVER}/${REGISTRY_USER}/ctl:${CALICO_VERSION} /calicoctl /tmp
rm -rf $(which calicoctl)
mv /tmp/calicoctl /usr/bin/  >/dev/null

docker run --net=host -d --restart=always  -v /tmp:/tmp  -v /etc/hosts:/etc/hosts -v /etc/kubernetes:/etc/kubernetes  -v /etc/resolv.conf:/etc/resolv.conf   --name agent  ${REGISTRY_SERVER}/${REGISTRY_USER}/agent:${AGENT_VERSION}  --role master --etcd-servers=http://127.0.0.1:2379 ${ADVERTISE_ADDRESSES_AGENT} --dns-enable true --ssl-enable=false >/dev/null
result=\$?
if [ \${result} -eq 0  ];then
   echo "TenxCloud Enterprise Cloud was deployed successfully"
else
   echo "TenxCloud Enterprise Cloud was deployed  failed!"
fi

EOF

  exit 0
  fi

  #Normal master mode
  PostToServer=""
  if [ -n "${SERVER_URL}" ] && [ -n "${CREDENTIAL}" ]; then
    PostToServer="--server ${SERVER_URL}  --server-credential ${CREDENTIAL}"
  fi

  local network_plugin_parameter=""
  if [ -n "${NETWORK}" ]; then
    network_plugin_parameter="--network-plugin ${NETWORK}"
  fi

  cat <<EOF
#!/bin/bash
$(welcome)
welcome
${Clean}
Clean

${PullImage}
PullImage ${REGISTRY_SERVER} ${REGISTRY_USER}  "master"
result=0
/tmp/kubeadm init  ${network_plugin_parameter}  ${pod_cidr_parameter} ${service_cidr_parameter} ${service_dns_domain_parameter}  ${apiserver_cert_extra_sans_parameter}   ${ADVERTISE_ADDRESSES} --kubernetes-version ${K8S_VERSION}  --token-ttl 0  ${PostToServer} --image-repository ${REGISTRY_SERVER}/${REGISTRY_USER}
result=\$?
rm -rf $(which kubeadm)
mv /tmp/kubeadm /usr/bin/  >/dev/null

docker run --rm -v /tmp:/tmp --entrypoint cp  ${REGISTRY_SERVER}/${REGISTRY_USER}/kubectl-amd64:${K8S_VERSION} /bin/kubectl /tmp
rm -rf $(which kubectl)
mv /tmp/kubectl /usr/bin/  >/dev/null

docker run --rm -v /tmp:/tmp --entrypoint cp  ${REGISTRY_SERVER}/${REGISTRY_USER}/etcd-amd64:${ETCD_VERSION}  /usr/local/bin/etcdctl /tmp
rm -rf $(which etcdctl)
mv /tmp/etcdctl /usr/bin/  >/dev/null

docker run --rm -v /tmp:/tmp --entrypoint cp  ${REGISTRY_SERVER}/${REGISTRY_USER}/ctl:${CALICO_VERSION} /calicoctl /tmp
rm -rf $(which calicoctl)
mv /tmp/calicoctl /usr/bin/  >/dev/null

docker run --net=host -d --restart=always -v /etc/kubernetes:/etc/kubernetes -v /etc/hosts:/etc/hosts -v /etc/resolv.conf:/etc/resolv.conf --name agent  ${REGISTRY_SERVER}/${REGISTRY_USER}/agent:${AGENT_VERSION} ${ADVERTISE_ADDRESSES_AGENT} --role master --etcd-servers=http://127.0.0.1:2379 --dns-enable true  --ssl-enable=false >/dev/null
result=\$?
if [ \${result} -eq 0  ];then
   echo "TenxCloud Enterprise Cloud was deployed successfully"
else
   echo "TenxCloud Enterprise Cloud was deployed  failed!"
fi
EOF
exit 0
}
Node() {
  #copy kubeadm from containers to /tmp
  cp /kubeadm  /tmp/ 2>/dev/null

  ADVERTISE_ADDRESSES_AGENT=""
  if [ -n "${ADDRESS}" ]; then
    ADVERTISE_ADDRESSES_AGENT="--advertise-address=${ADDRESS}"
  fi

  local pod_cidr_parameter=""
  if [ -n "${POD_CIDR}" ]; then
    pod_cidr_parameter="--pod-network-cidr ${POD_CIDR}"
  fi

  local service_cidr_parameter=""
  if [ -n "${SERVICE_CIDR}" ]; then
    service_cidr_parameter="--service-cidr ${SERVICE_CIDR}"
  fi

  local service_dns_domain_parameter=""
  if [ -n "${SERVICE_DNS_DOMAIN}" ]; then
    service_dns_domain_parameter="--service-dns-domain ${SERVICE_DNS_DOMAIN}"
  fi


  if [ -z "${K8S_TOKEN}" ]; then
    cat <<EOF
#!/bin/bash
echo "Please set kubernetes token with parameter --token <tokenstring>"
EOF
    exit 1
  fi

  ## loadbalancer node
  if [ "$ROLE" = "loadbalancer" ]; then
    cat <<EOF
#!/bin/bash
$(welcome)
welcome
${Clean}
Clean

result=0
echo "Deploying loadbalancer..."
docker run --net=host -d --restart=always -v /tmp:/tmp  -v /etc/hosts:/etc/hosts -v /etc/kubernetes:/etc/kubernetes  -v /etc/resolv.conf:/etc/resolv.conf --name agent ${REGISTRY_SERVER}/${REGISTRY_USER}/agent:${AGENT_VERSION}  ${ADVERTISE_ADDRESSES_AGENT} --role loadbalancer --etcd-servers=https://${MASTER}:2379 --accesstoken=${K8S_TOKEN} --cert-servers=${MASTER} --dns-enable false --ssl-enable=true >/dev/null
result=\$?
if [ \${result} -eq 0  ];then
   echo "TenxCloud Enterprise Cloud was deployed successfully"
else
   echo "TenxCloud Enterprise Cloud was deployed  failed!"
fi
EOF

   return
  fi


  ## Normal slave node
    if [ -z "${CA_CERT_HASH}" ]; then
    cat <<EOF
#!/bin/bash
echo "Please set kubernetes root ca cert hash with parameter --ca-cert-hash sha256:<hash>"
EOF
    exit 1
  fi

  cat <<EOF
#!/bin/bash
$(welcome)
welcome
${Clean}
Clean

${PullImage}
PullImage ${REGISTRY_SERVER} ${REGISTRY_USER}  "node"
result=0
/tmp/kubeadm join --token ${K8S_TOKEN} ${MASTER}:6443  --discovery-token-ca-cert-hash ${CA_CERT_HASH} --image-repository ${REGISTRY_SERVER}/${REGISTRY_USER}  --kubernetes-version ${K8S_VERSION}  ${pod_cidr_parameter} ${service_cidr_parameter} ${service_dns_domain_parameter}
result=\$?
rm /tmp/kubeadm 2>/dev/null
docker run --net=host -d --restart=always  -v /tmp:/tmp -v /etc/hosts:/etc/hosts -v /etc/kubernetes:/etc/kubernetes  -v /etc/resolv.conf:/etc/resolv.conf --name agent  ${REGISTRY_SERVER}/${REGISTRY_USER}/agent:${AGENT_VERSION} ${ADVERTISE_ADDRESSES_AGENT} --role node --etcd-servers=https://${MASTER}:2379 --dns-enable true --cert-dir /etc/kubernetes/pki  --accesstoken=${K8S_TOKEN} --cert-servers=${MASTER}  --ssl-enable=true >/dev/null
result=\$?
if [ \${result} -eq 0  ];then
   echo "TenxCloud Enterprise Cloud was deployed successfully"
else
   echo "TenxCloud Enterprise Cloud was deployed  failed!"
fi
EOF
exit 0
}



# if there's no valid parameter, it will show help message
if [ "$#" -le 0 ] ; then
  echo -e $(Usage)

  exit 0
fi
#welcome message
#welcome

#dispatch different parameters
 while(( $# > 0 ))
    do
        case "$1" in
          "--registry" )
              REGISTRY_SERVER="$2"
              shift 2;;
          "--address" )
              ADDRESS="$2"
              shift 2;;
          "--version" )
              K8S_VERSION="$2"
              shift 2 ;;
          "--token" )
              K8S_TOKEN="$2"
              shift 2 ;;
          "--ca-cert-hash" )
              CA_CERT_HASH="$2"
              shift 2 ;;
          "--credential" )
              CREDENTIAL="$2"
              shift 2 ;;
          "--ha-peer" )
              HA_PEER="$2"
              shift 2 ;;
          "--role" )
              ROLE="$2"
              shift 2 ;;
          "Join" )
              if [ "$#" -le 1 ]; then
                echo "Please Enter Master Address and Auth Token"
                exit
              fi
              MASTER="$2"
              Node
              exit 0
              shift 3;;
          "Init" )
              if [ "$#" -gt 1 ]; then
                SERVER_URL="$2"
              fi
              Master
              exit 0
              shift 2;;
          "Uninstall" )
              uninstall
              exit 0
              shift 1;;
          "welcome" )
              exit 0
              shift 1;;
            * )
                echo "Invalid parameter: $1"
                echo -e $(Usage)
                exit 1
        esac
    done # end while