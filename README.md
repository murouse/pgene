# pgo

Lightweight PostgreSQL client built on top of `pgx`, providing:

* simple configuration via options
* context-based transaction management
* integration with `squirrel` and raw SQL
* convenient query helpers (`Exec`, `Get`, `Select`)

---

## Installation

```bash
go get github.com/murouse/pgene
```

---

## Features

* ✅ Built on `pgx/v5`
* ✅ Connection pool configuration
* ✅ Context-driven transactions
* ✅ Nested transaction support (no-op)
* ✅ Works with `squirrel` and raw SQL
* ✅ Minimal and extensible API

---

## Quick Start

```go
ctx := context.Background()

client, err := pgo.New(ctx,
    pgo.WithConnString("postgres://user:pass@localhost:5432/db?sslmode=disable"),
)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

---

## Configuration

### Using connection string

```go
pgo.WithConnString("postgres://user:pass@localhost:5432/db?sslmode=disable")
```

### Using credentials

```go
pgo.WithCreds(&pgo.Creds{
    User: "user",
    Pass: "pass",
    Host: "localhost",
    Port: 5432,
    DB:   "db",
    SSLMode: "disable",
})
```

### Custom pool config

```go
pgo.WithPoolConfig(&pgo.PoolConfig{
    MaxConns: 10,
})
```

---

## Queries

### Exec

```go
_, err := client.Exec(ctx,
    pgo.Sql("INSERT INTO users(name) VALUES($1)", "john"),
)
```

### Get single row

```go
var user User

err := client.Get(ctx, &user,
    pgo.Sql("SELECT * FROM users WHERE id = $1", 1),
)
```

### Select multiple rows

```go
var users []User

err := client.Select(ctx, &users,
    pgo.Sql("SELECT * FROM users"),
)
```

---

## Using with squirrel

```go
query := pgo.Sq().
    Select("id", "name").
    From("users").
    Where(sq.Eq{"id": 1})

err := client.Get(ctx, &user, query)
```

---

## Transactions

### Basic usage

```go
err := pgo.InTx(ctx, client, func(ctx context.Context) error {
    if _, err := client.Exec(ctx,
        pgo.Sql("INSERT INTO users(name) VALUES($1)", "john"),
    ); err != nil {
        return err
    }

    return nil
})
```

---

### Nested transactions

Nested transactions are **no-op**:

```go
pgo.InTx(ctx, client, func(ctx context.Context) error {
    return pgo.InTx(ctx, client, func(ctx context.Context) error {
        // executed in the same transaction
        return nil
    })
})
```

* no new transaction is created
* errors are propagated to the outer scope

---

## Context-based transaction model

Transactions are stored inside `context.Context`.

All query methods automatically detect and use transaction if present:

```go
ctx, _ := client.BeginTx(ctx)

client.Exec(ctx, ...)
client.Get(ctx, ...)
```

---

## Advanced usage

### Manual transaction control

```go
ctx, err := client.BeginTx(ctx)
if err != nil {
    return err
}

defer client.RollbackTx(ctx)

if err := client.CommitTx(ctx); err != nil {
    return err
}
```

---

### Access underlying pool

```go
pool := client.Pool()
```

Use this for advanced `pgx` features.

---

## Design notes

* Transactions are propagated via context
* `InTx` operates on `TxController` interface for testability
* Query abstraction is based on `Sqlizer` interface
* Library avoids heavy abstractions and ORM patterns

---

## When to use

Use this library if you want:

* a thin layer over `pgx`
* structured transaction handling
* flexibility of raw SQL + builder

