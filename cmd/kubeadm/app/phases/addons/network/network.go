/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-02-05  @author weiwei@tenxcloud.com
 */
package network

import (
	"fmt"

	clientset "k8s.io/client-go/kubernetes"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/network/calico"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/network/flannel"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/network/canal"
)

const (
	// For backwards compatible, leave it empty if unset
	Calico string = "calico"
	// If nothing exists at the given path, an empty directory will be created there
	// as needed with file mode 0755, having the same group and ownership with Kubelet.
	Flannel string = "flannel"
	// A directory must exist at the given path
	Canal string = "canal"
	// If nothing exists at the given path, an empty file will be created there
	// as needed with file mode 0644, having the same group and ownership with Kubelet.
	Macvlan string = "macvlan"
)

func DeployNetworkAddons(cfg *kubeadmapi.MasterConfiguration, client clientset.Interface) error {
	//TODO: FIXME:
	//network plugin(calico,flannel,canal,macvlan)
	if cfg.Networking.Plugin == Calico || cfg.Networking.Plugin == ""{
		if err := calico.CreateCalicoAddon(cfg, client); err != nil {
			return fmt.Errorf("error setup calico addon: %v", err)
		}
	} else if cfg.Networking.Plugin == Flannel {
		if err := flannel.CreateFlannelAddon(cfg, client); err != nil {
			return fmt.Errorf("error setup flannel addon: %v", err)
		}
	} else if cfg.Networking.Plugin == Canal {
		if err := canal.CreateCanalAddon(cfg, client); err != nil {
			return fmt.Errorf("error setup canal addon: %v", err)
		}
	} else if cfg.Networking.Plugin == Macvlan {
		//TODO: FIXME
	} else {
		fmt.Errorf("Unsupported Network Plugin: %s !\n", cfg.Networking.Plugin)
	}
	return nil
}
