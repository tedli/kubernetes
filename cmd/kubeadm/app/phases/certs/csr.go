/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-01-30  @author weiwei@tenxcloud.com
 */
package certs

import (
	"os"
	"fmt"
	"path"
	"k8s.io/apimachinery/pkg/types"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/certificate/csr"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	kubeconfigutil "k8s.io/kubernetes/cmd/kubeadm/app/util/kubeconfig"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
)

// PerformTLSBootstrap executes a node certificate signing request.
func PerformTLSBootstrap(cfg *clientcmdapi.Config) error {
	client, err := kubeconfigutil.ToClientSet(cfg)
	if err != nil {
		return err
	}

	fmt.Println("[csr] Created API client to obtain client certificate for this node, generating keys and certificate signing request")

	key, err := certutil.MakeEllipticPrivateKeyPEM()
	if err != nil {
		return fmt.Errorf("failed to generate private key [%v]", err)
	}

	hostName, err := os.Hostname()
	cert, err := csr.RequestNodeCertificate(client.CertificatesV1beta1().CertificateSigningRequests(), key, types.NodeName(hostName))
	if err != nil {
		return fmt.Errorf("failed to request signed certificate from the API server [%v]", err)
	}
	fmt.Println("[csr] Received signed certificate from the API server")

	err = writeApiServerClientCert(cfg, cert, key)
	if err != nil {
		return fmt.Errorf("[csr] couldn't save client certificate,client key to disk: [%v]", err)
	}
	return nil
}

func writeApiServerClientCert(cfg *clientcmdapi.Config, certData, keyData []byte) error {
	//cluster := cfg.Contexts[cfg.CurrentContext].Cluster
	//err := certutil.WriteCert(kubeadmapi.DefaultCACertPath, cfg.Clusters[cluster].CertificateAuthorityData)
	//if err != nil {
	//	return fmt.Errorf("couldn't save the CA certificate to disk: %v", err)
	//}
    var err error
	clientPath := path.Join(kubeadmapi.DefaultCertificatesDir, kubeadmconstants.APIServerClientCertName)
	err = certutil.WriteCert(clientPath, certData)
	if err != nil {
		return fmt.Errorf("couldn't save the client certificate to disk: %v", err)
	}

	keyPath := path.Join(kubeadmapi.DefaultCertificatesDir, kubeadmconstants.APIServerClientKeyName)
	err = certutil.WriteKey(keyPath, keyData)
	if err != nil {
		return fmt.Errorf("couldn't save the client key to disk: %v", err)
	}
	return nil
}
