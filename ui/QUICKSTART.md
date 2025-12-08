# Quick Start Guide - AILANG UI

## Prerequisites

- Node.js 18+ installed
- The AILANG backend running (WebSocket server on port 8080)

## Installation

```bash
cd ui
npm install
```

## Running the UI

```bash
npm run dev
```

This will start the development server on **http://localhost:3000**

## What You'll See

The UI has two tabs:

### 💬 Messages Tab
- **Thread List** (left panel) - Shows all conversation threads
  - Mock threads are pre-loaded for demo
  - Unread count badges
  - Thread status indicators (🟢 active, 🟡 paused, ✅ resolved, 📁 archived)
- **Conversation View** (right panel) - Select a thread to view messages
  - Send messages with different kinds (directive, question, status, result)
  - Real-time message updates (when WebSocket is connected)
  - Auto-scrolling to latest messages

### 🔒 Approvals Tab
- **Approval Queue** - Review pending approval requests
  - 2 mock approvals pre-loaded for demo
  - Expandable cards showing effect details (capability type, paths, budget)
  - Impact indicators (🟢 low, 🟡 medium, 🔴 high)
  - Approve/Reject buttons with review notes

## Connection Status

Look for the connection indicator at the top:
- **● Connected** (green) - WebSocket connected to backend
- **○ Disconnected** (red) - WebSocket not connected

## Backend Setup (Optional)

To enable real-time features, you need the WebSocket server running:

```bash
# In the main ailang directory
go run cmd/ailang/main.go serve --ws-port 8080
```

**Note:** The WebSocket server implementation is not yet complete in the backend. The UI will work in demo mode with mock data for now.

## Features Working in Demo Mode

✅ Thread selection
✅ Message display with formatting
✅ Send messages (logged to console)
✅ Approval queue display
✅ Approve/reject actions (updates local state)
✅ Tab switching

## Features Requiring Backend

⏳ Real-time WebSocket message updates
⏳ Persistent message storage
⏳ Actual approval workflow with capability tokens
⏳ Thread creation
⏳ Message acknowledgements

## Development

```bash
# Run dev server
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview

# Type checking
npx tsc --noEmit
```

## Troubleshooting

**Port 3000 already in use:**
```bash
# Edit vite.config.ts and change the port:
server: {
  port: 3001, // or any other port
}
```

**WebSocket connection failed:**
- Make sure the backend is running on port 8080
- Check the console for connection errors
- The UI will still work in demo mode without the backend

**Styling issues:**
- The components use inline CSS-in-JS (`<style jsx>`)
- This requires the `styled-jsx` plugin, but we're using native React styles
- Some styles may not render correctly - this is expected in the MVP

## Next Steps

1. Install dependencies: `npm install`
2. Run the dev server: `npm run dev`
3. Open http://localhost:3000 in your browser
4. Explore the UI with mock data
5. (Optional) Start the backend WebSocket server for real-time features
