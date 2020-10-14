# Push notifications service

The push notifications service allows organizations to send pop up messages to their users devices. Organizations can send notifications at any time and their users do not have to be in the app or using their devices to receive them.

Currenty the push notifications system used is the Google Firebase Cloud Messassing but any other provider can be easily integrated.

Each user can be subscribed to any organization she wants and be notified for one or more events.

Event subscription is managed with a topic approach, currently supporting the following topics:
- `<entityID>_default_post-new`
- `<entityID>_default_process-new`

## Index

1. Ethereum
2. IPFS
3. Notifications service
4. API
    
### 1. Ethereum

Given that the Vocdoni platform uses different smart contracts deployed into an EVM compatible chain as the source of truth, it is a requirement to have access to an Ethereum JSON-RPC compatible endpoint in order to interact with these contracts.
Interacting with this contracts means to read, write and listen for events.

Some examples:
- New voting process creation
- A voting process is ended
- An ended voting process have the results available
- ...

Currently supported EVM logs: `NewProcess(bytes32 processId, uint16 namespace)`

### 2. IPFS

Another fundamental component of the notifications service is the IPFS node. Data is not stored in Ethereum but in IPFS, and given so, have access to an IPFS node is required in order to retrieve and store data in this distributed file system.

This data can be:
- Voting process metadata (type, mode, questions ...)
- Organization metadata (name, email, ID ...)
- Prefered organization bootnodes and gateways to connect with

The actual URI of this data is stored on Ethereum and changes every time any of the data is changed. This is an IPFS property which is a content-addressed system.

Currently supported file changes: News feed modifications on organizations metadata

### 3. Notifications service

The notifications service, in short, has two main different missions:

1. To track the`Process` smart contract events
    
    Example: An organization created a new process and wants to inform its members

2. To track IPFS files changes given its URI stored in Ethereum

    Example: An organization updated its news feed entry and wants to notify its members that a new post is added
    
    
Is composed by the following components:

![](https://hackmd.vocdoni.net/uploads/upload_3e17a65baa1f932ba61ad2be6d0fb741.png)

<br>

#### How it works

- **Private database**

    This database is used to get the organizations list and be able to track each ones metadata.

- **Ethereum**

    The Ethereum JSON-RPC endpoint is used to interact with the Ethereum network, the `Process` contract and the Ethereum Name Service (ENS)
    are those the notifications service uses.
    
    - The ENS smart contracts are used to fetch the canonical `Process` smart contract address under the `voting-process.vocdoni.eth` domain.
    - Once the `Process` contract address is fetched a component called `ethEvents` starts to listen for logs on the contract.
    - Each time a new process is created (the case currently supported) an Ethereum smart contract event is triggered and the `ethEvents` component
      handles it.
    - The event will be processed and a notification anouncing the creation of a process will be created and sended to Firebase.
    - The users subscribed to the corresponding topic will receive the notification
      

- **IPFS**

    The IPFS file tracking also relies on Ethereum as a source of truth and uses Ethereum to get the latest IPFS URI in the ENS Text List Record
    entry of an organization. This IPFS URI points to the entity metadata.
    As said, this metadata is stored on IPFS and its content is univocally identified by a hash.
    
    The file tracking mechanism works as follows:
    - Get the organizations list from the private database.
    - Get each entity IPFS metadata URI from the ENS Text list record corresponding entry.
    - Fetch the file from IPFS:
        - If the organization was not tracked before, just store the file and its hash.
        - Else the current hash is compared with the previously stored ones and if it is different the content is also different.
    - In case of the metadata was updated the current version checks if the news feed field is different.
    - Each time the news feed is modified a new Firebase notification is created.
    - The users subscribed to the corresponding topic will be notified that a new post is created or updated.

    This process runs forever and is repeated every scheduled time. 
          

### 4. API

The notifications API is used to generate user tokens that can be useful for certain circunstances.

Available by default under `/notifications`

##### register


- Request

     ```json
    {
      "id": "req-12345678",
      "request": {
        "method": "register",
        "timestamp": 1234567890
      },
      "signature": "0x12345"
    }
    ```
    
- Response:

    ```json
    {
      "response": {
        "ok": true,
        "request": "req-12345678",  
        "timestamp": 1556110671,
        "token": "xxx-yyy-zzz"
      },
      "id": "req-12345678",
      "signature": "0x12345"
    }
    ```


## Build and run

- Build
```bash
go build cmd/dvotenotif/dvotenotif.go
``` 
- Run

```bash
go run cmd/dvotenotif/dvotenotif.go

## or after build

./dvotenotif
```

- Options (.yaml config file is also created)

```bash
  --dataDir string                    directory where data is stored (default "/home/t480/.dvotenotif")
  --dbHost string                     DB server address (default "127.0.0.1")
  --dbName string                     DB database name (default "vocdoni")
  --dbPassword string                 DB password (default "vocdoni")
  --dbPort int                        DB server port (default 5432)
  --dbSslmode string                  DB postgres sslmode (default "prefer")
  --dbUser string                     DB Username (default "vocdoni")
  --ethBootNodes stringArray          Ethereum p2p custom bootstrap nodes (enode://<pubKey>@<ip>[:port])
  --ethCensusSync                     automatically import new census published on the smart contract
  --ethChain string                   Ethereum blockchain to use: [mainnet goerli xdai xdaistage sokol] (default "sokol")
  --ethChainLightMode                 synchronize Ethereum blockchain in light mode
  --ethNoWaitSync                     do not wait for Ethereum to synchronize (for testing only)
  --ethNodePort int                   Ethereum p2p node port to use (default 30303)
  --ethProcessDomain string           voting contract ENS domain (default "voting-process.vocdoni.eth")
  --ethSigningKey string              signing private Key (if not specified the Ethereum keystore will be used)
  --ethSubscribeOnly                  only subscribe to new ethereum events (do not read past log)
  --ethTrustedPeers stringArray       Ethereum p2p trusted peer nodes (enode://<pubKey>@<ip>[:port])
  --ipfsNoInit                        disables inter planetary file system support
  --ipfsSyncKey string                enable IPFS cluster synchronization using the given secret key
  --ipfsSyncPeers stringArray         use custom ipfsSync peers/bootnodes for accessing the DHT
  --logErrorFile string               Log errors and warnings to a file
  --logLevel string                   Log level (debug, info, warn, error, fatal) (default "info")
  --logOutput string                  Log output (stdout, stderr or filepath) (default "stdout")
  --metricsEnabled                    enable prometheus metrics (default true)
  --metricsRefreshInterval int        metrics refresh interval in seconds (default 10)
  --pushNotificationsKeyFile string   path to notifications service private key file
  --pushNotificationsService int      push notifications service, 1: Firebase (default 1)
  --saveConfig                        overwrites an existing config file with the CLI provided flags
  --w3Enabled                         if true, a web3 public endpoint will be enabled (default true)
  --w3External string                 use an external web3 endpoint instead of the local one. Supported protocols: http(s)://, ws(s):// and IPC filepath
  --w3RPCHost string                  web3 RPC host (default "127.0.0.1")
  --w3RPCPort int                     web3 RPC port (default 9091)
  --w3Route string                    web3 endpoint API route (default "/web3")
```