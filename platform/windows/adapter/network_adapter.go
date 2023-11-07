// Copyright 2017 Microsoft. All rights reserved.
// MIT License

package adapter

type NetworkAdapter interface {
	// GetAdapterNames returns array containing names of adapter if found
	// Must return error if adapter is not found or adapter name empty
	GetAdapterNames() ([]string, error)

	// Get PriorityVLANTag returns PriorityVLANTag value for Adapter
	GetPriorityVLANTag(adapterName string) (int, error)

	// Set adapter's PriorityVLANTag value to desired value if adapter exists
	SetPriorityVLANTag(adapterName string, value int) error
}
