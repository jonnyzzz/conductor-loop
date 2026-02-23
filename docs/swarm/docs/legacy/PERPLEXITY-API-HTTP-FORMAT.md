# Perplexity API HTTP Request Format

## Research Findings

Complete specification for making HTTP requests to Perplexity's chat completions API with streaming enabled.

---

## 1. API Endpoint URL

**Base URL:** `https://api.perplexity.ai`
**Endpoint:** `/chat/completions`
**Method:** `POST`
**Full URL:** `https://api.perplexity.ai/chat/completions`

---

## 2. HTTP Headers

### Required Headers

```
Authorization: Bearer YOUR_API_KEY
Content-Type: application/json
```

### Optional Headers

```
Accept: application/json
HTTP-Referer: your-app-url.com        (recommended for identification)
X-Title: Your Application Name         (recommended for identification)
```

**Authentication:** Uses HTTPBearer authentication. The API key should be prefixed with "Bearer ".

---

## 3. Request Body Structure

### Minimal Request (Required Fields Only)

```json
{
  "model": "sonar",
  "messages": [
    {
      "role": "user",
      "content": "Your question here"
    }
  ],
  "stream": true
}
```

### Complete Request Body Schema

#### Core Parameters

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `model` | string | ✓ | - | Model identifier (e.g., "sonar", "sonar-pro") |
| `messages` | array | ✓ | - | Array of ChatMessage objects |
| `stream` | boolean | ✗ | false | Enable streaming response |
| `max_tokens` | integer | ✗ | - | Max tokens to generate (0-128,000) |
| `temperature` | number | ✗ | 0.7 | Sampling temperature (0.0-2.0) |
| `n` | integer | ✗ | 1 | Number of completions to generate (1-10) |
| `stop` | string/array | ✗ | - | Stop sequence(s) |
| `response_format` | object | ✗ | - | Format: text, json_schema, or regex |

#### Message Object Structure

```json
{
  "role": "system" | "user" | "assistant",
  "content": "message text"
}
```

#### Search Parameters (Optional)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `disable_search` | boolean | false | Disable web search |
| `search_mode` | string | - | "web", "academic", or "sec" |
| `num_search_results` | integer | 10 | Number of search results |
| `search_domain_filter` | array | - | List of domains to filter |
| `search_recency_filter` | string | - | "hour", "day", "week", "month", "year" |

#### Tool/Function Calling (Optional)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `tools` | array | - | Array of ToolSpec objects |
| `tool_choice` | string | "auto" | "none", "auto", or "required" |
| `parallel_tool_calls` | boolean | - | Enable parallel tool execution |

---

## 4. Streaming Parameter

### Enabling Streaming

Set `"stream": true` in the request body JSON.

```json
{
  "model": "sonar",
  "messages": [...],
  "stream": true
}
```

### Behavior

- **When `stream: false` (default):** Response is returned as a single JSON object
- **When `stream: true`:** Response is streamed using Server-Sent Events (SSE) protocol

### Response Object Types

When streaming is enabled, each response chunk includes a `type` field:

- `"message"` - Content chunk
- `"info"` - Metadata information
- `"end_of_stream"` - Stream completion signal

---

## 5. Server-Sent Events (SSE) Format

### SSE Protocol

Responses use the Server-Sent Events (SSE) protocol with `Content-Type: text/event-stream`.

### Event Structure

Each SSE event follows this format:

```
data: {"id":"...","choices":[{"delta":{"content":"..."}}],...}

data: {"id":"...","choices":[{"delta":{"content":"..."}}],...}

data: [DONE]

```

- Events are separated by double newlines (`\n\n`)
- Each line starts with `data: `
- Stream terminates with `data: [DONE]`

### Response Chunk Structure

```json
{
  "id": "response-id",
  "model": "sonar",
  "object": "chat.completion.chunk",
  "created": 1234567890,
  "choices": [
    {
      "index": 0,
      "delta": {
        "role": "assistant",
        "content": "incremental text"
      },
      "finish_reason": null
    }
  ]
}
```

### Delta Content

**Important Note:** Perplexity's SSE implementation differs from OpenAI's:

- **OpenAI:** Each `delta.content` contains only the new tokens since the last chunk
- **Perplexity:** The `delta.content` may contain the complete content up to that point (accumulated)

### Termination Signal

The stream ends when receiving:

```
data: [DONE]
```

Clients should:
1. Check if `event.data === '[DONE]'`
2. Close the SSE connection
3. Stop processing events

---

## 6. Example cURL Requests

### Non-Streaming Request

```bash
curl -X POST https://api.perplexity.ai/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "sonar",
    "messages": [
      {
        "role": "system",
        "content": "Be precise and concise."
      },
      {
        "role": "user",
        "content": "What is the capital of France?"
      }
    ],
    "max_tokens": 1024,
    "temperature": 0.7,
    "stream": false
  }'
```

### Streaming Request

```bash
curl -X POST https://api.perplexity.ai/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -N \
  -d '{
    "model": "sonar",
    "messages": [
      {
        "role": "user",
        "content": "Tell me about artificial intelligence"
      }
    ],
    "stream": true,
    "max_tokens": 1024
  }'
```

**cURL Flags:**
- `-N` / `--no-buffer`: Disable output buffering for streaming

---

## 7. Go Implementation Guide

### Request Structure

```go
type ChatCompletionRequest struct {
    Model       string    `json:"model"`
    Messages    []Message `json:"messages"`
    Stream      bool      `json:"stream"`
    MaxTokens   int       `json:"max_tokens,omitempty"`
    Temperature float64   `json:"temperature,omitempty"`
}

type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}
```

### Making the Request

```go
import (
    "bytes"
    "encoding/json"
    "net/http"
)

func makeStreamingRequest(apiKey string) error {
    url := "https://api.perplexity.ai/chat/completions"

    payload := ChatCompletionRequest{
        Model: "sonar",
        Messages: []Message{
            {Role: "user", Content: "Hello"},
        },
        Stream: true,
        MaxTokens: 1024,
    }

    body, _ := json.Marshal(payload)

    req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
    req.Header.Set("Authorization", "Bearer "+apiKey)
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "text/event-stream")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    // Process SSE stream
    return processSSEStream(resp.Body)
}
```

### SSE Stream Processing

```go
import (
    "bufio"
    "io"
    "strings"
)

func processSSEStream(body io.Reader) error {
    scanner := bufio.NewScanner(body)

    for scanner.Scan() {
        line := scanner.Text()

        // SSE lines start with "data: "
        if strings.HasPrefix(line, "data: ") {
            data := strings.TrimPrefix(line, "data: ")

            // Check for stream termination
            if data == "[DONE]" {
                break
            }

            // Parse JSON chunk
            var chunk ChatCompletionChunk
            if err := json.Unmarshal([]byte(data), &chunk); err != nil {
                continue
            }

            // Extract content
            if len(chunk.Choices) > 0 {
                content := chunk.Choices[0].Delta.Content
                // Process content here
                fmt.Print(content)
            }
        }
    }

    return scanner.Err()
}

type ChatCompletionChunk struct {
    ID      string   `json:"id"`
    Choices []Choice `json:"choices"`
}

type Choice struct {
    Delta        Delta  `json:"delta"`
    FinishReason string `json:"finish_reason"`
}

type Delta struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}
```

---

## 8. Available Models

Common model identifiers:

- `sonar` - Standard model
- `sonar-pro` - Advanced model with better reasoning
- `llama-3.1-sonar-large-128k-chat` - Large context window model
- `sonar-reasoning-pro` - Specialized reasoning model

Check the [official documentation](https://docs.perplexity.ai/) for the complete and up-to-date model list.

---

## 9. Error Responses and Handling



### Standard Error Format



The API returns errors in the following JSON format:



```json

{

  "error": {

    "message": "Error description",

    "type": "error_type", // e.g., "invalid_request_error"

    "code": "error_code"  // e.g., "rate_limit_exceeded"

  }

}

```



### HTTP Status Codes & Actions



| Code | Meaning | Action |

|------|---------|--------|

| 200 | Success | Process response. |

| 400 | Bad Request | **Do not retry.** Fix parameters (model, max_tokens). |

| 401 | Unauthorized | **Do not retry.** Check API key/credits. |

| 403 | Forbidden | **Do not retry.** Access denied. |

| 429 | Rate Limit Exceeded | **Retry** with backoff. Check headers. |

| 500 | Internal Server Error | **Retry** with exponential backoff. |

| 502 | Bad Gateway | **Retry** with exponential backoff. |

| 503 | Service Unavailable | **Retry** with exponential backoff. |



### Streaming Errors



Errors can occur *during* a stream. The connection may close, or a final event might indicate an error.

**Crucial:** Check `finish_reason` in the final chunk. If it is `error` (or non-null/non-stop), handle as a failure.



---



## 10. Rate Limiting



### Limits (Approximate)

- **Sonar Online:** ~50 requests/minute (varies by tier).

- **Burst:** Small burst capacity allowed (leaky bucket).



### Headers

Perplexity uses standard rate limit headers (names may vary slightly by proxy, check case-insensitively):



- `x-ratelimit-limit` (or `ratelimit-limit`): Requests allowed per period.

- `x-ratelimit-remaining` (or `ratelimit-remaining`): Requests left.

- `x-ratelimit-reset` (or `ratelimit-reset`): Seconds until reset.

- `retry-after`: (Optional) Seconds to wait before retrying.



### Strategy

1.  **Read Headers:** Always parse `x-ratelimit-reset` or `retry-after` on 429 responses.

2.  **Backoff:** If headers are missing, use exponential backoff starting at 1s up to 32s.

3.  **Jitter:** Add random jitter (±10%) to wait times to prevent thundering herds.



---



## 11. Timeout Recommendations



| Timeout Type | Recommended Value | Reason |

|--------------|-------------------|--------|

| **Connect** | 10 seconds | Fast fail if API is unreachable. |

| **TLS Handshake** | 10 seconds | Standard security negotiation time. |

| **Response Header** | 10 seconds | Time to first byte. |

| **Idle (Stream)** | 30-60 seconds | `sonar-deep-research` can pause for long periods during "thinking". |

| **Total Request** | 120+ seconds | Complex queries/research models take time. |



**Go Implementation Note:**

Use `net/http.Client` with a custom `Transport` for granular control.

For `sonar-deep-research`, the server-side timeout is ~60s. Ensure your client waits at least this long.



---



## 12. Best Practices



### For Streaming Implementations



1.  **Connection Management**

    - Use `Connection: keep-alive`.

    - **Defensive Parsing:** Perplexity may send *accumulated* text in `delta.content` or standard deltas. Implement logic to detect if content is appended or replaced, or just treat as appended (standard OpenAI behavior) but verify.

    - **Markdown Cleaning:** Occasionally, JSON chunks might be wrapped in markdown code blocks if the model "leaks" raw output. Strip leading/trailing non-JSON characters if parsing fails.



2.  **Retry Logic (Go Example)**



    ```go

    func shouldRetry(err error, resp *http.Response) bool {

        if err != nil {

            return true // Network/Transport errors

        }

        if resp.StatusCode == 429 || resp.StatusCode >= 500 {

            return true

        }

        return false

    }

    ```



3.  **Stream Termination**

    - **Primary:** Stop on `data: [DONE]`.

    - **Secondary:** Stop on `finish_reason != null`.

    - **Safety:** Hard timeout if no data received for >60s.



---



## Sources

This documentation was compiled from the following sources:

- [Perplexity API Reference - Chat Completions](https://docs.perplexity.ai/api-reference/chat-completions-post)
- [Perplexity Streaming Responses Guide](https://docs.perplexity.ai/guides/streaming-responses)
- [Perplexity API Documentation](https://docs.perplexity.ai/)
- [Perplexity Python SDK](https://github.com/perplexityai/perplexity-py)
- [Perplexity Go Client - emmaly/perplexity](https://pkg.go.dev/github.com/emmaly/perplexity)
- [Perplexity Go Client - sgaunet/perplexity-go](https://github.com/sgaunet/perplexity-go)
- [Perplexity API Ultimate Guide - Zuplo](https://zuplo.com/learning-center/perplexity-api)
- [Mastering the Perplexity API - Neel Builds](https://blog.neelbuilds.com/comprehensive-guide-on-using-the-perplexity-api)
- [Community Discussion - SSE Implementation](https://community.perplexity.ai/t/bug-chat-completions-endpoint-implements-sse-via-post-instead-of-get/93)

---

**Last Updated:** 2026-02-04
**API Version:** Current (as of documentation date)
