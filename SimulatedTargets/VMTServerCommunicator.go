package main

import (
	"bytes"
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
// returns an array of *sdk.CommodityDTO
func CreateCommoditiesBought(comms_array []*Commodity_Params) []*sdk.CommodityDTO {
	var commoditiesBought []*sdk.CommodityDTO
	for _, comm := range comms_array {
		cpuUsed := float64(comm.used)
		cpuCapacity := float64(comm.cap)
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

// Creates reconciliation MetaData
func (nodeProbe *NodeProbe) generateReconcilationMetaData() *sdk.EntityDTO_ReplacementEntityMetaData {
	replacementEntityMetaDataBuilder := sdk.NewReplacementEntityMetaDataBuilder()
	replacementEntityMetaDataBuilder.Matching("IP")
	replacementEntityMetaDataBuilder.PatchSelling(sdk.CommodityDTO_CPU)
	metaData := replacementEntityMetaDataBuilder.Build()
	return metaData
}

// This function builds an EntityDTO object from information provided from the one of the entities at discovery time.
// It returns an *sdk.EntityDTO which points to the EntityDTO created.
func (e *Entity_Params) buildEntityDTO() *sdk.EntityDTO {
	entityDTOBuilder := sdk.NewEntityDTOBuilder(e.entityType, e.entityID)
	entityDTOBuilder.DisplayName(e.entityDisplayName)
	if e.Buyer == true {
		commoditiesbought := CreateCommoditiesBought(e.commoditiesBought)
		entityDTOBuilder = entityDTOBuilder.
			entityDTOBuilder.BuysCommodities(commoditiesbought)
	}
	if e.Seller == true {
		commoditiesSold := CreateCommoditiesSold(e.commoditiesSold)
		for _, curcomm := range commoditiesSold {
			entityDTOBuilder = entityDTOBuilder.Sells(*curcomm.CommodityType, *curcomm.Key).Capacity(*curcomm.Capacity).Used(*curcomm.Used)
		}
	}
	entityDTO := entityDTOBuilder.Create()
	return entityDTO
}

// This struct holds the array of Entity structs that are found in the simulated target
type NodeProbe struct {
	entities []*Entity_Params
}

// A struct that contains a Node probe, there is one Kubernetes probe and one NodeProbe for each target
type KubernetesProbe struct {
	nodeProbe *NodeProbe
}

// Struct that holds parameters for each commodity sold or bought by a given entity
type Commodity_Params struct {
	commType sdk.CommodityDTO_CommodityType
	commKey  string
	used     int
	cap      int
}

// Struct that holds a given entity's identifying and property information
type Entity_Params struct {
	Buyer             bool
	Seller            bool
	entityType        sdk.EntityDTO_EntityType
	entityID          string
	entityDisplayName string
	commoditiesSold   []*Commodity_Params
	commoditiesBought []*Commodity_Params
	providerID        string
	providerType      sdk.EntityDTO_EntityType
}

// Method that creates an array of entities found at this target
// Creates arrays of bought/sold commodities for each entity
// Sets the entities field of the NodeProbe it is called on to the
// newly created entity array
// Default: array of sold commodities contains a sdk.CommodityDTO_CPU
//	    and a sdk.CommodityDTO_MEM
//	    array of bought commodities contains a sdk.CommodityDTO_CPU
// 	    and asdk.CommodityDTO_MEM
//          Array of Entities contains 2 sellers, selling the same commodities
//          and 3 buyers, two buyers buying from seller 1 and one buyer buying from
//          seller 2.
func (nodeProbe *NodeProbe) SampleProbe() {
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

// this getter method returns the NodeProbe contained in the KubernetesProbe
func (kProbe *KubernetesProbe) getNodeProbe() *NodeProbe {
	return kProbe.nodeProbe
}

// this function turns our NodeArray from the Kubernetes.NodeProbe as a []*sdk.EntityDTO
func (kProbe *KubernetesProbe) getNodeEntityDTOs() []*sdk.EntityDTO {
	kProbe.getNodeProbe().SampleProbe()
	// create PM or VM EntityDTO
	var entityDTOarray []*sdk.EntityDTO
	for _, entity := range kProbe.getNodeProbe().entities {
		// we call createCommoditySold to get []*sdk.CommodityDTO
		newEntityDTO := entity.buildEntityDTO()
		entityDTOarray = append(entityDTOarray, newEntityDTO)
	}

	return entityDTOarray
}

// this helper function servers to send REST api calls to the VMTServer using opsmanager authentication
func (vmtapi *VMTApiRequestHandler) vmtApiPost(postPath, requestStr string) (*http.Response, error) {
	fullUrl := "http://" + vmtapi.vmtServerAddr + "/vmturbo/api" + postPath + requestStr
	req, err := http.NewRequest("POST", fullUrl, nil)
	req.SetBasicAuth(vmtapi.opsManagerUsername, vmtapi.opsManagerPassword)
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		glog.Infof("Log: error getting response")
		return nil, err
	}
	defer response.Body.Close()
	return response, nil
}

// Method used for adding a Target to a VMTServer
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
		glog.Infof(" postReply error")
	}

	if postReply.Status != "200 OK" {
		glog.Infof(" postReplyMessage error")
	}
}

// This Method validates our target which was previously added to the VMTServer
func (h *MsgHandler) Validate(serverMsg *communicator.MediationServerMessage) {
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
		glog.Infof(" error in validate response from server")
		return
	}

	if postReply.Status != "200 OK" {
		glog.Infof("Validate reply came in with error")
	}
	return
}

func (h *MsgHandler) HandleAction(serverMsg *communicator.MediationServerMessage) {
	glog.Infof("HandleAction called")
	return
}

// This Method sends all the topology entities and relationships found at
// this target to the VMTServer
func (h *MsgHandler) DiscoverTopology(serverMsg *communicator.MediationServerMessage) {

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
	probeInfo := communicator.NewProbeInfoBuilder(probeType, probeCat, templateDTOs, acctDefProps).Create()
	// Create container
	containerInfo := new(communicator.ContainerInfo)
	// Add probe to array of ProbeInfo* in container
	probes := append(containerInfo.Probes, probeInfo)
	containerInfo.Probes = probes
	return containerInfo
}

// SupplyChain definition: this function defines the buyer/seller relationships between each of
// the entity types in * the Target, the default Supply Chain definition in this function is:
// a Virtual Machine buyer, a Physical Machine seller and the commodities are CPU and Memory.
// Each entity type and the relationships are defined by a single TemplateDTO struct
// The function returns an array of TemplateDTO pointers
// TO MODIFY:
// For each entity: Create a supply chain builder object with sdk.NewSupplyChainNodeBuilder()
//		    Set a provider type if the new entity is a buyer , create commodity objects
//		    and add them to the entity's supply chain builder object
//                  Add commodity objects with the selling function to the entity you create if
//		    it is a seller.
//		    Add the new entity to the supplyChainBuilder instance with either the Top()
//		    or  Entity() methods
// The SupplyChainBuilder() function is only called once, in this function.
func createSupplyChain() []*sdk.TemplateDTO {
	//Commodity key is optional, when key is set, it serves as a constraint between seller and buyer
	//for example, the buyer can only go to a seller that sells the commodity with the required key
	optionalKey := "commodity_key"
	vmsupplyChainNodeBuilder := sdk.NewSupplyChainNodeBuilder()
	// Creates a Virtual Machine entity
	vmsupplyChainNodeBuilder = vmsupplyChainNodeBuilder.Entity(sdk.EntityDTO_VIRTUAL_MACHINE)
	cpuType := sdk.CommodityDTO_CPU
	cpuTemplateComm := &sdk.TemplateCommodity{
		Key:           &optionalKey,
		CommodityType: &cpuType,
	}

	memType := sdk.CommodityDTO_MEM
	memTemplateComm := &sdk.TemplateCommodity{
		Key:           &optionalKey,
		CommodityType: &memType,
	}
	// The Entity type for the Virtual Machine's commodity provider is defined by the Provider() method.
	// The Commodity type for Virtual Machine's buying relationship is define by the Buys() method
	vmsupplyChainNodeBuilder = vmsupplyChainNodeBuilder.Provider(sdk.EntityDTO_PHYSICAL_MACHINE, sdk.Provider_HOSTING).Buys(*cpuTemplateComm).Buys(*memTemplateComm)
	pmSupplyChainNodeBuilder := sdk.NewSupplyChainNodeBuilder()
	// Creates a Physical Machine entity and sets the type of commodity it sells to CPU
	pmSupplyChainNodeBuilder = pmSupplyChainNodeBuilder.Entity(sdk.EntityDTO_PHYSICAL_MACHINE).Selling(sdk.CommodityDTO_CPU, optionalKey).Selling(sdk.CommodityDTO_MEM, optionalKey)
	// SupplyChain building
	//  The last buyer in the supply chain is set as the top entity with the Top() method
	// All other entities are added to the SupplyChainBuilder with the Entity() method
	supplyChainBuilder := sdk.NewSupplyChainBuilder()
	supplyChainBuilder.Top(vmsupplyChainNodeBuilder)
	supplyChainBuilder.Entity(pmSupplyChainNodeBuilder)

	return supplyChainBuilder.Create()
}

func main() {
	//
	//User defined settings
	//
	local_IP := "172.16.162.133"
	VMTServer_IP := "10.10.200.98"
	TargetIdentifier := "userDefinedTarget"
	OpsManagerUsername := "administrator"
	OpsManagerPassword := "a"
	localAddress := "ws://" + local_IP
	VMTServerAddress := VMTServer_IP + ":8080"
	wsCommunicator := new(communicator.WebSocketCommunicator)
	wsCommunicator.SetDefaults()
	wsCommunicator.VmtServerAddress = VMTServerAddress
	wsCommunicator.LocalAddress = localAddress
	loginInfo := new(ConnectionInfo)
	loginInfo.Type = "Kubernetes"
	loginInfo.Name = "k8s_vmt"
	loginInfo.Username = "username"
	loginInfo.Password = "password"
	loginInfo.OpsManagerUsername = OpsManagerUsername
	loginInfo.OpsManagerPassword = OpsManagerPassword
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
