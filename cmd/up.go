package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/okteto/cnd/pkg/k8/client"
	"github.com/okteto/cnd/pkg/k8/deployments"
	"github.com/okteto/cnd/pkg/k8/forward"
	"github.com/okteto/cnd/pkg/storage"
	"github.com/okteto/cnd/pkg/syncthing"

	"github.com/okteto/cnd/pkg/model"
	"github.com/spf13/cobra"
)

//Up starts or upgrades a cloud native environment
func Up() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Starts or upgrades a cloud native environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeUp(devPath)
		},
	}

	return cmd
}

func executeUp(devPath string) error {
	fmt.Println("Activating dev mode...")

	namespace, client, restConfig, err := client.Get()
	if err != nil {
		return err
	}

	dev, err := model.ReadDev(devPath)
	if err != nil {
		return err
	}

	d, err := dev.DevDeployment()
	if err != nil {
		return err
	}

	err = deployments.Deploy(d, namespace, client)
	if err != nil {
		return err
	}

	pod, err := getCNDPod(client, namespace, d.Name, dev.Swap.Deployment.Container)
	if err != nil {
		return err
	}

	sy, err := syncthing.NewSyncthing(d.Name, namespace, dev.Mount.Source)
	if err != nil {
		return err
	}

	pf, err := forward.NewCNDPortForward(dev.Mount.Source, sy.RemoteAddress, deployments.GetFullName(namespace, d.Name))
	if err != nil {
		return err
	}

	if err := sy.Run(); err != nil {
		return err
	}

	err = storage.Insert(namespace, d.Name, dev.Swap.Deployment.Container, sy.LocalPath, sy.GUIAddress)
	if err != nil {
		return err
	}

	defer stop(sy, pf)
	err = pf.Start(client, restConfig, pod)
	return err
}

func stop(sy *syncthing.Syncthing, pf *forward.CNDPortForward) {
	if err := sy.Stop(); err != nil {
		log.Error(err)
	}

	pf.Stop()
}
