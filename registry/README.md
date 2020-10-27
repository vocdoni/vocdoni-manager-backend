# Registry API
The Registry API allows organizations to register new users into its database, the registration can be done by a generated token (the user can exists or not) or by just adding the user directly without any token validation. It also allows to fetch the current registration status of each user.

Available by default under `/registry`

## Methods

### register

- Request:

```json
{
  "id": "req-12345678",
  "request": {
    "method": "register",
    "entityId": "0x12345",
    "memberInfo": {
        "dateOfBirth": "1990-09-03",
        "email": "info@vocdoni.io",
        "firstName": "vocdoni",
        "lastName": "vocdoni",
        "phone": "+3674574221",
        "streetAddress": "Road av.",
        "origin": "Token",
        "customFields": {}
    }
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
    "timestamp": 1556110671
  },
  "id": "req-12345678",
  "signature": "0x123456"
}
```

### validate token

- Request

The automated tag called "PendingValidation" is removed (if exists) from the member.


```json
{
  "id": "req-12345678",
  "request": {
    "method": "validateToken",
    "entityId": "0x12345",
    "token": "xxx-yyy-zzz" 
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
    "timestamp": 1556110671
  },
  "id": "req-12345678",
  "signature": "0x123456"
}
```

### registration status

- Request

```json
{
  "id": "req-12345678",
  "request": {
    "method": "registrationStatus",
    "entityId": "0x12345",
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
    "status": {
        "registered": true,
        "needsUpdate": false
    }
    "timestamp": 1556110671
  },
  "id": "req-12345678",
  "signature": "0x123456"
}
```