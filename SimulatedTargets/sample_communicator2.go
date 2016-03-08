package main

import (
	"bytes"
	"fmt"
	"github.com/golang/glog"
	"github.com/vmturbo/vmturbo-go-sdk/communicator"
	"github.com/vmturbo/vmturbo-go-sdk/sdk"
	"net/http"
)

// Struct which holds identifying information for connecting to the VMTServer
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

// This Struct is the implementation of communicator.ServerMessageHandler interface
type MsgHandler struct {
	wscommunicator *communicator.WebSocketCommunicator
	cInfo          *ConnectionInfo
	vmtapi         *VMTApiRequestHandler
}

// This struct holds the authorization information and address for connecting to VMTurbo API
type VMTApiRequestHandler struct {
	vmtServerAddr      string
	opsManagerUsername string
	opsManagerPassword string
}

// A function that creates an array of *sdk.CommodityDTO , this array defines all the commodities bought by a single
// entity in the target supply chain.
func CreateCommoditiesBought(comms_array []*Commodity_Params) []*sdk.CommodityDTO {
	var commoditiesBought []*sdk.CommodityDTO
	for _, comm := range comms_array {
		cpuUsed := float64(comm.used)
		cpuCapacity := float64(comm.cap)
		// TODO correct spelling in github for vmturbo !
		cpuComm := sdk.NewCommodtiyDTOBuilder(comm.commType).Key(comm.commKey).Capacity(float64(cpuCapacity)).Used(cpuUsed).Create()
		commoditiesBought = append(commoditiesBought, cpuComm)
	}
	return commoditiesBought
}

// A function that creates an array of *sdk.CommodityDTO , this array defines all the commodities sold by a single
// entity
func CreateCommoditiesSold(comms_array []*Commodity_Params) []*sdk.CommodityDTO {
	var commoditiesSold []*sdk.CommodityDTO
	for _, comm := range comms_array {
		cpuUsed := float64(comm.used)
		cpuCapacity := float64(comm.cap)
		cpuComm := sdk.NewCommodtiyDTOBuilder(comm.commType).Key(comm.commKey).Capacity(float64(cpuCapacity)).Used(cpuUsed).Create()
		commoditiesSold = append(commoditiesSold, cpuComm)
	}
	return commoditiesSold
}

func (nodeProbe *NodeProbe) generateReconcilationMetaData() *sdk.EntityDTO_ReplacementEntityMetaData {
	replacementEntityMetaDataBuilder := sdk.NewReplacementEntityMetaDataBuilder()
	replacementEntityMetaDataBuilder.Matching("IP")
	replacementEntityMetaDataBuilder.PatchSelling(sdk.CommodityDTO_CPU)
	metaData := replacementEntityMetaDataBuilder.Build()
	return metaData
}

func (nodeProbe *NodeProbe) buildPMEntityDTO(PM_id, displayName string, commoditiesSold []*sdk.CommodityDTO) *sdk.EntityDTO {
	cpuUsed := float64(0)
	nodeCpuCapacity := float64(1000)
	entityDTOBuilder := sdk.NewEntityDTOBuilder(sdk.EntityDTO_PHYSICAL_MACHINE, PM_id)
	entityDTOBuilder.DisplayName(displayName)

	curcomm := commoditiesSold[0]
	entityDTOBuilder = entityDTOBuilder.Sells(*curcomm.CommodityType, *curcomm.Key).Capacity(float64(nodeCpuCapacity)).Used(cpuUsed)
	entityDTO := entityDTOBuilder.Create()
	return entityDTO
}

func (nodeProbe *NodeProbe) buildVMEntityDTO(VM_id, displayName, provider string, commoditiesbought []*sdk.CommodityDTO) *sdk.EntityDTO {
	entityDTOBuilder := sdk.NewEntityDTOBuilder(sdk.EntityDTO_VIRTUAL_MACHINE, VM_id)
	entityDTOBuilder.DisplayName(displayName)

	entityDTOBuilder = entityDTOBuilder.SetProvider(sdk.EntityDTO_PHYSICAL_MACHINE, provider)
	entityDTOBuilder.BuysCommodities(commoditiesbought)
	entityDTO := entityDTOBuilder.Create()

	return entityDTO
}

type NodeProbe struct {
	// nodesGetter func
	soldcommodities   []*Commodity_Params
	boughtcommodities []*Commodity_Params
}

type KubernetesProbe struct {
	nodeProbe *NodeProbe
}

type Commodity_Params struct {
	commType sdk.CommodityDTO_CommodityType
	commKey  string
	used     int
	cap      int
}

func (nodeProbe *NodeProbe) PopulateProbe() {
	var s_comms_array []*Commodity_Params
	var b_comms_array []*Commodity_Params
	comm1 := &Commodity_Params{
		commType: sdk.CommodityDTO_CPU,
		commKey:  "cpu_comm",
		used:     4,
		cap:      100,
	}
	comm2 := &Commodity_Params{
		commType: sdk.CommodityDTO_MEM,
		commKey:  "mem_comm",
		used:     10,
		cap:      100,
	}
	comm3 := &Commodity_Params{
		commType: sdk.CommodityDTO_CPU,
		commKey:  "cpu_comm",
		used:     0,
		cap:      1000,
	}
	comm4 := &Commodity_Params{
		commType: sdk.CommodityDTO_MEM,
		commKey:  "mem_comm",
		used:     0,
		cap:      1000,
	}
	s_comms_array = append(s_comms_array, comm1)
	s_comms_array = append(s_comms_array, comm2)
	b_comms_array = append(b_comms_array, comm3)
	b_comms_array = append(b_comms_array, comm4)
	nodeProbe.soldcommodities = s_comms_array
	nodeProbe.boughtcommodities = b_comms_array
}

func (kProbe *KubernetesProbe) getNodeProbe() *NodeProbe {
	return kProbe.nodeProbe
}

/*this function turns our NodeArray from the Kubernetes.NodeProbe as a []*sdk.EntityDTO */
func (kProbe *KubernetesProbe) getNodeEntityDTOs() []*sdk.EntityDTO {
	kProbe.getNodeProbe().PopulateProbe()
	/* if this was a real master and had >1 node then it would loop through nodearr type []*Node
	   for now we just harcode to the first Node*/
	// we call createCommoditySold to get []*sdk.CommodityDTO
	// for now commoditiesDTOSold and bought are array of size 1, TODO: modifify createCommodities.. and buildPM for array
	s_comms_array := kProbe.getNodeProbe().soldcommodities
	b_comms_array := kProbe.getNodeProbe().boughtcommodities
	commoditiesDTOsold := CreateCommoditiesSold(s_comms_array)
	commoditiesDTObought := CreateCommoditiesBought(b_comms_array)
	// create PM EntityDTO
	newPMEntityDTO1 := kProbe.getNodeProbe().buildPMEntityDTO("PM_seller1", "PAM_PM_seller1", commoditiesDTOsold)
	newVMEntityDTO1A := kProbe.getNodeProbe().buildVMEntityDTO("VM_buyer1A", "PAM_VM_buyer1A", "PM_seller1", commoditiesDTObought)
	newVMEntityDTO1B := kProbe.getNodeProbe().buildVMEntityDTO("VM_buyer1B", "PAM_VM_buyer1B", "PM_seller1", commoditiesDTObought)

	newPMEntityDTO2 := kProbe.getNodeProbe().buildPMEntityDTO("PM_seller2", "PAM_PM_seller2", commoditiesDTOsold)
	newVMEntityDTO2A := kProbe.getNodeProbe().buildVMEntityDTO("VM_buyer2A", "PAM_VM_buyer2A", "PM_seller2", commoditiesDTObought)
	newVMEntityDTO2B := kProbe.getNodeProbe().buildVMEntityDTO("VM_buyer2B", "PAM_VM_buyer2B", "PM_seller2", commoditiesDTObought)
	newPMEntityDTO3 := kProbe.getNodeProbe().buildPMEntityDTO("PM_seller3", "PAM_PM_seller3", commoditiesDTOsold)

	var entityDTOarray []*sdk.EntityDTO
	entityDTOarray = append(entityDTOarray, newPMEntityDTO1)
	entityDTOarray = append(entityDTOarray, newPMEntityDTO2)
	entityDTOarray = append(entityDTOarray, newVMEntityDTO1A)
	entityDTOarray = append(entityDTOarray, newVMEntityDTO1B)
	entityDTOarray = append(entityDTOarray, newVMEntityDTO2A)
	entityDTOarray = append(entityDTOarray, newVMEntityDTO2B)
	entityDTOarray = append(entityDTOarray, newPMEntityDTO3)
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

	// creates a ClientMessageBuilder and sets ClientMessageBuilder.clientMessage.MessageID = messageID
	// sets clientMessage.ValidationResponse = validationResponse
	// type of clientMessage is MediationClientMessage
	clientMsg := communicator.NewClientMessageBuilder(messageID).SetValidationResponse(validationResponse).Create()
	h.wscommunicator.SendClientMessage(clientMsg)
	glog.Infof("The client msg sent out is %++v", clientMsg)
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
	fmt.Println("DiscoverTopology called")

	messageID := serverMsg.GetMessageID()
	newNodeProbe := new(NodeProbe)
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
	glog.Infof("The client msg sent out is %++v", clientMsg)
	fmt.Println(clientMsg)
	fmt.Println("done with discover")
	return
}
func (h *MsgHandler) HandleAction(serverMsg *communicator.MediationServerMessage) {
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
	optionalKey := "commodity_key"
	vmsupplyChainNodeBuilder := sdk.NewSupplyChainNodeBuilder()
	vmsupplyChainNodeBuilder = vmsupplyChainNodeBuilder.Entity(sdk.EntityDTO_VIRTUAL_MACHINE)
	cpuType := sdk.CommodityDTO_CPU
	cpuTemplateComm := &sdk.TemplateCommodity{
		Key:           &optionalKey,
		CommodityType: &cpuType,
	}
	vmsupplyChainNodeBuilder = vmsupplyChainNodeBuilder.Provider(sdk.EntityDTO_PHYSICAL_MACHINE, sdk.Provider_HOSTING).Buys(*cpuTemplateComm)
	pmSupplyChainNodeBuilder := sdk.NewSupplyChainNodeBuilder()
	pmSupplyChainNodeBuilder = pmSupplyChainNodeBuilder.Entity(sdk.EntityDTO_PHYSICAL_MACHINE).Selling(sdk.CommodityDTO_CPU, optionalKey)
	/*SupplyChain building*/
	supplyChainBuilder := sdk.NewSupplyChainBuilder()
	supplyChainBuilder.Top(vmsupplyChainNodeBuilder)
	supplyChainBuilder.Entity(pmSupplyChainNodeBuilder)

	return supplyChainBuilder.Create()
}

func main() {

	wsCommunicator := new(communicator.WebSocketCommunicator)
	wsCommunicator.VmtServerAddress = "160.39.163.35:8080"
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
