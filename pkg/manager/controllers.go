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

	"k8s.io/client-go/rest"

	kcpdynamic "github.com/kcp-dev/client-go/dynamic"
	kcpkubernetesclientset "github.com/kcp-dev/client-go/kubernetes"

	kcpclusterclientset "github.com/kcp-dev/kcp/sdk/client/clientset/versioned/cluster"
	kcpapiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/kcp/clientset/versioned"
	kcpapiextensionsinformers "k8s.io/apiextensions-apiserver/pkg/client/kcp/informers/externalversions"

	tmcclusterclientset "github.com/kcp-dev/contrib-tmc/client/clientset/versioned/cluster"
	"github.com/kcp-dev/contrib-tmc/pkg/reconciler/apiresource"
	"github.com/kcp-dev/contrib-tmc/pkg/reconciler/scheduling/location"
	schedulingplacement "github.com/kcp-dev/contrib-tmc/pkg/reconciler/scheduling/placement"
	workloadsapiexport "github.com/kcp-dev/contrib-tmc/pkg/reconciler/workload/apiexport"
	workloadsdefaultlocation "github.com/kcp-dev/contrib-tmc/pkg/reconciler/workload/defaultlocation"
	"github.com/kcp-dev/contrib-tmc/pkg/reconciler/workload/heartbeat"
	workloadnamespace "github.com/kcp-dev/contrib-tmc/pkg/reconciler/workload/namespace"
	workloadplacement "github.com/kcp-dev/contrib-tmc/pkg/reconciler/workload/placement"
	workloadreplicateclusterrole "github.com/kcp-dev/contrib-tmc/pkg/reconciler/workload/replicateclusterrole"
	workloadreplicateclusterrolebinding "github.com/kcp-dev/contrib-tmc/pkg/reconciler/workload/replicateclusterrolebinding"
	workloadreplicatelogicalcluster "github.com/kcp-dev/contrib-tmc/pkg/reconciler/workload/replicatelogicalcluster"
	workloadresource "github.com/kcp-dev/contrib-tmc/pkg/reconciler/workload/resource"

	"github.com/kcp-dev/contrib-tmc/pkg/reconciler/workload/synctarget"
	"github.com/kcp-dev/contrib-tmc/pkg/reconciler/workload/synctargetexports"
)

func (m *Manager) installApiResourceController(ctx context.Context) error {
	config := rest.CopyConfig(m.clientConfig)
	config = rest.AddUserAgent(config, apiresource.ControllerName)
	kcpClusterClient, err := kcpclusterclientset.NewForConfig(config)
	if err != nil {
		return err
	}
	// TODO (FGI): Why is crdClusterClient not using the cluster aware version of the client?
	crdClusterClient, err := kcpapiextensionsclientset.NewForConfig(config)
	if err != nil {
		return err
	}
	cfg := rest.CopyConfig(m.clientConfig)
	cfg = rest.AddUserAgent(cfg, apiresource.ControllerName)
	tmcclusterclientset, err := tmcclusterclientset.NewForConfig(cfg)
	if err != nil {
		return err
	}

	crdSharedInformerFactory := kcpapiextensionsinformers.NewSharedInformerFactoryWithOptions(crdClusterClient, resyncPeriod)

	c, err := apiresource.NewController(
		crdClusterClient,
		kcpClusterClient,
		tmcclusterclientset,
		m.tmcSharedInformerFactory.Apiresource().V1alpha1().NegotiatedAPIResources(),
		m.tmcSharedInformerFactory.Apiresource().V1alpha1().APIResourceImports(),
		crdSharedInformerFactory.Apiextensions().V1().CustomResourceDefinitions(),
	)
	if err != nil {
		return err
	}

	go func() {
		// Wait for shared informer factories to by synced.
		<-m.syncedCh
		crdSharedInformerFactory.Start(ctx.Done())
		crdSharedInformerFactory.WaitForCacheSync(ctx.Done())
		c.Start(ctx, 2)
	}()

	return nil
}

func (m *Manager) installSyncTargetHeartbeatController(ctx context.Context) error {
	config := rest.CopyConfig(m.clientConfig)
	config = rest.AddUserAgent(config, heartbeat.ControllerName)
	kcpClusterClient, err := kcpclusterclientset.NewForConfig(config)
	if err != nil {
		return err
	}
	cfg := rest.CopyConfig(m.clientConfig)
	cfg = rest.AddUserAgent(cfg, heartbeat.ControllerName)
	tmcclusterclientset, err := tmcclusterclientset.NewForConfig(cfg)
	if err != nil {
		return err
	}

	c, err := heartbeat.NewController(
		kcpClusterClient,
		tmcclusterclientset,
		m.tmcSharedInformerFactory.Workload().V1alpha1().SyncTargets(),
		m.Options.Controllers.SyncTargetHeartbeat.HeartbeatThreshold,
	)
	if err != nil {
		return err
	}

	go func() {
		// Wait for shared informer factories to by synced.
		<-m.syncedCh
		c.Start(ctx)
	}()

	return nil
}

func (m *Manager) installSyncTargetController(ctx context.Context) error {
	config := rest.CopyConfig(m.clientConfig)
	config = rest.AddUserAgent(config, synctarget.ControllerName)
	kcpClusterClient, err := kcpclusterclientset.NewForConfig(config)
	if err != nil {
		return err
	}
	cfg := rest.CopyConfig(m.clientConfig)
	cfg = rest.AddUserAgent(cfg, synctarget.ControllerName)
	tmcclusterclientset, err := tmcclusterclientset.NewForConfig(cfg)
	if err != nil {
		return err
	}

	c := synctarget.NewController(
		kcpClusterClient,
		tmcclusterclientset,
		m.tmcSharedInformerFactory.Workload().V1alpha1().SyncTargets(),
		m.kcpSharedInformerFactory.Core().V1alpha1().Shards(),
		m.cacheKcpSharedInformerFactory.Core().V1alpha1().Shards(),
	)
	if err != nil {
		return err
	}
	go func() {
		// Wait for shared informer factories to by synced.
		<-m.syncedCh
		c.Start(ctx, 2)
	}()
	return nil
}

func (m *Manager) installSchedulingLocationStatusController(ctx context.Context) error {
	// TODO (FGI): the controller name should be a property of the controller
	controllerName := "tmc-scheduling-location-status-controller"
	config := rest.CopyConfig(m.clientConfig)
	config = rest.AddUserAgent(config, controllerName)
	kcpClusterClient, err := kcpclusterclientset.NewForConfig(config)
	if err != nil {
		return err
	}
	cfg := rest.CopyConfig(m.clientConfig)
	cfg = rest.AddUserAgent(cfg, controllerName)
	tmcclusterclientset, err := tmcclusterclientset.NewForConfig(cfg)
	if err != nil {
		return err
	}

	c, err := location.NewController(
		kcpClusterClient,
		tmcclusterclientset,
		m.tmcSharedInformerFactory.Scheduling().V1alpha1().Locations(),
		m.tmcSharedInformerFactory.Workload().V1alpha1().SyncTargets(),
	)
	if err != nil {
		return err
	}
	go func() {
		// Wait for shared informer factories to by synced.
		<-m.syncedCh
		c.Start(ctx, 2)
	}()
	return nil
}

func (m *Manager) installWorkloadResourceScheduler(ctx context.Context) error {
	config := rest.CopyConfig(m.clientConfig)
	config = rest.AddUserAgent(config, workloadresource.ControllerName)
	dynamicClusterClient, err := kcpdynamic.NewForConfig(config)
	if err != nil {
		return err
	}

	resourceScheduler, err := workloadresource.NewController(
		dynamicClusterClient,
		m.discoveringDynamicSharedInformerFactory,
		m.tmcSharedInformerFactory.Workload().V1alpha1().SyncTargets(),
		m.kubeSharedInformerFactory.Core().V1().Namespaces(),
		m.tmcSharedInformerFactory.Scheduling().V1alpha1().Placements(),
	)
	if err != nil {
		return err
	}

	go func() {
		// Wait for shared informer factories to by synced.
		<-m.syncedCh
		resourceScheduler.Start(ctx, 2)
	}()
	return nil
}

func (m *Manager) installWorkloadSyncTargetExportController(ctx context.Context) error {
	config := rest.CopyConfig(m.clientConfig)
	config = rest.AddUserAgent(config, synctargetexports.ControllerName)
	kcpClusterClient, err := kcpclusterclientset.NewForConfig(config)
	if err != nil {
		return err
	}
	cfg := rest.CopyConfig(m.clientConfig)
	cfg = rest.AddUserAgent(cfg, synctargetexports.ControllerName)
	tmcclusterclientset, err := tmcclusterclientset.NewForConfig(cfg)
	if err != nil {
		return err
	}

	c, err := synctargetexports.NewController(
		kcpClusterClient,
		tmcclusterclientset,
		m.tmcSharedInformerFactory.Workload().V1alpha1().SyncTargets(),
		m.kcpSharedInformerFactory.Apis().V1alpha1().APIExports(),
		m.cacheKcpSharedInformerFactory.Apis().V1alpha1().APIExports(),
		m.kcpSharedInformerFactory.Apis().V1alpha1().APIResourceSchemas(),
		m.cacheKcpSharedInformerFactory.Apis().V1alpha1().APIResourceSchemas(),
		m.tmcSharedInformerFactory.Apiresource().V1alpha1().APIResourceImports(),
	)
	if err != nil {
		return err
	}

	go func() {
		// Wait for shared informer factories to by synced.
		<-m.syncedCh
		c.Start(ctx, 2)
	}()
	return nil
}

func (m *Manager) installWorkloadReplicateClusterRoleControllers(ctx context.Context) error {
	config := rest.CopyConfig(m.clientConfig)
	config = rest.AddUserAgent(config, workloadreplicateclusterrole.ControllerName)
	kubeClusterClient, err := kcpkubernetesclientset.NewForConfig(config)
	if err != nil {
		return err
	}

	c := workloadreplicateclusterrole.NewController(
		kubeClusterClient,
		m.kubeSharedInformerFactory.Rbac().V1().ClusterRoles(),
		m.kubeSharedInformerFactory.Rbac().V1().ClusterRoleBindings(),
	)

	go func() {
		// Wait for shared informer factories to by synced.
		<-m.syncedCh
		c.Start(ctx, 2)
	}()
	return nil
}

func (m *Manager) installWorkloadReplicateClusterRoleBindingControllers(ctx context.Context) error {
	config := rest.CopyConfig(m.clientConfig)
	config = rest.AddUserAgent(config, workloadreplicateclusterrolebinding.ControllerName)
	kubeClusterClient, err := kcpkubernetesclientset.NewForConfig(config)
	if err != nil {
		return err
	}

	c := workloadreplicateclusterrolebinding.NewController(
		kubeClusterClient,
		m.kubeSharedInformerFactory.Rbac().V1().ClusterRoleBindings(),
		m.kubeSharedInformerFactory.Rbac().V1().ClusterRoles(),
	)

	go func() {
		// Wait for shared informer factories to by synced.
		<-m.syncedCh
		c.Start(ctx, 2)
	}()
	return nil
}

func (m *Manager) installWorkloadReplicateLogicalClusterControllers(ctx context.Context) error {
	config := rest.CopyConfig(m.clientConfig)
	config = rest.AddUserAgent(config, workloadreplicatelogicalcluster.ControllerName)
	kcpClusterClient, err := kcpclusterclientset.NewForConfig(config)
	if err != nil {
		return err
	}

	c := workloadreplicatelogicalcluster.NewController(
		kcpClusterClient,
		m.kcpSharedInformerFactory.Core().V1alpha1().LogicalClusters(),
		m.tmcSharedInformerFactory.Workload().V1alpha1().SyncTargets(),
	)

	go func() {
		// Wait for shared informer factories to by synced.
		<-m.syncedCh
		c.Start(ctx, 2)
	}()
	return nil
}

func (m *Manager) installWorkloadNamespaceScheduler(ctx context.Context) error {
	config := rest.CopyConfig(m.clientConfig)
	config = rest.AddUserAgent(config, workloadnamespace.ControllerName)
	kubeClusterClient, err := kcpkubernetesclientset.NewForConfig(config)
	if err != nil {
		return err
	}

	c, err := workloadnamespace.NewController(
		kubeClusterClient,
		m.kubeSharedInformerFactory.Core().V1().Namespaces(),
		m.tmcSharedInformerFactory.Scheduling().V1alpha1().Placements(),
	)
	if err != nil {
		return err
	}

	go func() {
		// Wait for shared informer factories to by synced.
		<-m.syncedCh
		c.Start(ctx, 2)
	}()
	return nil
}

func (m *Manager) installWorkloadPlacementScheduler(ctx context.Context) error {
	config := rest.CopyConfig(m.clientConfig)
	config = rest.AddUserAgent(config, workloadplacement.ControllerName)
	kcpClusterClient, err := kcpclusterclientset.NewForConfig(config)
	if err != nil {
		return err
	}
	cfg := rest.CopyConfig(m.clientConfig)
	cfg = rest.AddUserAgent(cfg, workloadplacement.ControllerName)
	tmcclusterclientset, err := tmcclusterclientset.NewForConfig(cfg)
	if err != nil {
		return err
	}

	c, err := workloadplacement.NewController(
		kcpClusterClient,
		tmcclusterclientset,
		m.kcpSharedInformerFactory.Core().V1alpha1().LogicalClusters(),
		m.tmcSharedInformerFactory.Scheduling().V1alpha1().Locations(),
		m.tmcSharedInformerFactory.Workload().V1alpha1().SyncTargets(),
		m.tmcSharedInformerFactory.Scheduling().V1alpha1().Placements(),
		m.kcpSharedInformerFactory.Apis().V1alpha1().APIBindings(),
	)
	if err != nil {
		return err
	}

	go func() {
		// Wait for shared informer factories to by synced.
		<-m.syncedCh
		c.Start(ctx, 2)
	}()
	return nil
}

func (m *Manager) installSchedulingPlacementController(ctx context.Context) error {
	config := rest.CopyConfig(m.clientConfig)
	config = rest.AddUserAgent(config, schedulingplacement.ControllerName)

	kcpClusterClient, err := kcpclusterclientset.NewForConfig(config)
	if err != nil {
		return err
	}
	cfg := rest.CopyConfig(m.clientConfig)
	cfg = rest.AddUserAgent(cfg, schedulingplacement.ControllerName)
	tmcclusterclientset, err := tmcclusterclientset.NewForConfig(cfg)
	if err != nil {
		return err
	}

	c, err := schedulingplacement.NewController(
		kcpClusterClient,
		tmcclusterclientset,
		m.kubeSharedInformerFactory.Core().V1().Namespaces(),
		m.tmcSharedInformerFactory.Scheduling().V1alpha1().Locations(),
		m.tmcSharedInformerFactory.Scheduling().V1alpha1().Placements(),
	)
	if err != nil {
		return err
	}

	go func() {
		// Wait for shared informer factories to by synced.
		<-m.syncedCh
		c.Start(ctx, 2)
	}()
	return nil
}

func (m *Manager) installWorkloadAPIExportController(ctx context.Context) error {
	config := rest.CopyConfig(m.clientConfig)
	config = rest.AddUserAgent(config, workloadsapiexport.ControllerName)
	kcpClusterClient, err := kcpclusterclientset.NewForConfig(config)
	if err != nil {
		return err
	}

	c, err := workloadsapiexport.NewController(
		kcpClusterClient,
		m.kcpSharedInformerFactory.Apis().V1alpha1().APIExports(),
		m.kcpSharedInformerFactory.Apis().V1alpha1().APIResourceSchemas(),
		m.tmcSharedInformerFactory.Apiresource().V1alpha1().NegotiatedAPIResources(),
		m.tmcSharedInformerFactory.Workload().V1alpha1().SyncTargets(),
	)
	if err != nil {
		return err
	}

	go func() {
		// Wait for shared informer factories to by synced.
		<-m.syncedCh
		c.Start(ctx, 2)
	}()
	return nil
}

func (m *Manager) installWorkloadDefaultLocationController(ctx context.Context) error {
	config := rest.CopyConfig(m.clientConfig)
	config = rest.AddUserAgent(config, workloadsdefaultlocation.ControllerName)
	kcpClusterClient, err := kcpclusterclientset.NewForConfig(config)
	if err != nil {
		return err
	}
	cfg := rest.CopyConfig(m.clientConfig)
	cfg = rest.AddUserAgent(cfg, workloadsdefaultlocation.ControllerName)
	tmcclusterclientset, err := tmcclusterclientset.NewForConfig(cfg)
	if err != nil {
		return err
	}

	c, err := workloadsdefaultlocation.NewController(
		kcpClusterClient,
		tmcclusterclientset,
		m.tmcSharedInformerFactory.Workload().V1alpha1().SyncTargets(),
		m.tmcSharedInformerFactory.Scheduling().V1alpha1().Locations(),
	)
	if err != nil {
		return err
	}

	go func() {
		// Wait for shared informer factories to by synced.
		<-m.syncedCh
		c.Start(ctx, 2)
	}()
	return nil
}
