# NetBackup™ 10.5 API - Getting Started

## Introduction

The NetBackup API provides a web-service based interface to configure and administer NetBackup, the industry leader in data protection for enterprise environments.

### NetBackup API is RESTful

The NetBackup API is built on the Representational State Transfer (REST) architecture, which is the most widely used style for building APIs. The NetBackup API uses the HTTP protocol to communicate with NetBackup. The NetBackup API is therefore easy to use in cloud-based applications, as well as across multiple platforms and programming languages.

### JSON message format

The NetBackup API uses JavaScript Object Notation (JSON) as the message format for request and response messages.

### The client-server relationship

The NetBackup API employs client-server communication in the form of HTTP requests and responses.

* The API client (your program) uses the HTTP protocol to make an API request to the NetBackup server.
* The NetBackup server processes the request. The server responds to the client with an appropriate HTTP status code indicating either success or failure. The client then extracts the required information from the server's response.

## Overview

### Authentication

NetBackup authenticates the incoming API requests based on a JSON Web Token (JWT) or an API key that needs to be provided in the `Authorization` HTTP header when making the API requests.

* A JSON Web Token (JWT) is acquired by executing a `login` API request.
* An API key is acquired by executing a `api-keys` API request.
  A NetBackup API key is a pre-authenticated token that lets a NetBackup user run NetBackup commands (such as nbcertcmd -createToken or nbcertcmd -revokeCertificate) or access NetBackup RESTful APIs.
  Unlike a password, an API key can exist for a long time and you can configure its expiration.
  Therefore, once an API key is configured, operations like automation can run for a long time using the API key.
  More details around API keys can be found in NetBackup Security and Encryption Guide: http://www.veritas.com/docs/DOC5332.

> **TIP:**
> The port that is used to access the NetBackup API is the standard NetBackup PBX port, `1556`.

## Authentication Examples

### Example 1: JWT Authentication

**Step 1** - Login to get JWT:

```bash
curl -X POST https://masterservername:1556/netbackup/login \
     -H 'content-type: application/vnd.netbackup+json;version=1.0' \
     -d '{
            "domainType":"vx",
            "domainName":"mydomain",
            "userName":"myusername",
            "password":"mypassword"
        }'
```

Response (JWT valid for 86400 seconds / 24 hours):

```json
{
    "token": "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsInppcCI6IkRFRiJ9...",
    "tokenType": "BEARER",
    "validity": 86400
}
```

**Step 2** - Use JWT to access API endpoints:

```bash
curl -X GET https://masterservername:1556/netbackup/admin/jobs/5 \
     -H 'Accept: application/vnd.netbackup+json;version=2.0' \
     -H 'Authorization: <JWT_TOKEN>'
```

### Example 2: API Key Authentication

**Step 1** - Login to get JWT (same as above)

**Step 2** - Create API key using JWT:

```bash
curl -X POST https://masterservername:1556/netbackup/security/api-keys \
     -H 'Content-Type: application/vnd.netbackup+json;version=3.0' \
     -H 'Authorization: <JWT_TOKEN>' \
     -d '{"data":{"type": "apiKeyCreationRequest","attributes":{"expireAfterDays":"P1D","description":"API key for user myusername with 1 year validity"}}}'
```

Response:

```json
{
    "data": {
        "type": "apiKeyCreationResponse",
        "id": "A1uMjmj93mo=",
        "attributes": {
            "apiKey": "A1uMjmj93mrpsW9qWroT-paNdMSDPdABCEDi7PRPD-Mc0mkFn6-KvHdXT1v8Wxx7",
            "expiryDateTime": "2019-04-25T08:28:30.294Z"
        }
    }
}
```

**Step 3** - Use API key for subsequent requests:

```bash
curl -X GET https://masterservername:1556/netbackup/admin/jobs/5 \
     -H 'Accept: application/vnd.netbackup+json;version=2.0' \
     -H 'Authorization: A1uMjmj93mrpsW9qWroT-paNdMSDPdABCEDi7PRPD-Mc0mkFn6-KvHdXT1v8Wxx7'
```

## Multifactor Authentication (MFA)

> **NOTE:** NetBackup supports multifactor authentication starting from version **10.3**.

### Two-Step MFA Process

**Step 1** - Get MFA token:

```bash
curl -X POST https://masterservername:1556/netbackup/login \
     -H 'content-type: application/vnd.netbackup+json;version=10.0' \
     -d '{
            "domainType":"unixpwd | NT | vx | ldap",
            "domainName":"mydomain",
            "userName":"myusername",
            "password":"mypassword"
        }'
```

Response (MFA token valid for 180 seconds):

```json
{
    "token": "AIwUEF-fbJEtnGtUhUBuadgTgMrRRTnjbKjFFhTyhZWwtMDgubmJ1c2VjLnZ4aW5kaWEudmVyaXRhAVAbbdmthbmRlOnVuaXhwd2Q6MTAwMjpmYWxzZQ==...",
    "tokenType": "MFA",
    "validity": 180
}
```

**Step 2** - Validate OTP to get JWT:

```bash
curl -X POST https://masterservername:1556/netbackup/login \
     -H 'content-type: application/vnd.netbackup+json;version=10.0' \
     -d '{
            "domainType":"vx",
            "domainName":"vrts.mfa.totp",
            "userName":"<MFA_TOKEN_FROM_STEP_1>",
            "password":"123456"
        }'
```

### Alternative: Password + OTP Combined

```bash
curl -X POST https://masterservername:1556/netbackup/login \
     -H 'content-type: application/vnd.netbackup+json;version=10.0' \
     -d '{
            "domainType":"vx",
            "domainName":"mydomain",
            "userName":"myusername",
            "password":"mypassword123456"
        }'
```

## Adaptive Multifactor Authentication

> **NOTE:** NetBackup supports Adaptive MFA starting from version **10.4** for critical operations.

Supported endpoints:
- `POST /netbackup/security/properties`
- `POST /netbackup/security/api-keys`

When a critical operation requires additional authentication, the API returns HTTP 202 with `X-NetBackup-MFA-Request` header containing a request ID.

**Validate the request:**

```bash
curl -X POST https://masterservername:1556/netbackup/security/adaptive-mfa/{requestId}/validate \
     -H 'content-type: application/vnd.netbackup+json;version=11.0' \
     -d '{
           "data": {
             "type": "mfaOTPValidateRequest",
             "attributes": {
                   "otpCode": "842815"
             }
           }
        }'
```

## SSL Certificate Validation

**Step 1** - Get CA certificate:

```bash
curl -X GET https://masterservername:1556/netbackup/security/cacert \
     -H 'content-type: application/vnd.netbackup+json;version=2.0' \
     --insecure
```

**Step 2** - Save the `webRootCert` to a file (e.g., `cacert.pem`)

**Step 3** - Use the certificate in subsequent requests:

```bash
curl -X GET https://masterservername:1556/netbackup/security/cacert \
     -H 'content-type: application/vnd.netbackup+json;version=2.0' \
     --cacert cacert.pem
```

## Authorization (RBAC)

### Key Concepts

- **Objects**: NetBackup resources (policies, jobs, assets, etc.)
- **Namespace**: Hierarchical identifiers for resources (e.g., `|ASSETS|MSSQL|INSTANCES|`)
- **Operations**: Actions on objects (e.g., `|OPERATIONS|VIEW|`, `|OPERATIONS|UPDATE|`)
- **Roles**: Job functions assigned to principals
- **ACL**: Access Control Lists defining who can do what
- **Inheritance**: Child resources inherit parent permissions
- **Propagation**: ACEs apply to self, children, or both

### Enforcement Levels

**API-Level**: Access to all objects managed by an endpoint
**Object-Level**: Granular access to specific resources

## API Features

### Versioning

Format: `application/vnd.netbackup+json;version=MAJOR.MINOR`

Example: `application/vnd.netbackup+json;version=2.0`

Version changes occur when:
- Endpoints are removed (Yes)
- Output fields are removed (Yes)
- Required input fields are added (Yes)
- New endpoints are added (No)
- Optional fields are added (No)

### Pagination

Query parameters:
- `page[offset]`: Starting point (default: 0)
- `page[limit]`: Page size (default: 10)

### Filtering

Uses OData syntax: http://docs.oasis-open.org/odata/odata/v4.01/

### Date/Time Format

ISO 8601 format in UTC with Z zone designator

### Integer Format

Default: 32-bit (`int32`) unless specified otherwise

### Request Headers

**X-Client-ID**: Identifies the consumer component (recommended for telemetry)
**X-Request-ID**: UUID for request tracking (auto-generated if not provided)

### URL Encoding

All URLs must be percent-encoded per RFC3986:
- Encode path parameters
- Encode query parameter names and values

Example:
```
Intended: filter=startswith(extendedAttributes/cluster,'Name-Ã  !@#$%^&*')
Encoded:  filter=startswith(extendedAttributes%2Fcluster,'Name-%C3%A0%20!%40%23%24%25%5E%26*')
```

## Asynchronous APIs

**Step 1** - Initiate operation:

```bash
GET /netbackup/library/async-articles-resource HTTP/1.1
Accept: application/vnd.netbackup+json; version=1.0
```

Response:
```
HTTP/1.1 202 Accepted
Location: /netbackup/library/async-articles-resource-results/10
Retry-after: 2022-01-14T14:46:01.311Z
```

**Step 2** - Check status:

```bash
GET /netbackup/library/async-articles-resource-results/10
```

**Step 3** - Cancel operation (optional):

```bash
DELETE /netbackup/library/async-articles-resource-results/10
```

## API Documentation

Access Swagger UI on master server:
`https://<master_server>:1556/api-docs/index.html`

> **WARNING:** This makes real API calls to your master server. Use in development environments only.

## Code Samples

GitHub repository: https://github.com/VeritasOS/netbackup-api-code-samples

> **Disclaimer:** Community-supported, open source (MIT license). Not officially supported by Veritas. For reference only.

## What's New in NetBackup 10.5

### New APIs

**Security Dashboard:**
- Retrieve security configuration risk score
- Set security baseline configuration values

**Key Management Services (NBKMS):**
- Recover, create, fetch, update, delete key groups
- Update and delete keys in key groups

**Aggregate Anomaly:**
- Retrieve, delete, and update aggregate job anomalies
- Bulk update anomaly status

**Anomaly File Extensions:**
- Retrieve system-defined ransomware file extensions
- Create, retrieve, and update user-defined ransomware file extensions

### Versioned APIs

Breaking changes in v10.5:
- `GET /security/mfa-status`: Replaced `isMfaNetBackupManaged` with `deploymentType` attribute

---

**Source:** https://sort.veritas.com/public/documents/nbu/10.5/windowsandunix/productguides/html/getting-started/
**Retrieved:** 2025-11-08
