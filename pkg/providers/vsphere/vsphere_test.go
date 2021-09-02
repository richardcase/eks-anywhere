package vsphere

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"path"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/golang/mock/gomock"
	etcdv1alpha3 "github.com/mrajashree/etcdadm-controller/api/v1alpha3"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api/api/v1alpha3"
	kubeadmnv1alpha3 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha3"

	"github.com/aws/eks-anywhere/internal/test"
	"github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/pkg/cluster"
	"github.com/aws/eks-anywhere/pkg/executables"
	"github.com/aws/eks-anywhere/pkg/providers/vsphere/mocks"
	"github.com/aws/eks-anywhere/pkg/types"
	releasev1alpha1 "github.com/aws/eks-anywhere/release/api/v1alpha1"
)

const (
	testClusterConfigMainFilename = "cluster_main.yaml"
	testDataDir                   = "testdata"
	expectedVSphereName           = "vsphere"
	expectedVSphereUsername       = "vsphere_username"
	expectedVSpherePassword       = "vsphere_password"
	expectedVSphereServer         = "vsphere_server"
	expectedExpClusterResourceSet = "expClusterResourceSetKey"
	eksd119Release                = "kubernetes-1-19-eks-4"
	eksd119ReleaseTag             = "eksdRelease:kubernetes-1-19-eks-4"
	eksd121ReleaseTag             = "eksdRelease:kubernetes-1-21-eks-4"
	ubuntuOSTag                   = "os:ubuntu"
	bottlerocketOSTag             = "os:bottlerocket"
	testTemplate                  = "/SDDC-Datacenter/vm/Templates/ubuntu-1804-kube-v1.19.6"
)

type DummyProviderGovcClient struct {
	osTag string
}

func NewDummyProviderGovcClient() *DummyProviderGovcClient {
	return &DummyProviderGovcClient{osTag: ubuntuOSTag}
}

func (pc *DummyProviderGovcClient) TemplateHasSnapshot(ctx context.Context, template string) (bool, error) {
	return false, nil
}

func (pc *DummyProviderGovcClient) GetWorkloadAvailableSpace(ctx context.Context, machineConfig *v1alpha1.VSphereMachineConfig) (float64, error) {
	return math.MaxFloat64, nil
}

func (pc *DummyProviderGovcClient) DeployTemplate(ctx context.Context, datacenterConfig *v1alpha1.VSphereDatacenterConfig) error {
	return nil
}

func (pc *DummyProviderGovcClient) ValidateVCenterSetup(ctx context.Context, datacenterConfig *v1alpha1.VSphereDatacenterConfig, selfSigned *bool) error {
	return nil
}

func (pc *DummyProviderGovcClient) ValidateVCenterSetupMachineConfig(ctx context.Context, datacenterConfig *v1alpha1.VSphereDatacenterConfig, machineConfig *v1alpha1.VSphereMachineConfig, selfSigned *bool) error {
	return nil
}

func (pc *DummyProviderGovcClient) SearchTemplate(ctx context.Context, datacenter string, machineConfig *v1alpha1.VSphereMachineConfig) (string, error) {
	return machineConfig.Spec.Template, nil
}

func (pc *DummyProviderGovcClient) LibraryElementExists(ctx context.Context, library string) (bool, error) {
	return true, nil
}

func (pc *DummyProviderGovcClient) CreateLibrary(ctx context.Context, datastore, library string) error {
	return nil
}

func (pc *DummyProviderGovcClient) DeployTemplateFromLibrary(ctx context.Context, templateDir, templateName, library, resourcePool string, resizeDisk2 bool) error {
	return nil
}

func (pc *DummyProviderGovcClient) ResizeDisk(ctx context.Context, template, diskName string, diskSizeInGB int) error {
	return nil
}

func (pc *DummyProviderGovcClient) ImportTemplate(ctx context.Context, library, ovaURL, name string) error {
	return nil
}

func (pc *DummyProviderGovcClient) GetTags(ctx context.Context, path string) (tags []string, err error) {
	return []string{eksd119ReleaseTag, eksd121ReleaseTag, pc.osTag}, nil
}

func (pc *DummyProviderGovcClient) ListTags(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (pc *DummyProviderGovcClient) CreateTag(ctx context.Context, tag, category string) error {
	return nil
}

func (pc *DummyProviderGovcClient) AddTag(ctx context.Context, path, tag string) error {
	return nil
}

func (pc *DummyProviderGovcClient) ListCategories(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (pc *DummyProviderGovcClient) CreateCategoryForVM(ctx context.Context, name string) error {
	return nil
}

type DummyNetClient struct{}

func (n *DummyNetClient) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	// add dummy case for coverage
	if address == "255.255.255.255:22" {
		return &net.IPConn{}, nil
	}
	return nil, errors.New("")
}

func givenClusterConfig(t *testing.T, fileName string) *v1alpha1.Cluster {
	return givenClusterSpec(t, fileName).Cluster
}

func givenClusterSpec(t *testing.T, fileName string) *cluster.Spec {
	return test.NewFullClusterSpec(t, path.Join(testDataDir, fileName))
}

func givenEmptyClusterSpec() *cluster.Spec {
	return test.NewClusterSpec(func(s *cluster.Spec) {
		s.VersionsBundle.KubeVersion = "1.19"
		s.VersionsBundle.EksD.Name = eksd119Release
	})
}

func fillClusterSpecWithClusterConfig(spec *cluster.Spec, clusterConfig *v1alpha1.Cluster) {
	spec.Spec = clusterConfig.Spec
}

func givenDatacenterConfig(t *testing.T, fileName string) *v1alpha1.VSphereDatacenterConfig {
	datacenterConfig, err := v1alpha1.GetVSphereDatacenterConfig(path.Join(testDataDir, fileName))
	if err != nil {
		t.Fatalf("unable to get datacenter config from file: %v", err)
	}
	return datacenterConfig
}

func givenMachineConfigs(t *testing.T, fileName string) map[string]*v1alpha1.VSphereMachineConfig {
	machineConfigs, err := v1alpha1.GetVSphereMachineConfigs(path.Join(testDataDir, fileName))
	if err != nil {
		t.Fatalf("unable to get machine configs from file")
	}
	return machineConfigs
}

func givenProvider(t *testing.T) *vsphereProvider {
	clusterConfig := givenClusterConfig(t, testClusterConfigMainFilename)
	datacenterConfig := givenDatacenterConfig(t, testClusterConfigMainFilename)
	machineConfigs := givenMachineConfigs(t, testClusterConfigMainFilename)
	_, writer := test.NewWriter(t)
	provider := NewProviderCustomNet(datacenterConfig, machineConfigs, clusterConfig, NewDummyProviderGovcClient(), nil, writer, &DummyNetClient{}, test.FakeNow, false)
	if provider == nil {
		t.Fatalf("provider object is nil")
	}
	return provider
}

func givenGovcMock(t *testing.T) *mocks.MockProviderGovcClient {
	ctrl := gomock.NewController(t)
	return mocks.NewMockProviderGovcClient(ctrl)
}

type testContext struct {
	oldUsername                string
	isUsernameSet              bool
	oldPassword                string
	isPasswordSet              bool
	oldServername              string
	isServernameSet            bool
	oldExpClusterResourceSet   string
	isExpClusterResourceSetSet bool
}

func (tctx *testContext) SaveContext() {
	tctx.oldUsername, tctx.isUsernameSet = os.LookupEnv(eksavSphereUsernameKey)
	tctx.oldPassword, tctx.isPasswordSet = os.LookupEnv(eksavSpherePasswordKey)
	tctx.oldServername, tctx.isServernameSet = os.LookupEnv(vSpherePasswordKey)
	tctx.oldExpClusterResourceSet, tctx.isExpClusterResourceSetSet = os.LookupEnv(vSpherePasswordKey)
	os.Setenv(eksavSphereUsernameKey, expectedVSphereUsername)
	os.Setenv(vSphereUsernameKey, os.Getenv(eksavSphereUsernameKey))
	os.Setenv(eksavSpherePasswordKey, expectedVSpherePassword)
	os.Setenv(vSpherePasswordKey, os.Getenv(eksavSpherePasswordKey))
	os.Setenv(vSphereServerKey, expectedVSphereServer)
	os.Setenv(expClusterResourceSetKey, expectedExpClusterResourceSet)
}

func (tctx *testContext) RestoreContext() {
	if tctx.isUsernameSet {
		os.Setenv(eksavSphereUsernameKey, tctx.oldUsername)
	} else {
		os.Unsetenv(eksavSphereUsernameKey)
	}
	if tctx.isPasswordSet {
		os.Setenv(eksavSpherePasswordKey, tctx.oldPassword)
	} else {
		os.Unsetenv(eksavSpherePasswordKey)
	}
}

func setupContext(t *testing.T) {
	var tctx testContext
	tctx.SaveContext()
	t.Cleanup(func() {
		tctx.RestoreContext()
	})
}

func TestNewProvider(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	clusterConfig := givenClusterConfig(t, testClusterConfigMainFilename)
	datacenterConfig := givenDatacenterConfig(t, testClusterConfigMainFilename)
	machineConfigs := givenMachineConfigs(t, testClusterConfigMainFilename)
	_, writer := test.NewWriter(t)
	kubectl := mocks.NewMockProviderKubectlClient(mockCtrl)
	provider := NewProviderCustomNet(datacenterConfig, machineConfigs, clusterConfig, NewDummyProviderGovcClient(), kubectl, writer, &DummyNetClient{}, test.FakeNow, false)

	if provider == nil {
		t.Fatalf("provider object is nil")
	}
}

func TestProviderGenerateDeploymentFileUpgradeCmdUpdateMachineTemplate(t *testing.T) {
	tests := []struct {
		testName           string
		clusterconfigFile  string
		wantDeploymentFile string
	}{
		{
			testName:           "minimal",
			clusterconfigFile:  "cluster_minimal.yaml",
			wantDeploymentFile: "testdata/expected_results_minimal.yaml",
		},
		{
			testName:           "with minimal oidc",
			clusterconfigFile:  "cluster_minimal_oidc.yaml",
			wantDeploymentFile: "testdata/expected_results_minimal_oidc.yaml",
		},
		{
			testName:           "with full oidc",
			clusterconfigFile:  "cluster_full_oidc.yaml",
			wantDeploymentFile: "testdata/expected_results_full_oidc.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			var tctx testContext
			tctx.SaveContext()
			defer tctx.RestoreContext()
			ctx := context.Background()
			kubectl := mocks.NewMockProviderKubectlClient(mockCtrl)
			cluster := &types.Cluster{
				Name: "test",
			}
			bootstrapCluster := &types.Cluster{
				Name: "bootstrap-test",
			}
			clusterSpec := givenClusterSpec(t, tt.clusterconfigFile)
			vsphereDatacenter := &v1alpha1.VSphereDatacenterConfig{
				Spec: v1alpha1.VSphereDatacenterConfigSpec{},
			}
			vsphereMachineConfig := &v1alpha1.VSphereMachineConfig{
				Spec: v1alpha1.VSphereMachineConfigSpec{},
			}

			kubectl.EXPECT().GetEksaCluster(ctx, cluster).Return(clusterSpec.Cluster, nil)
			kubectl.EXPECT().GetEksaVSphereDatacenterConfig(ctx, cluster.Name, cluster.KubeconfigFile).Return(vsphereDatacenter, nil)
			kubectl.EXPECT().GetEksaVSphereMachineConfig(ctx, clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name, cluster.KubeconfigFile).Return(vsphereMachineConfig, nil)
			kubectl.EXPECT().GetEksaVSphereMachineConfig(ctx, clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name, cluster.KubeconfigFile).Return(vsphereMachineConfig, nil)
			datacenterConfig := givenDatacenterConfig(t, tt.clusterconfigFile)
			machineConfigs := givenMachineConfigs(t, tt.clusterconfigFile)
			_, writer := test.NewWriter(t)
			provider := NewProviderCustomNet(datacenterConfig, machineConfigs, clusterSpec.Cluster, NewDummyProviderGovcClient(), kubectl, writer, &DummyNetClient{}, test.FakeNow, false)
			if provider == nil {
				t.Fatalf("provider object is nil")
			}

			err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
			if err != nil {
				t.Fatalf("failed to setup and validate: %v", err)
			}

			fileName := fmt.Sprintf("%s-eks-a-cluster.yaml", clusterSpec.ObjectMeta.Name)
			writtenFile, err := provider.GenerateDeploymentFileForUpgrade(context.Background(), bootstrapCluster, cluster, clusterSpec, fileName)
			if err != nil {
				t.Fatalf("failed to generate deployment file: %v", err)
			}
			if fileName == "" {
				t.Fatalf("empty fileName returned by GenerateDeploymentFile")
			}
			test.AssertFilesEquals(t, writtenFile, tt.wantDeploymentFile)
		})
	}
}

func TestProviderGenerateDeploymentFileUpgradeCmdUpdateMachineTemplateExternalEtcd(t *testing.T) {
	tests := []struct {
		testName           string
		clusterconfigFile  string
		wantDeploymentFile string
	}{
		{
			testName:           "main",
			clusterconfigFile:  testClusterConfigMainFilename,
			wantDeploymentFile: "testdata/expected_results_main.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			var tctx testContext
			tctx.SaveContext()
			defer tctx.RestoreContext()
			ctx := context.Background()
			kubectl := mocks.NewMockProviderKubectlClient(mockCtrl)
			cluster := &types.Cluster{
				Name: "test",
			}
			bootstrapCluster := &types.Cluster{
				Name: "bootstrap-test",
			}
			clusterSpec := givenClusterSpec(t, tt.clusterconfigFile)
			vsphereDatacenter := &v1alpha1.VSphereDatacenterConfig{
				Spec: v1alpha1.VSphereDatacenterConfigSpec{},
			}
			vsphereMachineConfig := &v1alpha1.VSphereMachineConfig{
				Spec: v1alpha1.VSphereMachineConfigSpec{},
			}

			kubectl.EXPECT().GetEksaCluster(ctx, cluster).Return(clusterSpec.Cluster, nil)
			kubectl.EXPECT().GetEksaVSphereDatacenterConfig(ctx, cluster.Name, cluster.KubeconfigFile).Return(vsphereDatacenter, nil)
			kubectl.EXPECT().GetEksaVSphereMachineConfig(ctx, clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name, cluster.KubeconfigFile).Return(vsphereMachineConfig, nil)
			kubectl.EXPECT().GetEksaVSphereMachineConfig(ctx, clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name, cluster.KubeconfigFile).Return(vsphereMachineConfig, nil)
			kubectl.EXPECT().GetEksaVSphereMachineConfig(ctx, clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name, cluster.KubeconfigFile).Return(vsphereMachineConfig, nil)
			kubectl.EXPECT().UpdateAnnotation(ctx, "etcdadmcluster", fmt.Sprintf("%s-etcd", cluster.Name), map[string]string{etcdv1alpha3.UpgradeInProgressAnnotation: "true"}, gomock.AssignableToTypeOf(executables.WithCluster(bootstrapCluster)))
			datacenterConfig := givenDatacenterConfig(t, tt.clusterconfigFile)
			machineConfigs := givenMachineConfigs(t, tt.clusterconfigFile)
			_, writer := test.NewWriter(t)
			provider := NewProviderCustomNet(datacenterConfig, machineConfigs, clusterSpec.Cluster, NewDummyProviderGovcClient(), kubectl, writer, &DummyNetClient{}, test.FakeNow, false)
			if provider == nil {
				t.Fatalf("provider object is nil")
			}

			err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
			if err != nil {
				t.Fatalf("failed to setup and validate: %v", err)
			}

			fileName := fmt.Sprintf("%s-eks-a-cluster.yaml", clusterSpec.ObjectMeta.Name)
			writtenFile, err := provider.GenerateDeploymentFileForUpgrade(context.Background(), bootstrapCluster, cluster, clusterSpec, fileName)
			if err != nil {
				t.Fatalf("failed to generate deployment file: %v", err)
			}
			if fileName == "" {
				t.Fatalf("empty fileName returned by GenerateDeploymentFile")
			}
			test.AssertFilesEquals(t, writtenFile, tt.wantDeploymentFile)
		})
	}
}

func TestProviderGenerateDeploymentFileCreateCmdSystemdCgroupForK8sVersion(t *testing.T) {
	tests := []struct {
		testName           string
		clusterconfigFile  string
		wantDeploymentFile string
	}{
		{
			testName:           "main",
			clusterconfigFile:  "cluster_main_121.yaml",
			wantDeploymentFile: "testdata/expected_results_121.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			var tctx testContext
			tctx.SaveContext()
			defer tctx.RestoreContext()
			ctx := context.Background()
			kubectl := mocks.NewMockProviderKubectlClient(mockCtrl)
			cluster := &types.Cluster{
				Name: "test",
			}
			clusterSpec := givenClusterSpec(t, tt.clusterconfigFile)

			datacenterConfig := givenDatacenterConfig(t, tt.clusterconfigFile)
			machineConfigs := givenMachineConfigs(t, tt.clusterconfigFile)
			_, writer := test.NewWriter(t)
			provider := NewProviderCustomNet(datacenterConfig, machineConfigs, clusterSpec.Cluster, NewDummyProviderGovcClient(), kubectl, writer, &DummyNetClient{}, test.FakeNow, false)
			if provider == nil {
				t.Fatalf("provider object is nil")
			}

			err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
			if err != nil {
				t.Fatalf("failed to setup and validate: %v", err)
			}

			fileName := fmt.Sprintf("%s-eks-a-cluster.yaml", clusterSpec.ObjectMeta.Name)
			writtenFile, err := provider.GenerateDeploymentFileForCreate(context.Background(), cluster, clusterSpec, fileName)
			if err != nil {
				t.Fatalf("failed to generate deployment file: %v", err)
			}
			if fileName == "" {
				t.Fatalf("empty fileName returned by GenerateDeploymentFile")
			}
			test.AssertFilesEquals(t, writtenFile, tt.wantDeploymentFile)
		})
	}
}

func TestProviderGenerateDeploymentFileUpgradeCmdNotUpdateMachineTemplate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	var tctx testContext
	tctx.SaveContext()
	defer tctx.RestoreContext()
	ctx := context.Background()
	kubectl := mocks.NewMockProviderKubectlClient(mockCtrl)
	cluster := &types.Cluster{
		Name: "test",
	}
	bootstrapCluster := &types.Cluster{
		Name: "bootstrap-test",
	}
	clusterSpec := givenClusterSpec(t, testClusterConfigMainFilename)

	cp := &kubeadmnv1alpha3.KubeadmControlPlane{
		Spec: kubeadmnv1alpha3.KubeadmControlPlaneSpec{
			InfrastructureTemplate: v1.ObjectReference{
				Name: "test-control-plane-template-original",
			},
		},
	}
	md := &v1alpha3.MachineDeployment{
		Spec: v1alpha3.MachineDeploymentSpec{
			Template: v1alpha3.MachineTemplateSpec{
				Spec: v1alpha3.MachineSpec{
					InfrastructureRef: v1.ObjectReference{
						Name: "test-worker-node-template-original",
					},
				},
			},
		},
	}
	etcdadmCluster := &etcdv1alpha3.EtcdadmCluster{
		Spec: etcdv1alpha3.EtcdadmClusterSpec{
			InfrastructureTemplate: v1.ObjectReference{
				Name: "test-etcd-template-original",
			},
		},
	}

	datacenterConfig := givenDatacenterConfig(t, testClusterConfigMainFilename)
	machineConfigs := givenMachineConfigs(t, testClusterConfigMainFilename)
	_, writer := test.NewWriter(t)
	provider := NewProviderCustomNet(datacenterConfig, machineConfigs, clusterSpec.Cluster, NewDummyProviderGovcClient(), kubectl, writer, &DummyNetClient{}, test.FakeNow, false)
	if provider == nil {
		t.Fatalf("provider object is nil")
	}

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("failed to setup and validate: %v", err)
	}

	controlPlaneMachineConfigName := clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name
	workerNodeMachineConfigName := clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name
	etcdMachineConfigName := clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name
	kubectl.EXPECT().GetEksaCluster(ctx, cluster).Return(clusterSpec.Cluster, nil)
	kubectl.EXPECT().GetEksaVSphereDatacenterConfig(ctx, cluster.Name, cluster.KubeconfigFile).Return(datacenterConfig, nil)
	kubectl.EXPECT().GetEksaVSphereMachineConfig(ctx, controlPlaneMachineConfigName, cluster.KubeconfigFile).Return(machineConfigs[controlPlaneMachineConfigName], nil)
	kubectl.EXPECT().GetEksaVSphereMachineConfig(ctx, workerNodeMachineConfigName, cluster.KubeconfigFile).Return(machineConfigs[workerNodeMachineConfigName], nil)
	kubectl.EXPECT().GetEksaVSphereMachineConfig(ctx, etcdMachineConfigName, cluster.KubeconfigFile).Return(machineConfigs[etcdMachineConfigName], nil)
	kubectl.EXPECT().GetKubeadmControlPlane(ctx, cluster, gomock.AssignableToTypeOf(executables.WithCluster(bootstrapCluster))).Return(cp, nil)
	kubectl.EXPECT().GetMachineDeployment(ctx, cluster, gomock.AssignableToTypeOf(executables.WithCluster(bootstrapCluster))).Return(md, nil)
	kubectl.EXPECT().GetEtcdadmCluster(ctx, cluster, gomock.AssignableToTypeOf(executables.WithCluster(bootstrapCluster))).Return(etcdadmCluster, nil)
	fileName := fmt.Sprintf("%s-eks-a-cluster.yaml", clusterSpec.ObjectMeta.Name)
	writtenFile, err := provider.GenerateDeploymentFileForUpgrade(context.Background(), bootstrapCluster, cluster, clusterSpec, fileName)
	if err != nil {
		t.Fatalf("failed to generate deployment file: %v", err)
	}
	if fileName == "" {
		t.Fatalf("empty fileName returned by GenerateDeploymentFile")
	}
	test.AssertFilesEquals(t, writtenFile, "testdata/expected_results_main_no_machinetemplate_update.yaml")
}

func TestProviderGenerateDeploymentFileCreateCmd(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	var tctx testContext
	tctx.SaveContext()
	defer tctx.RestoreContext()
	ctx := context.Background()
	kubectl := mocks.NewMockProviderKubectlClient(mockCtrl)
	cluster := &types.Cluster{
		Name: "test",
	}
	clusterSpec := givenClusterSpec(t, testClusterConfigMainFilename)

	datacenterConfig := givenDatacenterConfig(t, testClusterConfigMainFilename)
	machineConfigs := givenMachineConfigs(t, testClusterConfigMainFilename)
	_, writer := test.NewWriter(t)
	provider := NewProviderCustomNet(datacenterConfig, machineConfigs, clusterSpec.Cluster, NewDummyProviderGovcClient(), kubectl, writer, &DummyNetClient{}, test.FakeNow, false)
	if provider == nil {
		t.Fatalf("provider object is nil")
	}

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("failed to setup and validate: %v", err)
	}

	fileName := fmt.Sprintf("%s-eks-a-cluster.yaml", clusterSpec.ObjectMeta.Name)
	writtenFile, err := provider.GenerateDeploymentFileForCreate(context.Background(), cluster, clusterSpec, fileName)
	if err != nil {
		t.Fatalf("failed to generate deployment file: %v", err)
	}
	if fileName == "" {
		t.Fatalf("empty fileName returned by GenerateDeploymentFile")
	}
	test.AssertFilesEquals(t, writtenFile, "testdata/expected_results_main.yaml")
}

func TestProviderGenerateStorageClass(t *testing.T) {
	provider := givenProvider(t)

	storageClassManifest := provider.GenerateStorageClass()
	if storageClassManifest == nil {
		t.Fatalf("Expected storageClassManifest")
	}
}

func TestProviderGenerateDeploymentFileForCreateWithBottlerocketAndExternalEtcd(t *testing.T) {
	clusterSpecManifest := "cluster_bottlerocket_external_etcd.yaml"
	mockCtrl := gomock.NewController(t)
	setupContext(t)
	kubectl := mocks.NewMockProviderKubectlClient(mockCtrl)
	cluster := &types.Cluster{Name: "test"}
	clusterSpec := givenClusterSpec(t, clusterSpecManifest)
	datacenterConfig := givenDatacenterConfig(t, clusterSpecManifest)
	machineConfigs := givenMachineConfigs(t, clusterSpecManifest)
	ctx := context.Background()
	_, writer := test.NewWriter(t)
	govc := NewDummyProviderGovcClient()
	govc.osTag = bottlerocketOSTag
	provider := NewProviderCustomNet(datacenterConfig, machineConfigs, clusterSpec.Cluster, govc, kubectl, writer, &DummyNetClient{}, test.FakeNow, false)

	if err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec); err != nil {
		t.Fatalf("failed to setup and validate: %v", err)
	}

	fileName := fmt.Sprintf("%s-eks-a-cluster.yaml", clusterSpec.ObjectMeta.Name)
	writtenFile, err := provider.GenerateDeploymentFileForCreate(context.Background(), cluster, clusterSpec, fileName)
	if err != nil {
		t.Fatalf("failed to generate deployment file: %v", err)
	}
	if fileName == "" {
		t.Fatalf("empty fileName returned by GenerateDeploymentFile")
	}

	test.AssertFilesEquals(t, writtenFile, "testdata/expected_results_bottlerocket_external_etcd.yaml")
}

func TestUpdateKubeConfig(t *testing.T) {
	provider := givenProvider(t)
	content := []byte{}

	err := provider.UpdateKubeConfig(&content, "clusterName")
	if err != nil {
		t.Fatalf("failed UpdateKubeConfig: %v", err)
	}
}

func TestBootstrapClusterOpts(t *testing.T) {
	provider := givenProvider(t)

	bootstrapClusterOps, err := provider.BootstrapClusterOpts()
	if err != nil {
		t.Fatalf("failed BootstrapClusterOpts: %v", err)
	}
	if bootstrapClusterOps == nil {
		t.Fatalf("expected BootstrapClusterOpts")
	}
}

func TestName(t *testing.T) {
	provider := givenProvider(t)

	if provider.Name() != expectedVSphereName {
		t.Fatalf("unexpected Name %s!=%s", provider.Name(), expectedVSphereName)
	}
}

func TestSetupAndValidateCreateCluster(t *testing.T) {
	ctx := context.Background()
	provider := givenProvider(t)
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	var tctx testContext
	tctx.SaveContext()
	defer tctx.RestoreContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("unexpected failure %v", err)
	}
}

func thenErrorPrefixExpected(t *testing.T, expected string, err error) {
	if err == nil {
		t.Fatalf("Expected=<%s> actual=<nil>", expected)
	}
	actual := err.Error()
	if !strings.HasPrefix(actual, expected) {
		t.Fatalf("Expected=<%s...> actual=<%s...>", expected, actual)
	}
}

func thenErrorExpected(t *testing.T, expected string, err error) {
	if err == nil {
		t.Fatalf("Expected=<%s> actual=<nil>", expected)
	}
	actual := err.Error()
	if expected != actual {
		t.Fatalf("Expected=<%s> actual=<%s>", expected, actual)
	}
}

func TestSetupAndValidateCreateClusterNoUsername(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	provider := givenProvider(t)
	var tctx testContext
	tctx.SaveContext()
	defer tctx.RestoreContext()
	os.Unsetenv(eksavSphereUsernameKey)

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)

	thenErrorExpected(t, "failed setup and validations: EKSA_VSPHERE_USERNAME is not set or is empty", err)
}

func TestSetupAndValidateCreateClusterNoPassword(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	provider := givenProvider(t)
	var tctx testContext
	tctx.SaveContext()
	defer tctx.RestoreContext()
	os.Unsetenv(eksavSpherePasswordKey)

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)

	thenErrorExpected(t, "failed setup and validations: EKSA_VSPHERE_PASSWORD is not set or is empty", err)
}

func TestSetupAndValidateDeleteCluster(t *testing.T) {
	ctx := context.Background()
	provider := givenProvider(t)
	var tctx testContext
	tctx.SaveContext()
	defer tctx.RestoreContext()

	err := provider.SetupAndValidateDeleteCluster(ctx)
	if err != nil {
		t.Fatalf("unexpected failure %v", err)
	}
}

func TestSetupAndValidateDeleteClusterNoPassword(t *testing.T) {
	ctx := context.Background()
	provider := givenProvider(t)
	var tctx testContext
	tctx.SaveContext()
	defer tctx.RestoreContext()
	os.Unsetenv(eksavSpherePasswordKey)

	err := provider.SetupAndValidateDeleteCluster(ctx)

	thenErrorExpected(t, "failed setup and validations: EKSA_VSPHERE_PASSWORD is not set or is empty", err)
}

func TestSetupAndValidateUpgradeCluster(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	var tctx testContext
	tctx.SaveContext()
	defer tctx.RestoreContext()

	err := provider.SetupAndValidateUpgradeCluster(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("unexpected failure %v", err)
	}
}

func TestSetupAndValidateUpgradeClusterNoUsername(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	provider := givenProvider(t)
	var tctx testContext
	tctx.SaveContext()
	defer tctx.RestoreContext()
	os.Unsetenv(eksavSphereUsernameKey)

	err := provider.SetupAndValidateUpgradeCluster(ctx, clusterSpec)

	thenErrorExpected(t, "failed setup and validations: EKSA_VSPHERE_USERNAME is not set or is empty", err)
}

func TestSetupAndValidateUpgradeClusterNoPassword(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	provider := givenProvider(t)
	var tctx testContext
	tctx.SaveContext()
	defer tctx.RestoreContext()
	os.Unsetenv(eksavSpherePasswordKey)

	err := provider.SetupAndValidateUpgradeCluster(ctx, clusterSpec)

	thenErrorExpected(t, "failed setup and validations: EKSA_VSPHERE_PASSWORD is not set or is empty", err)
}

func TestSetupAndValidateUpgradeClusterIpExists(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateUpgradeCluster(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("unexpected failure %v", err)
	}
}

func TestSetupAndValidateUpgradeClusterCPSshNotExists(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	controlPlaneMachineConfigName := clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name
	provider.machineConfigs[controlPlaneMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] = ""
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateUpgradeCluster(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("unexpected failure %v", err)
	}
}

func TestSetupAndValidateUpgradeClusterWorkerSshNotExists(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	workerNodeMachineConfigName := clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name
	provider.machineConfigs[workerNodeMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] = ""
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateUpgradeCluster(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("unexpected failure %v", err)
	}
}

func TestSetupAndValidateUpgradeClusterEtcdSshNotExists(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	etcdMachineConfigName := clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name
	provider.machineConfigs[etcdMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] = ""
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateUpgradeCluster(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("unexpected failure %v", err)
	}
}

func TestCleanupProviderInfrastructure(t *testing.T) {
	ctx := context.Background()
	provider := givenProvider(t)
	var tctx testContext
	tctx.SaveContext()
	defer tctx.RestoreContext()

	err := provider.CleanupProviderInfrastructure(ctx)
	if err != nil {
		t.Fatalf("unexpected failure %v", err)
	}
}

func TestVersion(t *testing.T) {
	vSphereProviderVersion := "v0.7.10"
	provider := givenProvider(t)
	clusterSpec := givenEmptyClusterSpec()
	clusterSpec.VersionsBundle.VSphere.Version = vSphereProviderVersion
	var tctx testContext
	tctx.SaveContext()
	defer tctx.RestoreContext()

	result := provider.Version(clusterSpec)
	if result != vSphereProviderVersion {
		t.Fatalf("Unexpected version expected <%s> actual=<%s>", vSphereProviderVersion, result)
	}
}

func TestProviderBootstrapSetup(t *testing.T) {
	ctx := context.Background()
	datacenterConfig := givenDatacenterConfig(t, testClusterConfigMainFilename)
	machineConfigs := givenMachineConfigs(t, testClusterConfigMainFilename)
	clusterConfig := givenClusterConfig(t, testClusterConfigMainFilename)
	_, writer := test.NewWriter(t)
	mockCtrl := gomock.NewController(t)
	kubectl := mocks.NewMockProviderKubectlClient(mockCtrl)
	provider := NewProviderCustomNet(datacenterConfig, machineConfigs, clusterConfig, NewDummyProviderGovcClient(), kubectl, writer, &DummyNetClient{}, test.FakeNow, false)
	cluster := types.Cluster{
		Name:           "test",
		KubeconfigFile: "",
	}
	values := map[string]string{
		"clusterName":       clusterConfig.Name,
		"vspherePassword":   expectedVSphereUsername,
		"vsphereUsername":   expectedVSpherePassword,
		"vsphereServer":     datacenterConfig.Spec.Server,
		"vsphereDatacenter": datacenterConfig.Spec.Datacenter,
		"vsphereNetwork":    datacenterConfig.Spec.Network,
	}

	var tctx testContext
	tctx.SaveContext()
	defer tctx.RestoreContext()

	kubectl.EXPECT().LoadSecret(ctx, gomock.Any(), gomock.Any(), gomock.Any(), cluster.KubeconfigFile)

	template, err := template.New("test").Parse(defaultSecretObject)
	if err != nil {
		t.Fatalf("template create error: %v", err)
	}
	err = template.Execute(&bytes.Buffer{}, values)
	if err != nil {
		t.Fatalf("template execute error: %v", err)
	}

	err = provider.BootstrapSetup(ctx, clusterConfig, &cluster)
	if err != nil {
		t.Fatalf("BootstrapSetup error %v", err)
	}
}

func TestProviderUpdateSecret(t *testing.T) {
	ctx := context.Background()
	datacenterConfig := givenDatacenterConfig(t, testClusterConfigMainFilename)
	clusterConfig := givenClusterConfig(t, testClusterConfigMainFilename)
	machineConfigs := givenMachineConfigs(t, testClusterConfigMainFilename)
	_, writer := test.NewWriter(t)
	mockCtrl := gomock.NewController(t)
	kubectl := mocks.NewMockProviderKubectlClient(mockCtrl)
	provider := NewProviderCustomNet(datacenterConfig, machineConfigs, clusterConfig, NewDummyProviderGovcClient(), kubectl, writer, &DummyNetClient{}, test.FakeNow, false)
	cluster := types.Cluster{
		Name:           "test",
		KubeconfigFile: "",
	}
	values := map[string]string{
		"clusterName":       clusterConfig.Name,
		"vspherePassword":   expectedVSphereUsername,
		"vsphereUsername":   expectedVSpherePassword,
		"vsphereServer":     datacenterConfig.Spec.Server,
		"vsphereDatacenter": datacenterConfig.Spec.Datacenter,
		"vsphereNetwork":    datacenterConfig.Spec.Network,
	}

	var tctx testContext
	tctx.SaveContext()
	defer tctx.RestoreContext()

	kubectl.EXPECT().ApplyKubeSpecFromBytes(ctx, gomock.Any(), gomock.Any())

	template, err := template.New("test").Parse(defaultSecretObject)
	if err != nil {
		t.Fatalf("template create error: %v", err)
	}
	err = template.Execute(&bytes.Buffer{}, values)
	if err != nil {
		t.Fatalf("template execute error: %v", err)
	}

	err = provider.UpdateSecrets(ctx, &cluster)
	if err != nil {
		t.Fatalf("UpdateSecrets error %v", err)
	}
}

func TestSetupAndValidateCreateClusterNoServer(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	provider := givenProvider(t)
	provider.datacenterConfig.Spec.Server = ""
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)

	thenErrorExpected(t, "failed setup and validations: VSphereDatacenterConfig server is not set or is empty", err)
}

func TestSetupAndValidateCreateClusterInsecure(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	provider.datacenterConfig.Spec.Insecure = true
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("Unexpected error <%v>", err)
	}
	if provider.datacenterConfig.Spec.Thumbprint != "" {
		t.Fatalf("Expected=<> actual=<%s>", provider.datacenterConfig.Spec.Thumbprint)
	}
}

func TestSetupAndValidateCreateClusterNoControlPlaneEndpointIP(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	clusterSpec.Spec.ControlPlaneConfiguration.Endpoint.Host = ""
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)

	thenErrorExpected(t, "cluster controlPlaneConfiguration.Endpoint.Host is not set or is empty", err)
}

func TestSetupAndValidateCreateClusterNoDatacenter(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	provider.datacenterConfig.Spec.Datacenter = ""
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)

	thenErrorExpected(t, "VSphereDatacenterConfig datacenter is not set or is empty", err)
}

func TestSetupAndValidateCreateClusterNoDatastoreControlPlane(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	controlPlaneMachineConfigName := clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name
	provider.machineConfigs[controlPlaneMachineConfigName].Spec.Datastore = ""
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)

	thenErrorExpected(t, "VSphereMachineConfig datastore for control plane is not set or is empty", err)
}

func TestSetupAndValidateCreateClusterNoDatastoreWorker(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	workerMachineConfigName := clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name
	provider.machineConfigs[workerMachineConfigName].Spec.Datastore = ""
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)

	thenErrorExpected(t, "VSphereMachineConfig datastore for worker nodes is not set or is empty", err)
}

func TestSetupAndValidateCreateClusterNoDatastoreEtcd(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	etcdMachineConfigName := clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name
	provider.machineConfigs[etcdMachineConfigName].Spec.Datastore = ""
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)

	thenErrorExpected(t, "VSphereMachineConfig datastore for etcd machines is not set or is empty", err)
}

func TestSetupAndValidateCreateClusterNoResourcePoolControlPlane(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	controlPlaneMachineConfigName := clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name
	provider.machineConfigs[controlPlaneMachineConfigName].Spec.ResourcePool = ""
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)

	thenErrorExpected(t, "VSphereMachineConfig VM resourcePool for control plane is not set or is empty", err)
}

func TestSetupAndValidateCreateClusterNoResourcePoolWorker(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	workerMachineConfigName := clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name
	provider.machineConfigs[workerMachineConfigName].Spec.ResourcePool = ""
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)

	thenErrorExpected(t, "VSphereMachineConfig VM resourcePool for worker nodes is not set or is empty", err)
}

func TestSetupAndValidateCreateClusterNoResourcePoolEtcd(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	etcdMachineConfigName := clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name
	provider.machineConfigs[etcdMachineConfigName].Spec.ResourcePool = ""
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)

	thenErrorExpected(t, "VSphereMachineConfig VM resourcePool for etcd machines is not set or is empty", err)
}

func TestSetupAndValidateCreateClusterNoNetwork(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	provider.datacenterConfig.Spec.Network = ""
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)

	thenErrorExpected(t, "VSphereDatacenterConfig VM network is not set or is empty", err)
}

func TestSetupAndValidateCreateClusterNotControlPlaneVMsMemoryMiB(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	controlPlaneMachineConfigName := clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name
	provider.machineConfigs[controlPlaneMachineConfigName].Spec.MemoryMiB = 0
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("Unexpected error <%v>", err)
	}
	if provider.machineConfigs[controlPlaneMachineConfigName].Spec.MemoryMiB != 8192 {
		t.Fatalf("Expected=<8192> actual=<%d>", provider.machineConfigs[controlPlaneMachineConfigName].Spec.MemoryMiB)
	}
}

func TestSetupAndValidateCreateClusterNotControlPlaneVMsNumCPUs(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	controlPlaneMachineConfigName := clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name
	provider.machineConfigs[controlPlaneMachineConfigName].Spec.NumCPUs = 0
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("Unexpected error <%v>", err)
	}
	if provider.machineConfigs[controlPlaneMachineConfigName].Spec.NumCPUs != 2 {
		t.Fatalf("Expected=<2> actual=<%d>", provider.machineConfigs[controlPlaneMachineConfigName].Spec.NumCPUs)
	}
}

func TestSetupAndValidateCreateClusterNotWorkloadVMsMemoryMiB(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	workerNodeMachineConfigName := clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name
	provider.machineConfigs[workerNodeMachineConfigName].Spec.MemoryMiB = 0
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("Unexpected error <%v>", err)
	}
	if provider.machineConfigs[workerNodeMachineConfigName].Spec.MemoryMiB != 8192 {
		t.Fatalf("Expected=<8192> actual=<%d>", provider.machineConfigs[workerNodeMachineConfigName].Spec.MemoryMiB)
	}
}

func TestSetupAndValidateCreateClusterNotWorkloadVMsNumCPUs(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	workerNodeMachineConfigName := clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name
	provider.machineConfigs[workerNodeMachineConfigName].Spec.NumCPUs = 0
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("Unexpected error <%v>", err)
	}
	if provider.machineConfigs[workerNodeMachineConfigName].Spec.NumCPUs != 2 {
		t.Fatalf("Expected=<2> actual=<%d>", provider.machineConfigs[workerNodeMachineConfigName].Spec.NumCPUs)
	}
}

func TestSetupAndValidateCreateClusterNotEtcdVMsMemoryMiB(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	etcdMachineConfigName := clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name
	provider.machineConfigs[etcdMachineConfigName].Spec.MemoryMiB = 0
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("Unexpected error <%v>", err)
	}
	if provider.machineConfigs[etcdMachineConfigName].Spec.MemoryMiB != 8192 {
		t.Fatalf("Expected=<8192> actual=<%d>", provider.machineConfigs[etcdMachineConfigName].Spec.MemoryMiB)
	}
}

func TestSetupAndValidateCreateClusterNotEtcdVMsNumCPUs(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	etcdMachineConfigName := clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name
	provider.machineConfigs[etcdMachineConfigName].Spec.NumCPUs = 0
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("Unexpected error <%v>", err)
	}
	if provider.machineConfigs[etcdMachineConfigName].Spec.NumCPUs != 2 {
		t.Fatalf("Expected=<2> actual=<%d>", provider.machineConfigs[etcdMachineConfigName].Spec.NumCPUs)
	}
}

func TestSetupAndValidateCreateClusterBogusIp(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	clusterSpec.Spec.ControlPlaneConfiguration.Endpoint.Host = "bogus"
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)

	thenErrorExpected(t, "cluster controlPlaneConfiguration.Endpoint.Host is invalid: bogus", err)
}

func TestSetupAndValidateCreateClusterUsedIp(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	clusterSpec.Spec.ControlPlaneConfiguration.Endpoint.Host = "255.255.255.255"
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)

	thenErrorExpected(t, "cluster controlPlaneConfiguration.Endpoint.Host <255.255.255.255> is already in use, please provide a unique IP", err)
}

func TestSetupAndValidateForCreateSSHAuthorizedKeyInvalidCP(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	controlPlaneMachineConfigName := clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name
	tempKey := "ssh-rsa AAAA    B3NzaC1yc2EAAAADAQABAAACAQC1BK73XhIzjX+meUr7pIYh6RHbvI3tmHeQIXY5lv7aztN1UoX+bhPo3dwo2sfSQn5kuxgQdnxIZ/CTzy0p0GkEYVv3gwspCeurjmu0XmrdmaSGcGxCEWT/65NtvYrQtUE5ELxJ+N/aeZNlK2B7IWANnw/82913asXH4VksV1NYNduP0o1/G4XcwLLSyVFB078q/oEnmvdNIoS61j4/o36HVtENJgYr0idcBvwJdvcGxGnPaqOhx477t+kfJAa5n5dSA5wilIaoXH5i1Tf/HsTCM52L+iNCARvQzJYZhzbWI1MDQwzILtIBEQCJsl2XSqIupleY8CxqQ6jCXt2mhae+wPc3YmbO5rFvr2/EvC57kh3yDs1Nsuj8KOvD78KeeujbR8n8pScm3WDp62HFQ8lEKNdeRNj6kB8WnuaJvPnyZfvzOhwG65/9w13IBl7B1sWxbFnq2rMpm5uHVK7mAmjL0Tt8zoDhcE1YJEnp9xte3/pvmKPkST5Q/9ZtR9P5sI+02jY0fvPkPyC03j2gsPixG7rpOCwpOdbny4dcj0TDeeXJX8er+oVfJuLYz0pNWJcT2raDdFfcqvYA0B0IyNYlj5nWX4RuEcyT3qocLReWPnZojetvAG/H8XwOh7fEVGqHAKOVSnPXCSQJPl6s0H12jPJBDJMTydtYPEszl4/CeQ== testemail@test.com"
	provider.machineConfigs[controlPlaneMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] = tempKey
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	thenErrorExpected(t, "failed setup and validations: provided VSphereMachineConfig sshAuthorizedKey is invalid: ssh: no key found", err)
}

func TestSetupAndValidateForCreateSSHAuthorizedKeyInvalidWorker(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	workerNodeMachineConfigName := clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name
	tempKey := "ssh-rsa AAAA    B3NzaC1yc2EAAAADAQABAAACAQC1BK73XhIzjX+meUr7pIYh6RHbvI3tmHeQIXY5lv7aztN1UoX+bhPo3dwo2sfSQn5kuxgQdnxIZ/CTzy0p0GkEYVv3gwspCeurjmu0XmrdmaSGcGxCEWT/65NtvYrQtUE5ELxJ+N/aeZNlK2B7IWANnw/82913asXH4VksV1NYNduP0o1/G4XcwLLSyVFB078q/oEnmvdNIoS61j4/o36HVtENJgYr0idcBvwJdvcGxGnPaqOhx477t+kfJAa5n5dSA5wilIaoXH5i1Tf/HsTCM52L+iNCARvQzJYZhzbWI1MDQwzILtIBEQCJsl2XSqIupleY8CxqQ6jCXt2mhae+wPc3YmbO5rFvr2/EvC57kh3yDs1Nsuj8KOvD78KeeujbR8n8pScm3WDp62HFQ8lEKNdeRNj6kB8WnuaJvPnyZfvzOhwG65/9w13IBl7B1sWxbFnq2rMpm5uHVK7mAmjL0Tt8zoDhcE1YJEnp9xte3/pvmKPkST5Q/9ZtR9P5sI+02jY0fvPkPyC03j2gsPixG7rpOCwpOdbny4dcj0TDeeXJX8er+oVfJuLYz0pNWJcT2raDdFfcqvYA0B0IyNYlj5nWX4RuEcyT3qocLReWPnZojetvAG/H8XwOh7fEVGqHAKOVSnPXCSQJPl6s0H12jPJBDJMTydtYPEszl4/CeQ== testemail@test.com"
	provider.machineConfigs[workerNodeMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] = tempKey
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	thenErrorExpected(t, "failed setup and validations: provided VSphereMachineConfig sshAuthorizedKey is invalid: ssh: no key found", err)
}

func TestSetupAndValidateForCreateSSHAuthorizedKeyInvalidEtcd(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	etcdMachineConfigName := clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name
	tempKey := "ssh-rsa AAAA    B3NzaC1yc2EAAAADAQABAAACAQC1BK73XhIzjX+meUr7pIYh6RHbvI3tmHeQIXY5lv7aztN1UoX+bhPo3dwo2sfSQn5kuxgQdnxIZ/CTzy0p0GkEYVv3gwspCeurjmu0XmrdmaSGcGxCEWT/65NtvYrQtUE5ELxJ+N/aeZNlK2B7IWANnw/82913asXH4VksV1NYNduP0o1/G4XcwLLSyVFB078q/oEnmvdNIoS61j4/o36HVtENJgYr0idcBvwJdvcGxGnPaqOhx477t+kfJAa5n5dSA5wilIaoXH5i1Tf/HsTCM52L+iNCARvQzJYZhzbWI1MDQwzILtIBEQCJsl2XSqIupleY8CxqQ6jCXt2mhae+wPc3YmbO5rFvr2/EvC57kh3yDs1Nsuj8KOvD78KeeujbR8n8pScm3WDp62HFQ8lEKNdeRNj6kB8WnuaJvPnyZfvzOhwG65/9w13IBl7B1sWxbFnq2rMpm5uHVK7mAmjL0Tt8zoDhcE1YJEnp9xte3/pvmKPkST5Q/9ZtR9P5sI+02jY0fvPkPyC03j2gsPixG7rpOCwpOdbny4dcj0TDeeXJX8er+oVfJuLYz0pNWJcT2raDdFfcqvYA0B0IyNYlj5nWX4RuEcyT3qocLReWPnZojetvAG/H8XwOh7fEVGqHAKOVSnPXCSQJPl6s0H12jPJBDJMTydtYPEszl4/CeQ== testemail@test.com"
	provider.machineConfigs[etcdMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] = tempKey
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	thenErrorExpected(t, "failed setup and validations: provided VSphereMachineConfig sshAuthorizedKey is invalid: ssh: no key found", err)
}

func TestSetupAndValidateForUpgradeSSHAuthorizedKeyInvalidCP(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	controlPlaneMachineConfigName := clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name
	tempKey := "ssh-rsa AAAA    B3NzaC1yc2EAAAADAQABAAACAQC1BK73XhIzjX+meUr7pIYh6RHbvI3tmHeQIXY5lv7aztN1UoX+bhPo3dwo2sfSQn5kuxgQdnxIZ/CTzy0p0GkEYVv3gwspCeurjmu0XmrdmaSGcGxCEWT/65NtvYrQtUE5ELxJ+N/aeZNlK2B7IWANnw/82913asXH4VksV1NYNduP0o1/G4XcwLLSyVFB078q/oEnmvdNIoS61j4/o36HVtENJgYr0idcBvwJdvcGxGnPaqOhx477t+kfJAa5n5dSA5wilIaoXH5i1Tf/HsTCM52L+iNCARvQzJYZhzbWI1MDQwzILtIBEQCJsl2XSqIupleY8CxqQ6jCXt2mhae+wPc3YmbO5rFvr2/EvC57kh3yDs1Nsuj8KOvD78KeeujbR8n8pScm3WDp62HFQ8lEKNdeRNj6kB8WnuaJvPnyZfvzOhwG65/9w13IBl7B1sWxbFnq2rMpm5uHVK7mAmjL0Tt8zoDhcE1YJEnp9xte3/pvmKPkST5Q/9ZtR9P5sI+02jY0fvPkPyC03j2gsPixG7rpOCwpOdbny4dcj0TDeeXJX8er+oVfJuLYz0pNWJcT2raDdFfcqvYA0B0IyNYlj5nWX4RuEcyT3qocLReWPnZojetvAG/H8XwOh7fEVGqHAKOVSnPXCSQJPl6s0H12jPJBDJMTydtYPEszl4/CeQ== testemail@test.com"
	provider.machineConfigs[controlPlaneMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] = tempKey
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateUpgradeCluster(ctx, clusterSpec)
	thenErrorExpected(t, "failed setup and validations: provided VSphereMachineConfig sshAuthorizedKey is invalid: ssh: no key found", err)
}

func TestSetupAndValidateForUpgradeSSHAuthorizedKeyInvalidWorker(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	workerNodeMachineConfigName := clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name
	tempKey := "ssh-rsa AAAA    B3NzaC1yc2EAAAADAQABAAACAQC1BK73XhIzjX+meUr7pIYh6RHbvI3tmHeQIXY5lv7aztN1UoX+bhPo3dwo2sfSQn5kuxgQdnxIZ/CTzy0p0GkEYVv3gwspCeurjmu0XmrdmaSGcGxCEWT/65NtvYrQtUE5ELxJ+N/aeZNlK2B7IWANnw/82913asXH4VksV1NYNduP0o1/G4XcwLLSyVFB078q/oEnmvdNIoS61j4/o36HVtENJgYr0idcBvwJdvcGxGnPaqOhx477t+kfJAa5n5dSA5wilIaoXH5i1Tf/HsTCM52L+iNCARvQzJYZhzbWI1MDQwzILtIBEQCJsl2XSqIupleY8CxqQ6jCXt2mhae+wPc3YmbO5rFvr2/EvC57kh3yDs1Nsuj8KOvD78KeeujbR8n8pScm3WDp62HFQ8lEKNdeRNj6kB8WnuaJvPnyZfvzOhwG65/9w13IBl7B1sWxbFnq2rMpm5uHVK7mAmjL0Tt8zoDhcE1YJEnp9xte3/pvmKPkST5Q/9ZtR9P5sI+02jY0fvPkPyC03j2gsPixG7rpOCwpOdbny4dcj0TDeeXJX8er+oVfJuLYz0pNWJcT2raDdFfcqvYA0B0IyNYlj5nWX4RuEcyT3qocLReWPnZojetvAG/H8XwOh7fEVGqHAKOVSnPXCSQJPl6s0H12jPJBDJMTydtYPEszl4/CeQ== testemail@test.com"
	provider.machineConfigs[workerNodeMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] = tempKey
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateUpgradeCluster(ctx, clusterSpec)
	thenErrorExpected(t, "failed setup and validations: provided VSphereMachineConfig sshAuthorizedKey is invalid: ssh: no key found", err)
}

func TestSetupAndValidateForUpgradeSSHAuthorizedKeyInvalidEtcd(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	etcdMachineConfigName := clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name
	tempKey := "ssh-rsa AAAA    B3NzaC1yc2EAAAADAQABAAACAQC1BK73XhIzjX+meUr7pIYh6RHbvI3tmHeQIXY5lv7aztN1UoX+bhPo3dwo2sfSQn5kuxgQdnxIZ/CTzy0p0GkEYVv3gwspCeurjmu0XmrdmaSGcGxCEWT/65NtvYrQtUE5ELxJ+N/aeZNlK2B7IWANnw/82913asXH4VksV1NYNduP0o1/G4XcwLLSyVFB078q/oEnmvdNIoS61j4/o36HVtENJgYr0idcBvwJdvcGxGnPaqOhx477t+kfJAa5n5dSA5wilIaoXH5i1Tf/HsTCM52L+iNCARvQzJYZhzbWI1MDQwzILtIBEQCJsl2XSqIupleY8CxqQ6jCXt2mhae+wPc3YmbO5rFvr2/EvC57kh3yDs1Nsuj8KOvD78KeeujbR8n8pScm3WDp62HFQ8lEKNdeRNj6kB8WnuaJvPnyZfvzOhwG65/9w13IBl7B1sWxbFnq2rMpm5uHVK7mAmjL0Tt8zoDhcE1YJEnp9xte3/pvmKPkST5Q/9ZtR9P5sI+02jY0fvPkPyC03j2gsPixG7rpOCwpOdbny4dcj0TDeeXJX8er+oVfJuLYz0pNWJcT2raDdFfcqvYA0B0IyNYlj5nWX4RuEcyT3qocLReWPnZojetvAG/H8XwOh7fEVGqHAKOVSnPXCSQJPl6s0H12jPJBDJMTydtYPEszl4/CeQ== testemail@test.com"
	provider.machineConfigs[etcdMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] = tempKey
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateUpgradeCluster(ctx, clusterSpec)
	thenErrorExpected(t, "failed setup and validations: provided VSphereMachineConfig sshAuthorizedKey is invalid: ssh: no key found", err)
}

func TestSetupAndValidateSSHAuthorizedKeyEmptyCP(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	controlPlaneMachineConfigName := clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name
	provider.machineConfigs[controlPlaneMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] = ""
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("provider.SetupAndValidateCreateCluster() err = %v, want err = nil", err)
	}
	if provider.machineConfigs[controlPlaneMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] == "" {
		t.Fatalf("sshAuthorizedKey has not changed for control plane machine")
	}
}

func TestSetupAndValidateSSHAuthorizedKeyEmptyWorker(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	workerNodeMachineConfigName := clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name
	provider.machineConfigs[workerNodeMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] = ""
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("provider.SetupAndValidateCreateCluster() err = %v, want err = nil", err)
	}
	if provider.machineConfigs[workerNodeMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] == "" {
		t.Fatalf("sshAuthorizedKey has not changed for worker node machine")
	}
}

func TestSetupAndValidateSSHAuthorizedKeyEmptyEtcd(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	etcdMachineConfigName := clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name
	provider.machineConfigs[etcdMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] = ""
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("provider.SetupAndValidateCreateCluster() err = %v, want err = nil", err)
	}
	if provider.machineConfigs[etcdMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] == "" {
		t.Fatalf("sshAuthorizedKey did not get generated for etcd machine")
	}
}

func TestSetupAndValidateSSHAuthorizedKeyEmptyAllMachineConfigs(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	controlPlaneMachineConfigName := clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name
	provider.machineConfigs[controlPlaneMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] = ""
	workerNodeMachineConfigName := clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name
	provider.machineConfigs[workerNodeMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] = ""
	etcdMachineConfigName := clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name
	provider.machineConfigs[etcdMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] = ""

	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("provider.SetupAndValidateCreateCluster() err = %v, want err = nil", err)
	}
	if provider.machineConfigs[controlPlaneMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] == "" {
		t.Fatalf("sshAuthorizedKey has not changed for control plane machine")
	}
	if provider.machineConfigs[workerNodeMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] == "" {
		t.Fatalf("sshAuthorizedKey has not changed for worker node machine")
	}
	if provider.machineConfigs[etcdMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] == "" {
		t.Fatalf("sshAuthorizedKey not generated for etcd machines")
	}
	if provider.machineConfigs[controlPlaneMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] != provider.machineConfigs[workerNodeMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] {
		t.Fatalf("sshAuthorizedKey not the same for controlplane and worker machines")
	}
	if provider.machineConfigs[controlPlaneMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] != provider.machineConfigs[etcdMachineConfigName].Spec.Users[0].SshAuthorizedKeys[0] {
		t.Fatalf("sshAuthorizedKey not the same for controlplane and etcd machines")
	}
}

func TestSetupAndValidateUsersNil(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	controlPlaneMachineConfigName := clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name
	provider.machineConfigs[controlPlaneMachineConfigName].Spec.Users = nil
	workerNodeMachineConfigName := clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name
	provider.machineConfigs[workerNodeMachineConfigName].Spec.Users = nil
	etcdMachineConfigName := clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name
	provider.machineConfigs[etcdMachineConfigName].Spec.Users = nil
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("provider.SetupAndValidateCreateCluster() err = %v, want err = nil", err)
	}
}

func TestSetupAndValidateSshAuthorizedKeysNil(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	controlPlaneMachineConfigName := clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name
	provider.machineConfigs[controlPlaneMachineConfigName].Spec.Users[0].SshAuthorizedKeys = nil
	workerNodeMachineConfigName := clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name
	provider.machineConfigs[workerNodeMachineConfigName].Spec.Users[0].SshAuthorizedKeys = nil
	etcdMachineConfigName := clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name
	provider.machineConfigs[etcdMachineConfigName].Spec.Users[0].SshAuthorizedKeys = nil
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("provider.SetupAndValidateCreateCluster() err = %v, want err = nil", err)
	}
}

func TestSetupAndValidateCreateClusterCPMachineGroupRefNil(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef = nil
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		thenErrorExpected(t, "must specify machineGroupRef for control plane", err)
	}
}

func TestSetupAndValidateCreateClusterWorkerMachineGroupRefNil(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef = nil
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		thenErrorExpected(t, "must specify machineGroupRef for worker nodes", err)
	}
}

func TestSetupAndValidateCreateClusterEtcdMachineGroupRefNil(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef = nil
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		thenErrorExpected(t, "must specify machineGroupRef for etcd machines", err)
	}
}

func TestSetupAndValidateCreateClusterCPMachineGroupRefNonexistent(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name = "nonexistent"
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		thenErrorExpected(t, "cannot find VSphereMachineConfig nonexistent for control plane", err)
	}
}

func TestSetupAndValidateCreateClusterWorkerMachineGroupRefNonexistent(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name = "nonexistent"
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		thenErrorExpected(t, "cannot find VSphereMachineConfig nonexistent for worker nodes", err)
	}
}

func TestSetupAndValidateCreateClusterEtcdMachineGroupRefNonexistent(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name = "nonexistent"
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		thenErrorExpected(t, "cannot find VSphereMachineConfig nonexistent for etcd machines", err)
	}
}

func TestSetupAndValidateCreateClusterOsFamilyDifferent(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	controlPlaneMachineConfigName := clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name
	provider.machineConfigs[controlPlaneMachineConfigName].Spec.OSFamily = "bottlerocket"
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		thenErrorExpected(t, "control plane and worker nodes must have the same osFamily specified", err)
	}
}

func TestSetupAndValidateCreateClusterOsFamilyDifferentForEtcd(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	etcdMachineConfigName := clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name
	provider.machineConfigs[etcdMachineConfigName].Spec.OSFamily = "bottlerocket"
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		thenErrorExpected(t, "control plane and etcd machines must have the same osFamily specified", err)
	}
}

func TestSetupAndValidateCreateClusterOsFamilyEmpty(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	clusterConfig := givenClusterConfig(t, testClusterConfigMainFilename)
	datacenterConfig := givenDatacenterConfig(t, testClusterConfigMainFilename)
	machineConfigs := givenMachineConfigs(t, testClusterConfigMainFilename)
	_, writer := test.NewWriter(t)
	govc := NewDummyProviderGovcClient()
	govc.osTag = bottlerocketOSTag
	provider := NewProviderCustomNet(datacenterConfig, machineConfigs, clusterConfig, govc, nil, writer, &DummyNetClient{}, test.FakeNow, false)
	controlPlaneMachineConfigName := clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name
	provider.machineConfigs[controlPlaneMachineConfigName].Spec.OSFamily = ""
	provider.machineConfigs[controlPlaneMachineConfigName].Spec.Users[0].Name = ""
	workerNodeMachineConfigName := clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name
	provider.machineConfigs[workerNodeMachineConfigName].Spec.OSFamily = ""
	provider.machineConfigs[workerNodeMachineConfigName].Spec.Users[0].Name = ""
	etcdMachineConfigName := clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name
	provider.machineConfigs[etcdMachineConfigName].Spec.OSFamily = ""
	provider.machineConfigs[etcdMachineConfigName].Spec.Users[0].Name = ""
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		t.Fatalf("provider.SetupAndValidateCreateCluster() err = %v, want err = nil", err)
	}
	if provider.machineConfigs[controlPlaneMachineConfigName].Spec.OSFamily != v1alpha1.Bottlerocket {
		t.Fatalf("got osFamily for control plane machine as %v, want %v", provider.machineConfigs[controlPlaneMachineConfigName].Spec.OSFamily, v1alpha1.Bottlerocket)
	}
	if provider.machineConfigs[workerNodeMachineConfigName].Spec.OSFamily != v1alpha1.Bottlerocket {
		t.Fatalf("got osFamily for control plane machine as %v, want %v", provider.machineConfigs[controlPlaneMachineConfigName].Spec.OSFamily, v1alpha1.Bottlerocket)
	}
	if provider.machineConfigs[etcdMachineConfigName].Spec.OSFamily != v1alpha1.Bottlerocket {
		t.Fatalf("got osFamily for etcd machine as %v, want %v", provider.machineConfigs[etcdMachineConfigName].Spec.OSFamily, v1alpha1.Bottlerocket)
	}
}

func TestSetupAndValidateCreateClusterTemplateDifferent(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	controlPlaneMachineConfigName := clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name
	provider.machineConfigs[controlPlaneMachineConfigName].Spec.Template = "test"
	var tctx testContext
	tctx.SaveContext()

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)
	if err != nil {
		thenErrorExpected(t, "control plane and worker nodes must have the same template specified", err)
	}
}

func TestSetupAndValidateCreateClusterTemplateDoesNotExist(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	govc := givenGovcMock(t)
	provider.providerGovcClient = govc
	setupContext(t)

	govc.EXPECT().ValidateVCenterSetup(ctx, provider.datacenterConfig, &provider.selfSigned).Return(nil)
	govc.EXPECT().ValidateVCenterSetupMachineConfig(ctx, provider.datacenterConfig, provider.machineConfigs[clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name], &provider.selfSigned).Return(nil)
	govc.EXPECT().ValidateVCenterSetupMachineConfig(ctx, provider.datacenterConfig, provider.machineConfigs[clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name], &provider.selfSigned).Return(nil)
	govc.EXPECT().ValidateVCenterSetupMachineConfig(ctx, provider.datacenterConfig, provider.machineConfigs[clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name], &provider.selfSigned).Return(nil)
	govc.EXPECT().SearchTemplate(ctx, provider.datacenterConfig.Spec.Datacenter, provider.machineConfigs[clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name]).Return("", nil)

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)

	thenErrorExpected(t, "template <"+testTemplate+"> not found. Has the template been imported?", err)
}

func TestSetupAndValidateCreateClusterErrorCheckingTemplate(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	govc := givenGovcMock(t)
	provider.providerGovcClient = govc
	errorMessage := "failed getting template"
	setupContext(t)

	govc.EXPECT().ValidateVCenterSetup(ctx, provider.datacenterConfig, &provider.selfSigned).Return(nil)
	govc.EXPECT().ValidateVCenterSetupMachineConfig(ctx, provider.datacenterConfig, provider.machineConfigs[clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name], &provider.selfSigned).Return(nil)
	govc.EXPECT().ValidateVCenterSetupMachineConfig(ctx, provider.datacenterConfig, provider.machineConfigs[clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name], &provider.selfSigned).Return(nil)
	govc.EXPECT().ValidateVCenterSetupMachineConfig(ctx, provider.datacenterConfig, provider.machineConfigs[clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name], &provider.selfSigned).Return(nil)
	govc.EXPECT().SearchTemplate(ctx, provider.datacenterConfig.Spec.Datacenter, provider.machineConfigs[clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name]).Return("", errors.New(errorMessage))

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)

	thenErrorExpected(t, "error validating template: failed getting template", err)
}

func TestSetupAndValidateCreateClusterTemplateMissingTags(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	govc := givenGovcMock(t)
	provider.providerGovcClient = govc
	controlPlaneMachineConfigName := clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name
	provider.machineConfigs[controlPlaneMachineConfigName].Spec.Template = testTemplate
	setupContext(t)

	govc.EXPECT().ValidateVCenterSetup(ctx, provider.datacenterConfig, &provider.selfSigned).Return(nil)
	govc.EXPECT().ValidateVCenterSetupMachineConfig(ctx, provider.datacenterConfig, provider.machineConfigs[clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name], &provider.selfSigned).Return(nil)
	govc.EXPECT().ValidateVCenterSetupMachineConfig(ctx, provider.datacenterConfig, provider.machineConfigs[clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name], &provider.selfSigned).Return(nil)
	govc.EXPECT().ValidateVCenterSetupMachineConfig(ctx, provider.datacenterConfig, provider.machineConfigs[clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name], &provider.selfSigned).Return(nil)
	govc.EXPECT().SearchTemplate(ctx, provider.datacenterConfig.Spec.Datacenter, provider.machineConfigs[controlPlaneMachineConfigName]).Return(provider.machineConfigs[controlPlaneMachineConfigName].Spec.Template, nil)
	govc.EXPECT().GetTags(ctx, provider.machineConfigs[controlPlaneMachineConfigName].Spec.Template).Return(nil, nil)

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)

	thenErrorPrefixExpected(t, "template "+testTemplate+" is missing tag ", err)
}

func TestSetupAndValidateCreateClusterErrorGettingTags(t *testing.T) {
	ctx := context.Background()
	clusterSpec := givenEmptyClusterSpec()
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	govc := givenGovcMock(t)
	provider.providerGovcClient = govc
	controlPlaneMachineConfigName := clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name
	provider.machineConfigs[controlPlaneMachineConfigName].Spec.Template = testTemplate
	errorMessage := "failed getting tags"
	setupContext(t)

	govc.EXPECT().ValidateVCenterSetup(ctx, provider.datacenterConfig, &provider.selfSigned).Return(nil)
	govc.EXPECT().ValidateVCenterSetupMachineConfig(ctx, provider.datacenterConfig, provider.machineConfigs[clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name], &provider.selfSigned).Return(nil)
	govc.EXPECT().ValidateVCenterSetupMachineConfig(ctx, provider.datacenterConfig, provider.machineConfigs[clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name], &provider.selfSigned).Return(nil)
	govc.EXPECT().ValidateVCenterSetupMachineConfig(ctx, provider.datacenterConfig, provider.machineConfigs[clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name], &provider.selfSigned).Return(nil)
	govc.EXPECT().SearchTemplate(ctx, provider.datacenterConfig.Spec.Datacenter, provider.machineConfigs[controlPlaneMachineConfigName]).Return(provider.machineConfigs[controlPlaneMachineConfigName].Spec.Template, nil)
	govc.EXPECT().GetTags(ctx, provider.machineConfigs[controlPlaneMachineConfigName].Spec.Template).Return(nil, errors.New(errorMessage))

	err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec)

	thenErrorExpected(t, "error validating template tags: failed getting tags", err)
}

func TestSetupAndValidateCreateClusterDefaultTemplate(t *testing.T) {
	ctx := context.Background()
	clusterSpec := test.NewClusterSpec(func(s *cluster.Spec) {
		s.VersionsBundle.EksD.Ova.Ubuntu.URI = "https://amazonaws.com/artifacts/0.0.1/eks-distro/ova/1-19/1-19-4/ubuntu-v1.19.8-eks-d-1-19-4-eks-a-0.0.1.build.38-amd64.ova"
		s.VersionsBundle.EksD.Ova.Ubuntu.SHA256 = "63a8dce1683379cb8df7d15e9c5adf9462a2b9803a544dd79b16f19a4657967f"
		s.VersionsBundle.EksD.Ova.Ubuntu.Arch = []string{"amd64"}
		s.VersionsBundle.EksD.Name = eksd119Release
		s.VersionsBundle.EksD.KubeVersion = "v1.19.8"
		s.VersionsBundle.KubeVersion = "1.19"
	})
	fillClusterSpecWithClusterConfig(clusterSpec, givenClusterConfig(t, testClusterConfigMainFilename))
	provider := givenProvider(t)
	controlPlaneMachineConfigName := clusterSpec.Spec.ControlPlaneConfiguration.MachineGroupRef.Name
	provider.machineConfigs[controlPlaneMachineConfigName].Spec.Template = ""
	workerNodeMachineConfigName := clusterSpec.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name
	provider.machineConfigs[workerNodeMachineConfigName].Spec.Template = ""
	etcdMachineConfigName := clusterSpec.Spec.ExternalEtcdConfiguration.MachineGroupRef.Name
	provider.machineConfigs[etcdMachineConfigName].Spec.Template = ""
	wantTemplate := "/SDDC-Datacenter/vm/Templates/ubuntu-v1.19.8-kubernetes-1-19-eks-4-amd64-63a8dce"
	setupContext(t)

	if err := provider.SetupAndValidateCreateCluster(ctx, clusterSpec); err != nil {
		t.Fatalf("provider.SetupAndValidateCreateCluster() err = %v, want err = nil", err)
	}
	gotTemplate := provider.machineConfigs[controlPlaneMachineConfigName].Spec.Template
	if gotTemplate != wantTemplate {
		t.Fatalf("provider.SetupAndValidateCreateCluster() template = %s, want %s", gotTemplate, wantTemplate)
	}
	gotTemplate = provider.machineConfigs[workerNodeMachineConfigName].Spec.Template
	if gotTemplate != wantTemplate {
		t.Fatalf("provider.SetupAndValidateCreateCluster() template = %s, want %s", gotTemplate, wantTemplate)
	}
}

func TestGetInfrastructureBundleSuccess(t *testing.T) {
	tests := []struct {
		testName    string
		clusterSpec *cluster.Spec
	}{
		{
			testName: "correct Overrides layer",
			clusterSpec: test.NewClusterSpec(func(s *cluster.Spec) {
				s.VersionsBundle.VSphere = releasev1alpha1.VSphereBundle{
					Version: "v0.7.8",
					ClusterAPIController: releasev1alpha1.Image{
						URI: "public.ecr.aws/l0g8r8j6/kubernetes-sigs/cluster-api-provider-vsphere/release/manager:v0.7.8-35f54b0a7ff0f4f3cb0b8e30a0650acd0e55496a",
					},
					Manager: releasev1alpha1.Image{
						URI: "public.ecr.aws/l0g8r8j6/kubernetes/cloud-provider-vsphere/cpi/manager:v1.18.1-2093eaeda5a4567f0e516d652e0b25b1d7abc774",
					},
					KubeVip: releasev1alpha1.Image{
						URI: "public.ecr.aws/l0g8r8j6/plunder-app/kube-vip:v0.3.2-2093eaeda5a4567f0e516d652e0b25b1d7abc774",
					},
					Driver: releasev1alpha1.Image{
						URI: "public.ecr.aws/l0g8r8j6/kubernetes-sigs/vsphere-csi-driver/csi/driver:v2.2.0-7c2690c880c6521afdd9ffa8d90443a11c6b817b",
					},
					Syncer: releasev1alpha1.Image{
						URI: "public.ecr.aws/l0g8r8j6/kubernetes-sigs/vsphere-csi-driver/csi/syncer:v2.2.0-7c2690c880c6521afdd9ffa8d90443a11c6b817b",
					},
					Metadata: releasev1alpha1.Manifest{
						URI: "Metadata.yaml",
					},
					Components: releasev1alpha1.Manifest{
						URI: "Components.yaml",
					},
					ClusterTemplate: releasev1alpha1.Manifest{
						URI: "ClusterTemplate.yaml",
					},
				}
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			p := givenProvider(t)

			infraBundle := p.GetInfrastructureBundle(tt.clusterSpec)
			if infraBundle == nil {
				t.Fatalf("provider.GetInfrastructureBundle() should have an infrastructure bundle")
			}
			assert.Equal(t, "infrastructure-vsphere/v0.7.8/", infraBundle.FolderName, "Incorrect folder name")
			assert.Equal(t, len(infraBundle.Manifests), 3, "Wrong number of files in the infrastructure bundle")
			wantManifests := []releasev1alpha1.Manifest{
				tt.clusterSpec.VersionsBundle.VSphere.Components,
				tt.clusterSpec.VersionsBundle.VSphere.Metadata,
				tt.clusterSpec.VersionsBundle.VSphere.ClusterTemplate,
			}
			assert.ElementsMatch(t, infraBundle.Manifests, wantManifests, "Incorrect manifests")
		})
	}
}

func TestGetDatacenterConfig(t *testing.T) {
	provider := givenProvider(t)
	provider.datacenterConfig.TypeMeta.Kind = "kind"

	providerConfig := provider.DatacenterConfig()
	if providerConfig.Kind() != "kind" {
		t.Fatal("Unexpected error DatacenterConfig: kind field not found")
	}
}

func TestValidateNewSpecSuccess(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	clusterConfig := givenClusterConfig(t, testClusterConfigMainFilename)

	provider := givenProvider(t)
	kubectl := mocks.NewMockProviderKubectlClient(mockCtrl)
	provider.providerKubectlClient = kubectl

	newProviderConfig := givenDatacenterConfig(t, testClusterConfigMainFilename)
	newMachineConfigs := givenMachineConfigs(t, testClusterConfigMainFilename)

	clusterVsphereSecret := &v1.Secret{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Data: map[string][]byte{
			"username": []byte("vsphere_username"),
			"password": []byte("vsphere_password"),
		},
	}

	c := &types.Cluster{}

	kubectl.EXPECT().GetEksaCluster(context.TODO(), gomock.Any()).Return(clusterConfig, nil)
	kubectl.EXPECT().GetEksaVSphereDatacenterConfig(context.TODO(), clusterConfig.Spec.DatacenterRef.Name, gomock.Any()).Return(newProviderConfig, nil)
	for _, config := range newMachineConfigs {
		kubectl.EXPECT().GetEksaVSphereMachineConfig(context.TODO(), gomock.Any(), gomock.Any()).Return(config, nil)
	}
	kubectl.EXPECT().GetSecret(gomock.Any(), credentialsObjectName, gomock.Any()).Return(clusterVsphereSecret, nil)

	err := provider.ValidateNewSpec(context.TODO(), c)
	assert.NoError(t, err, "No error should be returned when previous spec == new spec")
}

func TestValidateNewSpecMutableFields(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	clusterConfig := givenClusterConfig(t, testClusterConfigMainFilename)

	provider := givenProvider(t)
	kubectl := mocks.NewMockProviderKubectlClient(mockCtrl)
	provider.providerKubectlClient = kubectl

	newDatacenterConfig := givenDatacenterConfig(t, testClusterConfigMainFilename)

	newMachineConfigs := givenMachineConfigs(t, testClusterConfigMainFilename)
	for _, config := range newMachineConfigs {
		config.Spec.ResourcePool = "new-" + config.Spec.ResourcePool
		config.Spec.Folder = "new=" + config.Spec.Folder
	}

	clusterVsphereSecret := &v1.Secret{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Data: map[string][]byte{
			"username": []byte("vsphere_username"),
			"password": []byte("vsphere_password"),
		},
	}

	kubectl.EXPECT().GetEksaCluster(context.TODO(), gomock.Any()).Return(clusterConfig, nil)
	kubectl.EXPECT().GetEksaVSphereDatacenterConfig(context.TODO(), clusterConfig.Spec.DatacenterRef.Name, gomock.Any()).Return(newDatacenterConfig, nil)
	for _, config := range newMachineConfigs {
		kubectl.EXPECT().GetEksaVSphereMachineConfig(context.TODO(), gomock.Any(), gomock.Any()).Return(config, nil)
	}
	kubectl.EXPECT().GetSecret(gomock.Any(), credentialsObjectName, gomock.Any()).Return(clusterVsphereSecret, nil)

	err := provider.ValidateNewSpec(context.TODO(), &types.Cluster{})
	assert.NoError(t, err, "No error should be returned when modifying mutable fields")
}

func TestValidateNewSpecDatacenterImmutable(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	clusterConfig := givenClusterConfig(t, testClusterConfigMainFilename)

	provider := givenProvider(t)
	kubectl := mocks.NewMockProviderKubectlClient(mockCtrl)
	provider.providerKubectlClient = kubectl

	newProviderConfig := givenDatacenterConfig(t, testClusterConfigMainFilename)
	newProviderConfig.Spec.Datacenter = "new-" + newProviderConfig.Spec.Datacenter

	newMachineConfigs := givenMachineConfigs(t, testClusterConfigMainFilename)

	kubectl.EXPECT().GetEksaCluster(context.TODO(), gomock.Any()).Return(clusterConfig, nil)
	kubectl.EXPECT().GetEksaVSphereDatacenterConfig(context.TODO(), clusterConfig.Spec.DatacenterRef.Name, gomock.Any()).Return(newProviderConfig, nil)
	for _, config := range newMachineConfigs {
		kubectl.EXPECT().GetEksaVSphereMachineConfig(context.TODO(), gomock.Any(), gomock.Any()).Return(config, nil)
	}

	err := provider.ValidateNewSpec(context.TODO(), &types.Cluster{})
	assert.Error(t, err, "Datacenter should be immutable")
}

func TestValidateNewSpecServerImmutable(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	clusterConfig := givenClusterConfig(t, testClusterConfigMainFilename)

	provider := givenProvider(t)
	kubectl := mocks.NewMockProviderKubectlClient(mockCtrl)
	provider.providerKubectlClient = kubectl

	newProviderConfig := givenDatacenterConfig(t, testClusterConfigMainFilename)
	newProviderConfig.Spec.Server = "new-" + newProviderConfig.Spec.Server

	newMachineConfigs := givenMachineConfigs(t, testClusterConfigMainFilename)

	kubectl.EXPECT().GetEksaCluster(context.TODO(), gomock.Any()).Return(clusterConfig, nil)
	kubectl.EXPECT().GetEksaVSphereDatacenterConfig(context.TODO(), clusterConfig.Spec.DatacenterRef.Name, gomock.Any()).Return(newProviderConfig, nil)
	for _, config := range newMachineConfigs {
		kubectl.EXPECT().GetEksaVSphereMachineConfig(context.TODO(), gomock.Any(), gomock.Any()).Return(config, nil)
	}

	err := provider.ValidateNewSpec(context.TODO(), &types.Cluster{})
	assert.Error(t, err, "Server should be immutable")
}

func TestValidateNewSpecStoragePolicyNameImmutableControlPlane(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	clusterConfig := givenClusterConfig(t, testClusterConfigMainFilename)

	provider := givenProvider(t)
	kubectl := mocks.NewMockProviderKubectlClient(mockCtrl)
	provider.providerKubectlClient = kubectl

	newProviderConfig := givenDatacenterConfig(t, testClusterConfigMainFilename)

	newMachineConfigs := givenMachineConfigs(t, testClusterConfigMainFilename)
	controlPlaneMachineConfigName := clusterConfig.Spec.ControlPlaneConfiguration.MachineGroupRef.Name
	newMachineConfigs[controlPlaneMachineConfigName].Spec.StoragePolicyName = "new-" + newMachineConfigs[controlPlaneMachineConfigName].Spec.StoragePolicyName

	kubectl.EXPECT().GetEksaCluster(context.TODO(), gomock.Any()).Return(clusterConfig, nil)
	kubectl.EXPECT().GetEksaVSphereDatacenterConfig(context.TODO(), clusterConfig.Spec.DatacenterRef.Name, gomock.Any()).Return(newProviderConfig, nil)
	kubectl.EXPECT().GetEksaVSphereMachineConfig(context.TODO(), gomock.Any(), gomock.Any()).Return(newMachineConfigs[controlPlaneMachineConfigName], nil)

	err := provider.ValidateNewSpec(context.TODO(), &types.Cluster{})
	assert.Error(t, err, "StoragePolicyName should be immutable")
}

func TestValidateNewSpecStoragePolicyNameImmutableWorker(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	clusterConfig := givenClusterConfig(t, testClusterConfigMainFilename)

	provider := givenProvider(t)
	kubectl := mocks.NewMockProviderKubectlClient(mockCtrl)
	provider.providerKubectlClient = kubectl

	newProviderConfig := givenDatacenterConfig(t, testClusterConfigMainFilename)

	newMachineConfigs := givenMachineConfigs(t, testClusterConfigMainFilename)
	workerMachineConfigName := clusterConfig.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name
	newMachineConfigs[workerMachineConfigName].Spec.StoragePolicyName = "new-" + newMachineConfigs[workerMachineConfigName].Spec.StoragePolicyName

	kubectl.EXPECT().GetEksaCluster(context.TODO(), gomock.Any()).Return(clusterConfig, nil)
	kubectl.EXPECT().GetEksaVSphereDatacenterConfig(context.TODO(), clusterConfig.Spec.DatacenterRef.Name, gomock.Any()).Return(newProviderConfig, nil)
	kubectl.EXPECT().GetEksaVSphereMachineConfig(context.TODO(), gomock.Any(), gomock.Any()).Return(newMachineConfigs[workerMachineConfigName], nil)

	err := provider.ValidateNewSpec(context.TODO(), &types.Cluster{})
	assert.Error(t, err, "StoragePolicyName should be immutable")
}

func TestValidateNewSpecTLSInsecureImmutable(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	clusterConfig := givenClusterConfig(t, testClusterConfigMainFilename)

	provider := givenProvider(t)
	kubectl := mocks.NewMockProviderKubectlClient(mockCtrl)
	provider.providerKubectlClient = kubectl

	newProviderConfig := givenDatacenterConfig(t, testClusterConfigMainFilename)
	newProviderConfig.Spec.Insecure = !newProviderConfig.Spec.Insecure

	newMachineConfigs := givenMachineConfigs(t, testClusterConfigMainFilename)

	kubectl.EXPECT().GetEksaCluster(context.TODO(), gomock.Any()).Return(clusterConfig, nil)
	kubectl.EXPECT().GetEksaVSphereDatacenterConfig(context.TODO(), clusterConfig.Spec.DatacenterRef.Name, gomock.Any()).Return(newProviderConfig, nil)
	for _, config := range newMachineConfigs {
		kubectl.EXPECT().GetEksaVSphereMachineConfig(context.TODO(), gomock.Any(), gomock.Any()).Return(config, nil)
	}
	err := provider.ValidateNewSpec(context.TODO(), &types.Cluster{})
	assert.Error(t, err, "Insecure should be immutable")
}

func TestValidateNewSpecTLSThumbprintImmutable(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	clusterConfig := givenClusterConfig(t, testClusterConfigMainFilename)

	provider := givenProvider(t)
	kubectl := mocks.NewMockProviderKubectlClient(mockCtrl)
	provider.providerKubectlClient = kubectl

	newProviderConfig := givenDatacenterConfig(t, testClusterConfigMainFilename)
	newProviderConfig.Spec.Thumbprint = "new-" + newProviderConfig.Spec.Thumbprint

	newMachineConfigs := givenMachineConfigs(t, testClusterConfigMainFilename)

	kubectl.EXPECT().GetEksaCluster(context.TODO(), gomock.Any()).Return(clusterConfig, nil)
	kubectl.EXPECT().GetEksaVSphereDatacenterConfig(context.TODO(), clusterConfig.Spec.DatacenterRef.Name, gomock.Any()).Return(newProviderConfig, nil)
	for _, config := range newMachineConfigs {
		kubectl.EXPECT().GetEksaVSphereMachineConfig(context.TODO(), gomock.Any(), gomock.Any()).Return(config, nil)
	}
	err := provider.ValidateNewSpec(context.TODO(), &types.Cluster{})
	assert.Error(t, err, "Thumbprint should be immutable")
}

func TestValidateNewSpecMachineConfigSshUsersImmutable(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	clusterConfig := givenClusterConfig(t, testClusterConfigMainFilename)

	provider := givenProvider(t)
	kubectl := mocks.NewMockProviderKubectlClient(mockCtrl)
	provider.providerKubectlClient = kubectl

	newProviderConfig := givenDatacenterConfig(t, testClusterConfigMainFilename)
	newProviderConfig.Spec.Datacenter = "new-" + newProviderConfig.Spec.Datacenter

	newMachineConfigs := givenMachineConfigs(t, testClusterConfigMainFilename)

	kubectl.EXPECT().GetEksaCluster(context.TODO(), gomock.Any()).Return(clusterConfig, nil)
	kubectl.EXPECT().GetEksaVSphereDatacenterConfig(context.TODO(), clusterConfig.Spec.DatacenterRef.Name, gomock.Any()).Return(newProviderConfig, nil)
	kubectl.EXPECT().GetEksaVSphereMachineConfig(context.TODO(), gomock.Any(), gomock.Any()).Return(newMachineConfigs["test-cp"], nil)

	newMachineConfigs["test-cp"].Spec.Users[0].Name = "newNameShouldNotBeAllowed"

	err := provider.ValidateNewSpec(context.TODO(), &types.Cluster{})
	assert.Error(t, err, "User should be immutable")
}

func TestValidateNewSpecMachineConfigSshAuthKeysImmutable(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	clusterConfig := givenClusterConfig(t, testClusterConfigMainFilename)

	provider := givenProvider(t)
	kubectl := mocks.NewMockProviderKubectlClient(mockCtrl)
	provider.providerKubectlClient = kubectl

	newProviderConfig := givenDatacenterConfig(t, testClusterConfigMainFilename)
	newProviderConfig.Spec.Datacenter = "new-" + newProviderConfig.Spec.Datacenter

	newMachineConfigs := givenMachineConfigs(t, testClusterConfigMainFilename)

	kubectl.EXPECT().GetEksaCluster(context.TODO(), gomock.Any()).Return(clusterConfig, nil)
	kubectl.EXPECT().GetEksaVSphereDatacenterConfig(context.TODO(), clusterConfig.Spec.DatacenterRef.Name, gomock.Any()).Return(newProviderConfig, nil)
	kubectl.EXPECT().GetEksaVSphereMachineConfig(context.TODO(), gomock.Any(), gomock.Any()).Return(newMachineConfigs["test-cp"], nil)
	newMachineConfigs["test-cp"].Spec.Users[0].SshAuthorizedKeys = []string{"rsa ssh-asd;lfajsfl;asjdfl;asjdlfajsdlfjasl;djf"}

	err := provider.ValidateNewSpec(context.TODO(), &types.Cluster{})
	assert.Error(t, err, "SSH Authorized Keys should be immutable")
}