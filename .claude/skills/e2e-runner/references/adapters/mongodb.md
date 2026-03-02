# MongoDB Adapter

For testing MongoDB document operations.

## Configuration

```yaml
environments:
  local:
    adapters:
      mongodb:
        connectionString: "mongodb://user:password@host:port"
        database: "mydb"      # Database name
```

**Peer dependency:** `npm install mongodb`

## Action: `insertOne`

Insert single document.

```yaml
- adapter: mongodb
  action: insertOne
  collection: "users"
  document:
    email: "test@example.com"
    name: "Test User"
    createdAt: "{{$isoDate()}}"
  capture:
    inserted_id: "insertedId"
```

## Action: `insertMany`

Insert multiple documents.

```yaml
- adapter: mongodb
  action: insertMany
  collection: "items"
  documents:
    - name: "Item 1"
      price: 10
    - name: "Item 2"
      price: 20
```

## Action: `findOne`

Find single document.

```yaml
- adapter: mongodb
  action: findOne
  collection: "users"
  filter:
    email: "test@example.com"
  capture:
    user_id: "_id"
    user_name: "name"
  assert:
    - path: "name"
      equals: "Test User"
```

## Action: `find`

Find all matching documents.

```yaml
- adapter: mongodb
  action: find
  collection: "orders"
  filter:
    status: "pending"
  capture:
    pending_count: "length"
  assert:
    - path: "[0].status"
      equals: "pending"
```

## Action: `updateOne`

Update single document.

```yaml
- adapter: mongodb
  action: updateOne
  collection: "users"
  filter:
    _id: "{{captured.user_id}}"
  update:
    $set:
      name: "Updated Name"
      updatedAt: "{{$isoDate()}}"
```

## Action: `updateMany`

Update multiple documents.

```yaml
- adapter: mongodb
  action: updateMany
  collection: "orders"
  filter:
    status: "pending"
  update:
    $set:
      status: "processed"
```

## Action: `deleteOne`

Delete single document.

```yaml
- adapter: mongodb
  action: deleteOne
  collection: "users"
  filter:
    email: "test@example.com"
```

## Action: `deleteMany`

Delete multiple documents.

```yaml
- adapter: mongodb
  action: deleteMany
  collection: "test_data"
  filter:
    testRun: "{{test_run_id}}"
```

## Action: `count`

Count documents.

```yaml
- adapter: mongodb
  action: count
  collection: "users"
  filter:
    status: "active"
  assert:
    - path: "count"
      greaterThan: 0
```

## Action: `aggregate`

Run aggregation pipeline.

```yaml
- adapter: mongodb
  action: aggregate
  collection: "orders"
  pipeline:
    - $match:
        status: "completed"
    - $group:
        _id: "$customerId"
        total: { $sum: "$amount" }
  capture:
    totals: "result"
```

## MongoDB Assertions

```yaml
assert:
  - path: "name"                     # Dot notation path
    equals: "expected"
  - path: "tags"
    length: 3
  - path: "nested.field"
    exists: true
  - path: "status"
    matches: "^(active|pending)$"
  - path: "deletedAt"
    isNull: true
```
