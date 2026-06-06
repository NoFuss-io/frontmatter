// Package store defines the pluggable data-source backend for fm.
// Third-party store authors import only this package — never internal/.
package store

// Store is the pluggable data-source backend. The executor calls it to
// enumerate items, read their fields, write mutations back, and label them
// in output tables.
type Store interface {
	// Enumerate resolves FROM-clause pattern tokens into opaque item keys.
	// File stores receive glob strings; API stores receive domain-specific
	// identifiers (project keys, JQL, URLs, …). The store interprets them.
	Enumerate(patterns []string, opts EnumOptions) ([]string, error)

	// Read returns the field map for one item. The key is a value previously
	// returned by Enumerate.
	Read(key string) (map[string]any, error)

	// Write persists a mutated field map. Only called when a mutation
	// statement succeeded and DryRun is false.
	Write(key string, fields map[string]any) error

	// Label returns a human-readable display name for the key, used as the
	// row identifier in output tables (the "filename" column).
	Label(key string) string
}

// EnumOptions carries per-enumeration settings that the store may consult.
type EnumOptions struct {
	// IncludeHidden requests that items whose names begin with '.' not be
	// filtered out. File stores apply this to basenames; API stores ignore it.
	IncludeHidden bool
}

// Format handles reading and writing a single file. Glob expansion and hidden-
// file filtering are provided by FileStore, so Format authors only need to
// care about the file contents.
type Format interface {
	Read(path string) (map[string]any, error)
	Write(path string, fields map[string]any) error
}

// FileStore is a concrete Store for any file-based format. It implements
// Enumerate (glob expansion + hidden-file filter) once, and delegates
// Read/Write to the supplied Format. Label returns the path unchanged.
type FileStore struct {
	Fmt Format
}

func (fs FileStore) Enumerate(patterns []string, opts EnumOptions) ([]string, error) {
	// Implementation filled in Phase B.
	panic("FileStore.Enumerate: not yet implemented")
}

func (fs FileStore) Read(key string) (map[string]any, error) {
	return fs.Fmt.Read(key)
}

func (fs FileStore) Write(key string, fields map[string]any) error {
	return fs.Fmt.Write(key, fields)
}

func (fs FileStore) Label(key string) string {
	return key
}
