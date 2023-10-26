/*
Copyright 2021 The KCP Authors.

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

package main

import (
	"math/rand"
	"os"
	"time"

	"github.com/spf13/cobra"

	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/component-base/cli"
	logsapiv1 "k8s.io/component-base/logs/api/v1"
	_ "k8s.io/component-base/logs/json/register"
	"k8s.io/component-base/version"

	"k8s.io/klog/v2"

	cacheclient "github.com/kcp-dev/kcp/pkg/cache/client"
	"github.com/kcp-dev/kcp/pkg/cache/client/shard"

	clientoptions "github.com/kcp-dev/contrib-tmc/cmd/tmc/options"
	"github.com/kcp-dev/contrib-tmc/pkg/features"
	"github.com/kcp-dev/contrib-tmc/pkg/manager"
	"github.com/kcp-dev/contrib-tmc/pkg/manager/options"
	"github.com/kcp-dev/contrib-tmc/tmc/cmd/help"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	cmd := &cobra.Command{
		Use:   "tmc",
		Short: "TMC - Transparent Multi Cluster - kcp sub-project for multi-cluster workload management",
		Long: help.Doc(`
			TMC is the easiest way to manage Kubernetes applications against one or
			more clusters, by giving you a personal control plane that schedules your
			workloads onto one or many clusters, and making it simple to pick up and
			move. It supports advanced use cases such as spreading your apps across
			clusters for resiliency, scheduling batch workloads onto clusters with
			free capacity, and enabling collaboration for individual teams without
			having access to the underlying clusters.

			To get started, launch a new cluster with 'tmc start', which will
			initialize tmc and bootstrap its resources inside kcp.
		`),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	clientOptions := clientoptions.NewOptions()
	clientOptions.Complete()
	clientOptions.Validate()

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the controller manager",
		Long: help.Doc(`
			Start the controller manager

			The controller manager is in charge of starting the controllers reconciliating TMC resources.
		`),
		PersistentPreRunE: func(*cobra.Command, []string) error {
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// run as early as possible to avoid races later when some components (e.g. grpc) start early using klog
			if err := logsapiv1.ValidateAndApply(clientOptions.Logs, features.DefaultFeatureGate); err != nil {
				return err
			}

			logger := klog.FromContext(cmd.Context())
			logger.Info("Instantiating the TMC manager")

			options := options.NewOptions()
			completedOptions, err := options.Complete()
			if err != nil {
				return err
			}
			config, err := manager.NewConfig(*completedOptions)
			if err != nil {
				return err
			}
			ctx := genericapiserver.SetupSignalContext()
			kcpClientConfigOverrides := &clientcmd.ConfigOverrides{
				CurrentContext: clientOptions.Context,
			}
			kcpClientConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
				&clientcmd.ClientConfigLoadingRules{ExplicitPath: clientOptions.Kubeconfig},
				kcpClientConfigOverrides).ClientConfig()
			if err != nil {
				return err
			}
			kcpClientConfig.QPS = clientOptions.QPS
			kcpClientConfig.Burst = clientOptions.Burst
			// TODO (FGI) this needs to be amended so that the flag --cache-config is
			// taken in consideration
			cacheClientConfigOverrides := &clientcmd.ConfigOverrides{
				CurrentContext: "system:admin",
			}
			cacheClientConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
				&clientcmd.ClientConfigLoadingRules{ExplicitPath: clientOptions.Kubeconfig},
				cacheClientConfigOverrides).ClientConfig()
			if err != nil {
				return err
			}
			cacheClientConfig.QPS = clientOptions.QPS
			cacheClientConfig.Burst = clientOptions.Burst
			cacheClientConfig = cacheclient.WithCacheServiceRoundTripper(cacheClientConfig)
			cacheClientConfig = cacheclient.WithShardNameFromContextRoundTripper(cacheClientConfig)
			cacheClientConfig = cacheclient.WithDefaultShardRoundTripper(cacheClientConfig, shard.Wildcard)
			mgr, err := manager.NewManager(ctx, config, kcpClientConfig, cacheClientConfig)
			if err != nil {
				return err
			} else {
				return mgr.Start(ctx)
			}
		},
	}

	clientOptions.AddFlags(startCmd.Flags())

	if v := version.Get().String(); len(v) == 0 {
		startCmd.Version = "<unknown>"
	} else {
		startCmd.Version = v
	}

	cmd.AddCommand(startCmd)

	help.FitTerminal(cmd.OutOrStdout())

	if v := version.Get().String(); len(v) == 0 {
		cmd.Version = "<unknown>"
	} else {
		cmd.Version = v
	}

	os.Exit(cli.Run(cmd))
}
