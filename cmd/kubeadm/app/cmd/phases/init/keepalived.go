package phases

import (
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/klog"
	"k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/workflow"
	cmdutil "k8s.io/kubernetes/cmd/kubeadm/app/cmd/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/controlplane"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	McastGroupFormat string = "239.%d.%d.%d"
	DefaultPassword         = "Dream001"
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
	data, ok := c.(InitData)
	if !ok {
		return errors.New("keepalived phase invoked with an invalid data struct")
	}
	cfg := data.Cfg()
	if cfg.ControlPlaneEndpoint == "" {
		fmt.Println("[keepalived] Skipping Kubernetes High Availability Cluster")
		klog.V(1).Infoln("[keepalived] Skipping Kubernetes High Availability Cluster")
		return nil
	}
	envs := initKeepAlivedEnv(cfg.ControlPlaneEndpoint)
	if len(envs) == 0 {
		return errors.New("[keepalived] ControlPlaneEndpoint is invalid.")
	}
	if err := controlplane.GenerateKeepAlivedEnv(envs); err != nil {
		return errors.Wrap(err, "[keepalived] failed to generate keepalived envs")
	}
	return controlplane.TryInstallKeepAlived(cfg.ImageRepository)
}

func initKeepAlivedEnv(controlPlaneEndpoint string) map[string]string {
	envs := make(map[string]string,5)
    ip := net.ParseIP(strings.Split(controlPlaneEndpoint,":")[0])
    if !ip.IsGlobalUnicast() {
    	fmt.Println("[keepalived] ControlPlaneEndpoint is invalid.")
		errors.New("[keepalived] ControlPlaneEndpoint is invalid.")
    	return envs
	}
    i := randInt(0,254)
    envs["KA_VIP"] = ip.String()
    envs["KA_MCASTGROUP"] = fmt.Sprintf(McastGroupFormat,i,i,i)
    envs["KA_VROUTERID"] = strconv.Itoa(i)
    envs["KA_PRIORITY"] = strconv.Itoa(i)
    envs["KA_PASSWORD"] = DefaultPassword
	//envs["KA_IFACE"] = "eth0" // keepalived auto detect itself
    return envs
}



func randInt(min int, max int) int {
	rand.Seed(time.Now().UTC().UnixNano())
	return min + rand.Intn(max-min)
}
