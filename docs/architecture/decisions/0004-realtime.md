# ADR 0004: Durable Messaging With Ephemeral Fan-out

Status: accepted

Messages are committed to PostgreSQL before any realtime signal is published. Redis Pub/Sub distributes new-message identifiers between instances. WebSocket delivery is best-effort; reconnect catch-up uses the PostgreSQL identity sequence as a cursor. Duplicate client submissions are rejected by a unique `(sender_id, client_message_id)` index.
