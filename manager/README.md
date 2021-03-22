# Manager API
The  Manager API allows organizations to perform all the necessary management actions on the backend that are related to their members and the corresponding censuses. This invloves:
- Registering new `Entities` and managing their info
- Managing `Members` and their information. Related to `Member` management also some `Token` calls are provided
- Creating and managing `Tags` for these members
- Creating and managing `Targets` that combine member attributes and tags in order to provide the ability to segment the members
- Creating and managing `Censuses` based that eacho of them is related with one concrete `Target` 

Available by default under `/manager`

## Entities
### sign Up
Registers an entity to the backend. The address/ID of the Entity is calculated by the signature of the request.
- Request
```json
{    
    "id": "req-12345678",
    "request": {
        "method": "signUp",
        "entity": {
            "name" : "Name",
            "email" : "email@email.com",
        }
    }
    "signature": "0x12345"
}
```
- Response
```json
{
    "id": "req-12345678",
    "response": {
        "ok": true,
        "count": 15000000,  // gas provided by faucet, if any
    },
     "signature": "0x12345"
}
```

### getEntity
- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "getEntity",
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
        "entity": {
            "callbackUrl": "http",
            "callbackSecret": "x45gse53wedfg",
            "email": "mail@entity.org",
            "name": "EntityName",
            "censusManagersAddresses": ["0x434223edfa","0x434223edfc"],
            "origin": "Token",
        }
    },
    "signature": "0x123456"
}
```
### updateEntity
- Request
```json
{    
    "id": "req-12345678",
    "request": {
        "method": "updateEntity",
        "entity": {
            "callbackUrl": "http",
            "callbackSecret": "x45gse53wedfg",
            "email": "mail@entity.org",
            "name": "EntityName",
            "censusManagersAddresses": ["0x434223edfa","0x434223edfc"],
            "origin": "Token",
        }
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
        "count": 1,
    },
     "signature": "0x12345"
}
```

## Members
### countMembers
Counts the number of members for a given entity.
- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "countMembers",
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
        "count": 10 // number of members
    },
     "signature": "0x12345"
    
}
```

### listMembers

Retrieve a list of members with the given constraints.

- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "listMembers",
        "listOptions": {
          "skip": 50,
          "count": 50,
          "sortBy": "lastName", // "name" | "lastName" | "email" | "dateOfBirth"
          "order": "asc",  // "asc" | "desc"
        },
        "filter": {  
             "target": "1234...",
             "tags": "2345...",
             "name": "John...",  
             "lastName": "Smith...",
             "email": "john@s..." ,
        }
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
        "members": [
            { "id": "1234...", "name": "John", "lastName": "Smith", }, //all member info
            { "id": "2345...", "name": "Jane", "lastName": "Smith",}
        ]
    }
    "signature": "0x123456"
}
```

### getMember
- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "getMember",
        "id": "1234-1234..."  // uuid
    },
    "signature": "0x12345"
}
```
- Response
```json
{ "id": "req-12345678",
    "response": {
        "ok": true,
        "member": {
            "id": "1234...", 
            "name": "John",
            "lastName": "Smith",
            "email": "john@smith.com",
            "phone": "+0123456789",
            "dateOfBirth": "2000-05-14T15:52:00.741Z" // ISO date
            ... 
        }
    },
    "signature": "0x123456"
}
```

### addMember
- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "addMember",
        "member": {
            "firstName": "John",
            "lastName": "Smith",
            "email": "john@smith.com",
            "phone": "+0123456789",
        }
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
        "member": {
            "id": "1234...",
            "firstName": "John",
            "lastName": "Smith",
            "email": "john@smith.com",
            "phone": "+0123456789", 
        }
    },
    "signature": "0x123456"
}
```


### updateMember
**Note**: All attributes execpet`tags`  if are included in the request but are empty they are ingored. If `tags == []` this value is stored in the database. 
- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "updateMember",
        "memberId": "1234",
        "member": {
           "email": "john@smith.com",
           "firstName": "John1",
           "tags": [1,2]
        }
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
        "count": 1,
    },
     "signature": "0x123456"
}
```


### deleteMembers
The calls fails if no `memberIds` are provided. Duplcate member IDs are ignored. 
The following constriant applies `length(memberIds) = count+length(invalidIds)+duplicates`. The duplicates are not provided by the response but they can be calculated from the above constraint.

- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "deleteMembers",
        "memberIds":  ["1234...","4567...."],
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
        "count": 2, // number of members deleted
        "invalidIds":["7890-cdefg-..."]:, // number of non-existing IDs that where included in the request

    },
     "signature": "0x123456"
}
```

### importMembers
Imports the given array of members with their info into the database.
- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "importMembers",
        "members": [
            {
                "firstName": "John",
                "lastName": "Smith",
                "email": "john@smith.com",
            }, 
            ...
        ]
    },
    "signature": "0x12345"
}
```
- Response
```json
{
    "id": "req-12345678",
    "response": {
        "ok": true
    },
     "signature": "0x123456"
}
```

### sendValidationLink
Uses the `SMTP` module to send an email to the  selected member, containing the necesary info to register his public key.  Members already verified are ingored (a corresponding message is returned). `ok:false` is returned only in the case that there memberIDs contains valid members, but no mail was succesfully sent for any of these IDs (either because they are already validated or because email sending failed). In contrast with other calls, a `message` can be present in the response also in the case of `ok:true`, the IDs to which an email was not sent and the corresponfing error.

Duplicate member IDs are ignored. The following constraint applies `length(memberIds) = count+length(invalidIds)+duplicates+length(errors)`.

An automated tag called "PendingValidation" is added to the members to which the emails were sent.

See also `validateToken`
- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "sendValidationLink",
        "memberIds": ["1234...","4567....","7890-cdefg-...","98769-adsdb-..."],
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
        "count": 2, // number of emails sent
        "invalidIds":["7890-cdefg-..."], // set of non-existing IDs that where included in the request
        "message": "... errors were found:\n [Error1, Error2]", // errors messages for emains not sent
    },
     "signature": "0x123456"
}
```

### `sendVotingLinks`
Uses the `SMTP` module to send emails containing one-time voting links to the ephemeral members (not having registered using `api/registry`) of a census. If the parameter `mail` exists in the request then the backend checks if there is a **unique**  **non-verified** user with that email and sends his the participation email. In contrast with other calls, a `message` can be present in the response also in the case of `ok:true`, the IDs to which an email was not sent and the corresponfing error.

- Request:
```json
{
    "id": "req-12345678",
    "request": {
        "method": "sendVotingLinks",
        "processId": "12345badc34..", // received from gateway
        "censusId": "12345badc34...", // target uuid
        "email": "mail@mail.org" // optional, if added then email is sent only to this member
    },
    "signature": "0x12345"
}
```
- Response:
~~~json
{
    "id": "req-12345678",
    "response": {
        "ok": true,
    },
    "signature": "0x123456"
}
~~~


## Tokens

### provisionTokens
Adds N new empty member entries, filled with the corresponding ID/token only.
- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "provisionTokens",
        "amount": 500
    },
    "signature": "0x12345"
}
```
- Response
```json
{
    "id": "req-12345678",
    "response": {
        "ok": true
    },
     "signature": "0x123456"
}
```

### exportTokens
The response contains a set of `{"email":"...","token":"..."}` for each member of the Entity,  that has no public key registered.
- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "exportTokens"
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
        "memberTokens" : [
            {
             "email" : "john@smith.org",
             "token" : "1234-abcde-...."
            },
            {
             "email" : "maria@smith.org",
             "token" : "5768-abcde-...."
            },
            ...
        ]
    },
    "signature": "0x123456"
}
```
## Targets
Each target contains a set of filters applied to members. 
A target can be used to create a list of members that can be used to create  a Census.
The target id is a UUID.

### listTargets
Retrieve a list of targets.

- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "listTargets",
        "listOptions": {
            "skip": 50,
            "count": 50,
            "sortBy": "name",    // "name"
            "order": "asc",     // "asc" | "desc"
        }    
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
        "targets": [
            { "id": "1234-abcd-...", "name": "People over 18",  }, // all member info
        ]
    },
    "signature": "0x123456"
}
```

### getTarget
- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "getTarget",
        "targetId": "1234-abcd-..."
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
        "target": {
            "id": "1234-abcd-...",
            "name": "People over 18",
            "filters": {
                "attributes": {},
                "tags": [1,15],
            }
            
        }
    },
    "signature": "0x123456"
}
```

### addTarget
- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "addTarget"
        "target": {
          "name": string
          "filters": {
              "attributes": {},
              "tags": [1,15],
          }
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
    "target": {
        "id": "1234-abcd-..."
        "name": "Over 18"
        "filters": {
          "attributes": {},
          "tags": [1,15],
        }
    },
    "signature": "0x123456"
}
```


### updateTarget
- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "updateTarget"
        "id": "1234-abcd-...",
        "target": {
            "name": "Over 18"
            "filters": {
              "attributes": {},
              "tags": [1,15],
        }
    },
     "signature": "0x12345"
}
```
- Response
```json
{
    "id": "req-12345678",
    "response": {
        "ok": true
    }
    "signature": "0x123456"
}
```


### deleteTarget
- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "deleteTarget",
        "id": "1234-abcd-..."
    },
    "signature": "0x12345"
}
```
- Response
```json
{
    "id": "req-12345678",
    "response": {
        "ok": true
    },
    "signature": "0x123456"
}
```

### dumpTarget
Dumps the public keys of the users that match the criteria of the target. The client then can then call the go-dvote `addCensus` call to add the keys to the Census Service.

- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "dumpTarget",
        "id": "1234-abcd-...",
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
        "claims": [
            "12345abccdef", //pubKey1
            "7890abccdeff", //pubKey2
            "34567abccdef", //pubKey3
            ...
        ]
    },
    "signature": "0x123456"
}
```

## Censuses

### addCensus
Add a census that is already published by a DvoteGW using the details provided by it.
- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "addCensus",
        "censusId": "abc342123dadf/cdb12341a", // received from gateway
        "targetID": "1234-acvbc-...", // target uuid
        "census": {
            "name": "CensusName",
            "merkleRoot": "0fa34cb...",    // hex received from gateway
            "merkleTreeUri": "ipfs://abc23454cbf",   // received from gateway
            "target": "1234", // targetId
            "ephemeral": true, // flag that decides wether ephemeral identities are created for the non validated members or not
            "createdAt": "2000-05-14T15:52:00.741Z" 
        }
    },
    "signature": "0x12345"
}
```
- Response
```json
{
    "id": "req-12345678",
    "response": {
        "ok": true
    },
     "signature": "0x123456"
}
```

### updateCensus
Updates the census info

- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "updateCensus",
        "censusId": "12345badc34...", // target uuid
        "census": {
            "merkleRoot": "0fa34cb...",    // hex received from gateway
            "merkleTreeUri": "ipfs://abc23454cbf",   // received from gateway
        },
        "invalidClaims": []
    },
    "signature": "0x12345"
}
```
- Response:
~~~json
{
    "id": "req-12345678",
    "response": {
        "ok": true,
        "count": 1,
    },
    "signature": "0x123456"
}
~~~

### listCensus
Retrieve a list of exported census.
- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "listCensus",
        "listOptions": {
            "skip": 50,
            "count": 50,
            "sortBy": "name", // "name" | "created"
            "order": "asc",  // "asc" | "desc"
        }
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
        "census": [
            { "id": "1234...", "name": "People over 18", "target": "1234..." },
            ...
        ]
    },
    "signature": "0x123456"
}
```

### getCensus
Returns requested census with the corresponding target
- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "getCensus",
        "censusId": "1234..."
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
        "census": {
            "id": "1234...",
            "name": "CensusName",
            "merkleRoot": "09cc09acb080/3543c34fe23da",   
            "merkleTreeUri": "ipfs://abc23454cbf", 
            "target": "1234",
            "createdAt": "2000-05-14T15:52:00.741Z" 
        }
        "target": {
            "id": "1234...",
            "name": "People over 18", 
        }
    },
    "signature": "0x123456"
}
```



### deleteCensus
- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "deleteCensus",
        "id": "1234..."
    },
    "signature": "0x12345"
}
```
- Response
```json
{
    "id": "req-12345678",
    "response": {
        "ok": true
    },
     "signature": "0x123456"
}
```

### dumpCensus
Closing the census populating the `census_members` filling with the necessary ephemeral identies for the members who have are not verified.

- Request
~~~json
{
    "id": "req-12345678",
    "request": {
        "method": "dumpCensus",
        "censusId": "0x080980/0x3243"
    },
    "signature": "0x12345"
}
~~~
- Response
~~~json
{
    "id": "req-12345678",
    "response": {
        "ok": true,
        "claims": [
            "12345abccdef", //pubKey1
            "7890abccdeff", //pubKey2
            "34567abccdef", //pubKey3
            ...
        ],
    },
    "signature": "0x123456"
}
~~~


### Tags
### listTags
- Request
```json
{
    "id": "req-12345678",
    "request": {
        method: "listTags",
    },
    "signature": "0x12345"
}
```
- Response
```json
{
    "id": "req-12345678",
    "response": {
        "ok": true
        tags: [
            { "id": "1234...", "name": "People over 18"},
            ...
        ]
    },
     "signature": "0x123456"
}
```
### createTag
- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "createTag",
        "tagName": "TestTag",
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
        "tag": {
            "id": 1234,
            "name": "TestTag",
        }
    },
     "signature": "0x123456"
}
```
### deleteTag
- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "deleteTag",
        "tagID": 1234
    },
    "signature": "0x12345"
}
```
- Response
```json
{
    "id": "req-12345678",
    "response": {
        "ok": true
    },
     "signature": "0x123456"
}
```
### addTag
Adds Tag to a list of Members. The calls fails if no `memberIds` are provided. Duplcate member IDs are ignored.
The following constriant applies `length(memberIds) = count+length(invalidIds)+duplicates`. The duplicates are not provided by the response but they can be calculated from the above constraint.

- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "addTag",
        "memberIds": ["1234-abcde-...","4567-bcdef-....",...]
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
        "count": 2, // number of members updated
        "invalidIds":["7890-cdefg-..."], // number of non-existing IDs that where included in the request
    },
     "signature": "0x123456"
}
```

### removeTag
The calls fails if no `memberIds` are provided. Duplcate member IDs are ignored. 
The following constriant applies `length(memberIds) = count+length(invalidIDs)+duplicates`. The duplicates are not provided by the response but they can be calculated from the above constraint.

- Request
```json
{
    "id": "req-12345678",
    "request": {
        "method": "removeTag",
        "memberIds": ["1234...","5678....."]
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
        "count": 2, // number of members updated
        "invalidIds":["7890-cdefg-..."], // number of non-existing IDs that where included in the request
    },
     "signature": "0x123456"
}
```