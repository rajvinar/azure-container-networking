package platform

import (
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/Azure/azure-container-networking/platform/windows/adapter/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTestFailure = errors.New("test failure")

// Test if hasNetworkAdapter returns false on actual error or empty adapter name(an error)
func TestHasNetworkAdapterReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNetworkAdapter := mocks.NewMockNetworkAdapter(ctrl)
	mockNetworkAdapter.EXPECT().GetAdapterNames().Return([]string{}, errTestFailure)

	result := hasNetworkAdapter(mockNetworkAdapter)
	assert.False(t, result)
}

// Test if hasNetworkAdapter returns false on actual error or empty adapter name(an error)
func TestHasNetworkAdapterAdapterReturnsEmptyAdapterName(t *testing.T) {
	t.Skip()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNetworkAdapter := mocks.NewMockNetworkAdapter(ctrl)
	mockNetworkAdapter.EXPECT().GetAdapterNames().Return([]string{"Ethernet 3", "Ethernet 2"}, nil)
	result := hasNetworkAdapter(mockNetworkAdapter)
	assert.True(t, result)
}

// Test if updatePriorityVLANTagIfRequired returns error on getting error on calling getpriorityvlantag
func TestUpdatePriorityVLANTagIfRequiredReturnsError(t *testing.T) {
	t.Skip()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNetworkAdapter := mocks.NewMockNetworkAdapter(ctrl)
	mockNetworkAdapter.EXPECT().GetAdapterNames().Return([]string{"Ethernet 3", "Ethernet 2"}, nil)
	mockNetworkAdapter.EXPECT().GetPriorityVLANTag("Ethernet 3").Return(0, errTestFailure)
	mockNetworkAdapter.EXPECT().GetPriorityVLANTag("Ethernet 2").Return(0, nil)
	updatePriorityVLANTagIfRequired(mockNetworkAdapter, 3)
}

// Test if updatePriorityVLANTagIfRequired returns nil if currentval == desiredvalue (SetPriorityVLANTag not being called)
func TestUpdatePriorityVLANTagIfRequiredIfCurrentValEqualDesiredValue(t *testing.T) {
	t.Skip()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNetworkAdapter := mocks.NewMockNetworkAdapter(ctrl)
	mockNetworkAdapter.EXPECT().GetAdapterNames().Return([]string{"Ethernet 3", "Ethernet 2"}, nil)
	mockNetworkAdapter.EXPECT().GetPriorityVLANTag("Ethernet 3").Return(4, nil)
	mockNetworkAdapter.EXPECT().GetPriorityVLANTag("Ethernet 2").Return(4, nil)
	updatePriorityVLANTagIfRequired(mockNetworkAdapter, 4)
}

// Test if updatePriorityVLANTagIfRequired returns nil if SetPriorityVLANTag being called to set value
func TestUpdatePriorityVLANTagIfRequiredIfCurrentValNotEqualDesiredValAndSetReturnsNoError(t *testing.T) {
	t.Skip()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNetworkAdapter := mocks.NewMockNetworkAdapter(ctrl)
	mockNetworkAdapter.EXPECT().GetAdapterNames().Return([]string{"Ethernet 3", "Ethernet 2"}, nil)
	mockNetworkAdapter.EXPECT().GetPriorityVLANTag("Ethernet 3").Return(1, nil)
	mockNetworkAdapter.EXPECT().SetPriorityVLANTag("Ethernet 3", 2).Return(nil)
	mockNetworkAdapter.EXPECT().GetPriorityVLANTag("Ethernet 2").Return(1, nil)
	mockNetworkAdapter.EXPECT().SetPriorityVLANTag("Ethernet 2", 2).Return(nil)
	updatePriorityVLANTagIfRequired(mockNetworkAdapter, 2)
}

// Test if updatePriorityVLANTagIfRequired returns error if SetPriorityVLANTag throwing error

func TestUpdatePriorityVLANTagIfRequiredIfCurrentValNotEqualDesiredValAndSetReturnsError(t *testing.T) {
	t.Skip()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNetworkAdapter := mocks.NewMockNetworkAdapter(ctrl)
	mockNetworkAdapter.EXPECT().GetAdapterNames().Return([]string{"Ethernet 3", "Ethernet 2"}, nil)
	mockNetworkAdapter.EXPECT().GetPriorityVLANTag("Ethernet 3").Return(1, nil)
	mockNetworkAdapter.EXPECT().SetPriorityVLANTag("Ethernet 3", 5).Return(errTestFailure)
	mockNetworkAdapter.EXPECT().GetPriorityVLANTag("Ethernet 2").Return(1, nil)
	mockNetworkAdapter.EXPECT().SetPriorityVLANTag("Ethernet 2", 5).Return(errTestFailure)
	updatePriorityVLANTagIfRequired(mockNetworkAdapter, 5)
}

func TestExecuteCommand(t *testing.T) {
	out, err := NewExecClient(nil).ExecuteCommand("dir")
	require.NoError(t, err)
	require.NotEmpty(t, out)
}

func TestExecuteCommandError(t *testing.T) {
	_, err := NewExecClient(nil).ExecuteCommand("dontaddtopath")
	require.Error(t, err)

	var xErr *exec.ExitError
	assert.ErrorAs(t, err, &xErr)
	assert.Equal(t, 1, xErr.ExitCode())
}

func TestSetSdnRemoteArpMacAddress_hnsNotEnabled(t *testing.T) {
	mockExecClient := NewMockExecClient(false)
	// testing skip setting SdnRemoteArpMacAddress when hns not enabled
	mockExecClient.SetPowershellCommandResponder(func(_ string) (string, error) {
		return "False", nil
	})
	err := SetSdnRemoteArpMacAddress(mockExecClient)
	assert.NoError(t, err)
	assert.Equal(t, false, sdnRemoteArpMacAddressSet)

	// testing the scenario when there is an error in checking if hns is enabled or not
	mockExecClient.SetPowershellCommandResponder(func(_ string) (string, error) {
		return "", errTestFailure
	})
	err = SetSdnRemoteArpMacAddress(mockExecClient)
	assert.ErrorAs(t, err, &errTestFailure)
	assert.Equal(t, false, sdnRemoteArpMacAddressSet)
}

func TestSetSdnRemoteArpMacAddress_hnsEnabled(t *testing.T) {
	mockExecClient := NewMockExecClient(false)
	// happy path
	mockExecClient.SetPowershellCommandResponder(func(cmd string) (string, error) {
		if strings.Contains(cmd, "Test-Path") {
			return "True", nil
		}
		return "", nil
	})
	err := SetSdnRemoteArpMacAddress(mockExecClient)
	assert.NoError(t, err)
	assert.Equal(t, true, sdnRemoteArpMacAddressSet)
	// reset sdnRemoteArpMacAddressSet
	sdnRemoteArpMacAddressSet = false
}
