# Research Findings

This document consolidates technical research conducted for the agent swarm subsystems.

## Go HCL Configuration Libraries

Research Date: 2026-02-04

### Key Findings

- **Library Choice**: Use `github.com/hashicorp/hcl/v2` (version 2 has an incompatible API with v1)
- **Simple Decoding**: `hclsimple` package for loading HCL directly into Go structs in one step
- **Schema Definition**: `gohcl` package provides struct field tag scheme for configuration schema
- **Validation**: `github.com/hashicorp/hcl-lang/validator` provides validators including:
  - BlockLabelsLength
  - DeprecatedAttribute/DeprecatedBlock
  - MaxBlocks/MinBlocks
  - MissingRequiredAttribute
  - UnexpectedAttribute/UnexpectedBlock

### Best Practices

- Write readable code over concise code
- Always validate configurations before use
- Incorrect configurations can cause disruptions
- HCL verifies structure and returns high-level objects for further processing

### Sources

- [Mastering HashiCorp Configuration Language](https://medium.com/@williamwarley/mastering-hashicorp-configuration-language-a-practical-guide-to-hcl-for-infrastructure-automation-cd0adb0d46c0)
- [GitHub - hashicorp/hcl](https://github.com/hashicorp/hcl)
- [gohcl package documentation](https://pkg.go.dev/github.com/hashicorp/hcl/v2/gohcl)
- [HashiCorp Configuration Language Tutorial](https://spacelift.io/blog/hcl-hashicorp-configuration-language)
- [hcl package documentation](https://pkg.go.dev/github.com/hashicorp/hcl/v2)
- [validator package documentation](https://pkg.go.dev/github.com/hashicorp/hcl-lang/validator)

## JetBrains Ring UI React Integration

Research Date: 2026-02-04

### Key Findings

- **Installation**: `npm install @jetbrains/ring-ui --save-exact`
- **Component Count**: 50+ React controls (links, buttons, Date Picker, Data List)
- **Build Integration**: Import ring-ui components where needed or create an entry point
- **Tooling**: Webpack configuration required; CodeSandbox example available based on create-react-app

### Resources

- Official GitHub: https://github.com/JetBrains/ring-ui
- Storybook Documentation: https://jetbrains.github.io/ring-ui/
- NPM Package: https://www.npmjs.com/package/@jetbrains/ring-ui

### Sources

- [GitHub - JetBrains/ring-ui](https://github.com/JetBrains/ring-ui)
- [Ring UI - Storybook](https://jetbrains.github.io/ring-ui/)
- [Ring UI NPM Package](https://www.npmjs.com/package/@jetbrains/ring-ui)

## Server-Sent Events vs WebSocket Comparison

Research Date: 2026-02-04

### Key Differences

**Communication Direction:**
- SSE: Server-to-client only (one-way)
- WebSocket: Full bidirectional communication

**Reconnection Handling:**
- SSE: EventSource automatically remembers last message ID and sends Last-Event-ID header on reconnect
- WebSocket: Requires custom reconnection logic

**Performance:**
- Tests show similar performance between SSE and WS
- WebSockets slightly more performance-efficient in simulated scenarios

### When to Choose SSE

- Only need server push
- Want automatic reconnection
- HTTP/2 compatibility important
- Simpler setup than WebSocket

**Ideal Use Cases:**
- Stock prices, analytics metrics, IoT sensor readings
- Dashboards, pricing displays, inventory monitors
- Alerting systems
- Any server-to-client streaming scenario

### When to Choose WebSocket

- Need client-to-server communication
- Chat, gaming, collaborative editing
- Trading systems, CRM automation
- Multiplayer platforms
- Need to transmit binary data

### Practical Recommendations

- SSE is simpler on a conceptual level with push model
- Automatic reconnection and stream continuity reduce implementation complexity
- Focus on functional requirements: bidirectional → WebSocket, server-push-only → SSE
- UTF-8 data from server to client → SSE
- Binary data or bidirectional → WebSocket

### Sources

- [WebSockets vs Server-Sent Events (SSE)](https://ably.com/blog/websockets-vs-sse)
- [WebSockets vs Server-Sent Events - WebSocket.org](https://websocket.org/comparisons/sse/)
- [WebSockets vs SSE vs Long-Polling vs WebRTC vs WebTransport](https://rxdb.info/articles/websockets-sse-polling-webrtc-webtransport.html)
- [Socket.io vs WebSockets vs Server-Sent Events 2026](https://www.index.dev/skill-vs-skill/socketio-vs-websockets-vs-server-sent-events)
- [WebSocket vs. Server-sent Events: Performance Comparison](https://www.timeplus.com/post/websocket-vs-sse)
- [SSE vs WebSockets: Comparing Real-Time Protocols](https://softwaremill.com/sse-vs-websockets-comparing-real-time-communication-protocols/)
- [Server-Sent Events vs WebSockets – freeCodeCamp](https://www.freecodecamp.org/news/server-sent-events-vs-websockets/)

## Message Bus Patterns & Event Sourcing

Research Date: 2026-02-04

### Event Sourcing & Append-Only Patterns

- Events are immutable and stored using append-only operations
- Only inserts/appends (Creates) are allowed in event sourcing
- Event store used with Event Sourcing + CQRS Architecture
- Write and read sides can scale independently

### Threading Considerations

- **Critical**: Consistency of events in the event store is vital
- Order of events affecting a specific entity must be preserved
- Be wary of multiple producers and multiple consumers
- Multi-threaded applications require careful consistency management

### Go Implementation

- Utilize Go channels and concurrency mechanisms for pub-sub pattern
- Event bus makes events asynchronous and untied from other components
- Multiple Go libraries available on GitHub for message bus implementations

### Sources

- [Microservices Pattern: Event sourcing](https://microservices.io/patterns/data/event-sourcing.html)
- [CQRS and Event Sourcing in Go](https://medium.com/@stani.petrosyan/how-to-implement-cqrs-and-event-sourcing-pattern-in-go-fd47dc0afd80)
- [Event Sourcing pattern - Azure](https://learn.microsoft.com/en-us/azure/architecture/patterns/event-sourcing)
- [Event Sourcing Microservices Pattern](https://akfpartners.com/growth-blog/event-sourcing-microservices-pattern)
- [Go Event-Driven Training](https://threedots.tech/event-driven/)
- [message-bus GitHub Topics](https://github.com/topics/message-bus?l=go)
- [Message Broker Patterns](https://blog.bytebytego.com/p/must-know-message-broker-patterns)

## Go Process Orchestration & Signal Handling

Research Date: 2026-02-04

### Key Challenges

- Child processes may continue running after cmd.Process.Kill()
- Programs implement custom behavior for SIGTERM (cleanup, persist state, close connections)
- Need proper handling of signals like SIGINT or SIGTERM

### Signal Differences

- **SIGKILL**: Cannot be ignored, terminates immediately, cannot be captured
- **SIGTERM**: Can be captured for graceful shutdown, allows cleanup routines

### Best Practices

1. Attempt graceful termination with SIGTERM first
2. Use process groups to manage process trees
3. Find all child PIDs recursively
4. Determine unique process groups
5. Send signal to each unique process group
6. Use negative PID to target entire process group: `kill -<signal> -<pgid>`

### Go Process Management

- **goprocess package**: Introduces Process abstraction for orchestration
  - Similar to WaitGroup and context.Context
  - Ensures safe exactly-once and well-ordered teardown
  - Can bind Process trees to syscall.SIGTERM, SIGKILL

### Sources

- [Managing Go Processes](https://medium.com/@keployio/managing-go-processes-4a488fccba74)
- [SIGKILL vs SIGTERM: Master Process Termination](https://www.suse.com/c/observability-sigkill-vs-sigterm-a-developers-guide-to-process-termination/)
- [goprocess package](https://pkg.go.dev/github.com/jbenet/goprocess)
- [Managing Linux Processes in Go](https://mezhenskyi.dev/posts/go-linux-processes/)
- [Terminating Processes in Go](https://bigkevmcd.github.io/go/pgrp/context/2019/02/19/terminating-processes-in-go.html)
- [signal package - os/signal](https://pkg.go.dev/os/signal)

## Implementation Recommendations

### For run-agent Configuration (HCL)

1. Use `github.com/hashicorp/hcl/v2` with `hclsimple` for straightforward decoding
2. Define schema using `gohcl` struct tags
3. Implement validation with `hcl-lang/validator`
4. Provide clear, explanatory error messages
5. Generate default config with comments using template

### For Monitoring UI (React + Ring UI)

1. Install Ring UI with exact version: `npm install @jetbrains/ring-ui --save-exact`
2. Use webpack for build configuration
3. Leverage official Storybook for component reference
4. Consider using Context + hooks for state management (no Redux/Zustand in MVP)

### For Message Bus Streaming (SSE vs WebSocket)

1. **Choose SSE** for run-agent serve:
   - Server-to-client only (monitoring UI reads from backend)
   - Automatic reconnection with Last-Event-ID
   - Simpler implementation
   - HTTP/2 compatible
   - 2s polling fallback for browsers without SSE support

2. **Consider WebSocket** if:
   - Future requirements need client-to-server streaming
   - Binary data transmission needed

### For Message Bus Implementation

1. Use append-only file writes for immutability
2. Ensure event ordering per entity
3. Handle multi-threaded access with atomic file operations (temp + rename)
4. Consider Go channels for internal event distribution
5. Implement sequential message ID generation (timestamp + PID + sequence)

### For Process Management in run-agent

1. Use process groups (PGID) for managing agent hierarchies
2. Graceful shutdown: SIGTERM → 30s wait → SIGKILL
3. Consider `goprocess` package for complex orchestration needs
4. Find and terminate all child PIDs recursively
5. Send signals to process groups, not individual PIDs
6. Log START/STOP/CRASH events to message bus for auditability
