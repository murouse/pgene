package link_test

import (
	"context"
	"errors"
	"testing"

	"github.com/murouse/pgene"
	"github.com/murouse/pgene/internal/testdb"
	"github.com/murouse/pgene/resource/link"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Стойки: 1, 2 (удалена). Теги: a, b, c, gone (удалён). Связи: 1 — b.
const schema = `
	DROP TABLE IF EXISTS link_rack_tags, link_racks, link_tags;
	CREATE TABLE link_racks (id bigint PRIMARY KEY, deleted_at timestamptz);
	CREATE TABLE link_tags (id text PRIMARY KEY, deleted_at timestamptz);
	CREATE TABLE link_rack_tags (
		rack_id bigint NOT NULL REFERENCES link_racks (id),
		tag_id  text NOT NULL REFERENCES link_tags (id)
	);
	CREATE UNIQUE INDEX link_rack_tags_uq ON link_rack_tags (rack_id, tag_id);
	INSERT INTO link_racks VALUES (1, NULL), (2, now());
	INSERT INTO link_tags VALUES ('a', NULL), ('b', NULL), ('c', NULL), ('gone', now());
	INSERT INTO link_rack_tags VALUES (1, 'b');`

var (
	errRackNotFound = errors.New("rack not found")
	errTagNotFound  = errors.New("tag not found")
)

// TagID — именованный строковый тип: такие ID раньше ломали каст в Replace.
type TagID string

func setup(t *testing.T) (*pgene.Client, *link.Resource[int64, TagID]) {
	c := testdb.Connect(t, schema)

	return c, link.New[int64, TagID](c, link.Config{
		Table:            "link_rack_tags",
		LeftColumn:       "rack_id",
		RightColumn:      "tag_id",
		LeftTable:        "link_racks",
		RightTable:       "link_tags",
		LeftNotFoundErr:  errRackNotFound,
		RightNotFoundErr: errTagNotFound,
	})
}

// rackTags возвращает теги стойки 1.
func rackTags(t *testing.T, c *pgene.Client) []TagID {
	t.Helper()

	var tags []TagID
	require.NoError(t, c.Select(t.Context(), pgene.Sql(`SELECT tag_id FROM link_rack_tags WHERE rack_id = 1`), &tags))
	return tags
}

func TestAdd(t *testing.T) {
	tests := []struct {
		name      string
		rack      int64
		tag       TagID
		wantAdded bool
		wantErr   error
	}{
		{name: "new", rack: 1, tag: "a", wantAdded: true},
		{name: "existing", rack: 1, tag: "b"},
		{name: "deleted rack", rack: 2, tag: "a", wantErr: errRackNotFound},
		{name: "missing rack", rack: 3, tag: "a", wantErr: errRackNotFound},
		{name: "deleted tag", rack: 1, tag: "gone", wantErr: errTagNotFound},
		{name: "missing tag", rack: 1, tag: "missing", wantErr: errTagNotFound},
	}

	_, links := setup(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			added, err := links.Add(t.Context(), tt.rack, tt.tag)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantAdded, added)
		})
	}
}

func TestRemove(t *testing.T) {
	tests := []struct {
		name        string
		tag         TagID
		wantRemoved bool
	}{
		{name: "existing", tag: "b", wantRemoved: true},
		{name: "missing", tag: "c"},
	}

	_, links := setup(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			removed, err := links.Remove(t.Context(), 1, tt.tag)
			require.NoError(t, err)
			assert.Equal(t, tt.wantRemoved, removed)
		})
	}
}

func TestRightsAndLefts(t *testing.T) {
	c, links := setup(t)
	_, err := c.Exec(t.Context(), pgene.Sql(`INSERT INTO link_rack_tags VALUES (1, 'a'), (1, 'gone'), (2, 'c')`))
	require.NoError(t, err)

	rights := []struct {
		name string
		rack int64
		want []TagID
	}{
		{name: "skips deleted tags", rack: 1, want: []TagID{"a", "b"}},
		{name: "no links", rack: 3},
	}
	for _, tt := range rights {
		t.Run("rights/"+tt.name, func(t *testing.T) {
			got, err := links.Rights(t.Context(), tt.rack)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	lefts := []struct {
		name string
		tag  TagID
		want []int64
	}{
		{name: "linked", tag: "b", want: []int64{1}},
		{name: "skips deleted racks", tag: "c"},
	}
	for _, tt := range lefts {
		t.Run("lefts/"+tt.name, func(t *testing.T) {
			got, err := links.Lefts(t.Context(), tt.tag)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestReplace(t *testing.T) {
	// шаги выполняются по порядку над одной и той же стойкой
	steps := []struct {
		name        string
		rack        int64
		tags        []TagID
		wantChanged bool
		wantErr     error
		wantTags    []TagID
	}{
		{name: "add with duplicates", rack: 1, tags: []TagID{"a", "b", "a"}, wantChanged: true, wantTags: []TagID{"a", "b"}},
		{name: "same set", rack: 1, tags: []TagID{"b", "a"}, wantTags: []TagID{"a", "b"}},
		{name: "swap", rack: 1, tags: []TagID{"b", "c"}, wantChanged: true, wantTags: []TagID{"b", "c"}},
		{name: "deleted tag", rack: 1, tags: []TagID{"a", "gone"}, wantErr: errTagNotFound, wantTags: []TagID{"b", "c"}},
		{name: "deleted rack", rack: 2, tags: []TagID{"a"}, wantErr: errRackNotFound, wantTags: []TagID{"b", "c"}},
		{name: "clear", rack: 1, wantChanged: true},
		{name: "clear again", rack: 1},
	}

	c, links := setup(t)
	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			changed, err := links.Replace(t.Context(), step.rack, step.tags)
			assert.ErrorIs(t, err, step.wantErr)
			assert.Equal(t, step.wantChanged, changed)
			assert.ElementsMatch(t, step.wantTags, rackTags(t, c))
		})
	}
}

func TestReplaceKeepsExistingRows(t *testing.T) {
	c, links := setup(t)

	ctid := func() string {
		var ctid string
		q := pgene.Sql(`SELECT ctid::text FROM link_rack_tags WHERE rack_id = 1 AND tag_id = 'b'`)
		require.NoError(t, c.Get(t.Context(), q, &ctid))
		return ctid
	}

	before := ctid()
	_, err := links.Replace(t.Context(), 1, []TagID{"b", "c"})
	require.NoError(t, err)

	assert.Equal(t, before, ctid())
}

func TestReplaceErrorKeepsOuterTx(t *testing.T) {
	c, links := setup(t)

	var replaceErr error
	err := pgene.InTx(t.Context(), c, func(ctx context.Context) error {
		_, replaceErr = links.Replace(ctx, 1, []TagID{"missing"})
		_, err := links.Add(ctx, 1, "a")
		return err
	})

	require.NoError(t, err)
	assert.ErrorIs(t, replaceErr, errTagNotFound)
	assert.ElementsMatch(t, []TagID{"a", "b"}, rackTags(t, c))
}
