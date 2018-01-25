/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-01-24  @author weiwei@tenxcloud.com
 */
package serviceproxy


const (

	TenxProxyVersion         = "v2.2.0"
	TenxProxyDomainConfigMap = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: kube-config
  namespace: kube-system
data:
  domain.json: '{"externalip":"","domain":""}'
`
	TenxProxyCertsConfigMap = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: kube-certs
  namespace: kube-system
data:
`

	TenxProxyDaemonSet = `
apiVersion: apps/v1beta2
kind: DaemonSet
metadata:
  labels:
    name: service-proxy
  name: service-proxy
  namespace: kube-system
spec:
  selector:
    matchLabels:
      name: service-proxy
  template:
    metadata:
      annotations:
        prometheus.io/scrape: "true"
      labels:
        name: service-proxy
    spec:
      serviceAccountName: service-proxy
      containers:
      - command:
        - /run.sh
        - --plugins=tenx-proxy --watch=watchsrvs --emailReceiver=yangle@tenxcloud.com
          --config=/etc/tenx/domain.json
        image: {{ .ImageRepository }}/tenx-proxy:{{ .Version }}
        imagePullPolicy: IfNotPresent
        name: service-proxy
        volumeMounts:
        - mountPath: /var/run/docker.sock
          name: docker-sock
        - mountPath: /etc/tenx/
          name: kube-config
        - mountPath: /etc/sslkeys/certs
          name: kube-cert
        - mountPath: /run/haproxy
          name: haproxy-sock
      - command:
        - sh
        - -c
        - sleep 10 && haproxy_exporter --haproxy.scrape-uri=unix:/run/haproxy/admin.sock
        image: {{ .ImageRepository }}/haproxy-exporter:v0.8.0
        imagePullPolicy: IfNotPresent
        name: exporter
        ports:
        - containerPort: 9101
          hostPort: 9101
          name: scrape
          protocol: TCP
        resources: {}
        volumeMounts:
        - mountPath: /run/haproxy
          name: haproxy-sock
      dnsPolicy: ClusterFirst
      hostNetwork: true
      nodeSelector:
        role: proxy
      restartPolicy: Always
      volumes:
      - emptyDir: {}
        name: docker-sock
      - hostPath:
          path: /var/run/docker.sock
        name: config-volume
      - configMap:
          defaultMode: 420
          name: kube-config
        name: kube-config
      - configMap:
          defaultMode: 420
          name: kube-certs
        name: kube-cert
      - emptyDir: {}
        name: haproxy-sock
      tolerations:
      - key: node-role.kubernetes.io/master
        effect: NoSchedule
      - key: CriticalAddonsOnly
        operator: Exists
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: role
                operator: In
                values:
                - proxy
`

	// for service-proxy
	ServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: service-proxy
  namespace: kube-system
`

	ClusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: system:service-proxy
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: service-proxy
  namespace: kube-system
`

)