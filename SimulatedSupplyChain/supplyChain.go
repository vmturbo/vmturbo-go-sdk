// Builds a fake supply chain to be sent to the VMTServer
package main

import (
	"bytes"
	"fmt"
	"github.com/vmturbo/vmturbo-go-sdk/communicator"
	"github.com/vmturbo/vmturbo-go-sdk/sdk"
	"net/http"
)

func createSupplyChain() []*sdk.TemplateDTO {
	supplyChainNodeBuilder := sdk.NewSupplyChainNodeBuilder()
	supplyChainNodeBuilder := supplyChainNodeBuilder.Entity(sdk.EntityDTO_VIRTUAL_MACHINE).Selling(sdk.CommodityDTO_CPU_ALLOCATION, "fake").Selling(sdk.CommodityDTO_MEM_ALLOCATION, "fake").Selling(sdk.CommodityDTO_VCPU, "fake").Selling(sdk.CommodityDTO_VMEM, "fake").Selling(sdk.CommodityDTO_APPLICATION, "fake")

}

func main() {
	wsCommunicator := new(communicator.WebSocketCommunicator)
	wsCommunicator.VmtServerAddress = "10.10.200.98:8080"
	wsCommunicator.LocalAddress = "ws://172.16.162.133"
	wsCommunicator.ServerUsername = "vmtRemoteMediation"
	wsCommunicator.ServerPassword = "vmtRemoteMediation"

}
