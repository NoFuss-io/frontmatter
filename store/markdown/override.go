package markdown

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/nofuss-io/frontmatter/store"
)

// Override pins the comment syntax for every file, instead of choosing it per
// file by extension or shebang. Lang names a row of the language table, or
// one of its extensions ("python", "py", ".py"). LineComment and
// BlockOpen/BlockClose describe a custom syntax instead; BlockClose defaults
// to BlockOpen. Lang cannot be combined with the custom markers.
type Override struct {
	Lang        string
	LineComment string
	BlockOpen   string
	BlockClose  string
}

func (o Override) isZero() bool { return o == Override{} }

// language resolves the override to a language row.
func (o Override) language() (language, error) {
	custom := o.LineComment != "" || o.BlockOpen != "" || o.BlockClose != ""
	switch {
	case o.Lang != "" && custom:
		return language{}, errors.New("--lang cannot be combined with --line-comment, --block-open or --block-close")
	case o.Lang != "":
		if l, ok := byName[strings.ToLower(o.Lang)]; ok {
			return l, nil
		}
		if l, ok := byExt["."+strings.TrimPrefix(strings.ToLower(o.Lang), ".")]; ok {
			return l, nil
		}
		return language{}, fmt.Errorf("unknown language %q: must be an extension or one of %s",
			o.Lang, strings.Join(Languages(), ", "))
	case o.BlockClose != "" && o.BlockOpen == "":
		return language{}, errors.New("--block-close requires --block-open")
	}
	l := language{name: "custom"}
	if o.LineComment != "" {
		l.line = []string{o.LineComment}
	}
	if o.BlockOpen != "" {
		closer := o.BlockClose
		if closer == "" {
			closer = o.BlockOpen
		}
		l.block = []blockComment{{o.BlockOpen, closer}}
	}
	return l, nil
}

// Languages returns the names accepted by Override.Lang, sorted.
func Languages() []string {
	names := make([]string, 0, len(languages))
	for _, l := range languages {
		names = append(names, l.name)
	}
	sort.Strings(names)
	return names
}

// NewWithOverride returns a store like New, except that every file is parsed
// with the syntax o describes. Since extensions no longer select the syntax,
// an extensionless glob such as `*` matches every file. A zero Override
// returns New().
func NewWithOverride(o Override) (store.Store, error) {
	if o.isZero() {
		return New(), nil
	}
	l, err := o.language()
	if err != nil {
		return nil, err
	}
	f := &Format{override: &l}
	return fileStore{store.FileStore{Fmt: f}, f}, nil
}
