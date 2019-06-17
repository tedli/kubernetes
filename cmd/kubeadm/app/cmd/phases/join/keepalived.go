package phases

import (
	"github.com/pkg/errors"
	"k8s.io/klog"
	"k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/workflow"
	cmdutil "k8s.io/kubernetes/cmd/kubeadm/app/cmd/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/controlplane"
)

// NewKeepAlivedPhase creates a kubeadm workflow phase that install and configure keepalived.
func NewKeepAlivedPhase() workflow.Phase {
	return workflow.Phase{
		Name:  "keepalived",
		Short: "Install and configure keepalived, Writes keepalived service file.",
		Long:  cmdutil.MacroCommandLongDescription,
		Hidden: true,
		Run:   runKeepAlived,
	}
}

func runKeepAlived(c workflow.RunData) error {
	data, ok := c.(JoinData)
	if !ok {
		return errors.New("keepalived phase invoked with an invalid data struct")
	}
	if data.Cfg().ControlPlane == nil {
		klog.V(1).Infoln("[keepalived] Skipping Kubernetes High Availability Cluster")
		return nil
	}
	cfg, err := data.InitCfg()
	if err != nil {
		return err
	}
	client, err := bootstrapClient(data)
	if err != nil {
		return err
	}
    if err := controlplane.DownloadEnvs(client); err != nil {
		return errors.Wrap(err,"[keepalived] failed to generate keepalived envs")
	}
	return controlplane.TryInstallKeepAlived(cfg.ImageRepository)
}
