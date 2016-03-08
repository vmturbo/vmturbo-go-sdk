

This go program sets up a websocket connection to a user's VMTServer.

Before running this program:

-Get your local IP address and update the local_IP variable in the main() function
-Get the IP address of the VMTServer host you want to connect to
and update the VMTServer_IP variable in the main() function.
-Create an arbitrary identifier for this simulated target.

Once the connection is established, a simulated Target Supply Chain Topology definition is 
created by the function createSupplyChain() and sent to the server.
The default simulated topology definition contains a seller entity of type Physical Machine
and a buyer entity of type Virtual Machine.
The createSupplyChain() function can be modified by the user to add buyer/seller entities 
and commodities.


