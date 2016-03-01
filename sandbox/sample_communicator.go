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
	Name               string
	Username           string
	Password           string
	TargetIdentifier   string
}

// implementation of communicator.ServerMessageHandler interface
type MsgHandler struct {
	wscommunicator *communicator.WebSocketCommunicator
	cInfo          *ConnectionInfo
}

type VMTApiRequestHandler struct {
	vmtServerAddr      string
	opsManagerUsername string
	opsManagerPassword string
}

func (VMTApiRequestHandler *vmtapi) VmtApiPost(postPath, requestStr string) (*http.Response, error) {
	fullUrl := "http://" + vmtapi.vmtServerAddr + "/vmturbo/api" + postPath + requestStr
}

func (h *MsgHandler) AddTarget() {

	vMTApiRequestHandler = new(VMTApiRequestHandler)
	vMTApiRequestHandler.vmtServerAddr = h.wscommunicator.VmtServerAddress
	vMTApiRequestHandler.opsManagerUsername = h.cInfo.OpsManagerUsername
	vMTApiRequestHandler.opsManagerPassword = h.cInfo.OpsManagerPassword
	//vmtServer := h.wscommunicator.VmtServerAddress
	// call vmtREST api
	// h.cInfo.Type h.cInfo.Name h.cInfo.Username h.cInfo.Password , h.cInfo.TargetIdentifier
	var requestDataB bytes.Buffer
	requestDataB.WriteString("?type=")
	requestDataB.WriteString(h.cInfo.Type)
	requestDataB.WriteString("&")
	requestDataB.WriteString("nameOrAddress=")
	requestDataB.WriteString(h.cInfo.Name)
	request.DataB.WriteString("&")
	requestDataB.WriteString("username=")
	requestDataB.WriteString(h.cInfo.Username)
	requestDataB.WriteString("&")
	requestDataB.WriteString("targetIdentifier=")
	requestDataB.WriteString(h.cInfo.TargetIdentifier)
	requestDataB.WriteString("&")
	requestDataB.WriteString("password")
	requestDataB.WriteString(h.cInfo.Password)
	str := requestDataB.String()
	postReply, err := vmtApiPost()
	fmt.Println("add target called")

}

func (h *MsgHandler) Validate(serverMsg *communicator.MediationServerMessage) {
	// TODO
	fmt.Println("validate called")

}
func (h *MsgHandler) DiscoverTopology(serverMsg *communicator.MediationServerMessage) {
	// TODO
	fmt.Println("DiscoverTopology called")

}
func (h *MsgHandler) HandleAction(serverMsg *communicator.MediationServerMessage) {
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
	loginInfo := new(ConnectionInfo)
	loginInfo.OpsManagerUsername = "administrator"
	loginInfo.OpsManagerPassword = "a"
	loginInfo.Type = "Kubernetes"
	loginInfo.Name = "kube_vmt"
	loginInfo.Username = "kubernetes_user"
	loginInfo.Password = "password"
	loginInfo.TargetIdentifier = "my_k8s"
	// ServerMessageHandler is implemented by MsgHandler
	msgHandler := new(MsgHandler)
	msgHandler.wscommunicator = wsCommunicator
	msgHandler.cInfo = loginInfo
	wsCommunicator.ServerMsgHandler = msgHandler

	containerInfo := CreateContainerInfo()
	fmt.Println("created container info ")
	wsCommunicator.RegisterAndListen(containerInfo)

}
