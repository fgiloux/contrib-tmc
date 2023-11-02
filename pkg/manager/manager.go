/*
Copyright 2023 The KCP Authors.

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

package manager

import (
	"context"
	"fmt"
	"time"

	kcpapiextensionsclientset "github.com/kcp-dev/client-go/apiextensions/client"
	kcpapiextensionsinformers "github.com/kcp-dev/client-go/apiextensions/informers"
	kcpkubernetesinformers "github.com/kcp-dev/client-go/informers"
	kcpkubernetesclientset "github.com/kcp-dev/client-go/kubernetes"
	"github.com/kcp-dev/kcp/pkg/virtual/framework/rootapiserver"
	apisv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha1"
	kcpclientset "github.com/kcp-dev/kcp/sdk/client/clientset/versioned"
	kcpclusterclientset "github.com/kcp-dev/kcp/sdk/client/clientset/versioned/cluster"
	kcpinformers "github.com/kcp-dev/kcp/sdk/client/informers/externalversions"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/version"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	tmcclusterclientset "github.com/kcp-dev/contrib-tmc/client/clientset/versioned/cluster"
	tmcinformers "github.com/kcp-dev/contrib-tmc/client/informers/externalversions"
	"github.com/kcp-dev/contrib-tmc/pkg/informer"
	metadataclient "github.com/kcp-dev/contrib-tmc/pkg/metadata"
)

/*
The manager is in charge of instantiating and starting the controllers
that do the reconciliation for TMC.
*/

const (
	apiExportName = "tmc.kcp.io"
	resyncPeriod  = 10 * time.Hour
)

type Manager struct {
	*Config
	clientConfig                            *rest.Config
	tmcSharedInformerFactory                tmcinformers.SharedInformerFactory
	kcpSharedInformerFactory                kcpinformers.SharedInformerFactory
	kubeSharedInformerFactory               kcpkubernetesinformers.SharedInformerFactory
	cacheKcpSharedInformerFactory           kcpinformers.SharedInformerFactory
	discoveringDynamicSharedInformerFactory *informer.DiscoveringDynamicSharedInformerFactory
	apiExtensionsClusterClient              kcpapiextensionsclientset.ClusterInterface
	apiExtensionsSharedInformerFactory      kcpapiextensionsinformers.SharedInformerFactory
	virtualWorkspaces                       []rootapiserver.NamedVirtualWorkspace
	stopCh                                  chan struct{}
	syncedCh                                chan struct{}
}

// NewManager creates a manager able to start controllers
func NewManager(ctx context.Context, cfg *Config, bootstrapClientConfig, cacheClientConfig *rest.Config) (*Manager, error) {
	var err error
	m := &Manager{
		Config:   cfg,
		stopCh:   make(chan struct{}),
		syncedCh: make(chan struct{}),
	}
	if m.clientConfig, err = restConfigForAPIExport(ctx, bootstrapClientConfig, apiExportName); err != nil {
		return nil, err
	}
	informerTmcClient, err := tmcclusterclientset.NewForConfig(m.clientConfig)
	if err != nil {
		return nil, err
	}
	m.tmcSharedInformerFactory = tmcinformers.NewSharedInformerFactoryWithOptions(
		informerTmcClient,
		resyncPeriod,
	)
	informerKcpClient, err := kcpclusterclientset.NewForConfig(m.clientConfig)
	if err != nil {
		return nil, err
	}
	m.kcpSharedInformerFactory = kcpinformers.NewSharedInformerFactoryWithOptions(
		informerKcpClient,
		resyncPeriod,
	)
	informerKubeClient, err := kcpkubernetesclientset.NewForConfig(m.clientConfig)
	if err != nil {
		return nil, err
	}
	m.kubeSharedInformerFactory = kcpkubernetesinformers.NewSharedInformerFactoryWithOptions(
		informerKubeClient,
		resyncPeriod,
	)
	cacheKcpClusterClient, err := kcpclusterclientset.NewForConfig(cacheClientConfig)
	if err != nil {
		return nil, err
	}
	m.cacheKcpSharedInformerFactory = kcpinformers.NewSharedInformerFactoryWithOptions(
		cacheKcpClusterClient,
		resyncPeriod,
	)

	metadataClusterClient, err := metadataclient.NewDynamicMetadataClusterClientForConfig(
		rest.AddUserAgent(rest.CopyConfig(m.clientConfig), "kcp-partial-metadata-informers"))
	if err != nil {
		return nil, err
	}
	// Setup apiextensions * informers
	m.apiExtensionsClusterClient, err = kcpapiextensionsclientset.NewForConfig(m.clientConfig)
	if err != nil {
		return nil, err
	}
	m.apiExtensionsSharedInformerFactory = kcpapiextensionsinformers.NewSharedInformerFactoryWithOptions(
		m.apiExtensionsClusterClient,
		resyncPeriod,
	)
	crdGVRSource, err := informer.NewCRDGVRSource(m.apiExtensionsSharedInformerFactory.Apiextensions().V1().CustomResourceDefinitions().Informer())
	if err != nil {
		return nil, err
	}
	m.discoveringDynamicSharedInformerFactory, err = informer.NewDiscoveringDynamicSharedInformerFactory(
		metadataClusterClient,
		func(obj interface{}) bool { return true },
		nil,
		crdGVRSource,
		cache.Indexers{},
	)
	if err != nil {
		return nil, err
	}

	// add tmc virtual workspaces
	virtualWorkspacesConfig := rest.CopyConfig(m.clientConfig)
	virtualWorkspacesConfig = rest.AddUserAgent(virtualWorkspacesConfig, "virtual-workspaces")

	m.virtualWorkspaces, err = m.Config.Options.TmcVirtualWorkspaces.NewVirtualWorkspaces(
		virtualWorkspacesConfig,
		m.Config.Options.VirtualWorkspaces.RootPathPrefix,
		func() string {
			return m.Config.Options.VirtualWorkspaces.ShardExternalURL
		},
		m.cacheKcpSharedInformerFactory,
		m.tmcSharedInformerFactory,
	)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// TODO (FGI): Need to define config.go with a Config struct
// including multiple sub-configs, one for each controller.
// they will get populated with NewConfig having Options as parameters

// Start starts informers and controller instances
func (m Manager) Start(ctx context.Context) error {

	logger := klog.FromContext(ctx)
	logger.V(2).Info("starting manager")

	if err := m.installApiResourceController(ctx); err != nil {
		return err
	}
	if err := m.installSyncTargetHeartbeatController(ctx); err != nil {
		return err
	}
	if err := m.installSyncTargetController(ctx); err != nil {
		return err
	}
	if err := m.installSchedulingLocationStatusController(ctx); err != nil {
		return err
	}
	if err := m.installWorkloadResourceScheduler(ctx); err != nil {
		return err
	}
	if err := m.installWorkloadSyncTargetExportController(ctx); err != nil {
		return err
	}
	// TODO(FGI): are separate controllers for replication needed?
	if err := m.installWorkloadReplicateClusterRoleControllers(ctx); err != nil {
		return err
	}
	if err := m.installWorkloadReplicateClusterRoleBindingControllers(ctx); err != nil {
		return err
	}
	if err := m.installWorkloadReplicateLogicalClusterControllers(ctx); err != nil {
		return err
	}
	if err := m.installWorkloadNamespaceScheduler(ctx); err != nil {
		return err
	}
	if err := m.installWorkloadPlacementScheduler(ctx); err != nil {
		return err
	}
	if err := m.installSchedulingPlacementController(ctx); err != nil {
		return err
	}
	if err := m.installWorkloadAPIExportController(ctx); err != nil {
		return err
	}
	if err := m.installWorkloadDefaultLocationController(ctx); err != nil {
		return err
	}

	logger.Info("starting kube informers")
	m.kubeSharedInformerFactory.Start(m.stopCh)
	m.kcpSharedInformerFactory.Start(m.stopCh)
	m.apiExtensionsSharedInformerFactory.Start(m.stopCh)
	m.cacheKcpSharedInformerFactory.Start(m.stopCh)
	m.tmcSharedInformerFactory.Start(m.stopCh)
	m.discoveringDynamicSharedInformerFactory.Start(m.stopCh)

	m.kubeSharedInformerFactory.WaitForCacheSync(m.stopCh)
	m.kcpSharedInformerFactory.WaitForCacheSync(m.stopCh)
	m.apiExtensionsSharedInformerFactory.WaitForCacheSync(m.stopCh)
	m.cacheKcpSharedInformerFactory.WaitForCacheSync(m.stopCh)
	m.tmcSharedInformerFactory.WaitForCacheSync(m.stopCh)
	logger.V(2).Info("starting dynamic metadata informer worker")
	go m.discoveringDynamicSharedInformerFactory.StartWorker(ctx)
	logger.V(2).Info("all informers synced, ready to start controllers")
	close(m.syncedCh)
	return nil
}

// restConfigForAPIExport returns a *rest.Config properly configured to communicate with the endpoint for the
// APIExport's virtual workspace.
func restConfigForAPIExport(ctx context.Context, cfg *rest.Config, apiExportName string) (*rest.Config, error) {

	logger := klog.FromContext(ctx)
	logger.V(2).Info("getting apiexport")

	bootstrapConfig := rest.CopyConfig(cfg)
	tmcVersion := version.Get().GitVersion
	rest.AddUserAgent(bootstrapConfig, "kcp#tmc/bootstrap/"+tmcVersion)
	bootstrapClient, err := kcpclientset.NewForConfig(bootstrapConfig)
	if err != nil {
		return nil, err
	}

	var apiExport *apisv1alpha1.APIExport

	if apiExportName != "" {

		if apiExport, err = bootstrapClient.ApisV1alpha1().APIExports().Get(ctx, apiExportName, metav1.GetOptions{}); err != nil {
			return nil, fmt.Errorf("error getting APIExport %q: %w", apiExportName, err)
		}
	} else {
		logger := klog.FromContext(ctx)
		logger.V(2).Info("api-export-name is empty - listing")
		exports := &apisv1alpha1.APIExportList{}
		if exports, err = bootstrapClient.ApisV1alpha1().APIExports().List(ctx, metav1.ListOptions{}); err != nil {
			return nil, fmt.Errorf("error listing APIExports: %w", err)
		}
		if len(exports.Items) == 0 {
			return nil, fmt.Errorf("no APIExport found")
		}
		if len(exports.Items) > 1 {
			return nil, fmt.Errorf("more than one APIExport found")
		}
		apiExport = &exports.Items[0]
	}

	if len(apiExport.Status.VirtualWorkspaces) < 1 {
		return nil, fmt.Errorf("APIExport %q status.virtualWorkspaces is empty", apiExportName)
	}

	cfg = rest.CopyConfig(cfg)
	// TODO(FGI): For sharding support we would need to interact with the APIExportEndpointSlice API
	// rather than APIExport. We would then have an URL per shard.
	cfg.Host = apiExport.Status.VirtualWorkspaces[0].URL

	return cfg, nil
}
