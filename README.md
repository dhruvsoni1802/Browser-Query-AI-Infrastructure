# Browser Query AI

A browser query AI service built with Go for learning distributed systems.

## Getting Started

Run the server:

```bash
go run ./cmd/server
```

Run the tests:
```bash
go test -v ./internal/session/...
```

Run the tests with race detection:

```bash
go test -race ./internal/session/...
```
## Environment Variables

The following environment variables can be set to configure the service:

### `ENV`
Sets the environment mode. Affects logging format.
- `production` - Uses JSON logging format
- `development` (default) - Uses human-readable text logging format

```bash
ENV=production go run ./cmd/server
```

### `CHROMIUM_PATH`
Optional. Path to the Chromium/Chrome binary. If not set, the service will automatically search common installation paths.

```bash
CHROMIUM_PATH="/path/to/chromium" go run ./cmd/server
```

### `SERVER_PORT`
Optional. Port number for the server to listen on.
- Default: `8080`

```bash
SERVER_PORT=3000 go run ./cmd/server
```

### `MAX_BROWSERS`
Optional. Maximum number of browser instances that can run concurrently.
- Default: `5`

```bash
MAX_BROWSERS=10 go run ./cmd/server
```

## Example with Multiple Environment Variables

```bash
ENV=production SERVER_PORT=3000 MAX_BROWSERS=10 go run ./cmd/server
```

# API Endpoints

## Create Session with Name

Request:
```bash

POST  http://{SERVER_URL}/sessions
{
  "agent_id": "Unique ID for the AI Agent",
  "session_name": "Unique Name for the Session of that AI Agent"
}
```

Example Request:
```bash
POST http://localhost:8080/sessions
{
  "agent_id": "agent-bob",
  "session_name": "research-task"
}
```

Response:

```json

{
    "session_id": "sess_058PUFuOLpmLD8WWp7Gw9g==",
    "session_name": "research-task",
    "agent_id": "agent-bob",
    "context_id": "55BEA2F416F7CCCCAE861453AA1C14BB",
    "created_at": "2026-02-09T00:43:29.821748-05:00"
}
```

Keep the session_name and agent_id unique for every AI Agent.

## Creat Session without Name

Request:
```bash

POST  http://{SERVER_URL}/sessions

{
  "agent_id": "Unique ID for the AI Agent",
}
```

Example Request:
```bash
POST http://localhost:8080/sessions
{
  "agent_id": "agent-alice"
}
```

Response:

```json

{
    "session_id": "sess_KMQR3ouLhf3aqJvP3L7Z3A==",
    "session_name": "session-2026-02-09-KMQR3ouL",
    "agent_id": "agent-alice",
    "context_id": "76BFA46581EF48B32632893DC29679CA",
    "created_at": "2026-02-09T00:43:34.757622-05:00"
}
```

The backend will automatically generate a session_name if not provided.

## Navigate to a URL in a Session

Request:
```bash
POST  http://{SERVER_URL}/sessions/{id}/navigate

{
  "url": "Any website URL you want to visit"
}
```

Example Request:
```bash
POST http://localhost:8080/sessions/sess_cOPHllumy5RIghDWWCrIlw==/navigate

{
  "url": "https://www.example.com"
}
```

Response:

```json

{
    "session_id": "sess_cOPHllumy5RIghDWWCrIlw==",
    "page_id": "BC22F0A8F5B43205C0A8FC920A1A8C51",
    "url": "https://example.com/"
}
```

Use the session_id returned from the Create session (with or without name) endpoint inside as {id} in the URL.


## Execute JavaScript on a Page in a Session

Request:

```bash
POST http://{SERVER_URL}/sessions/{id}/execute

{
  "page_id": "Any page ID you want to execute JavaScript on",
  "script": "DOM manipulation code you want to execute"
}
```

Example Request:
```bash
POST http://localhost:8080/sessions/sess_PhmTI_Pp7wVoC_YKDR1CJA==/execute

{
  "page_id": "F88D081D45FF710195145A522D524699",
  "script": "document.title"
}

```

Response:

```json

{
    "session_id": "sess_PhmTI_Pp7wVoC_YKDR1CJA==",
    "page_id": "F88D081D45FF710195145A522D524699",
    "result": "All Jobs From Hacker News 'Who is Hiring?' Posts | HNHIRING"
}
```

Use the session_id returned from the Create session (with or without name) endpoint inside as {id} in the URL.

Note that to get a page_id, you need to navigate to a URL first.

## Capture Screenshot of a Page in a Session

Request:

```bash
POST http://{SERVER_URL}/sessions/{id}/screenshot

{
  "page_id": "Any page ID you want to capture screenshot of",
}
```

Example Request:  

```bash
POST http://localhost:8080/sessions/sess_PhmTI_Pp7wVoC_YKDR1CJA==/screenshot
{
  "page_id": "F88D081D45FF710195145A522D524699"
}
```

Response:

```json 
{
    "session_id": "sess_PhmTI_Pp7wVoC_YKDR1CJA==",
    "page_id": "F88D081D45FF710195145A522D524699",
    "screenshot": "iVBORw0KGgoAAAANSUh....",
    "format": "png",
    "size": 39694
}
```

Use the session_id returned from the Create session (with or without name) endpoint inside as {id} in the URL.

Note that to get a page_id, you need to navigate to a URL first.

The screenshot is returned as a base64 encoded string. You can decode it to get the image data.

## Get Page Content of a Page in a Session

Request:

```bash
GET http://{SERVER_URL}/sessions/{id}/pages/{pageId}/content
```

Example Request:
```bash
GET http://localhost:8080/sessions/sess_cOPHllumy5RIghDWWCrIlw==/pages/BC22F0A8F5B43205C0A8FC920A1A8C51/content

Response:

```json
{
    "session_id": "sess_cOPHllumy5RIghDWWCrIlw==",
    "page_id": "BC22F0A8F5B43205C0A8FC920A1A8C51",
    "content": "<!DOCTYPE html><html lang=\"en\"><head><title>Example Domain</title><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"><style>body{background:#eee;width:60vw;margin:15vh auto;font-family:system-ui,sans-serif}h1{font-size:1.5em}div{opacity:0.8}a:link,a:visited{color:#348}</style></head><body><div><h1>Example Domain</h1><p>This domain is for use in documentation examples without needing permission. Avoid use in operations.</p><p><a href=\"https://iana.org/domains/example\">Learn more</a></p></div>\n</body></html>",
    "length": 528
}
```

Use the session_id returned from the Create session (with or without name) endpoint inside as {id} in the URL.

Note that to get a page_id, you need to navigate to a URL first.

The content is returned as a string. You can parse it to get the HTML content. 

## Get information about a Session

Request:

```bash
GET http://{SERVER_URL}/sessions/{id}
```

Example Request:
```bash
GET http://localhost:8080/sessions/sess_KMQR3ouLhf3aqJvP3L7Z3A==
```

Response: 
```json
{
    "session_id": "sess_KMQR3ouLhf3aqJvP3L7Z3A==",
    "session_name": "session-2026-02-09-KMQR3ouL",
    "agent_id": "agent-alice",
    "context_id": "76BFA46581EF48B32632893DC29679CA",
    "page_ids": [
        "9FD9F7BC07E73944525F05544DD7856D"
    ],
    "page_count": 1,
    "created_at": "2026-02-09T00:43:34.757622-05:00",
    "last_activity": "2026-02-09T00:45:54.038708-05:00",
    "status": "active"
}
```

Use the session_id returned from the Create session (with or without name) endpoint inside as {id} in the URL.

## List all Sessions  

Request:

```bash
GET http://{SERVER_URL}/sessions
```

Example Request:
```bash
GET http://localhost:8080/sessions
```

Response:
```json
{
    "sessions": [
        {
            "session_id": "sess_cOPHllumy5RIghDWWCrIlw==",
            "session_name": "research-task-2",
            "agent_id": "agent-alice",
            "context_id": "9C6312EC5A2E2F39049F90AB1F3CABC1",
            "page_count": 1,
            "created_at": "2026-02-09T00:43:14.409353-05:00",
            "last_activity": "2026-02-09T00:49:00.406728-05:00",
            "status": "active"
        },
        {
            "session_id": "sess_058PUFuOLpmLD8WWp7Gw9g==",
            "session_name": "research-task",
            "agent_id": "agent-bob",
            "context_id": "55BEA2F416F7CCCCAE861453AA1C14BB",
            "page_count": 0,
            "created_at": "2026-02-09T00:43:29.821748-05:00",
            "last_activity": "2026-02-09T00:43:29.821748-05:00",
            "status": "active"
        },
        {
            "session_id": "sess_KMQR3ouLhf3aqJvP3L7Z3A==",
            "session_name": "session-2026-02-09-KMQR3ouL",
            "agent_id": "agent-alice",
            "context_id": "76BFA46581EF48B32632893DC29679CA",
            "page_count": 1,
            "created_at": "2026-02-09T00:43:34.757622-05:00",
            "last_activity": "2026-02-09T00:45:54.038708-05:00",
            "status": "active"
        }
    ],
    "count": 3
}
```

## List all Sessions for an Agent

Request:

```bash
GET http://{SERVER_URL}/agents/{agentId}/sessions
```

Example Request:
```bash
GET http://localhost:8080/agents/agent-alice/sessions
```

Response:
```json
{
    "agent_id": "agent-alice",
    "sessions": [
        {
            "session_id": "sess_cOPHllumy5RIghDWWCrIlw==",
            "session_name": "research-task-2",
            "status": "active",
            "page_count": 1,
            "created_at": "2026-02-09T00:43:14.409353-05:00",
            "last_activity": "2026-02-09T00:49:37.40989-05:00"
        },
        {
            "session_id": "sess_KMQR3ouLhf3aqJvP3L7Z3A==",
            "session_name": "session-2026-02-09-KMQR3ouL",
            "status": "active",
            "page_count": 1,
            "created_at": "2026-02-09T00:43:34.757622-05:00",
            "last_activity": "2026-02-09T00:45:54.038708-05:00"
        }
    ],
    "count": 2
}
```

Use the same agent_id you used for creating sessions (with or without name) inside as {agentId} in the URL.

## Close a Page of a Session

Request:

```bash
DELETE http://{SERVER_URL}/sessions/{id}/pages/{pageId}
```

Example Request:
```bash
DELETE http://localhost:8080/sessions/sess_PhmTI_Pp7wVoC_YKDR1CJA==/pages/B4DD81BA63EF8533D9C13A76607CAB1A
``` 

Use the session_id returned from the Create session (with or without name) endpoint inside as {id} in the URL.

Note that to get a page_id, you need to navigate to a URL first. Also you can only delete a page that you have navigated to before.

The page will be closed and the page_id will be removed from the session. Also in case of success, the response will be just a 204 No Content.

## Delete a Session

Request:

```bash
DELETE http://{SERVER_URL}/sessions/{id}
```

Example Request:
```bash
DELETE http://localhost:8080/sessions/sess_PhmTI_Pp7wVoC_YKDR1CJA==
``` 

Use the session_id returned from the Create session (with or without name) endpoint inside as {id} in the URL.

The session will be deleted and all the pages will be closed. Also in case of success, the response will be just a 204 No Content.

The session data will also be deleted from Redis database and you won't we able to resume the session again.

## Close a Session

Request:

```bash
PUT http://{SERVER_URL}/sessions/{id}/close
```

Example Request:
```bash
PUT http://localhost:8080/sessions/sess_PhmTI_Pp7wVoC_YKDR1CJA==/close
``` 

Response:
```json
{
    "message": "Session closed. Pages were disposed and cannot be resumed.",
    "session_id": "sess_PhmTI_Pp7wVoC_YKDR1CJA==",
    "session_name": "research-task",
    "status": "idle"
}
```

Use the session_id returned from the Create session (with or without name) endpoint inside as {id} in the URL.  

The session will be closed and the session data will be kept in Redis database. You will be able to resume the session again.

However, the pages will be disposed and you won't be able to use the same pages again. 

## Resume a Session by Name

Request:

```bash
POST http://{SERVER_URL}/sessions/resume
{
  "agent_id": "Unique ID for the AI Agent",
  "session_name": "Unique Name for the Session of that AI Agent"
}
```

Example Request:
```bash
POST http://localhost:8080/sessions/resume
{
  "agent_id": "agent-alice",
  "session_name": "research-task"
}
``` 

Response: 
```json
{
    "session_id": "sess_PhmTI_Pp7wVoC_YKDR1CJA==",
    "session_name": "research-task",
    "resumed": true,
    "created_at": "2026-02-09T00:42:32-05:00"
}
``` 

Use the same agent_id you used for creating sessions (with or without name) inside as agent_id in the request body.

Use the same session_name you used for creating sessions (with or without name) inside as session_name in the request body.



The session will be resumed and the session data will be loaded from Redis database. However, you won't be able to use the same pages again.