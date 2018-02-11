/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-02-05  @author weiwei@tenxcloud.com
 */

/**
 *  Calico Version v2.6.7
 *  https://docs.projectcalico.org/v2.6/releases#v2.6.7
 *  This manifest includes the following component versions:
 *    quay.io/calico/node:v2.6.7
 *    quay.io/calico/cni:v1.11.2               # https://github.com/projectcalico/cni-plugin/blob/master/k8s-install/scripts/install-cni.sh
 *    quay.io/calico/ctl:v1.6.3
 *    quay.io/calico/kube-controllers:v1.0.3
 *    quay.io/calico/routereflector:v0.4.2
 *
 *    quay.io/coreos/flannel:v0.10.0
 *    quay.io/coreos/flannel:v0.10.0-amd64
 *
 *  https://github.com/kubernetes/contrib/tree/master/election
 *    gcr.io/google-containers/leader-elector:0.5
 */
package canal

const (

	//This ConfigMap is used to configure a self-hosted Canal Installation.
	ConfigMap =`
kind: ConfigMap
apiVersion: v1
metadata:
  name: canal-config
  namespace: kube-system
data:
  etcd_endpoints: "https://kubernetes.default.svc.cluster.local:2379"
  etcd_ca: "/etc/kubernetes/pki/ca.crt"
  etcd_cert: "/etc/kubernetes/pki/client.crt"
  etcd_key: "/etc/kubernetes/pki/client.key"
  canal_iface: ""
  masquerade: "true"
  cni_network_config: |-
    {
        "name": "canal",
        "cniVersion": "0.3.1",
        "plugins": [
            {
                "type": "flannel",
                "delegate": {
                    "type": "calico",
                    "etcd_endpoints": "__ETCD_ENDPOINTS__",
                    "etcd_key_file": "__ETCD_KEY_FILE__",
                    "etcd_cert_file": "__ETCD_CERT_FILE__",
                    "etcd_ca_cert_file": "__ETCD_CA_CERT_FILE__",
                    "log_level": "__LOG_LEVEL__",
                    "policy": {
                        "type": "k8s"
                    },
                    "kubernetes": {
                        "kubeconfig": "/etc/kubernetes/kubelet.conf"
                    }
                }
            },
            {
                "type": "portmap",
                "capabilities": {
                   "portMappings": true
                 },
                "snat": true
            }
        ]
    }`





    // This manifest installs the per-node agents, as well as the CNI plugins and network config on
    // each master and worker node in a Kubernetes cluster.
	DaemonSet =`
kind: DaemonSet
apiVersion: apps/v1beta2
metadata:
  name: canal-node
  namespace: kube-system
  labels:
    k8s-app: canal-node
spec:
  selector:
    matchLabels:
      k8s-app: canal-node
  template:
    metadata:
      annotations:
        scheduler.alpha.kubernetes.io/critical-pod: ''
      labels:
        k8s-app: canal-node
    spec:
      tolerations:
        - effect: NoSchedule
          operator: Exists
        - key: CriticalAddonsOnly
          operator: Exists
        - effect: NoExecute
          operator: Exists
      hostNetwork: true
      serviceAccountName: canal
      terminationGracePeriodSeconds: 0
      containers:
        - name: flannel
          image: {{ .ImageRepository }}/flannel:v0.10.0
          command:
          - /opt/bin/flanneld
          args:
          - --etcd-endpoints=https://kubernetes.default.svc.cluster.local:2379
          - --etcd-cafile=/etc/kubernetes/pki/ca.crt
          - --etcd-certfile=/etc/kubernetes/pki/client.crt
          - --etcd-keyfile=/etc/kubernetes/pki/client.key
          - --ip-masq
          env:
            - name: FLANNELD_ETCD_ENDPOINTS
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_endpoints
            - name: ETCD_CA_CERT_FILE
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_ca
            - name: ETCD_KEY_FILE
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_key
            - name: ETCD_CERT_FILE
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_cert
            - name: FLANNELD_ETCD_CAFILE
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_ca
            - name: FLANNELD_ETCD_KEYFILE
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_key
            - name: FLANNELD_ETCD_CERTFILE
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_cert
            - name: FLANNELD_IFACE
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: canal_iface
            - name: FLANNELD_IP_MASQ
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: masquerade
            - name: FLANNELD_SUBNET_FILE
              value: "/run/flannel/subnet.env"
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
          securityContext:
            privileged: true
          volumeMounts:
            - mountPath: /etc/resolv.conf
              name: etc-resolv-conf
              readOnly: true
            - mountPath: /run/flannel
              name: run-flannel
            - mountPath: /etc/kubernetes/pki
              name: k8s-certs
              readOnly: true
        - name: calico-node
          image: {{ .ImageRepository }}/node:v2.6.7
          env:
            - name: ETCD_ENDPOINTS
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_endpoints
            - name: ETCD_CA_CERT_FILE
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_ca
            - name: ETCD_KEY_FILE
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_key
            - name: ETCD_CERT_FILE
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_cert
            - name: CALICO_NETWORKING_BACKEND
              value: "none"
            - name: CLUSTER_TYPE
              value: "k8s,canal"
            - name: CALICO_DISABLE_FILE_LOGGING
              value: "true"
            - name: CALICO_K8S_NODE_REF
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: NO_DEFAULT_POOLS
              value: "true"
            - name: FELIX_IPV6SUPPORT
              value: "false"
            - name: FELIX_HEALTHENABLED
              value: "true"
          securityContext:
            privileged: true
          resources:
            requests:
              cpu: 250m
          livenessProbe:
            httpGet:
              path: /liveness
              port: 9099
            periodSeconds: 10
            initialDelaySeconds: 10
            failureThreshold: 6
          readinessProbe:
            httpGet:
              path: /readiness
              port: 9099
            periodSeconds: 10
          volumeMounts:
            - mountPath: /lib/modules
              name: lib-modules
              readOnly: true
            - mountPath: /var/run/calico
              name: var-run-calico
              readOnly: false
            - mountPath: /etc/kubernetes/pki
              name: k8s-certs
              readOnly: true
            - mountPath: /etc/resolv.conf
              name: etc-resolv-conf
              readOnly: true
        - name: install-calico-cni
          image: {{ .ImageRepository }}/cni:v1.11.2
          imagePullPolicy: IfNotPresent
          command: ["/install-cni.sh"]
          env:
            - name: CNI_CONF_NAME
              value: "10-canal.conflist"
            - name: ETCD_ENDPOINTS
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_endpoints
            - name: CNI_CONF_ETCD_CA
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_ca
            - name: CNI_CONF_ETCD_KEY
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_key
            - name: CNI_CONF_ETCD_CERT
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_cert
            - name: CNI_NETWORK_CONFIG
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: cni_network_config
          volumeMounts:
            - mountPath: /host/opt/cni/bin
              name: cni-bin-dir
            - mountPath: /host/etc/cni/net.d
              name: cni-net-dir
      volumes:
        - name: lib-modules
          hostPath:
            path: /lib/modules
            type: DirectoryOrCreate
        - name: var-run-calico
          hostPath:
            path: /var/run/calico
            type: DirectoryOrCreate
        - name: cni-bin-dir
          hostPath:
            path: /opt/cni/bin
            type: DirectoryOrCreate
        - name: cni-net-dir
          hostPath:
            path: /etc/cni/net.d
            type: DirectoryOrCreate
        - name: run-flannel
          hostPath:
            path: /run/flannel
            type: DirectoryOrCreate
        - name: etc-resolv-conf
          hostPath:
            path: /etc/resolv.conf
            type: FileOrCreate
        - name: k8s-certs
          hostPath:
            path: /etc/kubernetes/pki
            type: DirectoryOrCreate`

	// This manifest installs the calico/kube-controllers container on each master.
	// See https://github.com/projectcalico/kube-controllers
	//     https://github.com/kubernetes/contrib/tree/master/election
	KubeController = `
apiVersion: apps/v1beta2
kind: Deployment
metadata:
  name: kube-controller
  namespace: kube-system
  labels:
    k8s-app: kube-controller
spec:
  replicas: 1
  strategy:
    type: Recreate
  selector:
    matchLabels:
      k8s-app: kube-controller
  template:
    metadata:
      name: kube-controller
      namespace: kube-system
      labels:
        k8s-app: kube-controller
      annotations:
        scheduler.alpha.kubernetes.io/critical-pod: ''
    spec:
      hostNetwork: true
      nodeSelector:
        node-role.kubernetes.io/master: ""
      tolerations:
      - key: node.cloudprovider.kubernetes.io/uninitialized
        value: "true"
        effect: NoSchedule
      - key: node-role.kubernetes.io/master
        effect: NoSchedule
      - key: CriticalAddonsOnly
        operator: Exists
      serviceAccountName: kube-controllers
      containers:
      - name: kube-controller
        image: {{ .ImageRepository }}/kube-controllers:v1.0.3
        imagePullPolicy: IfNotPresent
        env:
          - name: ETCD_ENDPOINTS
            valueFrom:
              configMapKeyRef:
                name: canal-config
                key: etcd_endpoints
          - name: ETCD_CA_CERT_FILE
            valueFrom:
              configMapKeyRef:
                name: canal-config
                key: etcd_ca
          - name: ETCD_KEY_FILE
            valueFrom:
              configMapKeyRef:
                name: canal-config
                key: etcd_key
          - name: ETCD_CERT_FILE
            valueFrom:
              configMapKeyRef:
                name: canal-config
                key: etcd_cert
          - name: KUBECONFIG
            value: /etc/kubernetes/kubelet.conf
          - name: ENABLED_CONTROLLERS
            value: policy,profile,workloadendpoint,node
        volumeMounts:
          - mountPath: /etc/resolv.conf
            name: etc-resolv-conf
            readOnly: true
          - mountPath: /etc/kubernetes
            name: k8s-certs
            readOnly: true
      volumes:
        - name: etc-resolv-conf
          hostPath:
            path: /etc/resolv.conf
            type: FileOrCreate
        - name: k8s-certs
          hostPath:
            path: /etc/kubernetes
            type: DirectoryOrCreate`


    //This manifest deploys a Job which performs one time configuration of Canal.
    Job = `
apiVersion: batch/v1
kind: Job
metadata:
  name: configure-canal
  namespace: kube-system
  labels:
    k8s-app: canal
spec:
  template:
    metadata:
      name: configure-canal
    spec:
      hostNetwork: true
      restartPolicy: OnFailure
      tolerations:
      - key: node-role.kubernetes.io/master
        effect: NoSchedule
      containers:
        - name: configure-canal
          image: {{ .Image }}
          command:
          - "etcdctl"
          - "--endpoints=https://kubernetes.default.svc.cluster.local:2379"
          - "--cert-file=/etc/kubernetes/pki/client.crt"
          - "--key-file=/etc/kubernetes/pki/client.key"
          - "--ca-file=/etc/kubernetes/pki/ca.crt"
          - "--no-sync"
          - "set"
          - "/coreos.com/network/config"
          - '{ "Network": "{{ .PodSubnet }}", "Backend": {"Type": "vxlan"} }'
          volumeMounts:
            - mountPath: /etc/kubernetes/pki
              name: pki
              readOnly: true
            - name: etc-resolv-conf
              mountPath: /etc/resolv.conf
              readOnly: true
      volumes:
        - name: pki
          hostPath:
            path: /etc/kubernetes/pki
            type: DirectoryOrCreate
        - name: etc-resolv-conf
          hostPath:
            path: /etc/resolv.conf
            type: FileOrCreate`



	ClusterRole = `
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: system:canal
rules:
  - apiGroups: [""]
    resources:
      - pods
      - nodes
    verbs:
      - get`

    CalicoClusterRole = `
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: system:calico
rules:
  - apiGroups: [""]
    resources:
      - namespaces
    verbs:
      - get
      - list
      - watch
  - apiGroups: [""]
    resources:
      - pods/status
    verbs:
      - update
  - apiGroups: [""]
    resources:
      - pods
    verbs:
      - get
      - list
      - watch
  - apiGroups: [""]
    resources:
      - nodes
    verbs:
      - get
      - list
      - update
      - watch
  - apiGroups: ["extensions"]
    resources:
      - networkpolicies
    verbs:
      - get
      - list
      - watch
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - globalfelixconfigs
      - bgppeers
      - globalbgpconfigs
      - ippools
      - globalnetworkpolicies
    verbs:
      - create
      - get
      - list
      - update
      - watch`

    FlannelClusterRole = `
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: system:flannel
rules:
  - apiGroups:
      - ""
    resources:
      - pods
    verbs:
      - get
  - apiGroups:
      - ""
    resources:
      - nodes
    verbs:
      - list
      - watch
  - apiGroups:
      - ""
    resources:
      - nodes/status
    verbs:
      - patch`


	ServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: canal
  namespace: kube-system`

	ClusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: system:canal
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:canal
subjects:
- kind: ServiceAccount
  name: canal
  namespace: kube-system`

    FlannelClusterRoleBinding = `
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: system:canal-flannel
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:flannel
subjects:
- kind: ServiceAccount
  name: canal
  namespace: kube-system`

    CalicoClusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: system:canal-calico
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:calico
subjects:
- kind: ServiceAccount
  name: canal
  namespace: kube-system`


	// for calico/kube-controllers
	KubeControllersClusterRole = `
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: system:kube-controllers
rules:
  - apiGroups:
    - ""
    - extensions
    resources:
      - pods
      - namespaces
      - networkpolicies
      - nodes
    verbs:
      - watch
      - list`

	KubeControllersServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kube-controllers
  namespace: kube-system`

	KubeControllersClusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: system:kube-controllers
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:kube-controllers
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: Group
  name: system:masters
- apiGroup: rbac.authorization.k8s.io
  kind: Group
  name: system:nodes
- kind: ServiceAccount
  name: kube-controllers
  namespace: kube-system`


)