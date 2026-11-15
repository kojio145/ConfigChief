package schemas

import (
	"embed"
	"sync"

	"github.com/kojio145/configchief/internal/schema"
)

// One file per table. The number at the head of a file name gives the order
// of the sheets in the book.
//
//go:embed *.yaml
var files embed.FS

var (
	once sync.Once
	set  *schema.Set
	err  error
)

// Load gives the bundled table definitions. They are read once.
func Load() (*schema.Set, error) {
	once.Do(func() { set, err = schema.Load(files) })
	return set, err
}
