package main

import (
	"bytes"
	"fmt"
	"github.com/vmturbo/vmturbo-go-sdk/communicator"
	"github.com/vmturbo/vmturbo-go-sdk/sdk"
	//	"io/ioutil"
	"net/http"
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
	appComm := sdk.NewCommodtiyDTOBuilder(sdk.CommodityDTO_APPLICATION).Key(node.TypeMetaUID).Create()
	commoditiesSold = append(commoditiesSold, appComm)
	return commoditiesSold
}

func (nodeProbe *NodeProbe) buildVMEntityDTO(nodeID, displayName string, commoditiesSold []*sdk.CommodityDTO) *sdk.EntityDTO {
	entityDTOBuilder := sdk.NewEntityDTOBuilder(sdk.EntityDTO_VIRTUAL_MACHINE, nodeID)
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
	newEntityDTO := kProbe.getNodeProbe().buildVMEntityDTO(nodeID, dispName, commodityDTO)
	var entityDTOarray []*sdk.EntityDTO
	entityDTOarray = append(entityDTOarray, newEntityDTO)
	return entityDTOarray
}

/*
func (vmtapi *VMTApiRequestHandler) parseResponse(resp *http.Response) (string, error) {
	if resp == nil {
		return "", fmt.Errorf("response passed as argument is nil")
	}
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(content), nil
}*/

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
	//responseContent, _ := vmtapi.parsePostResponse(resp)
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
	//	postReplyMessage, err := vMTApiRequestHandler.parseResponse(postReply)

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
		TypeMetaUID:    "pamelatestNode2",
		ObjectMetaName: "randomName2",
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
	loginInfo.Name = "k8s_vmt_pam2"
	loginInfo.Username = "kubernetes_user"
	loginInfo.Password = "fake_password"
	loginInfo.TargetIdentifier = "my_k8s_other"
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
