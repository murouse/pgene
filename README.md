# pgene

Лёгкий клиент PostgreSQL поверх `pgx`:

* настройка через опции
* транзакции через контекст
* `squirrel` и сырой SQL (goqu и jet — через адаптеры)
* удобные методы `Exec`, `Get`, `Select`
* готовые ресурсы: справочники с soft delete и таблицы-связки

## Установка

```bash
go get github.com/murouse/pgene
```

## Возможности

* ✅ Построен на `pgx/v5`
* ✅ Настройка пула через конфиг pgx
* ✅ Транзакции через контекст
* ✅ Вложенные транзакции: присоединение к внешней (по умолчанию) или savepoint
* ✅ Хуки после коммита
* ✅ Тесты в транзакции с откатом
* ✅ `squirrel`, сырой SQL, goqu и jet
* ✅ Минимальный и расширяемый API

## Быстрый старт

```go
ctx := context.Background()

client, err := pgene.New(ctx,
    pgene.WithConnString("postgres://user:pass@localhost:5432/db?sslmode=disable"),
)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

## Настройка

### Строка подключения

```go
pgene.WithConnString("postgres://user:pass@localhost:5432/db?sslmode=disable")
```

Параметры пула можно задать прямо в строке: `pool_max_conns=10`, `default_query_exec_mode=simple_protocol` и т.д.

### Реквизиты

```go
pgene.WithCreds(&pgene.Creds{
    User:    "user",
    Pass:    "pass",
    Host:    "localhost",
    Port:    5432,
    DB:      "db",
    SSLMode: "disable",
})
```

### Конфиг pgx

`WithPgxConfig` получает конфиг, разобранный из строки подключения. Поля, которые вы не трогаете,
сохраняют значения из строки или значения pgx по умолчанию:

```go
pgene.WithPgxConfig(func(cfg *pgxpool.Config) {
    cfg.MaxConns = 10
    cfg.ConnConfig.Tracer = otelpgx.NewTracer()
})
```

## Запросы

### Exec

```go
_, err := client.Exec(ctx,
    pgene.Sql("INSERT INTO users(name) VALUES($1)", "john"),
)
```

### Одна строка

```go
var user User

err := client.Get(ctx,
    pgene.Sql("SELECT * FROM users WHERE id = $1", 1),
    &user,
)
```

### Несколько строк

```go
var users []User

err := client.Select(ctx,
    pgene.Sql("SELECT * FROM users"),
    &users,
)
```

`Get` и `Select` сканируют через scany: строки — в структуры по именам колонок, одну колонку — в простой тип.
Структуру вроде `time.Time` scany считает строкой, поэтому одну колонку времени сканируйте
в поле структуры, `sql.NullTime` или `pgtype.Timestamptz`.

## squirrel

```go
query := pgene.Sq().
    Select("id", "name").
    From("users").
    Where(sq.Eq{"id": 1})

err := client.Get(ctx, query, &user)
```

## goqu и jet

Адаптеры лежат в отдельных пакетах, поэтому ядро от этих библиотек не зависит:

```go
import (
    "github.com/murouse/pgene/adapter/goquadapter"
    "github.com/murouse/pgene/adapter/jetadapter"
)

ds := goquadapter.Dialect().From("users").Where(goqu.C("id").Eq(1)).Prepared(true)
err := client.Get(ctx, goquadapter.Wrap(ds), &user)

err = client.Get(ctx, jetadapter.Wrap(SELECT(Users.AllColumns).FROM(Users)), &user)
```

Для goqu вызывайте `Prepared(true)`: иначе goqu подставляет значения прямо в SQL,
и каждое значение даёт новый запрос в кеше pgx.

## Транзакции

### Обычное использование

```go
err := pgene.InTx(ctx, client, func(ctx context.Context) error {
    if _, err := client.Exec(ctx,
        pgene.Sql("INSERT INTO users(name) VALUES($1)", "john"),
    ); err != nil {
        return err
    }

    return nil
})
```

Ошибка обработчика возвращается как есть, без обёрток. Для обработчика, возвращающего значение, есть `GetInTx`.

### Вложенные транзакции

Режим задаётся опцией `WithNestedMode`. По умолчанию (`NestedJoin`) вложенная транзакция
**присоединяется** к внешней:

```go
pgene.InTx(ctx, client, func(ctx context.Context) error {
    return pgene.InTx(ctx, client, func(ctx context.Context) error {
        // выполняется в той же транзакции
        return nil
    })
})
```

* новая транзакция не создаётся, вложенные commit и rollback ничего не делают
* ошибки передаются во внешний обработчик
* упавший запрос ломает всю внешнюю транзакцию: следующие запросы получают
  `current transaction is aborted`

С `NestedSavepoint` вложенный блок выполняется в `SAVEPOINT`. Его откат отменяет только этот блок,
и внешняя транзакция может продолжаться после ожидаемой ошибки:

```go
pgene.InTx(ctx, client, func(ctx context.Context) error {
    err := pgene.InTx(ctx, client, func(ctx context.Context) error {
        _, err := client.Exec(ctx, pgene.Sql("INSERT INTO users(name) VALUES($1)", "john"))
        return err
    }, pgene.WithNestedMode(pgene.NestedSavepoint))
    if _, ok := pgene.IsDuplicateKeyError(err); ok {
        // пользователь уже есть, внешняя транзакция по-прежнему работает
    }
    ...
})
```

Уровень изоляции, режим доступа и deferrable применяются только к транзакции верхнего уровня.

### Хуки после коммита

Код глубоко в стеке часто хочет что-то сделать, когда данные точно сохранены: сбросить кеш, отправить событие.
Но он не знает, в транзакции ли он и когда будет настоящий коммит. `AfterCommit` откладывает действие до него:

```go
func (s *Service) RenameTag(ctx context.Context, id int64, name string) error {
    if _, err := s.tags.Update(ctx, id, map[string]any{"name": name}); err != nil {
        return err
    }
    s.db.AfterCommit(ctx, func() { s.cache.Drop(id) })
    return nil
}
```

* в транзакции хук выполнится после коммита верхнего уровня, при откате — нет
* в `NestedJoin` хук уходит во внешнюю транзакцию; в `NestedSavepoint` откат savepoint выбрасывает хуки этого блока
* вне транзакции хук выполняется сразу

Доставка не гарантирована: если процесс упадёт между коммитом и хуком, хук потеряется.
Для событий, которые нельзя терять, нужен outbox.

### Advisory lock

Когда блокировать нечего (например, «не больше одного активного заказа на пользователя»), работу можно
сериализовать по ключу. Блокировка держится до конца транзакции:

```go
err := pgene.InTx(ctx, client, func(ctx context.Context) error {
    if err := client.AdvisoryLock(ctx, fmt.Sprintf("orders:user:%d", userID)); err != nil {
        return err
    }
    // проверить, что активного заказа нет, и создать его
    ...
})
```

## Транзакции в контексте

Транзакция хранится в `context.Context` отдельно для каждого клиента:
транзакцию одного клиента другой клиент не использует.

Все методы запросов сами находят транзакцию в контексте:

```go
ctx, _ := client.BeginTx(ctx)

client.Exec(ctx, ...)
client.Get(ctx, ...)
```

Транзакция — одно соединение: не выполняйте запросы с её контекстом из нескольких горутин одновременно.

## Продвинутое использование

### Ручное управление транзакцией

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

### Доступ к пулу

```go
pool := client.Pool()
```

Для возможностей `pgx`, которых нет в клиенте. Запросы через пул идут мимо транзакции из контекста.

## Ресурсы

Пакеты `resource/dict` и `resource/link` закрывают типовые таблицы:

* `dict.Resource` — CRUD над справочником с soft delete: `Create`, `Upsert`, `Get`, `GetMany`, `List`, `Update`, `Delete`, `Lock`
* `link.Resource` — таблица-связка «многие ко многим» между двумя справочниками: `Add`, `Remove`, `Replace`, `Rights`, `Lefts`

Каждый метод — один запрос без собственной транзакции (кроме `link.Replace`; `Update` и `Upsert` делают второй запрос, если ничего не изменилось).

### Схема

Справочнику нужны колонки `id`, `updated_at` и `deleted_at`; активными считаются записи с `deleted_at IS NULL`.
Связке нужен уникальный индекс или первичный ключ на паре колонок:

```sql
CREATE TABLE racks (
    id         bigserial PRIMARY KEY,
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE TABLE tags (
    id         bigserial PRIMARY KEY,
    name       text NOT NULL,
    owner_id   bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

-- частичный индекс: имя удалённого тега можно занять снова
CREATE UNIQUE INDEX tags_name_uq ON tags (name) WHERE deleted_at IS NULL;

CREATE TABLE rack_tags (
    rack_id bigint NOT NULL REFERENCES racks (id),
    tag_id  bigint NOT NULL REFERENCES tags (id),
    PRIMARY KEY (rack_id, tag_id)
);
```

### Справочник

```go
var (
    ErrTagNotFound = errors.New("tag not found")
    ErrTagExists   = errors.New("tag already exists")
)

type Tag struct {
    ID      int64
    Name    string
    OwnerID int64
}

tags := dict.New[Tag, int64](client, dict.Config{
    Table:       "tags",
    NotFoundErr: ErrTagNotFound, // по умолчанию resource.ErrNotFound
    // нарушение ограничения → доменная ошибка (unique, foreign key, check)
    Constraints: resource.Constraints{"tags_name_uq": ErrTagExists},
})

cols := []string{"id", "name", "owner_id"}

id, err := tags.Create(ctx, map[string]any{"name": "outdoor", "owner_id": userID}) // ErrTagExists при дубле
tag, err := tags.Get(ctx, id, cols)                                                 // ErrTagNotFound, если нет или удалён
many, err := tags.GetMany(ctx, []int64{1, 2, 3}, cols)                              // одним запросом; нет или удалён — пропускается

// вставка или обновление активной записи с тем же name (нужен уникальный индекс по name);
// changed = false, если запись уже была с теми же значениями
id, changed, err := tags.Upsert(ctx, []string{"name"}, map[string]any{"name": "outdoor", "owner_id": userID})

// Where — дополнительное условие для любого метода, Extend — доработка SELECT в Get и List.
// List всегда досортировывает по id, поэтому порядок записей стабилен.
list, err := tags.List(ctx, cols,
    dict.Where(sq.Eq{"owner_id": userID}),
    dict.Extend(func(b sq.SelectBuilder) sq.SelectBuilder { return b.OrderBy("name").Limit(50) }),
)

// true, только если значения действительно изменились; updated_at обновляется только тогда
changed, err := tags.Update(ctx, id, map[string]any{"name": "indoor"}, dict.Where(sq.Eq{"owner_id": userID}))

err = tags.Delete(ctx, id) // soft delete: deleted_at = now()
```

Свои колонки ресурс уточняет именем таблицы (`tags.id`, `tags.deleted_at`), поэтому в `Extend` можно делать JOIN.
Ваши колонки при JOIN тоже уточняйте:

```go
// теги стойки
list, err := tags.List(ctx, []string{"tags.id", "tags.name", "tags.owner_id"},
    dict.Extend(func(b sq.SelectBuilder) sq.SelectBuilder {
        return b.Join("rack_tags rt ON rt.tag_id = tags.id").Where(sq.Eq{"rt.rack_id": rackID})
    }),
)
```

Ключи `data` экранируются как имена колонок. Колонки типа `json` (не `jsonb`) в `Update` не поддерживаются:
в PostgreSQL для `json` нет оператора сравнения.

### Связка

```go
var ErrRackNotFound = errors.New("rack not found")

rackTags := link.New[int64, int64](client, link.Config{
    Table:            "rack_tags",
    LeftColumn:       "rack_id",
    RightColumn:      "tag_id",
    LeftTable:        "racks",
    RightTable:       "tags",
    LeftNotFoundErr:  ErrRackNotFound, // стойки нет или она удалена
    RightNotFoundErr: ErrTagNotFound,  // тега нет или он удалён
})

added, err := rackTags.Add(ctx, rackID, tagID)       // false, если связь уже была
removed, err := rackTags.Remove(ctx, rackID, tagID)  // false, если связи не было

// Заменяет набор тегов стойки. Существующие связи не пересоздаются;
// если какого-то тега нет, возвращает ErrTagNotFound и ничего не меняет.
changed, err := rackTags.Replace(ctx, rackID, []int64{1, 2, 3})

tagIDs, err := rackTags.Rights(ctx, rackID) // теги стойки, без удалённых
rackIDs, err := rackTags.Lefts(ctx, tagID)  // стойки с тегом, без удалённых
```

Типы ID сторон могут различаться: `link.New[int64, uuid.UUID]`.

### Проверки и побочные действия

Проверки, аудит, outbox и т.п. оборачивайте в `pgene.InTx` вместе с вызовом метода.
Чтобы проверка не устарела к моменту изменения, сначала заблокируйте запись через `Lock`:

```go
var ErrTagInUse = errors.New("tag is in use")

err := pgene.InTx(ctx, client, func(ctx context.Context) error {
    if err := tags.Lock(ctx, tagID); err != nil {
        return err
    }

    var used bool
    q := pgene.Sql(`SELECT EXISTS (SELECT 1 FROM rack_tags WHERE tag_id = $1)`, tagID)
    if err := client.Get(ctx, q, &used); err != nil {
        return err
    }
    if used {
        return ErrTagInUse
    }

    return tags.Delete(ctx, tagID)
})
```

`link.Add` и `link.Replace` берут на записи справочников блокировку `FOR SHARE`. Поэтому параллельный `Add` дождётся
конца этой транзакции и получит `ErrTagNotFound`, а не создаст связь с удалённым тегом.

## Тесты с откатом

`pgtest.Tx` запускает тест в транзакции-песочнице и откатывает её в конце: таблицы не нужно чистить,
тесты не мешают друг другу на одной базе.

```go
func TestRenameTag(t *testing.T) {
    ctx := pgtest.Tx(t, client)

    require.NoError(t, svc.RenameTag(ctx, tagID, "indoor"))
    // проверки видят изменения; после теста они откатятся
}
```

Код под тестом работает так же, как без песочницы: его транзакции коммитятся (как savepoint'ы) и запускают
хуки `AfterCommit`, а ошибка внутри них не ломает тестовую транзакцию. Ограничения:

* каждый такой тест держит одно соединение пула до конца
* код под тестом не должен выполнять запросы из нескольких горутин одновременно
* не подходит для тестов, где нужны изменения, видимые другим соединениям (конкурентность, LISTEN/NOTIFY)

Та же песочница доступна напрямую — например, для dry run: `client.BeginSandbox(ctx)`, затем `client.RollbackTx(ctx)`.

## Тесты pgene

Интеграционным тестам нужен PostgreSQL; без него они пропускаются. Проще всего запустить их на временном
PostgreSQL в Docker:

```bash
task test:integration
```

Или на своей базе:

```bash
PGENE_TEST_DSN="postgres://user:pass@localhost:5432/db?sslmode=disable" go test ./...
```

## Принципы

* Транзакции передаются через контекст
* `InTx` работает с интерфейсом `TxController`, это упрощает тестирование
* Запросы описываются интерфейсом `Sqlizer`
* Без тяжёлых абстракций и ORM

## Когда использовать

Библиотека подойдёт, если нужны:

* тонкий слой над `pgx`
* понятная работа с транзакциями
* гибкость сырого SQL вместе с билдером
