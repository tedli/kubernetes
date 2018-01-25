/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-01-24  @author weiwei@tenxcloud.com
 */

package flannel

import (
	"fmt"
	"runtime"

	"k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	apps "k8s.io/api/apps/v1beta2"
	clientset "k8s.io/client-go/kubernetes"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
)

func CreateFlannelAddon(cfg *kubeadmapi.MasterConfiguration, client clientset.Interface) error {
	//PHASE 1: create flannel containers
	daemonSetBytes, err := kubeadmutil.ParseTemplate(DaemonSet, struct{ ImageRepository, Arch, Version string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Arch:            runtime.GOARCH,
		Version:         Version,
	})
	if err != nil {
		return fmt.Errorf("error when parsing flannel daemonset template: %v", err)
	}
	//PHASE 2: create flannel configMap
	configMapBytes, err := kubeadmutil.ParseTemplate(ConfigMap, struct{ PodSubnet, Backend string }{
		PodSubnet: cfg.Networking.PodSubnet,
		// TODO: FIXME
		Backend:   "vxlan", // vxlan,udp,host-gw,ipip,ali-vpc,aws-vpc,gce,alloc
	})
	if err != nil {
		return fmt.Errorf("error when parsing flannel configmap template: %v", err)
	}

	if err := createFlannel(daemonSetBytes, configMapBytes, client); err != nil {
		return err
	}
	fmt.Println("[addons] Applied essential addon: flannel")
	return nil
}

func createFlannel(daemonSetBytes, configBytes []byte, client clientset.Interface) error {
	//PHASE 1: create ConfigMap for flannel
	configMap := &v1.ConfigMap{}
	if err := kuberuntime.DecodeInto(legacyscheme.Codecs.UniversalDecoder(), configBytes, configMap); err != nil {
		return fmt.Errorf("unable to decode flannel configmap %v", err)
	}

	// Create the ConfigMap for flannel or update it in case it already exists
	if err := apiclient.CreateOrUpdateConfigMap(client, configMap); err != nil {
		return err
	}
	//PHASE 2: create RBAC rules
	clusterRoles := &rbac.ClusterRole{}
	if err := kuberuntime.DecodeInto(legacyscheme.Codecs.UniversalDecoder(), []byte(ClusterRole), clusterRoles); err != nil {
		return fmt.Errorf("unable to decode flannel clusterroles %v", err)
	}

	// Create the ClusterRoles for Calico Node or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRole(client, clusterRoles); err != nil {
		return err
	}

	clusterRolesBinding := &rbac.ClusterRoleBinding{}
	if err := kuberuntime.DecodeInto(legacyscheme.Codecs.UniversalDecoder(), []byte(ClusterRoleBinding), clusterRolesBinding); err != nil {
		return fmt.Errorf("unable to decode flannel clusterrolebindings %v", err)
	}

	// Create the ClusterRoleBindings for flannel or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRoleBinding(client, clusterRolesBinding); err != nil {
		return err
	}

	serviceAccount := &v1.ServiceAccount{}
	if err := kuberuntime.DecodeInto(legacyscheme.Codecs.UniversalDecoder(), []byte(ServiceAccount), serviceAccount); err != nil {
		return fmt.Errorf("unable to decode flannel serviceAccount %v", err)
	}

	// Create the ConfigMap for flannel or update it in case it already exists
	if err := apiclient.CreateOrUpdateServiceAccount(client, serviceAccount); err != nil {
		return err
	}

	//PHASE 3: create flannel daemonSet
	daemonSet := &apps.DaemonSet{}
	if err := kuberuntime.DecodeInto(legacyscheme.Codecs.UniversalDecoder(), daemonSetBytes, daemonSet); err != nil {
		return fmt.Errorf("unable to decode flannel daemonset %v", err)
	}

	// Create the DaemonSet for flannel or update it in case it already exists
	return apiclient.CreateOrUpdateDaemonSet(client, daemonSet)

}
