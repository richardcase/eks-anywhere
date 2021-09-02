package clustermanager

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/yaml"

	"github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/pkg/cluster"
	"github.com/aws/eks-anywhere/pkg/clustermarshaller"
	"github.com/aws/eks-anywhere/pkg/constants"
	"github.com/aws/eks-anywhere/pkg/filewriter"
	"github.com/aws/eks-anywhere/pkg/logger"
	"github.com/aws/eks-anywhere/pkg/providers"
	"github.com/aws/eks-anywhere/pkg/retrier"
	"github.com/aws/eks-anywhere/pkg/types"
)

const (
	maxRetries        = 30
	backOffPeriod     = 5 * time.Second
	machineMaxWait    = 10 * time.Minute
	machineBackoff    = 1 * time.Second
	machinesMinWait   = 30 * time.Minute
	moveCapiWait      = 5 * time.Minute
	logDir            = "logs"
	ctrlPlaneWaitStr  = "60m"
	etcdWaitStr       = "60m"
	deploymentWaitStr = "30m"
)

// map where key = namespace and value is a capi deployment
var capiDeployments = map[string][]string{
	"capi-kubeadm-bootstrap-system":     {"capi-kubeadm-bootstrap-controller-manager"},
	"capi-kubeadm-control-plane-system": {"capi-kubeadm-control-plane-controller-manager"},
	"capi-system":                       {"capi-controller-manager"},
	"capi-webhook-system":               {"capi-controller-manager", "capi-kubeadm-bootstrap-controller-manager", "capi-kubeadm-control-plane-controller-manager"},
	"cert-manager":                      {"cert-manager", "cert-manager-cainjector", "cert-manager-webhook"},
}

// map between file name and the capi/v deployments
var clusterDeployments = map[string]*types.Deployment{
	"kubeadm-bootstrap-controller-manager.log":         {Name: "capi-kubeadm-bootstrap-controller-manager", Namespace: "capi-kubeadm-bootstrap-system", Container: "manager"},
	"kubeadm-control-plane-controller-manager.log":     {Name: "capi-kubeadm-control-plane-controller-manager", Namespace: "capi-kubeadm-control-plane-system", Container: "manager"},
	"capi-controller-manager.log":                      {Name: "capi-controller-manager", Namespace: "capi-system", Container: "manager"},
	"wh-capi-controller-manager.log":                   {Name: "capi-controller-manager", Namespace: "capi-webhook-system", Container: "manager"},
	"wh-capi-kubeadm-bootstrap-controller-manager.log": {Name: "capi-kubeadm-bootstrap-controller-manager", Namespace: "capi-webhook-system", Container: "manager"},
	"wh-kubeadm-control-plane-controller-manager.log":  {Name: "capi-kubeadm-control-plane-controller-manager", Namespace: "capi-webhook-system", Container: "manager"},
	"cert-manager.log":                                 {Name: "cert-manager", Namespace: "cert-manager"},
	"cert-manager-cainjector.log":                      {Name: "cert-manager-cainjector", Namespace: "cert-manager"},
	"cert-manager-webhook.log":                         {Name: "cert-manager-webhook", Namespace: "cert-manager"},
	"coredns.log":                                      {Name: "coredns", Namespace: "kube-system"},
	"local-path-provisioner.log":                       {Name: "local-path-provisioner", Namespace: "local-path-storage"},
	"capv-controller-manager.log":                      {Name: "capv-controller-manager", Namespace: "capv-system", Container: "manager"},
	"wh-capv-controller-manager.log":                   {Name: "capv-controller-manager", Namespace: "capi-webhook-system", Container: "manager"},
}

type ClusterManager struct {
	clusterClient   ClusterClient
	writer          filewriter.FileWriter
	networking      Networking
	Retrier         *retrier.Retrier
	machineMaxWait  time.Duration
	machineBackoff  time.Duration
	machinesMinWait time.Duration
}

type ClusterClient interface {
	MoveManagement(ctx context.Context, org, target *types.Cluster) error
	ApplyKubeSpec(ctx context.Context, cluster *types.Cluster, kubeSpecFile string) error
	ApplyKubeSpecWithNamespace(ctx context.Context, cluster *types.Cluster, kubeSpecFile string, namespace string) error
	ApplyKubeSpecFromBytes(ctx context.Context, cluster *types.Cluster, data []byte) error
	ApplyKubeSpecFromBytesForce(ctx context.Context, cluster *types.Cluster, data []byte) error
	WaitForControlPlaneReady(ctx context.Context, cluster *types.Cluster, timeout string, newClusterName string) error
	WaitForManagedExternalEtcdReady(ctx context.Context, cluster *types.Cluster, timeout string, newClusterName string) error
	GetWorkloadKubeconfig(ctx context.Context, clusterName string, cluster *types.Cluster) ([]byte, error)
	DeleteCluster(ctx context.Context, managementCluster, clusterToDelete *types.Cluster) error
	InitInfrastructure(ctx context.Context, clusterSpec *cluster.Spec, cluster *types.Cluster, provider providers.Provider) error
	WaitForDeployment(ctx context.Context, cluster *types.Cluster, timeout string, condition string, target string, namespace string) error
	SaveLog(ctx context.Context, cluster *types.Cluster, deployment *types.Deployment, fileName string, writer filewriter.FileWriter) error
	GetMachines(ctx context.Context, cluster *types.Cluster) ([]types.Machine, error)
	GetClusters(ctx context.Context, cluster *types.Cluster) ([]types.CAPICluster, error)
	GetEksaCluster(ctx context.Context, cluster *types.Cluster) (*v1alpha1.Cluster, error)
	GetEksaVSphereDatacenterConfig(ctx context.Context, VSphereDatacenterName string, kubeconfigFile string) (*v1alpha1.VSphereDatacenterConfig, error)
	UpdateAnnotationInNamespace(ctx context.Context, resourceType, objectName string, annotations map[string]string, cluster *types.Cluster, namespace string) error
	RemoveAnnotationInNamespace(ctx context.Context, resourceType, objectName, key string, cluster *types.Cluster, namespace string) error
	GetEksaVSphereMachineConfig(ctx context.Context, VSphereDatacenterName string, kubeconfigFile string) (*v1alpha1.VSphereMachineConfig, error)
}

type Networking interface {
	GenerateManifest(clusterSpec *cluster.Spec) ([]byte, error)
}

type ClusterManagerOpt func(*ClusterManager)

func New(clusterClient ClusterClient, networking Networking, writer filewriter.FileWriter, opts ...ClusterManagerOpt) *ClusterManager {
	c := &ClusterManager{
		clusterClient:   clusterClient,
		writer:          writer,
		networking:      networking,
		Retrier:         retrier.NewWithMaxRetries(maxRetries, backOffPeriod),
		machineMaxWait:  machineMaxWait,
		machineBackoff:  machineBackoff,
		machinesMinWait: machinesMinWait,
	}

	for _, o := range opts {
		o(c)
	}

	return c
}

func WithWaitForMachines(machineBackoff, machineMaxWait, machinesMinWait time.Duration) ClusterManagerOpt {
	return func(c *ClusterManager) {
		c.machineBackoff = machineBackoff
		c.machineMaxWait = machineMaxWait
		c.machinesMinWait = machinesMinWait
	}
}

func (c *ClusterManager) MoveCapi(ctx context.Context, from, to *types.Cluster) error {
	logger.V(3).Info("Waiting for management machines to be ready before move")
	if err := c.waitForNodesReady(ctx, from); err != nil {
		return err
	}

	err := c.clusterClient.MoveManagement(ctx, from, to)
	if err != nil {
		return fmt.Errorf("error moving CAPI management from source to target: %v", err)
	}

	logger.V(3).Info("Waiting for control planes to be ready after move")
	err = c.waitForAllControlPlanes(ctx, to, moveCapiWait)
	if err != nil {
		return err
	}

	logger.V(3).Info("Waiting for machines to be ready after move")
	if err = c.waitForNodesReady(ctx, to); err != nil {
		return err
	}

	return nil
}

// CreateWorkloadCluster creates a workload cluster in the provider that the customer has specified.
// It applied the kubernetes manifest file on the management cluster, waits for the control plane to be ready,
// and then generates the kubeconfig for the cluster.
// It returns a struct of type Cluster containing the name and the kubeconfig of the cluster.
func (c *ClusterManager) CreateWorkloadCluster(ctx context.Context, managementCluster *types.Cluster, clusterSpec *cluster.Spec, provider providers.Provider) (*types.Cluster, error) {
	workloadCluster := &types.Cluster{
		Name: managementCluster.Name,
	}

	if err := c.applyCluster(ctx, managementCluster, workloadCluster, clusterSpec, provider, false); err != nil {
		return nil, err
	}

	err := cluster.ApplyExtraObjects(ctx, c.clusterClient, workloadCluster, clusterSpec)
	if err != nil {
		return nil, fmt.Errorf("error applying extra resources to workload cluster: %v", err)
	}

	return workloadCluster, nil
}

func (c *ClusterManager) generateWorkloadKubeconfig(ctx context.Context, clusterName string, cluster *types.Cluster, provider providers.Provider) (string, error) {
	fileName := fmt.Sprintf("%s-eks-a-cluster.kubeconfig", clusterName)
	kubeconfig, err := c.clusterClient.GetWorkloadKubeconfig(ctx, clusterName, cluster)
	if err != nil {
		return "", fmt.Errorf("error getting workload kubeconfig: %v", err)
	}
	if err := provider.UpdateKubeConfig(&kubeconfig, clusterName); err != nil {
		return "", err
	}

	writtenFile, err := c.writer.Write(fileName, kubeconfig, filewriter.PersistentFile, filewriter.Permission0600)
	if err != nil {
		return "", fmt.Errorf("error writing workload kubeconfig: %v", err)
	}
	return writtenFile, nil
}

func (c *ClusterManager) DeleteCluster(ctx context.Context, managementCluster, clusterToDelete *types.Cluster) error {
	return c.Retrier.Retry(
		func() error {
			return c.clusterClient.DeleteCluster(ctx, managementCluster, clusterToDelete)
		},
	)
}

func (c *ClusterManager) UpgradeCluster(ctx context.Context, managementCluster, workloadCluster *types.Cluster, clusterSpec *cluster.Spec, provider providers.Provider) error {
	if err := c.applyCluster(ctx, managementCluster, workloadCluster, clusterSpec, provider, true); err != nil {
		return err
	}

	if clusterSpec.Spec.ExternalEtcdConfiguration != nil {
		logger.V(3).Info("Waiting for external etcd to be ready after upgrade")
		if err := c.clusterClient.WaitForManagedExternalEtcdReady(ctx, managementCluster, etcdWaitStr, workloadCluster.Name); err != nil {
			return fmt.Errorf("error waiting for workload cluster etcd to be ready: %v", err)
		}
	}

	logger.V(3).Info("Waiting for control plane to be ready after upgrade")
	err := c.clusterClient.WaitForControlPlaneReady(ctx, managementCluster, ctrlPlaneWaitStr, workloadCluster.Name)
	if err != nil {
		return fmt.Errorf("error waiting for workload cluster control plane to be ready: %v", err)
	}

	logger.V(3).Info("Waiting for workload cluster capi components to be ready after upgrade")
	err = c.waitForCapi(ctx, workloadCluster, provider)
	if err != nil {
		return fmt.Errorf("error waiting for workload cluster capi components to be ready: %v", err)
	}

	err = cluster.ApplyExtraObjects(ctx, c.clusterClient, workloadCluster, clusterSpec)
	if err != nil {
		return fmt.Errorf("error applying extra resources to workload cluster: %v", err)
	}

	return nil
}

func (c *ClusterManager) EKSAClusterSpecChanged(ctx context.Context, cluster *types.Cluster, clusterSpec *cluster.Spec, datacenterConfig providers.DatacenterConfig, machineConfigs []providers.MachineConfig) (bool, error) {
	cc, err := c.clusterClient.GetEksaCluster(ctx, cluster)
	if err != nil {
		return false, err
	}

	if !reflect.DeepEqual(cc.Spec, clusterSpec.Spec) {
		logger.V(3).Info("Existing cluster and new cluster spec differ")
		return true, nil
	}
	logger.V(3).Info("Clusters are the same, checking provider spec")
	// compare provider spec
	switch cc.Spec.DatacenterRef.Kind {
	case v1alpha1.VSphereDatacenterKind:
		machineConfigMap := make(map[string]*v1alpha1.VSphereMachineConfig)

		existingVdc, err := c.clusterClient.GetEksaVSphereDatacenterConfig(ctx, cc.Spec.DatacenterRef.Name, cluster.KubeconfigFile)
		if err != nil {
			return false, err
		}
		vdc := datacenterConfig.(*v1alpha1.VSphereDatacenterConfig)
		if !reflect.DeepEqual(existingVdc.Spec, vdc.Spec) {
			logger.V(3).Info("New provider spec is different from the new spec")
			return true, nil
		}

		for _, config := range machineConfigs {
			mc := config.(*v1alpha1.VSphereMachineConfig)
			machineConfigMap[mc.Name] = mc
		}
		existingCpVmc, err := c.clusterClient.GetEksaVSphereMachineConfig(ctx, cc.Spec.ControlPlaneConfiguration.MachineGroupRef.Name, cluster.KubeconfigFile)
		if err != nil {
			return false, err
		}
		cpVmc := machineConfigMap[clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name]
		if !reflect.DeepEqual(existingCpVmc.Spec, cpVmc.Spec) {
			logger.V(3).Info("New control plane machine config spec is different from the existing spec")
			return true, nil
		}
		existingWnVmc, err := c.clusterClient.GetEksaVSphereMachineConfig(ctx, cc.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name, cluster.KubeconfigFile)
		if err != nil {
			return false, err
		}
		wnVmc := machineConfigMap[clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name]
		if !reflect.DeepEqual(existingWnVmc.Spec, wnVmc.Spec) {
			logger.V(3).Info("New worker node machine config spec is different from the existing spec")
			return true, nil
		}
		if cc.Spec.ExternalEtcdConfiguration != nil {
			existingEtcdVmc, err := c.clusterClient.GetEksaVSphereMachineConfig(ctx, cc.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name, cluster.KubeconfigFile)
			if err != nil {
				return false, err
			}
			etcdVmc := machineConfigMap[clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name]
			if !reflect.DeepEqual(existingEtcdVmc.Spec, etcdVmc.Spec) {
				logger.V(3).Info("New etcd machine config spec is different from the existing spec")
				return true, nil
			}
		}
	default:
		// Run upgrade flow
		return true, nil
	}

	return false, nil
}

func (c *ClusterManager) applyCluster(ctx context.Context, managementCluster, workloadCluster *types.Cluster, clusterSpec *cluster.Spec, provider providers.Provider, isUpgrade bool) error {
	clusterSpecFile, err := c.GenerateDeploymentFile(ctx, managementCluster, workloadCluster, clusterSpec, provider, isUpgrade)
	if err != nil {
		return fmt.Errorf("error generating workload spec: %v", err)
	}

	err = c.Retrier.Retry(
		func() error {
			return c.clusterClient.ApplyKubeSpecWithNamespace(ctx, managementCluster, clusterSpecFile, constants.EksaSystemNamespace)
		},
	)
	if err != nil {
		return fmt.Errorf("error applying workload spec: %v", err)
	}

	if clusterSpec.Spec.ExternalEtcdConfiguration != nil {
		logger.V(3).Info("Waiting for external etcd to be ready")
		err = c.clusterClient.WaitForManagedExternalEtcdReady(ctx, managementCluster, etcdWaitStr, workloadCluster.Name)
		if err != nil {
			return fmt.Errorf("error waiting for external etcd for workload cluster to be ready: %v", err)
		}
		logger.V(3).Info("External etcd is ready")
		// the condition external etcd ready if true indicates that all etcd machines are ready and the etcd cluster is ready to accept requests
	}

	logger.V(3).Info("Waiting for control plane to be ready")
	err = c.clusterClient.WaitForControlPlaneReady(ctx, managementCluster, ctrlPlaneWaitStr, workloadCluster.Name)
	if err != nil {
		return fmt.Errorf("error waiting for workload cluster control plane to be ready: %v", err)
	}

	if !isUpgrade {
		err = c.Retrier.Retry(
			func() error {
				workloadCluster.KubeconfigFile, err = c.generateWorkloadKubeconfig(ctx, workloadCluster.Name, managementCluster, provider)
				return err
			},
		)

		if err != nil {
			return fmt.Errorf("error generating workload kubeconfig: %v", err)
		}
	}

	logger.V(3).Info("Waiting for controlplane and worker machines to be ready")
	if err = c.waitForNodesReady(ctx, managementCluster); err != nil {
		return err
	}
	return nil
}

func (c *ClusterManager) InstallCapi(ctx context.Context, clusterSpec *cluster.Spec, cluster *types.Cluster, provider providers.Provider) error {
	err := c.clusterClient.InitInfrastructure(ctx, clusterSpec, cluster, provider)
	if err != nil {
		return fmt.Errorf("error initializing capi resources in cluster: %v", err)
	}

	return c.waitForCapi(ctx, cluster, provider)
}

func (c *ClusterManager) waitForCapi(ctx context.Context, cluster *types.Cluster, provider providers.Provider) error {
	err := c.waitForDeployments(ctx, capiDeployments, cluster)
	if err != nil {
		return err
	}

	err = c.waitForDeployments(ctx, provider.GetDeployments(), cluster)
	if err != nil {
		return err
	}

	return nil
}

func (c *ClusterManager) InstallNetworking(ctx context.Context, cluster *types.Cluster, clusterSpec *cluster.Spec) error {
	networkingManifestContent, err := c.networking.GenerateManifest(clusterSpec)
	if err != nil {
		return fmt.Errorf("error generating networking manifest: %v", err)
	}
	err = c.Retrier.Retry(
		func() error {
			return c.clusterClient.ApplyKubeSpecFromBytes(ctx, cluster, networkingManifestContent)
		},
	)
	if err != nil {
		return fmt.Errorf("error applying networking manifest spec: %v", err)
	}
	return nil
}

func (c *ClusterManager) InstallStorageClass(ctx context.Context, cluster *types.Cluster, provider providers.Provider) error {
	storageClass := provider.GenerateStorageClass()
	if storageClass == nil {
		return nil
	}

	err := c.Retrier.Retry(
		func() error {
			return c.clusterClient.ApplyKubeSpecFromBytes(ctx, cluster, storageClass)
		},
	)
	if err != nil {
		return fmt.Errorf("error applying storage class manifest: %v", err)
	}
	return nil
}

func (c *ClusterManager) InstallMachineHealthChecks(ctx context.Context, workloadCluster *types.Cluster, provider providers.Provider) error {
	mhc, err := provider.GenerateMHC()
	if err != nil {
		return err
	}
	if len(mhc) == 0 {
		logger.V(4).Info("Skipping machine health checks")
		return nil
	}
	err = c.Retrier.Retry(
		func() error {
			return c.clusterClient.ApplyKubeSpecFromBytes(ctx, workloadCluster, mhc)
		},
	)
	if err != nil {
		return fmt.Errorf("error applying machine health checks: %v", err)
	}
	return nil
}

func (c *ClusterManager) SaveLogs(ctx context.Context, cluster *types.Cluster) error {
	if c == nil || cluster == nil {
		return nil
	}
	var wg sync.WaitGroup
	wg.Add(len(clusterDeployments))

	w, err := c.writer.WithDir(logDir)
	if err != nil {
		return err
	}
	for fileName, deployment := range clusterDeployments {
		go func(dep *types.Deployment, f string) {
			// Ignoring error for now
			defer wg.Done()
			err := c.clusterClient.SaveLog(ctx, cluster, dep, f, w)
			if err != nil {
				logger.V(5).Info("Error saving logs", "error", err)
			}
		}(deployment, fileName)
	}
	wg.Wait()
	return nil
}

func (c *ClusterManager) waitForDeployments(ctx context.Context, deploymentsByNamespace map[string][]string, cluster *types.Cluster) error {
	for namespace, deployments := range deploymentsByNamespace {
		for _, deployment := range deployments {
			err := c.clusterClient.WaitForDeployment(ctx, cluster, deploymentWaitStr, "Available", deployment, namespace)
			if err != nil {
				return fmt.Errorf("error waiting for %s in namespace %s: %v", deployment, namespace, err)
			}
		}
	}
	return nil
}

func (c *ClusterManager) GenerateDeploymentFile(ctx context.Context, bootstrapCluster, workloadCluster *types.Cluster, clusterSpec *cluster.Spec, provider providers.Provider, isUpgrade bool) (string, error) {
	fileName := fmt.Sprintf("%s-eks-a-cluster.yaml", clusterSpec.ObjectMeta.Name)
	if !clusterSpec.HasOverrideClusterSpecFile() {
		if isUpgrade {
			return provider.GenerateDeploymentFileForUpgrade(ctx, bootstrapCluster, workloadCluster, clusterSpec, fileName)
		}
		return provider.GenerateDeploymentFileForCreate(ctx, workloadCluster, clusterSpec, fileName)
	}
	fileContent, err := clusterSpec.ReadOverrideClusterSpecFile()
	if err != nil {
		return "", err
	}
	writtenFile, err := c.writer.Write(fileName, []byte(fileContent))
	if err != nil {
		return "", fmt.Errorf("error creating cluster config file: %v", err)
	}
	return writtenFile, nil
}

func (c *ClusterManager) waitForNodesReady(ctx context.Context, managementCluster *types.Cluster) error {
	readyNodes, totalNodes := 0, 0
	policy := func(_ int, _ error) (bool, time.Duration) {
		return true, c.machineBackoff * time.Duration(totalNodes-readyNodes)
	}

	areNodesReady := func() error {
		var err error
		readyNodes, totalNodes, err = c.countNodesReady(ctx, managementCluster)
		if err != nil {
			return err
		}

		if readyNodes != totalNodes {
			logger.V(4).Info("Nodes are not ready yet", "total", totalNodes, "ready", readyNodes)
			return errors.New("nodes are not ready yet")
		}

		logger.V(4).Info("Nodes ready", "total", totalNodes)
		return nil
	}

	err := areNodesReady()
	if err == nil {
		return nil
	}

	timeout := time.Duration(totalNodes) * c.machineMaxWait
	if timeout <= c.machinesMinWait {
		timeout = c.machinesMinWait
	}

	r := retrier.New(timeout, retrier.WithRetryPolicy(policy))
	if err := r.Retry(areNodesReady); err != nil {
		return fmt.Errorf("retries exhausted waiting for machines to be ready: %v", err)
	}

	return nil
}

func (c *ClusterManager) countNodesReady(ctx context.Context, managementCluster *types.Cluster) (ready, total int, err error) {
	machines, err := c.clusterClient.GetMachines(ctx, managementCluster)
	if err != nil {
		return 0, 0, fmt.Errorf("error getting machines resources from management cluster: %v", err)
	}

	ready = 0
	var controlPlaneNodesCount, workerNodesCount int
	for _, m := range machines {
		// Extracted from cluster-api: NodeRef is considered a better signal than InfrastructureReady,
		// because it ensures the node in the workload cluster is up and running.
		if m.Status.NodeRef != nil {
			ready += 1
		}
		if _, ok := m.Metadata.Labels[clusterv1.MachineControlPlaneLabelName]; ok {
			controlPlaneNodesCount++
		}
		if _, ok := m.Metadata.Labels[clusterv1.MachineDeploymentLabelName]; ok {
			workerNodesCount++
		}
	}
	total = controlPlaneNodesCount + workerNodesCount
	return ready, total, nil
}

func (c *ClusterManager) waitForAllControlPlanes(ctx context.Context, cluster *types.Cluster, waitForCluster time.Duration) error {
	clusters, err := c.clusterClient.GetClusters(ctx, cluster)
	if err != nil {
		return fmt.Errorf("error getting clusters: %v", err)
	}

	for _, clu := range clusters {
		err = c.clusterClient.WaitForControlPlaneReady(ctx, cluster, waitForCluster.String(), clu.Metadata.Name)
		if err != nil {
			return fmt.Errorf("error waiting for workload cluster control plane for cluster %s to be ready: %v", clu.Metadata.Name, err)
		}
	}

	return nil
}

func (c *ClusterManager) InstallCustomComponents(ctx context.Context, clusterSpec *cluster.Spec, cluster *types.Cluster) error {
	componentsManifest, err := clusterSpec.LoadManifest(clusterSpec.VersionsBundle.Eksa.Components)
	if err != nil {
		return fmt.Errorf("failed loading manifest for eksa components: %v", err)
	}

	err = c.Retrier.Retry(
		func() error {
			return c.clusterClient.ApplyKubeSpecFromBytes(ctx, cluster, componentsManifest.Content)
		},
	)
	if err != nil {
		return fmt.Errorf("error applying eks-a components spec: %v", err)
	}
	return nil
}

func (c *ClusterManager) CreateEKSAResources(ctx context.Context, cluster *types.Cluster, clusterSpec *cluster.Spec,
	datacenterConfig providers.DatacenterConfig, machineConfigs []providers.MachineConfig) error {
	resourcesSpec, err := clustermarshaller.MarshalClusterSpec(clusterSpec, datacenterConfig, machineConfigs)
	if err != nil {
		return err
	}
	logger.V(4).Info("Applying eksa yaml resources to cluster")
	logger.V(6).Info(string(resourcesSpec))
	err = c.applyResource(ctx, cluster, resourcesSpec)
	if err != nil {
		return err
	}
	return c.applyVersionBundle(ctx, clusterSpec, cluster)
}

func (c *ClusterManager) applyVersionBundle(ctx context.Context, clusterSpec *cluster.Spec, cluster *types.Cluster) error {
	clusterSpec.Bundles.Name = clusterSpec.Name
	bundleObj, err := yaml.Marshal(clusterSpec.Bundles)
	if err != nil {
		return fmt.Errorf("error outputting bundle yaml: %v", err)
	}
	err = c.Retrier.Retry(
		func() error {
			return c.clusterClient.ApplyKubeSpecFromBytes(ctx, cluster, bundleObj)
		},
	)
	if err != nil {
		return fmt.Errorf("error applying bundle spec: %v", err)
	}
	return nil
}

func (c *ClusterManager) PauseEKSAControllerReconcile(ctx context.Context, cluster *types.Cluster, clusterSpec *cluster.Spec, provider providers.Provider) error {
	pausedAnnotation := map[string]string{clusterSpec.PausedAnnotation(): "true"}
	err := c.Retrier.Retry(
		func() error {
			return c.clusterClient.UpdateAnnotationInNamespace(ctx, provider.DatacenterResourceType(), clusterSpec.Spec.DatacenterRef.Name, pausedAnnotation, cluster, "")
		},
	)
	if err != nil {
		return fmt.Errorf("error updating annotation when pausing datacenterconfig reconciliation: %v", err)
	}
	if provider.MachineResourceType() != "" {
		if clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef == nil {
			return fmt.Errorf("machineGroupRef for control plane is not defined")
		}
		if len(clusterSpec.Spec.WorkerNodeGroupConfigurations) <= 0 || clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef == nil {
			return fmt.Errorf("machineGroupRef for worker nodes is not defined")
		}
		if clusterSpec.Spec.ExternalEtcdConfiguration != nil && clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef == nil {
			return fmt.Errorf("machineGroupRef for etcd machines is not defined")
		}
		err := c.Retrier.Retry(
			func() error {
				return c.clusterClient.UpdateAnnotationInNamespace(ctx, provider.MachineResourceType(), clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name, pausedAnnotation, cluster, "")
			},
		)
		if err != nil {
			return fmt.Errorf("error updating annotation when pausing control plane machineconfig reconciliation: %v", err)
		}
		if clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name != clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name {
			err := c.Retrier.Retry(
				func() error {
					return c.clusterClient.UpdateAnnotationInNamespace(ctx, provider.MachineResourceType(), clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name, pausedAnnotation, cluster, "")
				},
			)
			if err != nil {
				return fmt.Errorf("error updating annotation when pausing worker node machineconfig reconciliation: %v", err)
			}
		}
		if clusterSpec.Spec.ExternalEtcdConfiguration != nil {
			if clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name != clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name {
				// etcd machines have a separate machineGroupRef which hasn't been paused yet, so apply pause annotation
				err := c.Retrier.Retry(
					func() error {
						return c.clusterClient.UpdateAnnotationInNamespace(ctx, provider.MachineResourceType(), clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name, pausedAnnotation, cluster, "")
					},
				)
				if err != nil {
					return fmt.Errorf("error updating annotation when pausing etcd machineconfig reconciliation: %v", err)
				}
			}
		}
	}

	err = c.Retrier.Retry(
		func() error {
			return c.clusterClient.UpdateAnnotationInNamespace(ctx, clusterSpec.ResourceType(), cluster.Name, pausedAnnotation, cluster, "")
		},
	)
	if err != nil {
		return fmt.Errorf("error updating annotation when pausing cluster reconciliation: %v", err)
	}
	return nil
}

func (c *ClusterManager) ResumeEKSAControllerReconcile(ctx context.Context, cluster *types.Cluster, clusterSpec *cluster.Spec, provider providers.Provider) error {
	pausedAnnotation := clusterSpec.PausedAnnotation()
	err := c.Retrier.Retry(
		func() error {
			return c.clusterClient.RemoveAnnotationInNamespace(ctx, provider.DatacenterResourceType(), clusterSpec.Spec.DatacenterRef.Name, pausedAnnotation, cluster, "")
		},
	)
	if err != nil {
		return fmt.Errorf("error updating annotation when unpausing datacenterconfig reconciliation: %v", err)
	}
	if provider.MachineResourceType() != "" {
		if clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef == nil {
			return fmt.Errorf("machineGroupRef for control plane is not defined")
		}
		if len(clusterSpec.Spec.WorkerNodeGroupConfigurations) <= 0 || clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef == nil {
			return fmt.Errorf("machineGroupRef for worker nodes is not defined")
		}
		if clusterSpec.Spec.ExternalEtcdConfiguration != nil && clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef == nil {
			return fmt.Errorf("machineGroupRef for etcd machines is not defined")
		}
		err := c.Retrier.Retry(
			func() error {
				return c.clusterClient.RemoveAnnotationInNamespace(ctx, provider.MachineResourceType(), clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name, pausedAnnotation, cluster, "")
			},
		)
		if err != nil {
			return fmt.Errorf("error updating annotation when unpausing control plane machineconfig reconciliation: %v", err)
		}
		if clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name != clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name {
			err := c.Retrier.Retry(
				func() error {
					return c.clusterClient.RemoveAnnotationInNamespace(ctx, provider.MachineResourceType(), clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name, pausedAnnotation, cluster, "")
				},
			)
			if err != nil {
				return fmt.Errorf("error updating annotation when unpausing worker node machineconfig reconciliation: %v", err)
			}
		}
		if clusterSpec.Spec.ExternalEtcdConfiguration != nil {
			if clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name != clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name {
				// etcd machines have a separate machineGroupRef which hasn't been resumed yet, so apply pause annotation with false value
				err := c.Retrier.Retry(
					func() error {
						return c.clusterClient.RemoveAnnotationInNamespace(ctx, provider.MachineResourceType(), clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name, pausedAnnotation, cluster, "")
					},
				)
				if err != nil {
					return fmt.Errorf("error updating annotation when resuming etcd machineconfig reconciliation: %v", err)
				}
			}
		}
	}

	err = c.Retrier.Retry(
		func() error {
			return c.clusterClient.RemoveAnnotationInNamespace(ctx, clusterSpec.ResourceType(), cluster.Name, pausedAnnotation, cluster, "")
		},
	)
	if err != nil {
		return fmt.Errorf("error updating annotation when unpausing cluster reconciliation: %v", err)
	}
	// clear pause annotation
	clusterSpec.ClearPauseAnnotation()
	provider.DatacenterConfig().ClearPauseAnnotation()
	return nil
}

func (c *ClusterManager) applyResource(ctx context.Context, cluster *types.Cluster, resourcesSpec []byte) error {
	err := c.Retrier.Retry(
		func() error {
			return c.clusterClient.ApplyKubeSpecFromBytesForce(ctx, cluster, resourcesSpec)
		},
	)
	if err != nil {
		return fmt.Errorf("error applying eks-a spec: %v", err)
	}
	return nil
}