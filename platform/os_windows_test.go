package platform

import (
	"errors"
	"testing"

	"github.com/Azure/azure-container-networking/platform/windows/adapter/mellanox/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

// Test if HasMellanoxAdapter returns false on actual error or empty adapter name(an error)
func TestHasMellanoxAdapterReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMellanox := mocks.NewMockMellanox(ctrl)
	mockMellanox.EXPECT().GetAdapaterName().Return("", errors.New("failed to get adapter name"))

	result := hasMellanoxAdapter(mockMellanox)
	assert.False(t, result)
}

// Test if HasMellanoxAdapter returns false on actual error or empty adapter name(an error)
func TestHasMellanoxAdapterReturnsEmptyAdapterName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMellanox := mocks.NewMockMellanox(ctrl)
	mockMellanox.EXPECT().GetAdapaterName().Return("Ethernet 3", nil)

	result := hasMellanoxAdapter(mockMellanox)
	assert.True(t, result)
}
