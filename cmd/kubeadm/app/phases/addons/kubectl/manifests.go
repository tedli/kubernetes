/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-01-22  @author weiwei@tenxcloud.com
 */

/**
 *  Kubectl Version v1.9.2
 */

package kubectl



const (

  //This DaemonSet is used to WebTerminal installation.
  DaemonSet = `
kind: DaemonSet
apiVersion: apps/v1beta2
metadata:
  labels:
    app: kubectl
    use: webt
  name: kubectl
  namespace: kube-system
spec:
  selector:
    matchLabels:
      app: kubectl
      use: webt
  template:
    metadata:
      labels:
        app: kubectl
        use: webt
    spec:
      serviceAccountName: kubectl
      hostNetwork: true
      tolerations:
      - key: node-role.kubernetes.io/master
        effect: NoSchedule
      - key: CriticalAddonsOnly
        operator: Exists
      containers:
        - name: kubectl
          command:
            - /check.sh
            - "60"
          image: {{ .ImageRepository }}/kubectl-{{ .Arch }}:{{ .Version }}
          volumeMounts:
          - name: docker-sock
            mountPath: /var/run/docker.sock
          - name: localtime
            mountPath: /etc/localtime
          - mountPath: /etc/resolv.conf
            name: resolv
          - mountPath: /tmp/
            name: checklog
          - mountPath: /etc/kubernetes/manifests/
            name: k8s-manifests
      volumes:
      - name: docker-sock
        hostPath:
          path: /var/run/docker.sock
          type: FileOrCreate
      - name: localtime
        hostPath:
          path: /etc/localtime
          type: FileOrCreate
      - hostPath:
          path: /etc/resolv.conf
          type: FileOrCreate
        name: resolv
      - hostPath:
          path: /paas/agent_check/
          type: DirectoryOrCreate
        name: checklog
      - hostPath:
          path: /etc/kubernetes/manifests/
          type: DirectoryOrCreate
        name: k8s-manifests
`

  // for kubectl
  ServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kubectl
  namespace: kube-system`

  ClusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: system:kubectl
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: kubectl
  namespace: kube-system`

)