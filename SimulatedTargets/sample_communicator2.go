package main

import (
	"bytes"
	"fmt"
	"github.com/vmturbo/vmturbo-go-sdk/communicator"
	"github.com/vmturbo/vmturbo-go-sdk/sdk"
	"net/http"
	//	"github.com/pamelasanchezvi/vmturbo-go-sdk/communicator"

	"github.com/golang/glog"
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

func (node *Node) createCommoditiesBought() []*sdk.CommodityDTO {
	var commoditiesBought []*sdk.CommodityDTO
	cpuUsed := float64(10)
	cpuCapacity := float64(100)
	// TODO correct spelling in github for vmturbo !
	cpuComm := sdk.NewCommodtiyDTOBuilder(sdk.CommodityDTO_CPU).Key("cpu_comm").Capacity(float64(cpuCapacity)).Used(cpuUsed).Create()
	commoditiesBought = append(commoditiesBought, cpuComm)
	return commoditiesBought
}

func (node *Node) createCommoditiesSold() []*sdk.CommodityDTO {
	var commoditiesSold []*sdk.CommodityDTO
	cpuUsed := float64(0)
	cpuCapacity := float64(1000)
	// TODO should use an array of commodities sold by this node, find out
	cpuComm := sdk.NewCommodtiyDTOBuilder(sdk.CommodityDTO_CPU).Key("cpu_comm").Capacity(float64(cpuCapacity)).Used(cpuUsed).Create()
	commoditiesSold = append(commoditiesSold, cpuComm)
	return commoditiesSold
}

func (nodeProbe *NodeProbe) generateReconcilationMetaData() *sdk.EntityDTO_ReplacementEntityMetaData {
	replacementEntityMetaDataBuilder := sdk.NewReplacementEntityMetaDataBuilder()
	replacementEntityMetaDataBuilder.Matching("IP")
	replacementEntityMetaDataBuilder.PatchSelling(sdk.CommodityDTO_CPU)
	/*
		replacementEntityMetaDataBuilder.PatchSelling(sdk.CommodityDTO_MEM_ALLOCATION)
		replacementEntityMetaDataBuilder.PatchSelling(sdk.CommodityDTO_VCPU)
		replacementEntityMetaDataBuilder.PatchSelling(sdk.CommodityDTO_VMEM)
		replacementEntityMetaDataBuilder.PatchSelling(sdk.CommodityDTO_APPLICATION)
	*/
	metaData := replacementEntityMetaDataBuilder.Build()
	return metaData
}

func (nodeProbe *NodeProbe) buildPMEntityDTO(PM_id, displayName string, commoditiesSold []*sdk.CommodityDTO) *sdk.EntityDTO {
	cpuUsed := float64(0)
	//	memUsed := float64(0)
	//	nodeMemCapacity := float64(1000)
	nodeCpuCapacity := float64(1000)
	entityDTOBuilder := sdk.NewEntityDTOBuilder(sdk.EntityDTO_PHYSICAL_MACHINE, PM_id)
	entityDTOBuilder.DisplayName(displayName)

	curcomm := commoditiesSold[0]
	entityDTOBuilder = entityDTOBuilder.Sells(*curcomm.CommodityType, *curcomm.Key).Capacity(float64(nodeCpuCapacity)).Used(cpuUsed)
	//	entityDTOBuilder = entityDTOBuilder.SetProperty("IP", "172.16.162.133")
	//	metaData := nodeProbe.generateReconcilationMetaData()
	//	entityDTOBuilder = entityDTOBuilder.ReplacedBy(metaData)
	entityDTO := entityDTOBuilder.Create()
	return entityDTO
}

func (nodeProbe *NodeProbe) buildVMEntityDTO(VM_id, displayName, provider string, commoditiesbought []*sdk.CommodityDTO) *sdk.EntityDTO {
	entityDTOBuilder := sdk.NewEntityDTOBuilder(sdk.EntityDTO_VIRTUAL_MACHINE, VM_id)
	entityDTOBuilder.DisplayName(displayName)

	entityDTOBuilder = entityDTOBuilder.SetProvider(sdk.EntityDTO_PHYSICAL_MACHINE, provider)
	entityDTOBuilder.BuysCommodities(commoditiesbought)
	//ipAddress := "172.16.162.133" // ask Dongyi, getIPForStitching from pkg/vmturbo/vmt/probe/node_probe.go
	// not using nodeProbe.generateReconcilationMetaData()
	//Make this VM buy from a given PM , TODO check if this is the entity.Name for the PM
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

/*this function turns our NodeArray from the Kubernetes.NodeProbe as a []*sdk.EntityDTO */
func (kProbe *KubernetesProbe) getNodeEntityDTOs() []*sdk.EntityDTO {
	nodearr := kProbe.getNodeProbe().NodeArray
	/* if this was a real master and had >1 node then it would loop through nodearr type []*Node
	   for now we just harcode to the first Node*/
	//	nodeID := nodearr[0].TypeMetaUID
	//	dispName := nodearr[0].ObjectMetaName
	// we call createCommoditySold to get []*sdk.CommodityDTO
	// for now commoditiesDTOSold and bought are array of size 1, TODO: modifify createCommodities.. and buildPM for array
	commoditiesDTOsold := nodearr[0].createCommoditiesSold()
	commoditiesDTObought := nodearr[0].createCommoditiesBought()
	// create PM EntityDTO
	newPMEntityDTO := kProbe.getNodeProbe().buildPMEntityDTO("PM_seller", "PAM_PM_seller", commoditiesDTOsold)
	newEntityDTO := kProbe.getNodeProbe().buildVMEntityDTO("VM_buyer", "PAM_VM_buyer", "PM_seller", commoditiesDTObought)
	var entityDTOarray []*sdk.EntityDTO
	entityDTOarray = append(entityDTOarray, newPMEntityDTO)
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
	glog.Infof("The client msg sent out is %++v", clientMsg)
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
		TypeMetaUID:    "pamelatestNode",
		ObjectMetaName: "pamelatestNode_MetaName",
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
	glog.Infof("The client msg sent out is %++v", clientMsg)
	fmt.Println(clientMsg)
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
func (h *MsgHandler) CreateContainerInfo(localaddr string) *communicator.ContainerInfo {
	var acctDefProps []*communicator.AccountDefEntry
	targetIDAcctDefEntry := communicator.NewAccountDefEntryBuilder(h.cInfo.TargetIdentifier,
		h.cInfo.Name, localaddr, ".*", communicator.AccountDefEntry_OPTIONAL, false).Create()
	acctDefProps = append(acctDefProps, targetIDAcctDefEntry)
	usernameAcctDefEntry := communicator.NewAccountDefEntryBuilder("username", "Username", h.cInfo.Username, ".*", communicator.AccountDefEntry_OPTIONAL, false).Create()
	acctDefProps = append(acctDefProps, usernameAcctDefEntry)
	passwdAcctDefEntry := communicator.NewAccountDefEntryBuilder("password", "Password", h.cInfo.Password, ".*", communicator.AccountDefEntry_OPTIONAL, true).Create()
	acctDefProps = append(acctDefProps, passwdAcctDefEntry)
	//create the ProbeInfo struct with only type and category fields
	probeType := h.cInfo.Type
	probeCat := "Container"
	templateDTOs := createSupplyChain()
	fmt.Println(templateDTOs)
	probeInfo := communicator.NewProbeInfoBuilder(probeType, probeCat, templateDTOs, acctDefProps).Create()
	// Create container
	containerInfo := new(communicator.ContainerInfo)
	// Add probe to array of ProbeInfo* in container
	probes := append(containerInfo.Probes, probeInfo)
	containerInfo.Probes = probes
	return containerInfo
}

/*
* SupplyChain definition
 */
func createSupplyChain() []*sdk.TemplateDTO {
	//	fakestr := "fake"
	emptystr := "aaaa"
	/*
		cpuAllocationType := sdk.CommodityDTO_CPU_ALLOCATION
		cpuAllocationTemplateComm := &sdk.TemplateCommodity{
			Key:           &fakestr,
			CommodityType: &cpuAllocationType,
		}
		memAllocationType := sdk.CommodityDTO_MEM_ALLOCATION
		memAllocationTemplateComm := &sdk.TemplateCommodity{
			Key:           &fakestr,
			CommodityType: &memAllocationType,
		}*/
	vmsupplyChainNodeBuilder := sdk.NewSupplyChainNodeBuilder()
	vmsupplyChainNodeBuilder = vmsupplyChainNodeBuilder.Entity(sdk.EntityDTO_VIRTUAL_MACHINE)
	/*Selling(sdk.CommodityDTO_CPU_ALLOCATION, fakestr).Selling(sdk.CommodityDTO_MEM_ALLOCATION, fakestr).Selling(sdk.CommodityDTO_VCPU, fakestr).Selling(sdk.CommodityDTO_CPU, fakestr).Selling(sdk.CommodityDTO_VMEM, fakestr).Selling(sdk.CommodityDTO_APPLICATION, fakestr)
	 */
	cpuType := sdk.CommodityDTO_CPU
	cpuTemplateComm := &sdk.TemplateCommodity{
		Key:           &emptystr,
		CommodityType: &cpuType,
	}
	/*
		memType := sdk.CommodityDTO_MEM
		memTemplateComm := &sdk.TemplateCommodity{
			Key:           &emptystr,
			CommodityType: &memType,
		}*/
	/*
		vCpuType := sdk.CommodityDTO_VCPU
		vmVCpu := &sdk.TemplateCommodity{
			Key:           &fakestr,
			CommodityType: &vCpuType,
		}
		vMemType := sdk.CommodityDTO_VMEM
		vmVMem := &sdk.TemplateCommodity{
			Key:           &fakestr,
			CommodityType: &vMemType,
		}*/
	vmsupplyChainNodeBuilder = vmsupplyChainNodeBuilder.Provider(sdk.EntityDTO_PHYSICAL_MACHINE, sdk.Provider_HOSTING).Buys(*cpuTemplateComm)
	pmSupplyChainNodeBuilder := sdk.NewSupplyChainNodeBuilder()
	pmSupplyChainNodeBuilder = pmSupplyChainNodeBuilder.Entity(sdk.EntityDTO_PHYSICAL_MACHINE).Selling(sdk.CommodityDTO_CPU, emptystr)
	/*	pm := sdk.EntityDTO_PHYSICAL_MACHINE
		attr := ""
		externalEntityLink_SEPD := &sdk.ExternalEntityLink_ServerEntityPropDef{
			Entity:    &pm,
			Attribute: &attr,
		}
		pmVMExtLinkBuilder := sdk.NewExternalEntityLinkBuilder()
		pmVMExtLinkBuilder = pmVMExtLinkBuilder.Link(sdk.EntityDTO_VIRTUAL_MACHINE, sdk.EntityDTO_PHYSICAL_MACHINE, sdk.Provider_HOSTING).Commodity(vCpuType, true).Commodity(vMemType, true).Commodity(cpuAllocationType, true).Commodity(memAllocationType, true).ProbeEntityPropertyDef(sdk.SUPPLYCHAIN_CONSTANT_IP_ADDRESS, "172.16.162.135").ExternalEntityPropertyDef(externalEntityLink_SEPD)
		pmVMExternalLink := pmVMExtLinkBuilder.Build()
	*/
	/*SupplyChain building*/
	supplyChainBuilder := sdk.NewSupplyChainBuilder()
	supplyChainBuilder.Top(vmsupplyChainNodeBuilder)
	supplyChainBuilder.Entity(pmSupplyChainNodeBuilder)
	//	supplyChainBuilder.ConnectsTo(pmVMExternalLink)

	return supplyChainBuilder.Create()
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
	loginInfo.Name = "k8s_vmt_Enlin"
	loginInfo.Username = "kubernetes_user_ENlin"
	loginInfo.Password = "fake_password"
	loginInfo.TargetIdentifier = "my_k8s_VM_Enlin"
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

	containerInfo := msgHandler.CreateContainerInfo(wsCommunicator.LocalAddress)
	fmt.Println("created container info ")
	wsCommunicator.RegisterAndListen(containerInfo)

}
