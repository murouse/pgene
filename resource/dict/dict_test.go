package dict_test

import (
	"context"
	"errors"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/murouse/pgene"
	"github.com/murouse/pgene/internal/testdb"
	"github.com/murouse/pgene/resource"
	"github.com/murouse/pgene/resource/dict"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Записи: 1 — "b", 2 — "a", 3 — "gone" (удалена); у всех owner_id = 1.
const schema = `
	DROP TABLE IF EXISTS dict_tags, dict_utags;
	CREATE TABLE dict_tags (
		id         bigserial PRIMARY KEY,
		name       text NOT NULL,
		owner_id   int NOT NULL DEFAULT 0,
		updated_at timestamptz NOT NULL DEFAULT now(),
		deleted_at timestamptz
	);
	CREATE UNIQUE INDEX dict_tags_name_uq ON dict_tags (name) WHERE deleted_at IS NULL;
	INSERT INTO dict_tags (name, owner_id, deleted_at) VALUES ('b', 1, NULL), ('a', 1, NULL), ('gone', 1, now());

	CREATE TABLE dict_utags (
		id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
		name       text NOT NULL,
		updated_at timestamptz NOT NULL DEFAULT now(),
		deleted_at timestamptz
	);`

var (
	errTagNotFound = errors.New("tag not found")
	errTagExists   = errors.New("tag exists")

	cols = []string{"id", "name"}
)

type tag struct {
	ID   int64
	Name string
}

func setup(t *testing.T) *dict.Resource[tag, int64] {
	return dict.New[tag, int64](testdb.Connect(t, schema), dict.Config{
		Table:       "dict_tags",
		NotFoundErr: errTagNotFound,
		Constraints: resource.Constraints{"dict_tags_name_uq": errTagExists},
	})
}

func owner(id int) dict.Option {
	return dict.Where(sq.Eq{"owner_id": id})
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]any
		wantErr error
	}{
		{name: "new", data: map[string]any{"name": "c", "owner_id": 1}},
		{name: "name of deleted", data: map[string]any{"name": "gone", "owner_id": 1}},
		{name: "duplicate", data: map[string]any{"name": "a", "owner_id": 1}, wantErr: errTagExists},
	}

	tags := setup(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tags.Create(t.Context(), tt.data)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestCreateQuotesColumns(t *testing.T) {
	tags := setup(t)

	_, err := tags.Create(t.Context(), map[string]any{`name) VALUES ('x'); DROP TABLE dict_tags; --`: "y"})
	require.Error(t, err)

	_, err = tags.List(t.Context(), cols)
	assert.NoError(t, err, "table must survive")
}

func TestGet(t *testing.T) {
	tests := []struct {
		name    string
		id      int64
		want    tag
		wantErr error
	}{
		{name: "active", id: 1, want: tag{ID: 1, Name: "b"}},
		{name: "deleted", id: 3, wantErr: errTagNotFound},
		{name: "missing", id: 99, wantErr: errTagNotFound},
	}

	tags := setup(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tags.Get(t.Context(), tt.id, cols)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetMany(t *testing.T) {
	tests := []struct {
		name string
		ids  []int64
		want []tag
	}{
		{name: "existing", ids: []int64{2, 1}, want: []tag{{1, "b"}, {2, "a"}}},
		{name: "skips deleted and missing", ids: []int64{3, 99, 2}, want: []tag{{2, "a"}}},
		{name: "no ids"},
	}

	tags := setup(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tags.GetMany(t.Context(), tt.ids, cols)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestList(t *testing.T) {
	byName := dict.Extend(func(b sq.SelectBuilder) sq.SelectBuilder { return b.OrderBy("name") })

	tests := []struct {
		name string
		opts []dict.Option
		want []tag
	}{
		{name: "by id", want: []tag{{1, "b"}, {2, "a"}}},
		{name: "by name", opts: []dict.Option{byName}, want: []tag{{2, "a"}, {1, "b"}}},
		{name: "filtered", opts: []dict.Option{dict.Where(sq.Eq{"name": "a"})}, want: []tag{{2, "a"}}},
	}

	tags := setup(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tags.List(t.Context(), cols, tt.opts...)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name        string
		id          int64
		data        map[string]any
		opts        []dict.Option
		wantChanged bool
		wantErr     error
	}{
		{name: "new value", id: 1, data: map[string]any{"name": "x"}, opts: []dict.Option{owner(1)}, wantChanged: true},
		{name: "same value", id: 2, data: map[string]any{"name": "a"}},
		{name: "no data", id: 2},
		{name: "duplicate", id: 1, data: map[string]any{"name": "a"}, wantErr: errTagExists},
		{name: "other owner", id: 1, data: map[string]any{"name": "y"}, opts: []dict.Option{owner(2)}, wantErr: errTagNotFound},
		{name: "deleted", id: 3, data: map[string]any{"name": "y"}, wantErr: errTagNotFound},
		{name: "missing", id: 99, data: map[string]any{"name": "y"}, wantErr: errTagNotFound},
	}

	tags := setup(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changed, err := tags.Update(t.Context(), tt.id, tt.data, tt.opts...)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantChanged, changed)
		})
	}
}

func TestUpsertInserts(t *testing.T) {
	tests := []struct {
		name string
		data map[string]any
	}{
		{name: "new", data: map[string]any{"name": "c", "owner_id": 1}},
		{name: "name of deleted", data: map[string]any{"name": "gone", "owner_id": 1}},
	}

	tags := setup(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, changed, err := tags.Upsert(t.Context(), []string{"name"}, tt.data)
			require.NoError(t, err)
			assert.True(t, changed)
			assert.NotContains(t, []int64{1, 2, 3}, id)
		})
	}
}

func TestUpsertUpdates(t *testing.T) {
	tests := []struct {
		name        string
		data        map[string]any
		wantID      int64
		wantChanged bool
	}{
		{name: "new values", data: map[string]any{"name": "b", "owner_id": 5}, wantID: 1, wantChanged: true},
		{name: "same values", data: map[string]any{"name": "a", "owner_id": 1}, wantID: 2},
		{name: "only key", data: map[string]any{"name": "a"}, wantID: 2},
	}

	tags := setup(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, changed, err := tags.Upsert(t.Context(), []string{"name"}, tt.data)
			require.NoError(t, err)
			assert.Equal(t, tt.wantID, id)
			assert.Equal(t, tt.wantChanged, changed)
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name    string
		id      int64
		opts    []dict.Option
		wantErr error
	}{
		{name: "other owner", id: 1, opts: []dict.Option{owner(2)}, wantErr: errTagNotFound},
		{name: "owner", id: 1, opts: []dict.Option{owner(1)}},
		{name: "deleted", id: 3, wantErr: errTagNotFound},
		{name: "missing", id: 99, wantErr: errTagNotFound},
	}

	tags := setup(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ErrorIs(t, tags.Delete(t.Context(), tt.id, tt.opts...), tt.wantErr)
		})
	}
}

func TestLock(t *testing.T) {
	tests := []struct {
		name    string
		id      int64
		opts    []dict.Option
		wantErr error
	}{
		{name: "active", id: 1},
		{name: "other owner", id: 1, opts: []dict.Option{owner(2)}, wantErr: errTagNotFound},
		{name: "deleted", id: 3, wantErr: errTagNotFound},
		{name: "missing", id: 99, wantErr: errTagNotFound},
	}

	tags := setup(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ErrorIs(t, tags.Lock(t.Context(), tt.id, tt.opts...), tt.wantErr)
		})
	}
}

// TestUUIDKeys: Update и Lock раньше не работали с не числовыми ID.
func TestUUIDKeys(t *testing.T) {
	c := testdb.Connect(t, schema)
	utags := dict.New[struct{ ID uuid.UUID }, uuid.UUID](c, dict.Config{Table: "dict_utags"})

	id, err := utags.Create(t.Context(), map[string]any{"name": "a"})
	require.NoError(t, err)

	changed, err := utags.Update(t.Context(), id, map[string]any{"name": "b"})
	require.NoError(t, err)
	assert.True(t, changed)

	err = pgene.InTx(t.Context(), c, func(ctx context.Context) error { return utags.Lock(ctx, id) })
	assert.NoError(t, err)
	assert.ErrorIs(t, utags.Lock(t.Context(), uuid.New()), resource.ErrNotFound)
}
