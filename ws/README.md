### Example

You can test the server using a WebSocket client. For example, you can send the following JSON commands:
- **Subscribe to a channel:**
  ```json
  {"command": "subscribe", "identifier": "ChatChannel"}
  ```
- **Broadcast a message to the channel:**
  ```json
  {"command": "message", "identifier": "ChatChannel", "data": "Hello from Go!"}
  ```

### How It Works

1. **Database Setup (GORM):**  
   The `Message` model stores the channel, content, and creation time for each message.

2. **Hub:**  
   The `Hub` struct keeps track of active channels and client subscriptions. It listens for three types of events:
   - **Register:** When a client subscribes to a channel.
   - **Unregister:** When a client unsubscribes or disconnects.
   - **Broadcast:** When a message is sent on a channel. The hub persists the message using GORM and then sends it to every client subscribed to that channel.

3. **Client:**  
   A `Client` represents an individual websocket connection.  
   - The `readPump` listens for incoming messages, decodes the JSON command, and then either subscribes/unsubscribes the client or broadcasts a message.
   - The `writePump` sends messages (including periodic pings) back to the client.

4. **WebSocket Upgrade:**  
   The `/ws` HTTP endpoint upgrades HTTP connections to WebSocket connections using gorilla/websocket.

5. **Running the Server:**
   Finally, the server listens on port 9000 by default (configurable via `PORT`).
