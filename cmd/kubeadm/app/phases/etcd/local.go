/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package etcd

import (
	"fmt"
	"path"
	"strings"

	"k8s.io/api/core/v1"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/util/wait"
	"github.com/coreos/etcd/clientv3"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/images"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	staticpodutil "k8s.io/kubernetes/cmd/kubeadm/app/util/staticpod"
)

const (
	etcdVolumeName    = "etcd"
	etcdPkiVolumeName = "pki"
	etcdPkiPath       = "/etc/kubernetes/pki"
)

// CreateLocalEtcdStaticPodManifestFile will write local etcd static pod manifest file.
func CreateLocalEtcdStaticPodManifestFile(manifestDir string, cfg *kubeadmapi.MasterConfiguration) error {

	// gets etcd StaticPodSpec, actualized for the current MasterConfiguration
	spec := GetEtcdPodSpec(cfg)
	// writes etcd StaticPod to disk
	if err := staticpodutil.WriteStaticPodToDisk(kubeadmconstants.Etcd, manifestDir, spec); err != nil {
		return err
	}

	fmt.Printf("[etcd] Wrote Static Pod manifest for a local etcd instance to %q\n", kubeadmconstants.GetStaticPodFilepath(kubeadmconstants.Etcd, manifestDir))
	return nil
}

// GetEtcdPodSpec returns the etcd static Pod actualized to the context of the current MasterConfiguration
// NB. GetEtcdPodSpec methods holds the information about how kubeadm creates etcd static pod mainfests.
func GetEtcdPodSpec(cfg *kubeadmapi.MasterConfiguration) v1.Pod {
	etcdMounts := map[string]v1.Volume{
		etcdVolumeName: staticpodutil.NewVolume(etcdVolumeName, cfg.Etcd.DataDir, &v1.HostPathDirectoryOrCreate),
		etcdPkiVolumeName: staticpodutil.NewVolume(etcdPkiVolumeName,etcdPkiPath,&v1.HostPathFileOrCreate),
	}
	etcdVolumeMounts := []v1.VolumeMount{
		// Mount the etcd datadir path read-write so etcd can store data in a more persistent manner
		staticpodutil.NewVolumeMount(etcdVolumeName, cfg.Etcd.DataDir, false),
		staticpodutil.NewVolumeMount(etcdPkiVolumeName, etcdPkiPath, true),
	}
	return staticpodutil.ComponentPod(v1.Container{
		Name:    kubeadmconstants.Etcd,
		Command: getEtcdCommand(cfg),
		Image:   images.GetCoreImage(kubeadmconstants.Etcd, cfg.ImageRepository, cfg.KubernetesVersion, cfg.Etcd.Image),
		VolumeMounts:  etcdVolumeMounts,
		LivenessProbe: staticpodutil.ComponentProbe(cfg, kubeadmconstants.Etcd, 2379, "/health", v1.URISchemeHTTP),
	}, etcdMounts)
}

// getEtcdCommand builds the right etcd command from the given config object
func getEtcdCommand(cfg *kubeadmapi.MasterConfiguration) []string {
	AdvertiseAddr := "127.0.0.1"
	if len(cfg.API.AdvertiseAddress) > 0 {
		AdvertiseAddr = cfg.API.AdvertiseAddress
	}

	NewMemberName := "etcd-" + AdvertiseAddr
	NewMemberPeerUrl := "https://" + AdvertiseAddr + ":2380"
	InitialClusterFlag := "etcd-" + AdvertiseAddr + "=https://" + AdvertiseAddr + ":2380"
	InitialClusterStatus := "new"

	if cfg.HighAvailabilityPeer != "" {
		wait.PollImmediateInfinite(kubeadmconstants.DiscoveryRetryInterval, func() (bool, error) {
			endpoints := strings.Split(cfg.HighAvailabilityPeer, ",")
			socket := endpoints[0]
			ip := socket[:strings.Index(socket, ":")]
			existingMember := fmt.Sprintf("https://%s:2379", ip)
			//fmt.Printf("[manifests] Adding etcd member [ %s ] into an existing cluster [ %s ] !\n",NewMemberPeerUrl,existingMember)
			client, err := kubeadmutil.NewEtcdClient([]string{existingMember},
				path.Join(cfg.CertificatesDir, kubeadmconstants.APIServerClientCertName),
				path.Join(cfg.CertificatesDir, kubeadmconstants.APIServerClientKeyName),
				path.Join(cfg.CertificatesDir, kubeadmconstants.CACertName))
			if err != nil {
				return false, fmt.Errorf("[etcd] Fail to retrieve client from etcd [%v]", err)
			}
			var clusterflag = ""
			cluster := clientv3.NewCluster(client)
			ctx, cancel := context.WithTimeout(context.Background(), kubeadmconstants.DiscoveryRetryInterval)
			defer cancel()
			if memberListResponse, err := cluster.MemberList(ctx); err != nil {
				return false, fmt.Errorf("[etcd]  Fail to retrieve members of etcd,%s", err)
			} else {
				//m, _ := json.Marshal(members)
				//fmt.Printf("[manifests] Etcd members existed :[%s]\n", string(m))
				isMemberAdded := false
				for _, member := range memberListResponse.Members {
					if member.Name == "" && member.PeerURLs[0] != NewMemberPeerUrl {
						return false, fmt.Errorf("[etcd]  There's a member not ready, please wait the previous installation finished or remove it mannually")
					} else if member.PeerURLs[0] != NewMemberPeerUrl {
						clusterflag += "," + member.Name + "=" + member.PeerURLs[0]
					} else {
						isMemberAdded = true
					}
				}
				InitialClusterFlag += clusterflag
				InitialClusterStatus = "existing"
				if isMemberAdded == false {
					cluster.MemberAdd(ctx,[]string{NewMemberPeerUrl})
				}
				return true, nil
			}
			})
		}


	defaultArguments := map[string]string{
		"name":                  NewMemberName,
		"data-dir":              cfg.Etcd.DataDir,

		"trusted-ca-file":       path.Join(cfg.CertificatesDir, kubeadmconstants.CACertName),
		"key-file":              path.Join(cfg.CertificatesDir, kubeadmconstants.APIServerKeyName),
		"cert-file":             path.Join(cfg.CertificatesDir, kubeadmconstants.APIServerCertName),
		"client-cert-auth":      "true",
		"peer-trusted-ca-file":  path.Join(cfg.CertificatesDir, kubeadmconstants.CACertName),
		"peer-key-file":         path.Join(cfg.CertificatesDir, kubeadmconstants.APIServerKeyName),
		"peer-cert-file":        path.Join(cfg.CertificatesDir, kubeadmconstants.APIServerCertName),
		"peer-client-cert-auth": "true",

		"initial-advertise-peer-urls": "https://" + AdvertiseAddr + ":2380",
		"listen-peer-urls":            "https://" + AdvertiseAddr + ":2380",
		"listen-client-urls":          "https://" + AdvertiseAddr + ":2379,http://127.0.0.1:2379",
		"advertise-client-urls":       "https://" + AdvertiseAddr + ":2379,http://127.0.0.1:2379",
		"initial-cluster-token":       "k8s",
		"initial-cluster":             InitialClusterFlag,
		"initial-cluster-state":       InitialClusterStatus,
	}

	command := []string{"etcd"}
	command = append(command, kubeadmutil.BuildArgumentListFromMap(defaultArguments, cfg.Etcd.ExtraArgs)...)
	return command
}
