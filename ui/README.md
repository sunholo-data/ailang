# AILANG UI Collaboration Hub

React-based frontend for the AILANG Collaboration Hub, enabling real-time human-AI communication.

## Components

### Message Center

**Location:** `src/components/MessageCenter/`

The Message Center provides real-time messaging between humans and AILANG instances.

#### ThreadList (`ThreadList.tsx`)
- Displays all conversation threads
- Shows unread message counts
- Thread status indicators (active, paused, resolved, archived)
- Click to select thread

#### ConversationView (`ConversationView.tsx`)
- Displays messages in chronological order
- Auto-scrolls to latest message
- Send messages with different kinds (directive, question, status, result)
- Message formatting with icons and timestamps
- Delivery state indicators

#### MessageCenter (`MessageCenter.tsx`)
- Main container component
- Manages WebSocket connection
- Handles thread selection and message routing
- Real-time message updates

### Approval Queue

**Location:** `src/components/ApprovalQueue/`

The Approval Queue allows humans to review and approve/reject effect-gated actions requested by AILANG instances.

#### ApprovalQueue (`ApprovalQueue.tsx`)
- Lists pending approval requests
- Expandable cards with full effect details
- Shows capability type, paths, budget delta
- Impact level indicators (low, medium, high)
- Review notes textarea
- Approve/reject actions

## Hooks

### useWebSocket

**Location:** `src/hooks/useWebSocket.ts`

Custom React hook for WebSocket communication with cursor-based resumption.

**Features:**
- Automatic connection and reconnection
- Subscribe to threads with `from_seq` for missed messages
- Acknowledge messages to track progress
- Ping/pong heartbeat for connection health
- Event-based message handling
- Maintains subscription state across reconnects

**Usage:**
```typescript
const { isConnected, subscribe, acknowledge } = useWebSocket({
  url: 'ws://localhost:8080/ws',
  instanceId: 'my-instance',
  onMessage: (msg) => console.log('New message:', msg),
  onBatch: (batch) => console.log('Message batch:', batch),
  onError: (err) => console.error('Error:', err),
});

// Subscribe to a thread
subscribe('thread_123', 0); // from seq 0

// Acknowledge up to seq 5
acknowledge('thread_123', 5);
```

## Types

**Location:** `src/types/index.ts`

TypeScript type definitions for all data structures:
- `Thread` - Conversation threads
- `Message` - Individual messages
- `Approval` - Approval requests
- `EffectDelta` - Requested capabilities
- `WSEvent` - WebSocket event types
- Event-specific types: `SubscribeEvent`, `AckEvent`, `MessageEvent`, `BatchEvent`, etc.

## Architecture

```
┌─────────────────────────────────────────┐
│          MessageCenter                  │
│  ┌──────────────┐   ┌────────────────┐  │
│  │  ThreadList  │   │ConversationView│  │
│  └──────────────┘   └────────────────┘  │
└─────────────────────────────────────────┘
                 │
                 ▼
         useWebSocket Hook
                 │
                 ▼
       WebSocket Server (Go)
                 │
                 ▼
         SQLite Message Bus
```

### Data Flow

1. **Real-time Updates:**
   - WebSocket server sends `batch` events with new messages
   - `useWebSocket` hook receives and parses events
   - `MessageCenter` updates state and re-renders UI
   - Auto-scrolls to latest message

2. **Cursor-based Resumption:**
   - On subscribe, client sends `from_seq` (last acknowledged sequence)
   - Server replays all messages since `from_seq`
   - Client deduplicates by `message_seq`
   - At-least-once delivery guaranteed

3. **Message Sending:**
   - User types message in `ConversationView`
   - TODO: Send to REST API endpoint
   - Server creates message in SQLite
   - WebSocket broadcasts to all subscribers

4. **Approval Workflow:**
   - AILANG instance requests approval via API
   - Approval appears in `ApprovalQueue`
   - Human reviews effect delta and impact
   - Approve/reject via API
   - Server generates capability token (if approved)
   - Instance retrieves token and proceeds

## Styling

Currently using inline CSS-in-JS with `<style jsx>` for component-scoped styles.

**Features:**
- Responsive layout
- Clean, minimal design
- Status indicators with emojis
- Color-coded message kinds
- Impact-based approval coloring

**Future:** Consider migrating to CSS Modules or styled-components for better maintainability.

## TODO

- [ ] Implement REST API calls for message sending
- [ ] Add thread creation UI
- [ ] Implement approval approve/reject API calls
- [ ] Add loading states
- [ ] Error handling and retry logic
- [ ] Message attachments support
- [ ] Markdown rendering in messages
- [ ] Code syntax highlighting
- [ ] Search/filter threads
- [ ] Archive/resolve threads
- [ ] User authentication
- [ ] Responsive mobile layout

## Development

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build

# Run tests
npm test
```

## Integration

The UI connects to the AILANG WebSocket server and REST API:

- **WebSocket**: `ws://localhost:8080/ws`
- **REST API**: `http://localhost:8080/api/`

See `internal/websocket/` and future HTTP API implementation for backend details.
