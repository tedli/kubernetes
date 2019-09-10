package copycerts

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	"k8s.io/klog"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pkiutil"
	"path"
)

// CreateEtcdClientCerts generates the certificates needed by network plugins.
func CreateEtcdClientCerts(client clientset.Interface, cfg *kubeadmapi.InitConfiguration) error {
	fmt.Printf("[etcd-client-certs] Downloading the certificates in Secret %q in the %q Namespace\n", kubeadmconstants.KubeadmCertsSecret, metav1.NamespaceSystem)

	secret, err := getSecret(client)
	if err != nil {
		return errors.Wrap(err, "error downloading the secret")
	}

	decodedKey, err := hex.DecodeString(secret.Annotations[certificateKey])
	if err != nil {
		return errors.Wrap(err, "error decoding certificate key")
	}

	secretData, err := getDataFromSecret(secret, decodedKey)
	if err != nil {
		return errors.Wrap(err, "error decoding secret data with provided key")
	}

	for certOrKeyName, certOrKeyPath := range certsToPath(cfg) {
		certOrKeyData, found := secretData[certOrKeyNameToSecretName(certOrKeyName)]
		if !found {
			return errors.New("couldn't find required certificate or key in Secret")
		}
		if len(certOrKeyData) == 0 {
			klog.V(1).Infof("[download-certs] Not saving %q to disk, since it is empty in the %q Secret\n", certOrKeyName, kubeadmconstants.KubeadmCertsSecret)
			continue
		}
		if err := writeCertOrKey(certOrKeyPath, certOrKeyData); err != nil {
			return err
		}
	}
	etcdCaCert, err := cert.ParseCertsPEM(secretData[certOrKeyNameToSecretName(kubeadmconstants.EtcdCACertName)])
	if err != nil {
		return fmt.Errorf("[etcd-client-certs] failed to Parse etcd ca cert [%v]", err)
	}
	etcdCaKey, err := keyutil.ParsePrivateKeyPEM(secretData[certOrKeyNameToSecretName(kubeadmconstants.EtcdCAKeyName)])
	if err != nil {
		return fmt.Errorf("[etcd-client-certs] failed to Parse etcd ca key [%v]", err)
	}
	// sign etcd client certificate with etcd ca
	certCfg := &cert.Config{
		CommonName:   kubeadmconstants.EtcdClientCertCommonName,
		Organization: []string{kubeadmconstants.NodesGroup},
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	etcdClientCert, etcdClientKey, err := pkiutil.NewCertAndKey(etcdCaCert[0], etcdCaKey.(*rsa.PrivateKey), certCfg)
	if err != nil {
		return fmt.Errorf("[etcd-client-certs] failed to Create etcd client certificate & key   [%v]", err)
	}
	if err := pkiutil.WriteCertAndKey(cfg.CertificatesDir, kubeadmconstants.EtcdClientCertAndKeyBaseName, etcdClientCert, etcdClientKey); err != nil {
		return fmt.Errorf("[etcd-client-certs] failed while saving %s certificate and key: %v", kubeadmconstants.EtcdClientCertAndKeyBaseName, err)
	}
	return nil
}

func certsToPath(cfg *kubeadmapi.InitConfiguration) map[string]string {
	certsDir := cfg.CertificatesDir
	certs := map[string]string{}
	if cfg.Etcd.External == nil {
		certs[kubeadmconstants.EtcdCACertName] = path.Join(certsDir, kubeadmconstants.EtcdCACertName)
		certs[kubeadmconstants.EtcdCAKeyName] = path.Join(certsDir, kubeadmconstants.EtcdCAKeyName)
	}
	return certs
}
