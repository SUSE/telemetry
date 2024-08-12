# /register

Type: **POST**

## Parameters

| Name | Type | Description | Example |
| ---- | ---- | ----------- | ------- |
| body | object | {<br>&nbsp;&nbsp;clientInstanceId: string<br>} | {<br>&nbsp;&nbsp;"clientInstanceId": "ba2cb9f4927441602a385b27f502134902b636f395cadb3ea1438084dba29c8c"<br>}|

Request body type `ClientRegistrationRequest` defined in [restapi module](pkg/restapi)

## Responses

| Code | Description | Example |
| ---- | ----------- | ------- |
| 200  | Success<br>`Content-Type: application/json`<br>{<br>&nbsp;&nbsp;clientId integer($int64)<br>&nbsp;&nbsp;authToken string<br>&nbsp;&nbsp;issueDate string<br>} | {<br>&nbsp;&nbsp;"clientId": 1234567890<br>&nbsp;&nbsp;"authToken": "encoded.JWT.token"<br>&nbsp;&nbsp;"issueDate": "2024-08-01T01:02:03.000000Z"<br>} |
| 400  | Bad Request<br>Missing or incompatible body<br>`Content-Type: application/json`<br>{<br>&nbsp;&nbsp;error string<br>} | {<br>&nbsp;&nbsp;"error": "no clientInstanceId value provided"<br>} |
| 409  | Conflict<br>Client Instance Id already registered<br>`Content-Type: application/json`<br>{<br>&nbsp;&nbsp;error string<br>} | {<br>&nbsp;&nbsp;"error": "specified clientInstanceId already exists"<br>} |

Response success body type `ClientRegistrationResponse` defined in [restapi module](pkg/restapi)