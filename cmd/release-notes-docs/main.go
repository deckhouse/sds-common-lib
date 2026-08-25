// Command release-notes-docs renders the release-notes pages of a storage module.
//
// It reads the module's release notes from git and writes the two documentation pages the
// module bundle ships:
//
//	.release-notes/vX.Y.Z.yaml, vX.Y.Z.ru.yaml   sectioned notes (current format)
//	CHANGELOG/vX.Y.Z.yml, vX.Y.Z.ru.yml          flat list (releases cut before the move)
//	                    |
//	                    v
//	docs/RELEASE_NOTES.md, docs/RELEASE_NOTES.ru.md
//
// The module's docs image runs it during the werf build, so the pages are never committed:
//
//	shell:
//	  setup:
//	    - go run github.com/deckhouse/sds-common-lib/cmd/release-notes-docs@vX.Y.Z --root /
//
// It exits non-zero when the notes are incomplete -- a locale missing, an empty summary or
// highlights, a mistyped section key -- so a broken page fails the build instead of being
// published half empty.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/deckhouse/sds-common-lib/releasenotes"
)

func main() {
	root := flag.String("root", ".", "module root")
	notesDir := flag.String("notes-dir", "", "release-notes directory (default <root>/"+releasenotes.DefaultNotesDir+")")
	legacyDir := flag.String("legacy-dir", "", "legacy changelog directory (default <root>/"+releasenotes.DefaultLegacyDir+")")
	outEN := flag.String("out-en", "", "English page (default <root>/docs/RELEASE_NOTES.md)")
	outRU := flag.String("out-ru", "", "Russian page (default <root>/docs/RELEASE_NOTES.ru.md)")
	flag.Parse()

	if err := run(*root, *notesDir, *legacyDir, *outEN, *outRU); err != nil {
		fmt.Fprintln(os.Stderr, "release-notes-docs:", err)
		os.Exit(1)
	}
}

func run(root, notesDir, legacyDir, outEN, outRU string) error {
	opts := releasenotes.Options{Root: root, NotesDir: notesDir, LegacyDir: legacyDir}
	releases, err := releasenotes.Collect(opts)
	if err != nil {
		return err
	}

	if outEN == "" {
		outEN = filepath.Join(root, "docs", "RELEASE_NOTES.md")
	}
	if outRU == "" {
		outRU = filepath.Join(root, "docs", "RELEASE_NOTES.ru.md")
	}

	sectioned := 0
	for i := range releases {
		if releases[i].Sectioned() {
			sectioned++
		}
	}
	fmt.Printf("rendering %d releases (%d sectioned, %d legacy)\n", len(releases), sectioned, len(releases)-sectioned)

	for _, out := range []struct {
		path string
		lang releasenotes.Lang
	}{{outEN, releasenotes.LangEN}, {outRU, releasenotes.LangRU}} {
		if err := os.MkdirAll(filepath.Dir(out.path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(out.path, []byte(releasenotes.Render(releases, out.lang)), 0o644); err != nil {
			return err
		}
		fmt.Println("wrote", out.path)
	}
	return nil
}
