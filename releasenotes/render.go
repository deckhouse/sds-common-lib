package releasenotes

import (
	"fmt"
	"strings"
)

// section binds a heading and its lead-in sentence to the field it renders. Highlights is
// rendered unconditionally -- the format requires it -- and every other section is omitted
// when empty, which is also how Console treats an empty list.
type section struct {
	headingEN string
	headingRU string
	leadEN    string
	leadRU    string
	items     func(*Note) []string
}

// The wording follows the release-notes pages the format was approved with, so a reader
// moving between two modules' pages sees the same headings.
var sections = []section{
	{"Highlights", "Ключевые изменения", "Changes in this release:", "Изменения в этом релизе:",
		func(n *Note) []string { return n.Highlights }},
	{"New features", "Новый функционал", "This release adds:", "В этом релизе добавлено:",
		func(n *Note) []string { return n.NewFeatures }},
	{"Improvements", "Улучшения", "This release improves:", "В этом релизе улучшено:",
		func(n *Note) []string { return n.Improvements }},
	{"Fixes", "Исправления", "This release fixes:", "В этом релизе исправлено:",
		func(n *Note) []string { return n.Fixes }},
	{"Security updates", "Обновления безопасности", "Security updates in this release:", "Обновления безопасности в этом релизе:",
		func(n *Note) []string { return n.Security }},
	{"Breaking changes", "Несовместимые изменения", "Changes that affect backward compatibility:", "Изменения, влияющие на обратную совместимость:",
		func(n *Note) []string { return n.Breaking }},
	{"Upgrade notes", "Рекомендации по обновлению", "Before upgrading, note the following:", "Перед обновлением учтите следующее:",
		func(n *Note) []string { return n.UpgradeNotes }},
	{"Known issues", "Известные ограничения", "Known limitations of this release:", "Известные ограничения этого релиза:",
		func(n *Note) []string { return n.KnownIssues }},
	{"Docs", "Документация", "Documentation changes:", "Изменения в документации:",
		func(n *Note) []string { return n.Docs }},
}

// Render builds one release-notes page, newest release first.
//
// The front matter is what the documentation build needs to turn the file into a page; the
// title is the one the storage modules have always published under.
func Render(releases []Release, lang Lang) string {
	title := `title: "Release Notes"`
	if lang == LangRU {
		title = `title: "Релизы"`
	}
	lines := []string{"---", title, "---", ""}

	for i := range releases {
		r := &releases[i]
		lines = append(lines, "## "+r.Version, "")
		if r.Sectioned() {
			note := r.EN
			if lang == LangRU {
				note = r.RU
			}
			lines = renderSectioned(lines, note, lang)
			continue
		}
		items := r.LegacyEN
		if lang == LangRU {
			items = r.LegacyRU
		}
		lines = renderLegacy(lines, items)
	}

	return strings.Join(lines, "\n")
}

// renderLegacy emits a version stored in the flat format exactly as the pages carried it
// before the sectioned format arrived, so migrating a module does not rewrite its history.
func renderLegacy(lines []string, items []string) []string {
	for _, item := range items {
		lines = append(lines, "* "+item)
	}
	return append(lines, "")
}

func renderSectioned(lines []string, note *Note, lang Lang) []string {
	if note.Date != "" {
		label := "**Release date:**"
		if lang == LangRU {
			label = "**Дата релиза:**"
		}
		lines = append(lines, label+" "+note.Date, "")
	}
	if len(note.Channels) > 0 {
		label := "**Channels:**"
		if lang == LangRU {
			label = "**Каналы:**"
		}
		quoted := make([]string, 0, len(note.Channels))
		for _, c := range note.Channels {
			quoted = append(quoted, "`"+c+"`")
		}
		lines = append(lines, label+" "+strings.Join(quoted, ", "), "")
	}

	lines = append(lines, note.Summary, "")

	for _, s := range sections {
		items := s.items(note)
		if len(items) == 0 {
			continue
		}
		heading, lead := s.headingEN, s.leadEN
		if lang == LangRU {
			heading, lead = s.headingRU, s.leadRU
		}
		lines = append(lines, "### "+heading, "", lead, "")
		for _, item := range items {
			lines = append(lines, "- "+item)
		}
		lines = append(lines, "")
	}

	if len(note.Dependencies) > 0 {
		heading, lead := "Dependencies", "Dependency updates:"
		if lang == LangRU {
			heading, lead = "Зависимости", "Обновления зависимостей:"
		}
		lines = append(lines, "### "+heading, "", lead, "")
		for _, d := range note.Dependencies {
			item := fmt.Sprintf("- `%s`: `%s` → `%s`", d.Name, d.From, d.To)
			if d.ChangelogURL != "" {
				item += fmt.Sprintf(" ([changelog](%s))", d.ChangelogURL)
			}
			lines = append(lines, item)
			if d.Notes != "" {
				lines = append(lines, "  - "+d.Notes)
			}
		}
		lines = append(lines, "")
	}

	return lines
}
