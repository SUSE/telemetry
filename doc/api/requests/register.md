# /register

Type: **POST**

## Parameters

| Name | Type | Description | Example |
| ---- | ---- | ----------- | ------- |
| body | object | {<br>&nbsp;&nbsp;clientRegistration: {<br>&nbsp;&nbsp;&nbsp;&nbsp;clientId: string<br>&nbsp;&nbsp;&nbsp;&nbsp;systemUUID: string<br>&nbsp;&nbsp;&nbsp;&nbsp;timestamp: string($[rfc3339nano](https://pkg.go.dev/time#pkg-constants))<br>&nbsp;&nbsp;}<br>} | {<br>&nbsp;&nbsp;"clientRegistration": {<br>&nbsp;&nbsp;&nbsp;&nbsp;"clientId": "f323628e-c1cc-45d4-824d-22d4d6f0fd01"<br>&nbsp;&nbsp;&nbsp;&nbsp;"systemUUID": "74f0f0b0-fb29-4405-a0b8-4e7747bdfd8a"<br>&nbsp;&nbsp;&nbsp;&nbsp;"timestamp": "2024-08-01T00:01:02.000000Z"<br>&nbsp;&nbsp;}<br>}|

Request body type `ClientRegistrationRequest` defined in [restapi module](../../../pkg/restapi/)

## Responses

| Code | Description | Example |
| ---- | ----------- | ------- |
| 200  | Success<br>`Content-Type: application/json`<br>{<br>&nbsp;&nbsp;registrationId integer($int64)<br>&nbsp;&nbsp;authToken string<br>&nbsp;&nbsp;registrationDate string<br>} | {<br>&nbsp;&nbsp;"registrationId": 1234567890<br>&nbsp;&nbsp;"authToken": "encoded.JWT.token"<br>&nbsp;&nbsp;"registrationDate": "2024-08-01T01:02:03.000000Z"<br>} |
| 400  | Bad Request<br>Missing or incompatible body<br>`Content-Type: application/json`<br>{<br>&nbsp;&nbsp;error string<br>} | {<br>&nbsp;&nbsp;"error": "missing registration clientId"<br>}<br>or<br>{<br>&nbsp;&nbsp;"error": "missing registration timestamp"<br>} |
| 409  | Conflict<br>Client Registration or Registration's Client Id already registered<br>`Content-Type: application/json`<br>{<br>&nbsp;&nbsp;error string<br>} | {<br>&nbsp;&nbsp;"error": "specified registration already exists"<br>}<br>or<br>{<br>&nbsp;&nbsp;"error": "specified registration clientId already exists"<br>} |

Response success body type `ClientRegistrationResponse` defined in [restapi module](../../../pkg/restapi/)