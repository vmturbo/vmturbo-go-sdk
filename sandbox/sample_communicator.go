package main

import (
	"bytes"
	"fmt"
	"github.com/vmturbo/vmturbo-go-sdk/communicator"
	"github.com/vmturbo/vmturbo-go-sdk/sdk"
	"net/http"
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
	vmtapi         *VMTApiRequestHandler
}

type VMTApiRequestHandler struct {
	vmtServerAddr      string
	opsManagerUsername string
	opsManagerPassword string
}

type Node struct {
	TypeMetaUID    string
	ObjectMetaName string
	// Spec of type NodeSpec defines the behavior of a node.
	NodeSpecPodCIDR       string
	NodeSpecExternalID    string
	NodeSpecProviderID    string
	NodeSpecUnschedulable bool

	// Status describes the current status of a Node
	//	Status NodeStatus `json:"status,omitempty"`
}

func (node *Node) createCommoditySold() []*sdk.CommodityDTO {
	var commoditiesSold []*sdk.CommodityDTO
	memUsed := float64(0)
	nodeMemCapacity := float64(1000)
	memAllocationComm := sdk.NewCommodtiyDTOBuilder(sdk.CommodityDTO_MEM_ALLOCATION).Key("Container").Capacity(float64(nodeMemCapacity)).Used(memUsed).Create()
	commoditiesSold = append(commoditiesSold, memAllocationComm)

	//appComm := sdk.NewCommodtiyDTOBuilder(sdk.CommodityDTO_APPLICATION).Key(node.TypeMetaUID).Create()
	//commoditiesSold = append(commoditiesSold, appComm)
	return commoditiesSold
}

func (nodeProbe *NodeProbe) buildPMEntityDTO(nodeID, displayName string, commoditiesSold []*sdk.CommodityDTO) *sdk.EntityDTO {
	cpuUsed := float64(0)
	memUsed := float64(0)
	nodeMemCapacity := float64(1000)
	nodeCpuCapacity := float64(1000)
	entityDTOBuilder := sdk.NewEntityDTOBuilder(sdk.EntityDTO_PHYSICAL_MACHINE, nodeID)
	entityDTOBuilder.DisplayName(displayName)
	entityDTOBuilder.SellsCommodities(commoditiesSold)
	entityDTOBuilder = entityDTOBuilder.Sells(sdk.CommodityDTO_MEM_ALLOCATION, "Container").Capacity(float64(nodeMemCapacity)).Used(memUsed)
	entityDTOBuilder = entityDTOBuilder.Sells(sdk.CommodityDTO_CPU_ALLOCATION, "Container").Capacity(float64(nodeCpuCapacity)).Used(cpuUsed)
	entityDTOBuilder = entityDTOBuilder.Sells(sdk.CommodityDTO_VMEM, nodeID).Capacity(float64(nodeMemCapacity)).Used(memUsed)
	entityDTOBuilder = entityDTOBuilder.Sells(sdk.CommodityDTO_VCPU, nodeID).Capacity(float64(nodeCpuCapacity)).Used(cpuUsed)
	entityDTOBuilder = entityDTOBuilder.SetProperty("IP", "172.16.162.133")
	metaData := nodeProbe.generateReconcilationMetaData()
	entityDTOBuilder = entityDTOBuilder.ReplacedBy(metaData)
	entityDTO := entityDTOBuilder.Create()
	return entityDTO
}

func (nodeProbe *NodeProbe) generateReconcilationMetaData() *sdk.EntityDTO_ReplacementEntityMetaData {
	replacementEntityMetaDataBuilder := sdk.NewReplacementEntityMetaDataBuilder()
	replacementEntityMetaDataBuilder.Matching("IP")
	replacementEntityMetaDataBuilder.PatchSelling(sdk.CommodityDTO_CPU_ALLOCATION)
	replacementEntityMetaDataBuilder.PatchSelling(sdk.CommodityDTO_MEM_ALLOCATION)
	replacementEntityMetaDataBuilder.PatchSelling(sdk.CommodityDTO_VCPU)
	replacementEntityMetaDataBuilder.PatchSelling(sdk.CommodityDTO_VMEM)
	replacementEntityMetaDataBuilder.PatchSelling(sdk.CommodityDTO_APPLICATION)
	metaData := replacementEntityMetaDataBuilder.Build()
	return metaData
}

func (nodeProbe *NodeProbe) buildVMEntityDTO(nodeID, displayName string, commoditiesSold []*sdk.CommodityDTO) *sdk.EntityDTO {
	// create fake VM
	entityDTOBuilder := sdk.NewEntityDTOBuilder(sdk.EntityDTO_VIRTUAL_MACHINE, nodeID)
	// Find out the used value for each commodity

	entityDTOBuilder.DisplayName(displayName)
	entityDTOBuilder.SellsCommodities(commoditiesSold)
	ipAddress := "10.10.173.131" // ask Dongyi, getIPForStitching from pkg/vmturbo/vmt/probe/node_probe.go
	entityDTOBuilder = entityDTOBuilder.SetProperty("IP", ipAddress)
	// not using nodeProbe.generateReconcilationMetaData()
	entityDTO := entityDTOBuilder.Create()

	return entityDTO
}

type NodeProbe struct {
	// nodesGetter func
	NodeArray []*Node // pkg.api.Node ?
}

type KubernetesProbe struct {
	//RestClient  KubeClient *client.Client
	//	GetNodes() returns a  []*api.Node made from  *api.NodeList .Items[] using label field  ,
	//	Items is a []Node
	nodeProbe *NodeProbe
}

func (kProbe *KubernetesProbe) getNodeProbe() *NodeProbe {
	return kProbe.nodeProbe
}

func (kProbe *KubernetesProbe) getNodeEntityDTOs() []*sdk.EntityDTO {
	// return NodeArray as []*sdk.EntityDTO
	nodearr := kProbe.getNodeProbe().NodeArray
	// loops through nodearr type []*Node
	nodeID := nodearr[0].TypeMetaUID
	dispName := nodearr[0].ObjectMetaName
	// call createCommoditySold to get []*sdk.CommodityDTO
	commodityDTO := nodearr[0].createCommoditySold()
	newEntityDTO := kProbe.getNodeProbe().buildPMEntityDTO(nodeID, dispName, commodityDTO)
	var entityDTOarray []*sdk.EntityDTO
	entityDTOarray = append(entityDTOarray, newEntityDTO)
	return entityDTOarray
}

func (vmtapi *VMTApiRequestHandler) vmtApiPost(postPath, requestStr string) (*http.Response, error) {
	fullUrl := "http://" + vmtapi.vmtServerAddr + "/vmturbo/api" + postPath + requestStr
	fmt.Println("Log: The ful Url is " + fullUrl)
	req, err := http.NewRequest("POST", fullUrl, nil)
	req.SetBasicAuth(vmtapi.opsManagerUsername, vmtapi.opsManagerPassword)
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		fmt.Println("Log: error getting response")
		return nil, err
	}
	defer response.Body.Close()
	return response, nil
}

func (h *MsgHandler) AddTarget() {

	//vmtServer := h.wscommunicator.VmtServerAddress
	// call vmtREST api
	// h.cInfo.Type h.cInfo.Name h.cInfo.Username h.cInfo.Password , h.cInfo.TargetIdentifier
	var requestDataB bytes.Buffer
	requestDataB.WriteString("?type=")
	requestDataB.WriteString(h.cInfo.Type)
	requestDataB.WriteString("&")
	requestDataB.WriteString("nameOrAddress=")
	requestDataB.WriteString(h.cInfo.Name)
	requestDataB.WriteString("&")
	requestDataB.WriteString("username=")
	requestDataB.WriteString(h.cInfo.Username)
	requestDataB.WriteString("&")
	requestDataB.WriteString("targetIdentifier=")
	requestDataB.WriteString(h.cInfo.TargetIdentifier)
	requestDataB.WriteString("&")
	requestDataB.WriteString("password=")
	requestDataB.WriteString(h.cInfo.Password)
	str := requestDataB.String()
	postReply, err := h.vmtapi.vmtApiPost("/externaltargets", str)
	if err != nil {
		fmt.Println(" postReply error")
	}
	fmt.Println("Printing AddTarget postReply:")
	fmt.Println(postReply)

	if postReply.Status != "200 OK" {
		fmt.Println(" postReplyMessage error")
	}
	fmt.Println("Add target response is " + postReply.Status)
}

func (h *MsgHandler) Validate(serverMsg *communicator.MediationServerMessage) {
	fmt.Println("validate called")
	// messageID is a int32 , if nil then 0
	messageID := serverMsg.GetMessageID()
	validationResponse := new(communicator.ValidationResponse)
	// add something in validation response fields?? TODO

	// creates a ClientMessageBuilder and sets ClientMessageBuilder.clientMessage.MessageID = messageID
	// sets clientMessage.ValidationResponse = validationResponse
	// type of clientMessage is MediationClientMessage
	clientMsg := communicator.NewClientMessageBuilder(messageID).SetValidationResponse(validationResponse).Create()
	h.wscommunicator.SendClientMessage(clientMsg)
	// discover TODO
	//  handler.meta.NameOrAddress passed to discoverTarget
	var requestDataB bytes.Buffer
	requestDataB.WriteString("?type=")
	requestDataB.WriteString(h.cInfo.Type)
	requestDataB.WriteString("&")
	requestDataB.WriteString("nameOrAddress=")
	requestDataB.WriteString(h.cInfo.Name)
	requestDataB.WriteString("&")
	requestDataB.WriteString("username=")
	requestDataB.WriteString(h.cInfo.Username)
	requestDataB.WriteString("&")
	requestDataB.WriteString("targetIdentifier=")
	requestDataB.WriteString(h.cInfo.TargetIdentifier)
	requestDataB.WriteString("&")
	requestDataB.WriteString("password=")
	requestDataB.WriteString(h.cInfo.Password)
	str := requestDataB.String()

	postReply, err := h.vmtapi.vmtApiPost("/targets", str)
	if err != nil {
		fmt.Println(" error in validate response from server")
		return
	}

	fmt.Println("Printing Validate postReply:")
	fmt.Println(postReply)
	if postReply.Status != "200 OK" {
		fmt.Println("Validate reply came in with error")
	}
	return
}
func (h *MsgHandler) DiscoverTopology(serverMsg *communicator.MediationServerMessage) {
	// TODO
	fmt.Println("DiscoverTopology called")

	messageID := serverMsg.GetMessageID()
	newNode := &Node{
		TypeMetaUID:    "pamelatestNode_PM_3",
		ObjectMetaName: "randomName_PM_3",
		// add more fields for this Node TODO

	}
	// make new NodeArray
	var newNodeArray []*Node
	newNodeArray = append(newNodeArray, newNode)
	newNodeProbe := &NodeProbe{
		NodeArray: newNodeArray,
	}
	simulatedProbe := &KubernetesProbe{
		nodeProbe: newNodeProbe,
	}
	// add some fake nodes to simulatdProbe or just created it in getNodeEntityDTOs
	nodeEntityDTOs := simulatedProbe.getNodeEntityDTOs() // []*sdk.EntityDTO
	//  use simulated kubeclient to do ParseNode and ParsePod
	discoveryResponse := &communicator.DiscoveryResponse{
		EntityDTO: nodeEntityDTOs,
	}
	clientMsg := communicator.NewClientMessageBuilder(messageID).SetDiscoveryResponse(discoveryResponse).Create()
	h.wscommunicator.SendClientMessage(clientMsg)
	// TODO h.DiscoverTarget()
	fmt.Println("done with discover")
	return
}
func (h *MsgHandler) HandleAction(serverMsg *communicator.MediationServerMessage) {
	// TODO
	fmt.Println("HandleAction called")
	return
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
	wsCommunicator.LocalAddress = "ws://172.16.162.133"
	wsCommunicator.ServerUsername = "vmtRemoteMediation"
	wsCommunicator.ServerPassword = "vmtRemoteMediation"
	loginInfo := new(ConnectionInfo)
	loginInfo.OpsManagerUsername = "administrator"
	loginInfo.OpsManagerPassword = "a"
	loginInfo.Type = "Kubernetes"
	loginInfo.Name = "k8s_vmt_pam_PM_4"
	loginInfo.Username = "kubernetes_user_pm4"
	loginInfo.Password = "fake_password"
	loginInfo.TargetIdentifier = "my_k8s_PM4"
	// ServerMessageHandler is implemented by MsgHandler
	msgHandler := new(MsgHandler)
	msgHandler.wscommunicator = wsCommunicator
	msgHandler.cInfo = loginInfo
	vMTApiRequestHandler := new(VMTApiRequestHandler)
	vMTApiRequestHandler.vmtServerAddr = wsCommunicator.VmtServerAddress
	vMTApiRequestHandler.opsManagerUsername = loginInfo.OpsManagerUsername
	vMTApiRequestHandler.opsManagerPassword = loginInfo.OpsManagerPassword
	msgHandler.vmtapi = vMTApiRequestHandler
	wsCommunicator.ServerMsgHandler = msgHandler

	containerInfo := CreateContainerInfo()
	fmt.Println("created container info ")
	wsCommunicator.RegisterAndListen(containerInfo)

}
