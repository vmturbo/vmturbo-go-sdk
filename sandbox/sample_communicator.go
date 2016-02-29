package main

import (
	"fmt"
	"github.com/vmturbo/vmturbo-go-sdk/communicator"
	//	"github.com/pamelasanchezvi/vmturbo-go-sdk/communicator"
)

// Struct which hold identifying information for connecting to the VMTServer
type ConnectionInfo struct {
	ServerAddr         string
	LocalAddr          string
	Type               string
	OpsManagerUsername string
	OpsManagerPassword string
}

// implementation of communicator.ServerMessageHandler interface
type MsgHandler struct {
	wscommunicator *communicator.WebSocketCommunicator
	cInfo          *ConnectionInfo
}

func (h *MsgHandler) AddTarget() {
	configMap := make(map[string]string)
	configMap["Username"] = h.OpsManagerUsername
	configMap["Password"] = h.OpsManagerPassword
	vmtServer := h.wscommunicator.VmtServerAddress
	// call vmtREST api
	fmt.Println("add target called")

}

func (h Handler) Validate(serverMsg *communicator.MediationServerMessage) {
	// TODO
	fmt.Println("validate called")

}
func (h Handler) DiscoverTopology(serverMsg *communicator.MediationServerMessage) {
	// TODO
	fmt.Println("DiscoverTopology called")

}
func (h Handler) HandleAction(serverMsg *communicator.MediationServerMessage) {
	// TODO
	fmt.Println("HandleAction called")

}

// Function Creates ContainerInfo struct, sets Kubernetes Container Probe Information
// Returns pointer to newly created ContainerInfo
func CreateContainerInfo() *communicator.ContainerInfo {
	strtype := "Kubernetes"
	strcat := "Container"
	//create the ProbeInfo struct with only type and category fields
	probeInfo := &communicator.ProbeInfo{
		ProbeType:     &strtype,
		ProbeCategory: &strcat,
		// SupplyChainDefinitionSet
		// AccountDefinition
		// XXX_unrecognized
	}
	// Create container
	containerInfo := new(communicator.ContainerInfo)
	// Add probe to array of ProbeInfo* in container
	probes := append(containerInfo.Probes, probeInfo)
	containerInfo.Probes = probes
	return containerInfo
}

func main() {

	wsCommunicator := new(communicator.WebSocketCommunicator)
	wsCommunicator.VmtServerAddress = "10.10.200.98:8080"
	wsCommunicator.LocalAddress = "ws://172.16.162.131"
	wsCommunicator.ServerUsername = "vmtRemoteMediation"
	wsCommunicator.ServerPassword = "vmtRemoteMediation"
	// ServerMessageHandler is implemented by MsgHandler
	msgHandler := new(MsgHandler)
	msgHandler.wscommunicator = wsCommunicator
	wsCommunicator.ServerMsgHandler = *msgHandler

	containerInfo := CreateContainerInfo()
	fmt.Println("created container info ")
	wsCommunicator.RegisterAndListen(containerInfo)

}
