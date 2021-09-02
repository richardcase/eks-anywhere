// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/aws/eks-anywhere/pkg/clustermanager (interfaces: ClusterClient,Networking)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	v1alpha1 "github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	cluster "github.com/aws/eks-anywhere/pkg/cluster"
	filewriter "github.com/aws/eks-anywhere/pkg/filewriter"
	providers "github.com/aws/eks-anywhere/pkg/providers"
	types "github.com/aws/eks-anywhere/pkg/types"
	gomock "github.com/golang/mock/gomock"
)

// MockClusterClient is a mock of ClusterClient interface.
type MockClusterClient struct {
	ctrl     *gomock.Controller
	recorder *MockClusterClientMockRecorder
}

// MockClusterClientMockRecorder is the mock recorder for MockClusterClient.
type MockClusterClientMockRecorder struct {
	mock *MockClusterClient
}

// NewMockClusterClient creates a new mock instance.
func NewMockClusterClient(ctrl *gomock.Controller) *MockClusterClient {
	mock := &MockClusterClient{ctrl: ctrl}
	mock.recorder = &MockClusterClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClusterClient) EXPECT() *MockClusterClientMockRecorder {
	return m.recorder
}

// ApplyKubeSpec mocks base method.
func (m *MockClusterClient) ApplyKubeSpec(arg0 context.Context, arg1 *types.Cluster, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ApplyKubeSpec", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// ApplyKubeSpec indicates an expected call of ApplyKubeSpec.
func (mr *MockClusterClientMockRecorder) ApplyKubeSpec(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ApplyKubeSpec", reflect.TypeOf((*MockClusterClient)(nil).ApplyKubeSpec), arg0, arg1, arg2)
}

// ApplyKubeSpecFromBytes mocks base method.
func (m *MockClusterClient) ApplyKubeSpecFromBytes(arg0 context.Context, arg1 *types.Cluster, arg2 []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ApplyKubeSpecFromBytes", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// ApplyKubeSpecFromBytes indicates an expected call of ApplyKubeSpecFromBytes.
func (mr *MockClusterClientMockRecorder) ApplyKubeSpecFromBytes(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ApplyKubeSpecFromBytes", reflect.TypeOf((*MockClusterClient)(nil).ApplyKubeSpecFromBytes), arg0, arg1, arg2)
}

// ApplyKubeSpecFromBytesForce mocks base method.
func (m *MockClusterClient) ApplyKubeSpecFromBytesForce(arg0 context.Context, arg1 *types.Cluster, arg2 []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ApplyKubeSpecFromBytesForce", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// ApplyKubeSpecFromBytesForce indicates an expected call of ApplyKubeSpecFromBytesForce.
func (mr *MockClusterClientMockRecorder) ApplyKubeSpecFromBytesForce(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ApplyKubeSpecFromBytesForce", reflect.TypeOf((*MockClusterClient)(nil).ApplyKubeSpecFromBytesForce), arg0, arg1, arg2)
}

// ApplyKubeSpecWithNamespace mocks base method.
func (m *MockClusterClient) ApplyKubeSpecWithNamespace(arg0 context.Context, arg1 *types.Cluster, arg2, arg3 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ApplyKubeSpecWithNamespace", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// ApplyKubeSpecWithNamespace indicates an expected call of ApplyKubeSpecWithNamespace.
func (mr *MockClusterClientMockRecorder) ApplyKubeSpecWithNamespace(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ApplyKubeSpecWithNamespace", reflect.TypeOf((*MockClusterClient)(nil).ApplyKubeSpecWithNamespace), arg0, arg1, arg2, arg3)
}

// DeleteCluster mocks base method.
func (m *MockClusterClient) DeleteCluster(arg0 context.Context, arg1, arg2 *types.Cluster) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteCluster", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteCluster indicates an expected call of DeleteCluster.
func (mr *MockClusterClientMockRecorder) DeleteCluster(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteCluster", reflect.TypeOf((*MockClusterClient)(nil).DeleteCluster), arg0, arg1, arg2)
}

// GetClusters mocks base method.
func (m *MockClusterClient) GetClusters(arg0 context.Context, arg1 *types.Cluster) ([]types.CAPICluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetClusters", arg0, arg1)
	ret0, _ := ret[0].([]types.CAPICluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetClusters indicates an expected call of GetClusters.
func (mr *MockClusterClientMockRecorder) GetClusters(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetClusters", reflect.TypeOf((*MockClusterClient)(nil).GetClusters), arg0, arg1)
}

// GetEksaCluster mocks base method.
func (m *MockClusterClient) GetEksaCluster(arg0 context.Context, arg1 *types.Cluster) (*v1alpha1.Cluster, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEksaCluster", arg0, arg1)
	ret0, _ := ret[0].(*v1alpha1.Cluster)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetEksaCluster indicates an expected call of GetEksaCluster.
func (mr *MockClusterClientMockRecorder) GetEksaCluster(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEksaCluster", reflect.TypeOf((*MockClusterClient)(nil).GetEksaCluster), arg0, arg1)
}

// GetEksaVSphereDatacenterConfig mocks base method.
func (m *MockClusterClient) GetEksaVSphereDatacenterConfig(arg0 context.Context, arg1, arg2 string) (*v1alpha1.VSphereDatacenterConfig, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEksaVSphereDatacenterConfig", arg0, arg1, arg2)
	ret0, _ := ret[0].(*v1alpha1.VSphereDatacenterConfig)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetEksaVSphereDatacenterConfig indicates an expected call of GetEksaVSphereDatacenterConfig.
func (mr *MockClusterClientMockRecorder) GetEksaVSphereDatacenterConfig(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEksaVSphereDatacenterConfig", reflect.TypeOf((*MockClusterClient)(nil).GetEksaVSphereDatacenterConfig), arg0, arg1, arg2)
}

// GetEksaVSphereMachineConfig mocks base method.
func (m *MockClusterClient) GetEksaVSphereMachineConfig(arg0 context.Context, arg1, arg2 string) (*v1alpha1.VSphereMachineConfig, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEksaVSphereMachineConfig", arg0, arg1, arg2)
	ret0, _ := ret[0].(*v1alpha1.VSphereMachineConfig)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetEksaVSphereMachineConfig indicates an expected call of GetEksaVSphereMachineConfig.
func (mr *MockClusterClientMockRecorder) GetEksaVSphereMachineConfig(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEksaVSphereMachineConfig", reflect.TypeOf((*MockClusterClient)(nil).GetEksaVSphereMachineConfig), arg0, arg1, arg2)
}

// GetMachines mocks base method.
func (m *MockClusterClient) GetMachines(arg0 context.Context, arg1 *types.Cluster) ([]types.Machine, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMachines", arg0, arg1)
	ret0, _ := ret[0].([]types.Machine)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMachines indicates an expected call of GetMachines.
func (mr *MockClusterClientMockRecorder) GetMachines(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMachines", reflect.TypeOf((*MockClusterClient)(nil).GetMachines), arg0, arg1)
}

// GetWorkloadKubeconfig mocks base method.
func (m *MockClusterClient) GetWorkloadKubeconfig(arg0 context.Context, arg1 string, arg2 *types.Cluster) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetWorkloadKubeconfig", arg0, arg1, arg2)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetWorkloadKubeconfig indicates an expected call of GetWorkloadKubeconfig.
func (mr *MockClusterClientMockRecorder) GetWorkloadKubeconfig(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetWorkloadKubeconfig", reflect.TypeOf((*MockClusterClient)(nil).GetWorkloadKubeconfig), arg0, arg1, arg2)
}

// InitInfrastructure mocks base method.
func (m *MockClusterClient) InitInfrastructure(arg0 context.Context, arg1 *cluster.Spec, arg2 *types.Cluster, arg3 providers.Provider) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InitInfrastructure", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// InitInfrastructure indicates an expected call of InitInfrastructure.
func (mr *MockClusterClientMockRecorder) InitInfrastructure(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InitInfrastructure", reflect.TypeOf((*MockClusterClient)(nil).InitInfrastructure), arg0, arg1, arg2, arg3)
}

// MoveManagement mocks base method.
func (m *MockClusterClient) MoveManagement(arg0 context.Context, arg1, arg2 *types.Cluster) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MoveManagement", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// MoveManagement indicates an expected call of MoveManagement.
func (mr *MockClusterClientMockRecorder) MoveManagement(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MoveManagement", reflect.TypeOf((*MockClusterClient)(nil).MoveManagement), arg0, arg1, arg2)
}

// RemoveAnnotationInNamespace mocks base method.
func (m *MockClusterClient) RemoveAnnotationInNamespace(arg0 context.Context, arg1, arg2, arg3 string, arg4 *types.Cluster, arg5 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveAnnotationInNamespace", arg0, arg1, arg2, arg3, arg4, arg5)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveAnnotationInNamespace indicates an expected call of RemoveAnnotationInNamespace.
func (mr *MockClusterClientMockRecorder) RemoveAnnotationInNamespace(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveAnnotationInNamespace", reflect.TypeOf((*MockClusterClient)(nil).RemoveAnnotationInNamespace), arg0, arg1, arg2, arg3, arg4, arg5)
}

// SaveLog mocks base method.
func (m *MockClusterClient) SaveLog(arg0 context.Context, arg1 *types.Cluster, arg2 *types.Deployment, arg3 string, arg4 filewriter.FileWriter) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveLog", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveLog indicates an expected call of SaveLog.
func (mr *MockClusterClientMockRecorder) SaveLog(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveLog", reflect.TypeOf((*MockClusterClient)(nil).SaveLog), arg0, arg1, arg2, arg3, arg4)
}

// UpdateAnnotationInNamespace mocks base method.
func (m *MockClusterClient) UpdateAnnotationInNamespace(arg0 context.Context, arg1, arg2 string, arg3 map[string]string, arg4 *types.Cluster, arg5 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateAnnotationInNamespace", arg0, arg1, arg2, arg3, arg4, arg5)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateAnnotationInNamespace indicates an expected call of UpdateAnnotationInNamespace.
func (mr *MockClusterClientMockRecorder) UpdateAnnotationInNamespace(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateAnnotationInNamespace", reflect.TypeOf((*MockClusterClient)(nil).UpdateAnnotationInNamespace), arg0, arg1, arg2, arg3, arg4, arg5)
}

// WaitForControlPlaneReady mocks base method.
func (m *MockClusterClient) WaitForControlPlaneReady(arg0 context.Context, arg1 *types.Cluster, arg2, arg3 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WaitForControlPlaneReady", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// WaitForControlPlaneReady indicates an expected call of WaitForControlPlaneReady.
func (mr *MockClusterClientMockRecorder) WaitForControlPlaneReady(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WaitForControlPlaneReady", reflect.TypeOf((*MockClusterClient)(nil).WaitForControlPlaneReady), arg0, arg1, arg2, arg3)
}

// WaitForDeployment mocks base method.
func (m *MockClusterClient) WaitForDeployment(arg0 context.Context, arg1 *types.Cluster, arg2, arg3, arg4, arg5 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WaitForDeployment", arg0, arg1, arg2, arg3, arg4, arg5)
	ret0, _ := ret[0].(error)
	return ret0
}

// WaitForDeployment indicates an expected call of WaitForDeployment.
func (mr *MockClusterClientMockRecorder) WaitForDeployment(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WaitForDeployment", reflect.TypeOf((*MockClusterClient)(nil).WaitForDeployment), arg0, arg1, arg2, arg3, arg4, arg5)
}

// WaitForManagedExternalEtcdReady mocks base method.
func (m *MockClusterClient) WaitForManagedExternalEtcdReady(arg0 context.Context, arg1 *types.Cluster, arg2, arg3 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WaitForManagedExternalEtcdReady", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// WaitForManagedExternalEtcdReady indicates an expected call of WaitForManagedExternalEtcdReady.
func (mr *MockClusterClientMockRecorder) WaitForManagedExternalEtcdReady(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WaitForManagedExternalEtcdReady", reflect.TypeOf((*MockClusterClient)(nil).WaitForManagedExternalEtcdReady), arg0, arg1, arg2, arg3)
}

// MockNetworking is a mock of Networking interface.
type MockNetworking struct {
	ctrl     *gomock.Controller
	recorder *MockNetworkingMockRecorder
}

// MockNetworkingMockRecorder is the mock recorder for MockNetworking.
type MockNetworkingMockRecorder struct {
	mock *MockNetworking
}

// NewMockNetworking creates a new mock instance.
func NewMockNetworking(ctrl *gomock.Controller) *MockNetworking {
	mock := &MockNetworking{ctrl: ctrl}
	mock.recorder = &MockNetworkingMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockNetworking) EXPECT() *MockNetworkingMockRecorder {
	return m.recorder
}

// GenerateManifest mocks base method.
func (m *MockNetworking) GenerateManifest(arg0 *cluster.Spec) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GenerateManifest", arg0)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GenerateManifest indicates an expected call of GenerateManifest.
func (mr *MockNetworkingMockRecorder) GenerateManifest(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GenerateManifest", reflect.TypeOf((*MockNetworking)(nil).GenerateManifest), arg0)
}