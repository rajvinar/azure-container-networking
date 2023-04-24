// Copyright 2017 Microsoft. All rights reserved.
// MIT License

package platform

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Azure/azure-container-networking/log"
	"golang.org/x/sys/windows"
)

const (
	// CNMRuntimePath is the path where CNM state files are stored.
	CNMRuntimePath = ""

	// CNIRuntimePath is the path where CNI state files are stored.
	CNIRuntimePath = ""

	// CNILockPath is the path where CNI lock files are stored.
	CNILockPath = ""

	// CNIStateFilePath is the path to the CNI state file
	CNIStateFilePath = "C:\\k\\azure-vnet.json"

	// CNIIpamStatePath is the name of IPAM state file
	CNIIpamStatePath = "C:\\k\\azure-vnet-ipam.json"

	// CNIBinaryPath is the path to the CNI binary
	CNIBinaryPath = "C:\\k\\azurecni\\bin\\azure-vnet.exe"

	// CNI runtime path on a Kubernetes cluster
	K8SCNIRuntimePath = "C:\\k\\azurecni\\bin"

	// Network configuration file path on a Kubernetes cluster
	K8SNetConfigPath = "C:\\k\\azurecni\\netconf"

	// CNSRuntimePath is the path where CNS state files are stored.
	CNSRuntimePath = ""

	// NPMRuntimePath is the path where NPM state files are stored.
	NPMRuntimePath = ""

	// DNCRuntimePath is the path where DNC state files are stored.
	DNCRuntimePath = ""

	// SDNRemoteArpMacAddress is the registry key for the remote arp mac address.
	// This is set for multitenancy to get arp response from within VM
	// for vlan tagged arp requests
	SDNRemoteArpMacAddress = "12-34-56-78-9a-bc"

	// Command to get SDNRemoteArpMacAddress registry key
	GetSdnRemoteArpMacAddressCommand = "(Get-ItemProperty " +
		"-Path HKLM:\\SYSTEM\\CurrentControlSet\\Services\\hns\\State -Name SDNRemoteArpMacAddress).SDNRemoteArpMacAddress"

	// Command to set SDNRemoteArpMacAddress registry key
	SetSdnRemoteArpMacAddressCommand = "Set-ItemProperty " +
		"-Path HKLM:\\SYSTEM\\CurrentControlSet\\Services\\hns\\State -Name SDNRemoteArpMacAddress -Value \"12-34-56-78-9a-bc\""

	// Command to restart HNS service
	RestartHnsServiceCommand = "Restart-Service -Name hns"

	// Search string to find adapter having Mellanox in description
	mellanoxSearchString = "*Mellanox*"

	// PriorityVlanTag reg key for adapter
	priorityVLANTagIdentifier = "*PriorityVLANTag"

	// Registry key Path Prefix
	registryKeyPrefix = "HKLM:\\System\\CurrentControlSet\\Control\\Class\\"

	// Value for reg key: PriorityVLANTag for adapter
	// reg key value for PriorityVLANTag = 3  --> Packet priority and VLAN enabled
	// for more details goto https://learn.microsoft.com/en-us/windows-hardware/drivers/network/standardized-inf-keywords-for-ndis-qos
	desiredRegValueForVLANTag = 3

	// Interval between successive checks for mellanox adapter's PriorityVLANTag value
	defaultMellanoxMonitorInterval = 30 * time.Second
)

// Flag to check if sdnRemoteArpMacAddress registry key is set
var sdnRemoteArpMacAddressSet = false

// GetOSInfo returns OS version information.
func GetOSInfo() string {
	return "windows"
}

func GetProcessSupport() error {
	cmd := fmt.Sprintf("Get-Process -Id %v", os.Getpid())
	_, err := ExecutePowershellCommand(cmd)
	return err
}

var tickCount = syscall.NewLazyDLL("kernel32.dll").NewProc("GetTickCount64")

// GetLastRebootTime returns the last time the system rebooted.
func GetLastRebootTime() (time.Time, error) {
	currentTime := time.Now()
	output, _, err := tickCount.Call()
	if errno, ok := err.(syscall.Errno); !ok || errno != 0 {
		log.Printf("Failed to call GetTickCount64, err: %v", err)
		return time.Time{}.UTC(), err
	}
	rebootTime := currentTime.Add(-time.Duration(output) * time.Millisecond).Truncate(time.Second)
	log.Printf("Formatted Boot time: %s", rebootTime.Format(time.RFC3339))
	return rebootTime.UTC(), nil
}

func (p *execClient) ExecuteCommand(command string) (string, error) {
	log.Printf("[Azure-Utils] %s", command)

	var stderr bytes.Buffer
	var out bytes.Buffer
	cmd := exec.Command("cmd", "/c", command)
	cmd.Stderr = &stderr
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%s:%s", err.Error(), stderr.String())
	}

	return out.String(), nil
}

func SetOutboundSNAT(subnet string) error {
	return nil
}

// ClearNetworkConfiguration clears the azure-vnet.json contents.
// This will be called only when reboot is detected - This is windows specific
func ClearNetworkConfiguration() (bool, error) {
	jsonStore := CNIRuntimePath + "azure-vnet.json"
	log.Printf("Deleting the json store %s", jsonStore)
	cmd := exec.Command("cmd", "/c", "del", jsonStore)

	if err := cmd.Run(); err != nil {
		log.Printf("Error deleting the json store %s", jsonStore)
		return true, err
	}

	return true, nil
}

func KillProcessByName(processName string) {
	p := NewExecClient()
	cmd := fmt.Sprintf("taskkill /IM %v /F", processName)
	p.ExecuteCommand(cmd)
}

// ExecutePowershellCommand executes powershell command
func ExecutePowershellCommand(command string) (string, error) {
	ps, err := exec.LookPath("powershell.exe")
	if err != nil {
		return "", fmt.Errorf("Failed to find powershell executable")
	}

	log.Printf("[Azure-Utils] %s", command)

	cmd := exec.Command(ps, command)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%s:%s", err.Error(), stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// SetSdnRemoteArpMacAddress sets the regkey for SDNRemoteArpMacAddress needed for multitenancy
func SetSdnRemoteArpMacAddress() error {
	if !sdnRemoteArpMacAddressSet {
		result, err := ExecutePowershellCommand(GetSdnRemoteArpMacAddressCommand)
		if err != nil {
			return err
		}

		// Set the reg key if not already set or has incorrect value
		if result != SDNRemoteArpMacAddress {
			if _, err = ExecutePowershellCommand(SetSdnRemoteArpMacAddressCommand); err != nil {
				log.Printf("Failed to set SDNRemoteArpMacAddress due to error %s", err.Error())
				return err
			}

			log.Printf("[Azure CNS] SDNRemoteArpMacAddress regKey set successfully. Restarting hns service.")
			if _, err := ExecutePowershellCommand(RestartHnsServiceCommand); err != nil {
				log.Printf("Failed to Restart HNS Service due to error %s", err.Error())
				return err
			}
		}

		sdnRemoteArpMacAddressSet = true
	}

	return nil
}

func HasMellanoxAdapter() bool {
	adapterName, err := getMellanoxAdapterName()
	if err != nil {
		log.Errorf("Error while getting mellanox adapter name: %v", err)
		return false
	}
	log.Printf("Name of Mellanox adapter : %v", adapterName)
	return true
}

// Regularly monitors the Mellanox PriorityVLANGTag registry value and sets it to desired value if needed
func MonitorAndSetMellanoxRegKeyPriorityVLANTag(ctx context.Context, intervalSecs int) {
	interval := defaultMellanoxMonitorInterval
	if intervalSecs > 0 {
		interval = time.Duration(intervalSecs) * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Printf("context cancelled, stopping Mellanox Monitoring:", ctx.Err())
			return
		case <-ticker.C:
			adapterName, err := getMellanoxAdapterName()
			if err != nil {
				log.Errorf("getMellanoxAdapterName returned err: %v and adapterName: %s", err, adapterName)
			}

			err = SetMellanoxPriorityVLANTag(adapterName)
			if err != nil {
				log.Errorf("error while monitoring and setting Mellanox Reg Key value: %v", err)
			}
		}
	}
}

func getMellanoxAdapterName() (string, error) {
	//get mellanox adapter name
	cmd := fmt.Sprintf(`Get-NetAdapter | Where-Object { $_.InterfaceDescription -like "%s" } | Select-Object -ExpandProperty Name`, mellanoxSearchString)
	adapterName, err := ExecutePowershellCommand(cmd)
	if err != nil {
		return "", fmt.Errorf("error while executing powershell command to get net adapter list: %w", err)
	}
	if adapterName == "" {
		return "", fmt.Errorf("no network adapter found with %s in description", mellanoxSearchString)
	}
	return adapterName, nil
}

// Set Mellanox adapter's PriorityVLANTag value to 3 if adapter exists
// reg key value for PriorityVLANTag = 3  --> Packet priority and VLAN enabled
// for more details goto https://docs.nvidia.com/networking/display/winof2v230/Configuring+the+Driver+Registry+Keys#ConfiguringtheDriverRegistryKeys-GeneralRegistryKeysGeneralRegistryKeys
func SetMellanoxPriorityVLANTag(adapterName string) error {
	//Find if adapter has property PriorityVLANTag (version 4 or up) or not (version 3)
	cmd := fmt.Sprintf(`Get-NetAdapterAdvancedProperty | Where-Object { $_.RegistryKeyword -like "%s" -and $_.Name -eq "%s" } | Select-Object -ExpandProperty Name`, priorityVLANTagIdentifier, adapterName)
	adapterNameWithVLANTag, err := ExecutePowershellCommand(cmd)
	if err != nil {
		return fmt.Errorf("error while executing powershell command to get VLAN Tag advance property of %s: %w", adapterName, err)
	}

	if adapterNameWithVLANTag != "" {
		err = setMellanoxPriorityVLANTagValueForV4(adapterNameWithVLANTag)
	} else {
		err = setMellanoxPriorityVLANTagValueForV3(adapterName)
	}

	return err
}

// Checks if a Mellanox adapter's PriorityVLANTag value
// for version 4 and up is set to the given expected value
func getMellanoxPriorityVLANTagValueForV4(adapterName string) (int, error) {
	cmd := fmt.Sprintf(
		`Get-NetAdapterAdvancedProperty | Where-Object { $_.RegistryKeyword -like "%s" -and $_.Name -eq "%s" } | Select-Object -ExpandProperty RegistryValue`,
		priorityVLANTagIdentifier, adapterName)

	regvalue, err := ExecutePowershellCommand(cmd)
	if err != nil {
		return 0, err
	}

	intValue, err := strconv.Atoi(regvalue)
	if err != nil {
		return 0, fmt.Errorf("failed to convert PriorityVLANTag value to integer: %w", err)
	}

	return intValue, nil
}

// Checks if a Mellanox adapter's PriorityVLANTag value
// for version 3 and below is set to the given expected value
func getMellanoxPriorityVLANTagValueForV3(registryKeyFullPath, adapterName string) (int, error) {
	cmd := fmt.Sprintf(
		`Get-ItemProperty -Path "%s" -Name "%s" | Select-Object -ExpandProperty "%s"`, registryKeyFullPath, priorityVLANTagIdentifier, priorityVLANTagIdentifier)
	regvalue, err := ExecutePowershellCommand(cmd)
	if err != nil {
		return 0, err
	}

	intValue, err := strconv.Atoi(regvalue)
	if err != nil {
		return 0, fmt.Errorf("failed to convert PriorityVLANTag value to integer: %w", err)
	}

	return intValue, nil
}

// adapter is version 4 and up since adapter's advance property consists of reg key : PriorityVLANTag
// set reg value for Priorityvlantag of adapter to 3 if not set already
func setMellanoxPriorityVLANTagValueForV4(adapterName string) error {
	currentVLANTagValue, err := getMellanoxPriorityVLANTagValueForV4(adapterName)
	if err != nil {
		return fmt.Errorf("error while checking registry value for PriorityVLANTag for adapter: %v", err)
	}

	if currentVLANTagValue == desiredRegValueForVLANTag {
		log.Printf("Mellanox PriorityVLANTag is already set to %v, skipping reset", desiredRegValueForVLANTag)
		return nil
	}

	cmd := fmt.Sprintf(
		`Set-NetAdapterAdvancedProperty -Name "%s" -RegistryKeyword "%s" -RegistryValue %d`, adapterName, priorityVLANTagIdentifier, desiredRegValueForVLANTag)
	_, err = ExecutePowershellCommand(cmd)
	if err != nil {
		return fmt.Errorf("error while setting up registry value for PriorityVLANTag for adapter: %w", err)
	}

	log.Printf("Successfully set Mellanox Network Adapter: %s with %s property value as %d", adapterName, priorityVLANTagIdentifier, desiredRegValueForVLANTag)
	return nil
}

// Adapter is version 3 or less as PriorityVLANTag was not found in advanced properties of mellanox adpater
func setMellanoxPriorityVLANTagValueForV3(adapterName string) error {
	log.Printf("Searching through CIM instances for Network devices with %s in the name", mellanoxSearchString)
	cmd := fmt.Sprintf(`Get-CimInstance -Namespace root/cimv2 -ClassName Win32_PNPEntity | Where-Object PNPClass -EQ "Net" | Where-Object { $_.Name -like "%s" } | Select-Object -ExpandProperty DeviceID`, mellanoxSearchString)
	deviceid, err := ExecutePowershellCommand(cmd)

	if err != nil {
		return fmt.Errorf("error while executing powershell command to get device id of %s: %w", adapterName, err)
	}
	if deviceid == "" {
		return fmt.Errorf("no network device found with %s in description", mellanoxSearchString)
	}

	log.Printf("Device ID found and Getting PNP device properites for %s", deviceid)
	cmd = fmt.Sprintf(`Get-PnpDeviceProperty -InstanceId "%s" | Where-Object KeyName -EQ "DEVPKEY_Device_Driver" | Select-Object -ExpandProperty Data`, deviceid)
	registryKeySuffix, err := ExecutePowershellCommand(cmd)
	if err != nil {
		return fmt.Errorf("error while executing powershell command to get registry suffix of device id %s: %w", deviceid, err)
	}

	registryKeyFullPath := registryKeyPrefix + registryKeySuffix

	currentVLANTagValue, err := getMellanoxPriorityVLANTagValueForV3(registryKeyFullPath, adapterName)
	if err != nil {
		return fmt.Errorf("error while checking registry value for PriorityVLANTag for adapter: %v", err)
	}

	if currentVLANTagValue == desiredRegValueForVLANTag {
		log.Printf("Mellanox PriorityVLANTag is already set to %v, skipping reset", desiredRegValueForVLANTag)
		return nil
	}

	cmd = fmt.Sprintf(`New-ItemProperty -Path "%s" -Name "%s" -Value %d -PropertyType String -Force`, registryKeyFullPath, priorityVLANTagIdentifier, desiredRegValueForVLANTag)
	_, err = ExecutePowershellCommand(cmd)
	if err != nil {
		return fmt.Errorf("error while executing powershell command to set Item property for device id  %s: %w", deviceid, err)
	}

	log.Printf("Restarting Mellanox network adapter for regkey change to take effect")
	cmd = fmt.Sprintf(`Restart-NetAdapter -Name "%s"`, adapterName)
	_, err = ExecutePowershellCommand(cmd)
	if err != nil {
		return fmt.Errorf("error while executing powershell command to restart net adapter  %s: %w", adapterName, err)
	}
	log.Printf("For Mellanox CX-3 adapters, the reg key set to %d", desiredRegValueForVLANTag)
	return nil
}

func GetOSDetails() (map[string]string, error) {
	return nil, nil
}

func GetProcessNameByID(pidstr string) (string, error) {
	pidstr = strings.Trim(pidstr, "\r\n")
	cmd := fmt.Sprintf("Get-Process -Id %s|Format-List", pidstr)
	out, err := ExecutePowershellCommand(cmd)
	if err != nil {
		log.Printf("Process is not running. Output:%v, Error %v", out, err)
		return "", err
	}

	if len(out) <= 0 {
		log.Printf("Output length is 0")
		return "", fmt.Errorf("get-process output length is 0")
	}

	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Name") {
			pName := strings.Split(line, ":")
			if len(pName) > 1 {
				return strings.TrimSpace(pName[1]), nil
			}
		}
	}

	return "", fmt.Errorf("Process not found")
}

func PrintDependencyPackageDetails() {
}

// https://docs.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-movefileexw
func ReplaceFile(source, destination string) error {
	src, err := syscall.UTF16PtrFromString(source)
	if err != nil {
		return err
	}

	dest, err := syscall.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}

	return windows.MoveFileEx(src, dest, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}
