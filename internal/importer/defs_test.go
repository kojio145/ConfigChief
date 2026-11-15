package importer

import (
	"testing"

	"github.com/kojio145/configchief/internal/schema"
	"github.com/kojio145/configchief/internal/schemas"
)

// defs gives the bundled table definitions.
func defs(t *testing.T) *schema.Set {
	t.Helper()
	sc, err := schemas.Load()
	if err != nil {
		t.Fatalf("schemas: %v", err)
	}
	return sc
}
