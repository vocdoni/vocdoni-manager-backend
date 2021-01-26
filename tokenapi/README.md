# Token API

The Token API allows organizations to take control over the user tokens required for user registration on the Vocdoni registry backend.

The API can be accessed on a url like this `wss://vocdoni-registry-service/api/token`. It uses websockets with a JSON payload.

## Authentication

There is a common known secret between the entity and the vocdoni backend. This secret is used to compute the authHash field.

`authHash: hexadecimal(keccak256(field1+field2+fieldN+secret))`

The fields are concatenated strings. One of the fields must be a 32 bits `timestamp` (seconds).
If a field is an array, then all the elements of the array should be included as strings in the secret.

When computing the authHash the order of the fields is alphabetical using the field name, except from the last word that is is always the secret. This order in not obligatorily applied in the request body, just for the authHash calculation.

The timestamp window tolerance is 3 seconds. If the request is processed outside of this tolerance window, the packet will be discarted.

```json
{
  "id": "req-12345678",
  "request": {
    "method": "revoke",
    "entityId": "0x12345",
    "token": "xxx-yyy-zzz",
    "authHash": "0x123456789",
    "timestamp": 1234567890
  }
}
```

secret = "hello"

`authHash = keccack256("0x123456789"+"revoke"+"1234567890"+"xxx-yyy-zzz"+"hello")`

### Error

+ `response.ok` A bool indicating if the request failed
+ `response.request` The value given in the request `id` field
+ `response.message` Explanation of what went wrong

```json
{
  "id": "req-12345678",                 // ID of the originating request
  "response": {
	"ok": false,                        // false if error found
    "request": "req-12345678",          // Request ID here as well
    "message": "Unknown method"         // What went wrong
  }
}
```

## Methods

### revoke a token
- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "revoke",
        "entityId": "0x12345",
        "token": "xxx-yyy-zzz",
        "timestamp": 1234567890,
        "authHash": "0x123456789"
    },
    "signature": "0x12345"
}
```
- Response
```json
{
    "id": "req-12345678",    // ID of the originating request
    "response": {
        "ok": true,
        "message": "...",     // Error message available only in case ok:false
        "timestamp": 1234567890
    },
    "signature": "0x123456"
}
```

### status of a token
- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "status",
        "entityId": "0x12345",
        "token": "xxx-yyy-zzz",
        "timestamp": 1234567890,
        "authHash": "0x123456789"
    },
    "signature": "0x12345"   
}
```
- Response
```json
{
    "id": "req-12345678", 
    "response": {
        "ok": true,
        "tokenStatus": "available",    // available | "registered" | "invalid"
        "timestamp": 1234567890
    },
    "signature": "0x123456"
}
```

### generate a batch of tokens
- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "generate",
        "amount": 1000, 
        "entityId": "0x12345",
        "timestamp": 1234567890,
        "authHash": "0x123456789"
    },
    "signature": "0x12345"
}
```
- Response
```json
{
    "id": "req-12345678", // ID of the originating request
    "response": {
        "ok": true,
        "tokens": ["xxx-yyy-zzz", "yyy-zzz-xxx"],
        "timestamp": 1234567890
    },
    "signature": "0x123456"
}
```

### import members by public keys
Populate the entity members/users importing based on **non-digested** public keys
- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "importKeysBulk",
        "entityId": "0x12345",
        "keys": ["590289d82938","850c7ea5a360e599dbbd"],
        "timestamp": 1234567890,
        "authHash": "0x123456789"
    },
    "signature": "0x12345"
}
```
- Response
```json
{
    "id": "req-12345678", // ID of the originating request
    "response": {
        "ok": true,
    },
    "signature": "0x123456"
}
```

### list by public keys
List the entity's **non-digested** public keys. For the authentication the `listOptions` arguments should be added separately in the position of `l` and then, in alphabetical order between them. For the following example the sequence would be:
`$entityID$listOptions.count$listOptions.skip$method$timestamp`
or
`0x12345220x123451234567890`

- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "listKeys",
        "entityId": "0x12345",
        "listOptions": {
          "skip": 2,
          "count": 2,
        },
        "timestamp": 1234567890,
        "authHash": "0x123456789"
    },
    "signature": "0x12345"
}
```
- Response
```json
{
    "id": "req-12345678", // ID of the originating request
    "response": {
        "ok": true,
        "count": 1, // Number of keys deleted
        "keys", ["590289d82938", "850c7ea5a360e599dbbd"] // list of invalid keys
    },
    "signature": "0x123456"
}
```

### delete members by public keys
Delete  entity members indentified by their **non-digested** public keys
- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "deleteKeys",
        "entityId": "0x12345",
        "keys": ["590289d82938","850c7ea5a360e599dbbd"],
        "timestamp": 1234567890,
        "authHash": "0x123456789"
    },
    "signature": "0x12345"
}
```
- Response
```json
{
    "id": "req-12345678", // ID of the originating request
    "response": {
        "ok": true,
        "count": 1, // Number of keys deleted
        "invalidKeys", ["590289d82938"] // list of invalid keys
    },
    "signature": "0x123456"
}
```

### Full example 

A full working example with secret=`test`.

```json
{
"request":{
    "amount":5,
    "authHash":"6853b0b189bd0b69a288e458299b2f8ea4a2ee2f08e0d88a255edf10b891e9c9",
    "entityId":"590289d82938b894c816d814244e616a893a0bf39117f80a21815179c5c01c8c",
    "method":"generate",
    "timestamp":1595323066
    },
    "id":"req-814"
}

{
"response":{
    "ok":true,
    "request":"814",
    "timestamp":1595323066,
    "tokens":["f45a5966-f44f-4c7b-b70e-900ca49f18f7","a209816a-2999-441a-b921-0c8037814673","b3ed5bbd-6e28-43eb-806c-dedae2bc8f21","9435e251-73a1-48e4-bf57-fdfb29b9df2b","02d5af0a-0128-467c-9ab0-63787c0b62e6"]
    },
    "id":"req-814",
    "signature":"850c7ea5a360e599dbbd1d3c3b8428ead6025e460a2f33be0bb2e44ff94e0cd97fc48da44d4cfd69d9d1ad99df03e8ff3c74442d314d25a5d88608c1a0da018801"
}
```

### reference javascript client

Dependencies (tested versions): [ethers@4.0.47](https://docs.ethers.io/v5/api/utils/hashing/#utils-keccak256), [ws@7.3.1](https://github.com/websockets/ws)

```js
const axios = require("axios")
const utils = require("ethers").utils
const createWallet  = require("ethers").Wallet.createRandom

// Basic parameters
const url = "http://localhost:9000"
const entityId = "bda15be769a73fd2fc556f3c194c812dd321b056cb089cb83cede5edde6d16d8"

const generateAuthHash = (fields, secret) => {
  str = fields.reduce((x,y) => String(x)+String(y)) + secret
  return utils.keccak256(utils.toUtf8Bytes(str))  
}

// Example 1
// Send a token status request
const statusRequest = () => {
    var secret = "test"
    // Generate request
    var request = {
        method: "status",
        entityId,
        token: "69cdfe74-8f1f-45fa-9074-c82d5aade894",
    }
    // adjust timestamp precision to the server
    request.timestamp = Math.floor(Date.now() / 1000)
    request.authHash = generateAuthHash(new Array(request.entityId, request.method, request.timestamp, request.token), secret)  

    // Generate random request id
    var rand = Math.random().toString(16).split('.')[1]
    var requestId = utils.keccak256('0x' + rand).substr(2, 10)
    var body = {
        "id": requestId,
        "request": request
    }
    console.log("REQUEST\n",body)
    return axios.post(`${url}/api/token`, body)
}

statusRequest()
    .then(res => {
        response = res.data.response
        if (response.ok) console.log("RESPONSE\n",JSON.stringify(response,null,2))
        else  console.error("ERROR\n",JSON.stringify(response.message,null,2))
    })
    .catch(err => console.error("ERROR\n",JSON.stringify(err,null,2)))


// Example 2
// Send an importKeysBulk request
const importKeysRequest = () => {
    const createKeys = (size) => {
        var keys = Array.from({length: size}, () => createWallet().signingKey.publicKey)
        return keys
    }

    const secret = "test"
    // For number of keys higher than 10000 the socket timeout may need to be adjusted 
    const keys = createKeys(100)
    // Generate request
    var request = {
        method: "importKeysBulk",
        entityId,
        keys,
    }
    // adjust timestamp precision to the server
    request.timestamp = Math.floor(Date.now() / 1000)
    request.authHash = generateAuthHash(keys.concat(request.entityId, request.method, request.timestamp), secret)  
    // Generate random request id
    var rand = Math.random().toString(16).split('.')[1]
    var requestId = utils.keccak256('0x' + rand).substr(2, 10)
    var body = {
        "id": requestId,
        "request": request
    }
    console.log("REQUEST\n",body)
    return axios.post(`${url}/api/token`, body)
}

importKeysRequest()
    .then(res => {
        response = res.data.response
        if (response.ok) console.log("RESPONSE\n",JSON.stringify(response,null,2))
        else  console.error("ERROR\n",JSON.stringify(response.message,null,2))
    })
    .catch(err => console.error("ERROR\n",JSON.stringify(err,null,2)))
```

### Callback

The organization can define an HTTP callback that will be triggered on some registration events.

The callback definition looks like this:

`https://domain.com/callback?authHash={AUTHASH}&event={EVENT}&timestamp={TIMESTAMP}&token={TOKEN}`

The parameters between braces will be replaced on each callback call. So `{EVENT}` will became `register` on the register callback.

The authHash is calculated the same way described above `alphabetical order of field values + shared secret` thus `AUTHASH = Keccak256(event + ts + token + secret)`.