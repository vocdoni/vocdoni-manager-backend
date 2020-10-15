# Token API

The Token API allows organizations to take control over the user tokens required for user registration on the Vocdoni registry backend.

The API can be accessed on a url like this `wss://vocdoni-registry-service/api/token`. It uses websockets with a JSON payload.

## Authentication

There is a common known secret between the entity and the vocdoni backend. This secret is used to compute the authHash field.

`authHash: hexadecimal(keccak256(field1+field2+fieldN+secret))`

The fields are concatenated strings. One of the fields must be a 32 bits `timestamp` (seconds).

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

`authHash = keccack256("0x123456"+"revoke"+"1234567890"+"xxx-yyy-zzz"+"hello")`

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
const WebSocket = require('ws')
const keccak256 = require("ethers").utils.keccak256
const toUtf8Bytes = require("ethers").utils.toUtf8Bytes

const ws = new WebSocket('ws://localhost:8000/api/token')

const generateAuthHash = (fields, secret) => {
  str = fields.reduce((x,y) => String(x)+String(y)) + secret
  return keccak256(toUtf8Bytes(str))  
}

ws.on('message', function incoming(message) {
    console.log('received: %s', message);
    ws.close()
});

ws.on('open', function open() {
  var secret = "test"
  // Generate request
  var request = {}
  request.method= "status"
  // adjust timestamp precision to the server
  request.timestamp = Math.floor(Date.now() / 1000)
  request.authHash = generateAuthHash(new Array(request.entityId, request.method, request.timestamp, request.token), secret)  

  // Generate random request id
  var rand = Math.random().toString(16).split('.')[1]
  var requestId = keccak256('0x' + rand).substr(2, 10)
  var msg = {
  "id": requestId,
  "request": request
  }
  console.log(msg)
  ws.send(JSON.stringify(msg));

ws.on('error', function error(err) {
  console.error(err)
  })
});
```

### Callback

The organization can define an HTTP callback that will be triggered on some registration events.

The callback definition looks like this:

`https://domain.com/callback?authHash={AUTHASH}&event={EVENT}&timestamp={TIMESTAMP}&token={TOKEN}`

The parameters between braces will be replaced on each callback call. So `{EVENT}` will became `register` on the register callback.

The authHash is calculated the same way described above `alphabetical order of field values + shared secret` thus `AUTHASH = Keccak256(event + ts + token + secret)`.