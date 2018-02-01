/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-01-22  @author weiwei@tenxcloud.com
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
  *  https://github.com/kubernetes/contrib/tree/master/election
  *    gcr.io/google-containers/leader-elector:0.5
  */

package calico



const (

    //This ConfigMap is used to configure a self-hosted Calico installation.
	NodeConfigMap = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: cni-config
  namespace: kube-system
data:
  cni_network_config: |-
    {
        "name": "k8s-pod-network",
        "cniVersion": "0.3.1",
        "type": "calico",
        "etcd_endpoints": "__ETCD_ENDPOINTS__",
        "etcd_key_file": "__ETCD_KEY_FILE__",
        "etcd_cert_file": "__ETCD_CERT_FILE__",
        "etcd_ca_cert_file ": "__ETCD_CA_CERT_FILE__",
        "log_level": "info",
        "mtu": 1500,
        "ipam": {
            "type": "calico-ipam"
        },
        "policy": {
            "type": "k8s"
        },
        "kubernetes": {
            "kubeconfig": "/etc/kubernetes/kubelet.conf"
        }
    }`

    // This manifest installs the calico/node container,
    // as well as the Calico CNI plugins and network config on
    // each master and worker node in a Kubernetes cluster.
	Node = `
apiVersion: apps/v1beta2
kind: DaemonSet
metadata:
  name: calico-node
  namespace: kube-system
  labels:
    k8s-app: calico-node
spec:
  selector:
    matchLabels:
      k8s-app: calico-node
  template:
    metadata:
      labels:
        k8s-app: calico-node
        component: calico
        tier: control-plane
      annotations:
        scheduler.alpha.kubernetes.io/critical-pod: ''
    spec:
      hostNetwork: true
      tolerations:
      - key: node.cloudprovider.kubernetes.io/uninitialized
        value: "true"
        effect: NoSchedule
      - key: node-role.kubernetes.io/master
        effect: NoSchedule
      - key: CriticalAddonsOnly
        operator: Exists
      serviceAccountName: calico-cni-plugin
      terminationGracePeriodSeconds: 0
      containers:
        - name: calico-node
          image: {{ .ImageRepository }}/node:v2.6.7
          env:
            - name: ETCD_ENDPOINTS
              value: https://kubernetes.default.svc.cluster.local:2379
            - name: CALICO_NETWORKING_BACKEND
              value: bird
            - name: CLUSTER_TYPE
              value: "kubeadm,bgp"
            - name: CALICO_K8S_NODE_REF
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: CALICO_DISABLE_FILE_LOGGING
              value: "true"
            - name: FELIX_DEFAULTENDPOINTTOHOSTACTION
              value: "ACCEPT"
            - name: NO_DEFAULT_POOLS
              value: "true"
            - name: FELIX_IPV6SUPPORT
              value: "false"
            - name: FELIX_IPINIPMTU
              value: "1440"
            - name: FELIX_LOGSEVERITYSCREEN
              value: "info"
            - name: IP
              value: ""
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
            - mountPath: /etc/kubernetes/
              name: k8s-certs
              readOnly: true
            - mountPath: /etc/resolv.conf
              name: etc-resolv-conf
              readOnly: true
        - name: install-cni
          image: {{ .ImageRepository }}/cni:v1.11.2
          command: ["/install-cni.sh"]
          env:
            - name: ETCD_ENDPOINTS
              value: https://kubernetes.default.svc.cluster.local:2379
            - name: CNI_CONF_ETCD_CERT
              value: /etc/kubernetes/pki/client.crt
            - name: CNI_CONF_ETCD_KEY
              value: /etc/kubernetes/pki/client.key
            - name: CNI_CONF_ETCD_CA
              value: /etc/kubernetes/pki/ca.crt
            - name: CNI_NETWORK_CONFIG
              valueFrom:
                configMapKeyRef:
                  name: cni-config
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
        - name: var-run-calico
          hostPath:
            path: /var/run/calico
        - name: cni-bin-dir
          hostPath:
            path: /opt/cni/bin
        - name: cni-net-dir
          hostPath:
            path: /etc/cni/net.d
        - name: k8s-certs
          hostPath:
            path: /etc/kubernetes
        - name: etc-resolv-conf
          hostPath:
            path: /etc/resolv.conf`

    // This manifest installs the calico/kube-controllers container on each master.
    // See https://github.com/projectcalico/kube-controllers
    //     https://github.com/kubernetes/contrib/tree/master/election
    KubeController = `
apiVersion: apps/v1beta2
kind: DaemonSet
metadata:
  name: kube-policy-controller
  namespace: kube-system
  labels:
    k8s-app: kube-policy-controller
spec:
  selector:
    matchLabels:
      k8s-app: kube-policy-controller
  template:
    metadata:
      name: kube-policy-controller
      namespace: kube-system
      labels:
        k8s-app: kube-policy-controller
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
      serviceAccountName: calico-kube-controllers
      containers:
      - name: kube-policy-controller
        image: {{ .ImageRepository }}/kube-controllers:v1.0.3
        imagePullPolicy: IfNotPresent
        env:
          - name: ETCD_ENDPOINTS
            value: http://127.0.0.1:2379
          - name: K8S_API
            value: https://kubernetes.default.svc.cluster.local:6443
          - name: LEADER_ELECTION
            value: "true"
          - name: ENABLED_CONTROLLERS
            value: policy,profile,workloadendpoint,node
      - name: leader-elector
        image: {{ .ImageRepository }}/leader-elector:0.5
        imagePullPolicy: IfNotPresent
        args:
        - --election=kube-policy-election
        - --election-namespace=kube-system
        - --http=127.0.0.1:4040
        securityContext:
          privileged: true`


	CtlConfigMap = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: ippool
  namespace: kube-system
data:
  ippool.yaml: |-
    apiVersion: v1
    kind: ipPool
    metadata:
      cidr: {{ .PodSubnet }}
    spec:
      nat-outgoing: true
`

	CtlJob = `
apiVersion: batch/v1
kind: Job
metadata:
  labels:
    k8s-app: calico
  name: configure-calico
  namespace: kube-system
spec:
  completions: 1
  parallelism: 1
  template:
    metadata:
      labels:
        k8s-app: calico
    spec:
      containers:
      - args:
        - apply
        - -f
        - /etc/config/calico/ippool.yaml
        env:
        - name: ETCD_ENDPOINTS
          value: http://127.0.0.1:2379
        image: {{ .ImageRepository }}/ctl:v1.6.3
        imagePullPolicy: IfNotPresent
        name: configure-calico
        volumeMounts:
        - mountPath: /etc/config
          name: config-volume
      hostNetwork: true
      nodeSelector:
        node-role.kubernetes.io/master: ""
      tolerations:
      - key: node-role.kubernetes.io/master
        effect: NoSchedule
      restartPolicy: OnFailure
      volumes:
      - configMap:
          defaultMode: 420
          items:
          - key: ippool.yaml
            path: calico/ippool.yaml
          name: ippool
        name: config-volume
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: {{ .LabelNodeRoleMaster }}
                operator: Exists
`
	// for calico/node
    CalicoClusterRole = `
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: calico-cni-plugin
rules:
  - apiGroups: [""]
    resources:
      - pods
      - nodes
    verbs:
      - get`

    CalicoServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: calico-cni-plugin
  namespace: kube-system`

	CalicoClusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: calico-cni-plugin
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: calico-cni-plugin
subjects:
- kind: ServiceAccount
  name: calico-cni-plugin
  namespace: kube-system`

    // for calico/kube-controllers
	CalicoControllersClusterRole = `
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: calico-kube-controllers
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

	CalicoControllersServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: calico-kube-controllers
  namespace: kube-system`

	CalicoControllersClusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: calico-kube-controllers
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: calico-kube-controllers
subjects:
- kind: ServiceAccount
  name: calico-kube-controllers
  namespace: kube-system`

)
