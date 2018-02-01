/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-01-24  @author weiwei@tenxcloud.com
 */
package flannel


/**
 *  quay.io/coreos/flannel:v0.10.0-amd64
 *  https://github.com/coreos/flannel/blob/master/Documentation/configuration.md
 *  https://github.com/coreos/flannel/blob/master/Documentation/backends.md
 *  https://github.com/coreos/flannel/blob/master/Documentation/kube-flannel.yml
 */

const (

	ConfigMap = `
kind: ConfigMap
apiVersion: v1
metadata:
  name: flannel
  namespace: kube-system
  labels:
    tier: node
    app: flannel
data:
  cni-conf.json: |
    {
      "name": "cbr0",
      "cniVersion":"0.3.1",
      "plugins": [
        {
          "type": "flannel",
          "delegate": {
            "isDefaultGateway": true
          }
        },
        {
          "type": "portmap",
          "capabilities": {
            "portMappings": true
          }
        }
      ]
    }
  net-conf.json: |
    {
      "Network": {{ .PodSubnet }},
      "Backend": {
        "Type": {{ .Backend }}
      }
    }`

    Version = "v0.10.0"

	DaemonSet = `
apiVersion: apps/v1beta2
kind: DaemonSet
metadata:
  name: flannel
  namespace: kube-system
  labels:
    tier: node
    app: flannel
spec:
  selector:
    matchLabels:
      tier: node
	  app: flannel
  template:
    metadata:
      labels:
        tier: node
        app: flannel
    spec:
      hostNetwork: true
      nodeSelector:
        beta.kubernetes.io/arch: amd64
      tolerations:
      - key: node-role.kubernetes.io/master
        operator: Exists
        effect: NoSchedule
      serviceAccountName: flannel
      initContainers:
      - name: install-cni
        image: {{ .ImageRepository }}/flannel:{{ .Version }}-{{ .Arch }}
        command:
        - cp
        args:
        - -f
        - /etc/kube-flannel/cni-conf.json
        - /etc/cni/net.d/10-flannel.conflist
        volumeMounts:
        - name: cni
          mountPath: /etc/cni/net.d
        - name: config
          mountPath: /etc/kube-flannel/
      containers:
      - name: flannel
        image: quay.io/coreos/flannel:v0.10.0-amd64
        command:
        - /opt/bin/flanneld
        args:
        - --ip-masq
        - --kube-subnet-mgr
        resources:
          requests:
            cpu: "100m"
            memory: "50Mi"
          limits:
            cpu: "100m"
            memory: "50Mi"
        securityContext:
          privileged: true
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        volumeMounts:
        - name: run
          mountPath: /run
        - name: config
          mountPath: /etc/kube-flannel/
      volumes:
        - name: run
          hostPath:
            path: /run
        - name: cni
          hostPath:
            path: /etc/cni/net.d
        - name: config
          configMap:
            name: flannel
`

	// for flannel
	ServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: flannel
  namespace: kube-system
`


	ClusterRole=`kind: ClusterRole
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
      - patch
`

	ClusterRoleBinding = `
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: system:flannel
subjects:
  - kind: ServiceAccount
    name: flannel
    namespace: kube-system
roleRef:
  kind: ClusterRole
  name: system:flannel
  apiGroup: rbac.authorization.k8s.io
`

)
