/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-01-23  @author weiwei@tenxcloud.com
 */

package kubelet

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/golang/glog"

	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/util/initsystem"
	"k8s.io/kubernetes/pkg/volume/util"
)

var (
	kubeletServicePath     = "/etc/systemd/system"
	ServiceName            = "kubelet"
	ConfigName             = "10-kubeadm.conf"
	kubeletServiceConfPath = kubeletServicePath + "/" + ServiceName + ".service.d"
)

func TryInstallKubelet(serviceSubnet, DNSDomain, imageRepository, kubernetesVersion string) error {
	// PHASE 1: Write Kubelet Service to /etc/systemd/system/kubelet.service
	err := writeKubeletService(serviceSubnet, DNSDomain, imageRepository, kubernetesVersion)
	if err != nil {
		fmt.Println("[kubelet] Write kubelet service to /etc/systemd/system/kubelet.service failed.")
		return err
	}
	// PHASE 2: If we notice that the kubelet service is inactive, try to start it
	initSystem, err := initsystem.GetInitSystem()
	initSystem.DaemonReload()
	if err != nil {
		fmt.Println("[kubelet] No supported init system detected, won't ensure kubelet is running.")
		return err
	} else if initSystem.ServiceExists(ServiceName) && !initSystem.ServiceIsActive(ServiceName) {
		fmt.Println("[kubelet] Starting the kubelet service")
		if err := initSystem.ServiceStart(ServiceName); err != nil {
			fmt.Printf("[kubelet] WARNING: Unable to start the kubelet service: [%v]\n", err)
			fmt.Println("[kubelet] WARNING: Please ensure kubelet is running manually.")
			return err
		} else {
			if !initSystem.ServiceIsEnabled(ServiceName) {
				initSystem.ServiceEnable(ServiceName)
				fmt.Println("[kubelet] kubelet is enabled.")
			}
		}
	}
	return nil
}

// /etc/systemd/system/kubelet.service
func writeKubeletService(serviceSubnet, DNSDomain, imageRepository, kubernetesVersion string) error {
	dnsIP, err := getKubeDNSServiceIP(serviceSubnet)
	if err != nil {
		return fmt.Errorf("could not parse dns ip %q", dnsIP)
	}
	kubeletservice := `
[Unit]
Description=kubelet: The Kubernetes Node Agent
Documentation=http://kubernetes.io/docs/
After=network.target docker.service

[Service]
ExecStart=/usr/bin/kubelet
Restart=on-failure
StartLimitInterval=0
RestartSec=10

[Install]
WantedBy=multi-user.target
`
	buf := bytes.Buffer{}
	buf.WriteString(kubeletservice)
	filename := filepath.Join(kubeletServicePath, ServiceName+".service")
	if err := cmdutil.DumpReaderToFile(bytes.NewReader(buf.Bytes()), filename); err != nil {
		return fmt.Errorf("failed to create kubelet.service file for (%q) [%v] \n", filename, err)
	} else {
		fmt.Printf("[kubelet] Write kubelet service to %q Successfully.\n", filename)
	}

	if exist, _ := util.PathExists(kubeletServiceConfPath); exist == false {
		err := os.MkdirAll(kubeletServiceConfPath, 0755)
		if err != nil {
			glog.Error(err)
			return err
		}
	}

	buf = bytes.Buffer{}
	buf.WriteString("[Service]\n")
	buf.WriteString("Environment=\"KUBELET_KUBECONFIG_ARGS=--bootstrap-kubeconfig=/etc/kubernetes/bootstrap-kubelet.conf --kubeconfig=/etc/kubernetes/kubelet.conf \"\n")
	buf.WriteString("Environment=\"KUBELET_SYSTEM_PODS_ARGS=--pod-manifest-path=/etc/kubernetes/manifests --allow-privileged=true\"\n")
	buf.WriteString("Environment=\"KUBELET_NETWORK_ARGS=--network-plugin=cni --cni-conf-dir=/etc/cni/net.d --cni-bin-dir=/opt/cni/bin\"\n")
	buf.WriteString("Environment=\"KUBELET_DNS_ARGS=--cluster-dns=" + dnsIP.String() + "   --cluster-domain=" + DNSDomain + "\"\n")
	buf.WriteString("Environment=\"KUBELET_AUTHZ_ARGS=--client-ca-file=/etc/kubernetes/pki/ca.crt  --anonymous-auth=false --authorization-mode=Webhook --authentication-token-webhook \"\n")
	buf.WriteString("Environment=\"KUBELET_CADVISOR_ARGS=--cadvisor-port=0 --housekeeping-interval=5s  --global-housekeeping-interval=5s \"\n")
	buf.WriteString("Environment=\"KUBELET_CERT_ARGS=--rotate-certificates=true --cert-dir=/var/lib/kubelet/pki --feature-gates=RotateKubeletServerCertificate=true  \"\n")
	buf.WriteString("Environment=\"KUBELET_CGROUP_ARGS=--cgroup-driver=cgroupfs  --pod-infra-container-image=" + fmt.Sprintf("%s/pause-%s:3.0", imageRepository, runtime.GOARCH) + "\"\n")
	buf.WriteString("Environment=\"KUBELET_PERFORMANCE_ARGS=--kube-reserved=cpu=200m,memory=512Mi  \"\n")
	buf.WriteString("ExecStartPre=/usr/bin/docker run --rm -v /opt/tmp/bin/:/opt/tmp/bin/   ")
	buf.WriteString(fmt.Sprintf("%s/hyperkube-%s:%s", imageRepository, runtime.GOARCH, kubernetesVersion))
	buf.WriteString(" /bin/bash -c \"mkdir -p /opt/tmp/bin && cp /opt/cni/bin/* /opt/tmp/bin/ && cp /usr/bin/nsenter /opt/tmp/bin/\" \n")
	buf.WriteString("ExecStartPre=/bin/bash -c \"mkdir -p /opt/cni/bin && cp -r /opt/tmp/bin/ /opt/cni/ && cp /opt/tmp/bin/nsenter /usr/bin/ && rm -r /opt/tmp\"\n")
	buf.WriteString("ExecStartPre=/bin/bash -c \"docker inspect kubelet >/dev/null 2>&1 && docker rm -f kubelet || true \" \n")
	buf.WriteString("ExecStart= \n")
	buf.WriteString("ExecStart=/bin/bash -c \"docker run --name kubelet --net=host --privileged --pid=host -v /:/rootfs:ro ")
	buf.WriteString("-v /dev:/dev -v /var/log:/var/log:shared -v /var/lib/docker/:/var/lib/docker:rw  ")
	buf.WriteString("-v /var/lib/kubelet/:/var/lib/kubelet:shared -v /etc/kubernetes:/etc/kubernetes:ro ")
	buf.WriteString("-v /etc/cni:/etc/cni:rw -v /sys:/sys:ro -v /var/run:/var/run:rw -v /opt/cni/bin/:/opt/cni/bin/ ")
	buf.WriteString("-v /srv/kubernetes:/srv/kubernetes:ro ")
	buf.WriteString(fmt.Sprintf("%s/hyperkube-%s:%s", imageRepository, runtime.GOARCH, kubernetesVersion))
	buf.WriteString(" nsenter --target=1 --mount --wd=./ -- ./hyperkube kubelet ")
	buf.WriteString(" $KUBELET_KUBECONFIG_ARGS $KUBELET_SYSTEM_PODS_ARGS $KUBELET_NETWORK_ARGS $KUBELET_DNS_ARGS $KUBELET_AUTHZ_ARGS  $KUBELET_CADVISOR_ARGS $KUBELET_CERT_ARGS $KUBELET_CGROUP_ARGS $KUBELET_EXTRA_ARGS $KUBELET_PERFORMANCE_ARGS \" \n")
	buf.WriteString("ExecStop=/usr/bin/docker stop kubelet \n")
	buf.WriteString("ExecStopPost=/usr/bin/docker rm -f kubelet \n")
	buf.WriteString("Restart=on-failure \n")
	buf.WriteString("StartLimitInterval=0 \n")
	buf.WriteString("RestartSec=10 \n")
	buf.WriteString("\n")
	buf.WriteString("[Install]\n")
	buf.WriteString("WantedBy=multi-user.target\n")
	buf.WriteString("\n")
	filename = kubeletServiceConfPath + "/" + ConfigName
	if err := cmdutil.DumpReaderToFile(bytes.NewReader(buf.Bytes()), filename); err != nil {
		return fmt.Errorf("failed to create kubelet.service file for (%q) [%v] \n", filename, err)
	} else {
		fmt.Printf("[kubelet] Write kubelet service conf to %q Successfully.\n", filename)
	}
	return nil
}

func getKubeDNSServiceIP(serviceSubnet string) (net.IP, error) {
	index := strings.LastIndex(serviceSubnet, ".")
	// Build an IP by taking the kubernetes service's clusterIP and appending a "0" and checking that it's valid
	dnsIP := net.ParseIP(fmt.Sprintf("%s.10", serviceSubnet[:index]))
	if dnsIP == nil {
		return nil, fmt.Errorf("could not parse dns ip %q", dnsIP)
	}
	return dnsIP, nil
}
