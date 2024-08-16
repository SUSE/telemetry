# /authenticate

Type: **POST**

## Parameters

| Name | Type | Description | Example |
| ---- | ---- | ----------- | ------- |
| body | object | {<br>&nbsp;&nbsp;clientId integer($int64)<br>&nbsp;&nbsp;instIdHash {<br>&nbsp;&nbsp;&nbsp;&nbsp;method string<br>&nbsp;&nbsp;&nbsp;&nbsp;value string<br>&nbsp;&nbsp;}<br>} | {<br>&nbsp;&nbsp;"clientId": 1234567890<br>&nbsp;&nbsp;"instIdHash": {<br>&nbsp;&nbsp;&nbsp;&nbsp;"method": "sha256"<br>&nbsp;&nbsp;&nbsp;&nbsp;"value": "984271ec70628b47995fdf9271ded6274c2b104ce201164a9b63cfefef7f40d0"<br>&nbsp;&nbsp;}<br>}|

Request body type `ClientAuthenticationRequest` defined in [restapi module](../../../pkg/restapi)

## Responses

| Code | Description | Example |
| ---- | ----------- | ------- |
| 200  | Success<br>`Content-Type: application/json`<br>{<br>&nbsp;&nbsp;clientId integer($int64)<br>&nbsp;&nbsp;authToken string<br>&nbsp;&nbsp;registrationDate string<br>} | {<br>&nbsp;&nbsp;"clientId": 1234567890<br>&nbsp;&nbsp;"authToken": "encoded.JWT.token"<br>&nbsp;&nbsp;"registrationDate": "2024-08-01T01:02:03.000000Z"<br>} |
| 400  | Bad Request<br>Missing or incompatible body<br>`Content-Type: application/json`<br>{<br>&nbsp;&nbsp;error string<br>} | {<br>&nbsp;&nbsp;"error": "no clientInstanceId value provided"<br>} |
| 401  | Unauthorized<br>Client (re-)registration required due to one of:<br>- specified client is not registered<br>- invalid clientId provided<br>- provided clientInstanceId hash doesn't match<br>[WWW-Authenticate](../headers/www-authenticate.md) response header will specify recovery action<br>`Content-Type: application/json`<br>{<br>&nbsp;&nbsp;error string<br>} | {<br>&nbsp;&nbsp;"error": "Client not registered"<br>} |

Response success body type `ClientAuthenticationResponse` defined in [restapi module](../../../pkg/restapi)

If the response is a 401 Unauthorized then the `WWW-Authenticate` reponse header will be set.
See [www-authenticate.md](../headers/www-authenticate.md) for more details.
