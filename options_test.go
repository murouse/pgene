package pgene_test

import (
	"testing"

	"github.com/murouse/pgene"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithCreds(t *testing.T) {
	tests := []struct {
		name  string
		creds pgene.Creds
	}{
		{name: "plain", creds: pgene.Creds{User: "user", Pass: "pass", Host: "localhost", Port: 5432, DB: "db", SSLMode: "disable"}},
		{name: "special chars", creds: pgene.Creds{User: "us@er", Pass: "p@ss/w#rd?:%", Host: "localhost", Port: 5432, DB: "my db", SSLMode: "disable"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := pgene.New(t.Context(),
				pgene.WithCreds(&tt.creds),
				pgene.WithConstructorConfig(&pgene.ConstructorConfig{Ping: false}),
			)
			require.NoError(t, err)
			t.Cleanup(func() { _ = c.Close() })

			cfg := c.Pool().Config().ConnConfig
			got := pgene.Creds{User: cfg.User, Pass: cfg.Password, Host: cfg.Host, Port: int(cfg.Port), DB: cfg.Database, SSLMode: tt.creds.SSLMode}
			assert.Equal(t, tt.creds, got)
		})
	}
}
