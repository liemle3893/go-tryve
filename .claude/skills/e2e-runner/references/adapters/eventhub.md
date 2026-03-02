# EventHub Adapter

For testing Azure EventHub messaging.

## Configuration

```yaml
environments:
  local:
    adapters:
      eventhub:
        connectionString: "Endpoint=sb://namespace.servicebus.windows.net/;SharedAccessKeyName=...;SharedAccessKey=...;EntityPath=hub-name"
        consumerGroup: "$Default"
```

For local development with EventHub emulator:

```yaml
eventhub:
  connectionString: "Endpoint=sb://localhost;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=SAS_KEY_VALUE;UseDevelopmentEmulator=true;EntityPath=events"
  consumerGroup: "$Default"
```

**Peer dependency:** `npm install @azure/event-hubs`

## Action: `publish`

Publish message(s) to EventHub.

```yaml
- adapter: eventhub
  action: publish
  topic: "events"
  message:
    type: "user.created"
    data:
      userId: "{{captured.user_id}}"
      email: "{{user_email}}"
  partitionKey: "user-partition"
```

Publish multiple messages:

```yaml
- adapter: eventhub
  action: publish
  topic: "events"
  messages:
    - type: "event.one"
      data: { id: 1 }
    - type: "event.two"
      data: { id: 2 }
```

## Action: `consume`

Consume N messages from EventHub.

```yaml
- adapter: eventhub
  action: consume
  topic: "events"
  count: 5                           # Number of messages to consume
  timeout: 10000                     # Timeout in ms
  capture:
    messages: "result"
```

## Action: `waitFor`

Wait for message matching filter.

```yaml
- adapter: eventhub
  action: waitFor
  topic: "events"
  timeout: 30000
  filter:
    type: "user.created"
    data.userId: "{{captured.user_id}}"
  capture:
    event_data: "data"
  assert:
    - path: "type"
      equals: "user.created"
```

## Action: `clear`

Clear received messages buffer.

```yaml
- adapter: eventhub
  action: clear
  topic: "events"
```

## EventHub Assertions

```yaml
assert:
  - path: "type"
    equals: "user.created"
  - path: "data.userId"
    exists: true
  - path: "data.email"
    contains: "@example.com"
```
