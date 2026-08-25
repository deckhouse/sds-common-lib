// Package releasenotes renders the release-notes pages of a storage module from the
// YAML the module keeps in git.
//
// Two sources are read, both optional:
//
//	.release-notes/vX.Y.Z.yaml + vX.Y.Z.ru.yaml   sectioned notes (current format)
//	CHANGELOG/vX.Y.Z.yml      + vX.Y.Z.ru.yml     flat Changes/Изменения list (legacy)
//
// The legacy source exists because the released history is not converted: a version cut
// before a module moved to the sectioned format keeps its file where it is and is rendered
// the way it always was. A version present in both sources is taken from the sectioned one.
//
// The package lives here rather than in each module because the renderer is the same for
// the whole fleet: a module keeps its notes and nothing else. The consumer is
// cmd/release-notes-docs, which the module's docs image runs during the build.
package releasenotes

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Default directory names, relative to the module root.
const (
	DefaultNotesDir  = ".release-notes"
	DefaultLegacyDir = "CHANGELOG"
)

// Lang selects the locale of a rendered page.
type Lang string

const (
	LangEN Lang = "en"
	LangRU Lang = "ru"
)

// Dep is one entry of the dependencies section.
type Dep struct {
	Name         string `yaml:"name"`
	From         string `yaml:"from"`
	To           string `yaml:"to"`
	ChangelogURL string `yaml:"changelog_url"`
	Notes        string `yaml:"notes"`
}

// Note is one locale file of the sectioned format. The field set is the approved section
// list and decoding is strict, so a mistyped key fails the build instead of silently
// dropping a section from the published page.
type Note struct {
	Date         string   `yaml:"date"`
	Channels     []string `yaml:"channels"`
	Summary      string   `yaml:"summary"`
	Highlights   []string `yaml:"highlights"`
	NewFeatures  []string `yaml:"new_features"`
	Improvements []string `yaml:"improvements"`
	Fixes        []string `yaml:"fixes"`
	Security     []string `yaml:"security"`
	Breaking     []string `yaml:"breaking"`
	UpgradeNotes []string `yaml:"upgrade_notes"`
	KnownIssues  []string `yaml:"known_issues"`
	Docs         []string `yaml:"docs"`
	Dependencies []Dep    `yaml:"dependencies"`

	// SkipDocs keeps a release out of the pages while still describing it for the
	// release image and the announcements. Used for internal builds.
	SkipDocs bool `yaml:"skip_docs"`

	// Version and Module are accepted because some files carry them, but the page uses
	// the file name: the tag is the identity of a release.
	Version string `yaml:"version"`
	Module  string `yaml:"module"`
}

// Release is one version, from whichever source carries it.
type Release struct {
	Version string

	// Sectioned format.
	EN *Note
	RU *Note

	// Legacy format: the items of the flat list, already flattened to strings.
	LegacyEN []string
	LegacyRU []string
}

// Sectioned reports whether the release is described in the sectioned format.
func (r *Release) Sectioned() bool { return r.EN != nil && r.RU != nil }

// Options tells Collect where to look. An empty field falls back to the default.
type Options struct {
	Root      string
	NotesDir  string
	LegacyDir string
}

func (o Options) notesDir() string {
	if o.NotesDir == "" {
		return filepath.Join(o.Root, DefaultNotesDir)
	}
	return o.NotesDir
}

func (o Options) legacyDir() string {
	if o.LegacyDir == "" {
		return filepath.Join(o.Root, DefaultLegacyDir)
	}
	return o.LegacyDir
}

var (
	leadingDigits  = regexp.MustCompile(`^(\d+)`)
	notesFileName  = regexp.MustCompile(`^(v\d+\.\d+\.\d+)(\.ru)?\.yaml$`)
	legacyFileName = regexp.MustCompile(`^(v?\d+\.\d+\.\d+[^.]*)(\.ru)?\.yml$`)
)

// Collect reads both sources and returns the releases newest first, having checked that
// each one is complete: both locales present, summary and highlights filled in, skip_docs
// agreeing between the locales. Releases marked skip_docs are left out.
func Collect(opts Options) ([]Release, error) {
	byVersion := map[string]*Release{}
	get := func(version string) *Release {
		r, ok := byVersion[version]
		if !ok {
			r = &Release{Version: version}
			byVersion[version] = r
		}
		return r
	}

	if err := readNotes(opts.notesDir(), get); err != nil {
		return nil, err
	}
	if err := readLegacy(opts.legacyDir(), get); err != nil {
		return nil, err
	}
	if len(byVersion) == 0 {
		return nil, fmt.Errorf("no release notes found in %s or %s", opts.notesDir(), opts.legacyDir())
	}

	return order(byVersion, opts)
}

func readNotes(dir string, get func(string) *Release) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := notesFileName.FindStringSubmatch(e.Name())
		if m == nil {
			// README.md and the next-release scaffolds live here too.
			continue
		}
		note, err := loadNote(filepath.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		r := get(m[1])
		if m[2] == ".ru" {
			r.RU = note
		} else {
			r.EN = note
		}
	}
	return nil
}

func readLegacy(dir string, get func(string) *Release) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := legacyFileName.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		lang := LangEN
		if m[2] == ".ru" {
			lang = LangRU
		}
		items, err := loadLegacy(filepath.Join(dir, e.Name()), lang)
		if err != nil {
			return err
		}
		r := get(m[1])
		if lang == LangRU {
			r.LegacyRU = items
		} else {
			r.LegacyEN = items
		}
	}
	return nil
}

func loadNote(path string) (*Note, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)

	var n Note
	if err := dec.Decode(&n); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &n, nil
}

// loadLegacy reads the flat list of one legacy file.
//
// An item is either a string or a single-key mapping: the house style
// `- Исправление: текст` parses as a mapping, because YAML reads a colon followed by a
// space that way. Nothing else occurs -- across the 894 legacy files in the fleet at the
// time of writing there were 1845 strings and 229 single-key mappings and no other shape
// -- so a mapping is flattened to "key: value" and anything else is an error rather than
// a silently stringified surprise.
func loadLegacy(path string, lang Lang) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc map[string]yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	key := "Changes"
	if lang == LangRU {
		key = "Изменения"
	}
	section, ok := doc[key]
	if !ok {
		return nil, fmt.Errorf("%s: has no %q key", path, key)
	}
	if section.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("%s: %q is not a list", path, key)
	}

	items := make([]string, 0, len(section.Content))
	for i, node := range section.Content {
		item, err := legacyItem(node)
		if err != nil {
			return nil, fmt.Errorf("%s: item %d: %w", path, i+1, err)
		}
		items = append(items, item)
	}
	return items, nil
}

func legacyItem(node *yaml.Node) (string, error) {
	switch node.Kind {
	case yaml.ScalarNode:
		return node.Value, nil
	case yaml.MappingNode:
		parts := make([]string, 0, len(node.Content)/2)
		for i := 0; i+1 < len(node.Content); i += 2 {
			value, err := legacyItem(node.Content[i+1])
			if err != nil {
				return "", err
			}
			parts = append(parts, node.Content[i].Value+": "+value)
		}
		return strings.Join(parts, ", "), nil
	default:
		return "", fmt.Errorf("is neither a string nor a mapping")
	}
}

func order(byVersion map[string]*Release, opts Options) ([]Release, error) {
	versions := make([]string, 0, len(byVersion))
	for v := range byVersion {
		versions = append(versions, v)
	}
	sort.SliceStable(versions, func(i, j int) bool {
		return compareVersions(versions[i], versions[j]) > 0
	})

	out := make([]Release, 0, len(versions))
	var problems []string

	for _, version := range versions {
		r := byVersion[version]

		switch {
		case r.EN != nil && r.RU == nil:
			problems = append(problems, fmt.Sprintf("%s: %s/%s.ru.yaml is missing", version, opts.notesDir(), version))
			continue
		case r.RU != nil && r.EN == nil:
			problems = append(problems, fmt.Sprintf("%s: %s/%s.yaml is missing", version, opts.notesDir(), version))
			continue
		}

		if r.Sectioned() {
			if r.EN.SkipDocs != r.RU.SkipDocs {
				problems = append(problems, fmt.Sprintf("%s: skip_docs differs between the locales", version))
				continue
			}
			if r.EN.SkipDocs {
				continue
			}
			for lang, note := range map[Lang]*Note{LangEN: r.EN, LangRU: r.RU} {
				if strings.TrimSpace(note.Summary) == "" {
					problems = append(problems, fmt.Sprintf("%s (%s): summary is empty", version, lang))
				}
				if len(note.Highlights) == 0 {
					problems = append(problems, fmt.Sprintf("%s (%s): highlights is empty", version, lang))
				}
			}
			out = append(out, *r)
			continue
		}

		switch {
		case len(r.LegacyEN) == 0:
			problems = append(problems, fmt.Sprintf("%s: %s/%s.yml is missing or empty", version, opts.legacyDir(), version))
		case len(r.LegacyRU) == 0:
			problems = append(problems, fmt.Sprintf("%s: %s/%s.ru.yml is missing or empty", version, opts.legacyDir(), version))
		default:
			out = append(out, *r)
		}
	}

	if len(problems) > 0 {
		sort.Strings(problems)
		return nil, fmt.Errorf("release notes are incomplete:\n  %s", strings.Join(problems, "\n  "))
	}
	if len(out) == 0 {
		return nil, errors.New("every version was skipped, nothing to render")
	}
	return out, nil
}

// compareVersions orders two version strings by their numeric components, the way
// packaging.version.parse did in the Python generator these pages come from.
func compareVersions(a, b string) int {
	ka, kb := versionKey(a), versionKey(b)
	n := max(len(ka), len(kb))
	for i := range n {
		var ai, bi int
		if i < len(ka) {
			ai = ka[i]
		}
		if i < len(kb) {
			bi = kb[i]
		}
		if ai != bi {
			if ai < bi {
				return -1
			}
			return 1
		}
	}
	return 0
}

func versionKey(version string) []int {
	parts := strings.Split(strings.TrimPrefix(version, "v"), ".")
	keys := make([]int, 0, len(parts))
	for _, p := range parts {
		m := leadingDigits.FindStringSubmatch(p)
		if m == nil {
			keys = append(keys, 0)
			continue
		}
		n, _ := strconv.Atoi(m[1])
		keys = append(keys, n)
	}
	return keys
}
