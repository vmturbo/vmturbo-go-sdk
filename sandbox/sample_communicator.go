package main

import (
	"fmt"
	"github.com/vmturbo/vmturbo-go-sdk/communicator"
)

type Handler struct {
	// TODO
}

func (h Handler) AddTarget() {
	// TODO
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

func CreateContainerInfo() *communicator.ContainerInfo {
	strtype := new(string)
	strcat := new(string)
	*strtype = "type1"
	*strcat = "cat1"

	probeInfo := &communicator.ProbeInfo{
		ProbeType:     strtype,
		ProbeCategory: strcat,
		// SupplyChainDefinitionSet
		// AccountDefinition
		// XXX_unrecognized
	}
	containerInfo := new(communicator.ContainerInfo)
	probes := append(containerInfo.Probes, probeInfo)
	containerInfo.Probes = probes

	// set the MediationClientMessage member variables

	//m := &communicator.MediationClientMessage{
	//      ValidationResponse:     ,
	//      DiscoveryResponse:      ,
	//      KeepAlive:      ,
	//      ActionProgress:         ,
	//      MessageID:      ,
	//}
	return containerInfo
}

func main() {

	wsCommunicator := new(communicator.WebSocketCommunicator)
	wsCommunicator.VmtServerAddress = "192.168.1.105:9400"
	// ex: "ws://172.16.162.244"
	wsCommunicator.LocalAddress = "ws://172.16.162.131"

	wsCommunicator.ServerUsername = "administrator"
	wsCommunicator.ServerPassword = "a"
	// ServerMessageHandler is implemented by Handler for now
	// set wsCommnunicator.ServerMsgHandler = ....
	msgHandler := new(Handler)
	wsCommunicator.ServerMsgHandler = *msgHandler
	//	registrationMessage := CreateMediationClientMessage()
	containerInfo := CreateContainerInfo()
	fmt.Println("created container info ")
	wsCommunicator.RegisterAndListen(containerInfo)

}
