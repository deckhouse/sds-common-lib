package releasenotes_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deckhouse/sds-common-lib/releasenotes"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// legacyPair writes a version in the pre-migration flat format, including the
// `Исправление: текст` house style that YAML parses as a mapping.
func legacyPair(t *testing.T, root, version string) {
	t.Helper()
	write(t, filepath.Join(root, releasenotes.DefaultLegacyDir, version+".yml"),
		"Changes:\n  - Build changes\n  - CI: fix foo\n")
	write(t, filepath.Join(root, releasenotes.DefaultLegacyDir, version+".ru.yml"),
		"Изменения:\n  - Изменения в сборке\n  - CI: правка foo\n")
}

// sectionedPair writes a version in the current format; extra is appended to both locales.
func sectionedPair(t *testing.T, root, version, extra string) {
	t.Helper()
	write(t, filepath.Join(root, releasenotes.DefaultNotesDir, version+".yaml"),
		"date: \"2026-08-25\"\nsummary: \"Summary line.\"\nhighlights:\n  - \"Highlight one.\"\nfixes:\n  - \"Fixed a thing.\"\nnew_features: []\n"+extra)
	write(t, filepath.Join(root, releasenotes.DefaultNotesDir, version+".ru.yaml"),
		"date: \"2026-08-25\"\nsummary: \"Строка summary.\"\nhighlights:\n  - \"Ключевое изменение.\"\nfixes:\n  - \"Исправлено кое-что.\"\nnew_features: []\n"+extra)
}

func render(t *testing.T, root string) (en, ru string) {
	t.Helper()
	releases, err := releasenotes.Collect(releasenotes.Options{Root: root})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	return releasenotes.Render(releases, releasenotes.LangEN), releasenotes.Render(releases, releasenotes.LangRU)
}

func TestLegacyRenderingIsUnchanged(t *testing.T) {
	root := t.TempDir()
	legacyPair(t, root, "v0.0.1")

	en, ru := render(t, root)

	wantEN := "---\ntitle: \"Release Notes\"\n---\n\n## v0.0.1\n\n* Build changes\n* CI: fix foo\n"
	if en != wantEN {
		t.Errorf("English page changed:\n got %q\nwant %q", en, wantEN)
	}
	wantRU := "---\ntitle: \"Релизы\"\n---\n\n## v0.0.1\n\n* Изменения в сборке\n* CI: правка foo\n"
	if ru != wantRU {
		t.Errorf("Russian page changed:\n got %q\nwant %q", ru, wantRU)
	}
}

func TestBothFormatsOnOnePageNewestFirst(t *testing.T) {
	root := t.TempDir()
	legacyPair(t, root, "v0.0.9")
	sectionedPair(t, root, "v0.1.0", "")

	en, ru := render(t, root)

	if newer, older := strings.Index(en, "## v0.1.0"), strings.Index(en, "## v0.0.9"); newer > older {
		t.Errorf("v0.1.0 must precede v0.0.9, got positions %d and %d", newer, older)
	}
	for _, want := range []string{
		"## v0.1.0",
		"**Release date:** 2026-08-25",
		"Summary line.",
		"### Highlights",
		"- Highlight one.",
		"### Fixes",
		"- Fixed a thing.",
		"## v0.0.9",
		"* Build changes",
	} {
		if !strings.Contains(en, want) {
			t.Errorf("English page is missing %q", want)
		}
	}
	if strings.Contains(en, "### New features") {
		t.Error("an empty new_features section produced a heading")
	}
	for _, want := range []string{"**Дата релиза:** 2026-08-25", "### Ключевые изменения", "- Ключевое изменение.", "### Исправления"} {
		if !strings.Contains(ru, want) {
			t.Errorf("Russian page is missing %q", want)
		}
	}
}

func TestChannelsAndDependencies(t *testing.T) {
	root := t.TempDir()
	sectionedPair(t, root, "v0.1.0", "channels:\n  - Stable\ndependencies:\n  - name: ceph\n    from: \"19.2.1\"\n    to: \"19.2.2\"\n    changelog_url: \"https://example.invalid/ceph\"\n    notes: \"Patch release.\"\n")

	en, ru := render(t, root)

	for _, want := range []string{
		"**Channels:** `Stable`",
		"### Dependencies",
		"- `ceph`: `19.2.1` → `19.2.2` ([changelog](https://example.invalid/ceph))",
		"  - Patch release.",
	} {
		if !strings.Contains(en, want) {
			t.Errorf("English page is missing %q", want)
		}
	}
	if !strings.Contains(ru, "**Каналы:** `Stable`") || !strings.Contains(ru, "### Зависимости") {
		t.Error("Russian page is missing the channels or dependencies block")
	}
}

func TestSkipDocsOmitsTheVersion(t *testing.T) {
	root := t.TempDir()
	legacyPair(t, root, "v0.0.9")
	sectionedPair(t, root, "v0.1.0", "skip_docs: true\n")

	en, _ := render(t, root)

	if strings.Contains(en, "## v0.1.0") {
		t.Error("a skip_docs release reached the page")
	}
	if !strings.Contains(en, "## v0.0.9") {
		t.Error("skip_docs dropped an unrelated release")
	}
}

func TestSectionedPairWins(t *testing.T) {
	root := t.TempDir()
	legacyPair(t, root, "v0.1.0")
	sectionedPair(t, root, "v0.1.0", "")

	en, _ := render(t, root)

	if strings.Contains(en, "* Build changes") {
		t.Error("the legacy file was rendered for a version that has a sectioned pair")
	}
	if got := strings.Count(en, "## v0.1.0"); got != 1 {
		t.Errorf("version rendered %d times, want 1", got)
	}
}

func TestCollectErrors(t *testing.T) {
	tests := map[string]func(t *testing.T, root string){
		"no sources": func(t *testing.T, root string) {},
		"missing Russian locale": func(t *testing.T, root string) {
			write(t, filepath.Join(root, releasenotes.DefaultNotesDir, "v0.1.0.yaml"), "summary: \"x\"\nhighlights:\n  - \"y\"\n")
		},
		"empty highlights": func(t *testing.T, root string) {
			write(t, filepath.Join(root, releasenotes.DefaultNotesDir, "v0.1.0.yaml"), "summary: \"x\"\nhighlights: []\n")
			write(t, filepath.Join(root, releasenotes.DefaultNotesDir, "v0.1.0.ru.yaml"), "summary: \"х\"\nhighlights: []\n")
		},
		"unknown key": func(t *testing.T, root string) {
			write(t, filepath.Join(root, releasenotes.DefaultNotesDir, "v0.1.0.yaml"), "summary: \"x\"\nhighlights:\n  - \"y\"\nfixess:\n  - \"typo\"\n")
			write(t, filepath.Join(root, releasenotes.DefaultNotesDir, "v0.1.0.ru.yaml"), "summary: \"х\"\nhighlights:\n  - \"у\"\n")
		},
		"skip_docs mismatch": func(t *testing.T, root string) {
			write(t, filepath.Join(root, releasenotes.DefaultNotesDir, "v0.1.0.yaml"), "summary: \"x\"\nhighlights:\n  - \"y\"\nskip_docs: true\n")
			write(t, filepath.Join(root, releasenotes.DefaultNotesDir, "v0.1.0.ru.yaml"), "summary: \"х\"\nhighlights:\n  - \"у\"\n")
		},
		"legacy half pair": func(t *testing.T, root string) {
			write(t, filepath.Join(root, releasenotes.DefaultLegacyDir, "v0.0.1.yml"), "Changes:\n  - only English\n")
		},
		"legacy item is a list": func(t *testing.T, root string) {
			write(t, filepath.Join(root, releasenotes.DefaultLegacyDir, "v0.0.1.yml"), "Changes:\n  - [nested, list]\n")
			write(t, filepath.Join(root, releasenotes.DefaultLegacyDir, "v0.0.1.ru.yml"), "Изменения:\n  - строка\n")
		},
		"only skip_docs releases": func(t *testing.T, root string) {
			sectionedPair(t, root, "v0.1.0", "skip_docs: true\n")
		},
	}

	for name, setup := range tests {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			setup(t, root)
			if _, err := releasenotes.Collect(releasenotes.Options{Root: root}); err == nil {
				t.Fatal("want an error, got none")
			}
		})
	}
}
