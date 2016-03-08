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
	entities          []*Entity_Params
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

type Entity_Params struct {
	Buyer             bool
	Seller            bool
	entityType        sdk.EntityDTO_EntityType
	entityID          string
	entityDisplayName string
	commoditiesSold   []*Commodity_Params
	commoditiesBought []*Commodity_Params
	providerID        string
}

func (nodeProbe *NodeProbe) PopulateProbe() {
	var s_comms_array []*Commodity_Params
	var b_comms_array []*Commodity_Params
	var entities []*Entity_Params
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

	newSeller1 := &Entity_Params{
		Buyer:             false,
		Seller:            true,
		entityType:        sdk.EntityDTO_PHYSICAL_MACHINE,
		entityID:          "PM_seller_1",
		entityDisplayName: "test_PM_seller_1",
		commoditiesSold:   s_comms_array,
	}
	newSeller2 := &Entity_Params{
		Buyer:             false,
		Seller:            true,
		entityType:        sdk.EntityDTO_PHYSICAL_MACHINE,
		entityID:          "PM_seller_2",
		entityDisplayName: "test_PM_seller_2",
		commoditiesSold:   s_comms_array,
	}
	newBuyer1 := &Entity_Params{
		Buyer:             true,
		Seller:            false,
		entityType:        sdk.EntityDTO_VIRTUAL_MACHINE,
		entityID:          "VM_buyer_1A",
		entityDisplayName: "test_VM_buyer_1A",
		commoditiesBought: b_comms_array,
		providerID:        "PM_seller_1",
	}
	newBuyer2 := *newBuyer1
	newBuyer2.entityID = "VM_buyer_1B"
	newBuyer2.entityDisplayName = "test_VM_buyer_1B"
	newBuyer3 := *newBuyer1
	newBuyer3.entityID = "VM_buyer_2A"
	newBuyer3.entityDisplayName = "test_VM_buyer_2A"
	newBuyer3.providerID = "PM_seller_2"
	entities = append(entities, newSeller1)
	entities = append(entities, newSeller2)
	entities = append(entities, newBuyer1)
	entities = append(entities, &newBuyer2)
	entities = append(entities, &newBuyer3)
	nodeProbe.entities = entities
}

func (kProbe *KubernetesProbe) getNodeProbe() *NodeProbe {
	return kProbe.nodeProbe
}

/*this function turns our NodeArray from the Kubernetes.NodeProbe as a []*sdk.EntityDTO */
func (kProbe *KubernetesProbe) getNodeEntityDTOs() []*sdk.EntityDTO {
	kProbe.getNodeProbe().PopulateProbe()
	// create PM or VM EntityDTO
	var entityDTOarray []*sdk.EntityDTO
	for _, entity := range kProbe.getNodeProbe().entities {
		// here the only sellers are Physical Machines
		if entity.Seller == true {
			// we call createCommoditySold to get []*sdk.CommodityDTO
			commoditiesDTOSold := CreateCommoditiesSold(entity.commoditiesSold)
			newEntityDTO := kProbe.getNodeProbe().buildPMEntityDTO(entity.entityID, entity.entityDisplayName, commoditiesDTOSold)
			entityDTOarray = append(entityDTOarray, newEntityDTO)
		}
		if entity.Buyer == true {
			commoditiesDTOBought := CreateCommoditiesBought(entity.commoditiesBought)
			newEntityDTO := kProbe.getNodeProbe().buildVMEntityDTO(entity.entityID, entity.entityDisplayName, entity.providerID, commoditiesDTOBought)
			entityDTOarray = append(entityDTOarray, newEntityDTO)
		}
	}

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
* SupplyChain definition: this function defines the buyer/seller relationships between each of the entity types in * the Target, the default Supply Chain definition in this function is Virtual Machine buyer, a Physical Machine
* seller and the commodities are CPU and Memory.
* Each entity type and the relationships are defined by a single TemplateDTO struct
* The function returns an array of TemplateDTO pointers
 */
func createSupplyChain() []*sdk.TemplateDTO {
	optionalKey := "commodity_key"
	vmsupplyChainNodeBuilder := sdk.NewSupplyChainNodeBuilder()
	// Creates a Virtual Machine entity
	vmsupplyChainNodeBuilder = vmsupplyChainNodeBuilder.Entity(sdk.EntityDTO_VIRTUAL_MACHINE)
	cpuType := sdk.CommodityDTO_CPU
	cpuTemplateComm := &sdk.TemplateCommodity{
		Key:           &optionalKey,
		CommodityType: &cpuType,
	}
	// The Entity type for the Virtual Machine's commodity provider is defined by the Provider() method.
	// The Commodity type for Virtual Machine's buying relationship is define by the Buys() method
	vmsupplyChainNodeBuilder = vmsupplyChainNodeBuilder.Provider(sdk.EntityDTO_PHYSICAL_MACHINE, sdk.Provider_HOSTING).Buys(*cpuTemplateComm)
	pmSupplyChainNodeBuilder := sdk.NewSupplyChainNodeBuilder()
	// Creates a Physical Machine entity and sets the type of commodity it sells to CPU
	pmSupplyChainNodeBuilder = pmSupplyChainNodeBuilder.Entity(sdk.EntityDTO_PHYSICAL_MACHINE).Selling(sdk.CommodityDTO_CPU, optionalKey)
	// SupplyChain building
	//  The last buyer in the supply chain is set as the top entity with the Top() method
	// All other entities are added to the SupplyChainBuilder with the Entity() method
	supplyChainBuilder := sdk.NewSupplyChainBuilder()
	supplyChainBuilder.Top(vmsupplyChainNodeBuilder)
	supplyChainBuilder.Entity(pmSupplyChainNodeBuilder)

	return supplyChainBuilder.Create()
}

func main() {
	/*
	* User defined settings
	 */
	local_IP := "172.16.162.133"
	VMTServer_IP := "160.39.162.134"
	TargetIdentifier := "userDefinedTarget"

	/*
	* Do Not Modify below this point
	 */
	localAddress := "ws://" + local_IP
	VMTServerAddress := VMTServer_IP + ":8080"
	wsCommunicator := new(communicator.WebSocketCommunicator)
	wsCommunicator.VmtServerAddress = VMTServerAddress
	wsCommunicator.LocalAddress = localAddress
	wsCommunicator.ServerUsername = "vmtRemoteMediation"
	wsCommunicator.ServerPassword = "vmtRemoteMediation"
	loginInfo := new(ConnectionInfo)
	loginInfo.OpsManagerUsername = "administrator"
	loginInfo.OpsManagerPassword = "a"
	loginInfo.Type = "Kubernetes"
	loginInfo.Name = "k8s_vmt"
	loginInfo.Username = "username"
	loginInfo.Password = "password"
	loginInfo.TargetIdentifier = TargetIdentifier
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
	wsCommunicator.RegisterAndListen(containerInfo)

}
