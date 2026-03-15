package obsidianmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDocumentWithoutFrontmatter(t *testing.T) {
	doc, err := ParseDocument("# Title\n\nBody\n")
	require.NoError(t, err)
	require.Nil(t, doc.Frontmatter)
	require.Equal(t, "# Title\n\nBody\n", doc.Body)
}

func TestParseDocumentWithFrontmatter(t *testing.T) {
	doc, err := ParseDocument("---\nstatus: draft\ntags:\n  - foo\n---\n# Title\n\nBody\n")
	require.NoError(t, err)
	require.Equal(t, "draft", doc.Frontmatter["status"])
	require.Equal(t, "# Title\n\nBody\n", doc.Body)
}

func TestExtractWikilinks(t *testing.T) {
	links := ExtractWikilinks("See [[Category Theory]] and [[2a0 - Systems|systems note]] plus [[Heading#Part]].")
	require.Equal(t, []string{"Category Theory", "2a0 - Systems", "Heading"}, links)
}

func TestExtractHeadings(t *testing.T) {
	headings := ExtractHeadings("# Title\n\n## Links\ntext\n### Deep\n")
	require.Equal(t, []Heading{
		{Level: 1, Text: "Title", Line: 1},
		{Level: 2, Text: "Links", Line: 3},
		{Level: 3, Text: "Deep", Line: 5},
	}, headings)
}

func TestExtractTagsSkipsHeadings(t *testing.T) {
	tags := ExtractTags("# Heading\n\nText with #software and #systems-thinking\n#not-heading in sentence\n")
	require.Equal(t, []string{"software", "systems-thinking", "not-heading"}, tags)
}

func TestExtractTasks(t *testing.T) {
	tasks := ExtractTasks("intro\n- [ ] first task\n- [x] done task\n")
	require.Equal(t, []Task{
		{Text: "first task", Done: false, Line: 2},
		{Text: "done task", Done: true, Line: 3},
	}, tasks)
}

func TestBuildNoteZKStyle(t *testing.T) {
	note, err := BuildNote(NoteTemplate{
		Title:    "New Claim",
		WikiTags: []string{"Software", "Architecture"},
		Body:     "Architecture is about boundaries.",
		Sections: []NoteSection{
			{Title: "Logs", Body: "[[2026-03-15]]\n- Created"},
			{Title: "Brainstorm", Body: "- Explore boundary patterns"},
			{Title: "Links", Body: "- [[2g - Architecture]]"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "# New Claim\n\n[[Software]] [[Architecture]]\n\nArchitecture is about boundaries.\n\n## Brainstorm\n- Explore boundary patterns\n\n## Links\n- [[2g - Architecture]]\n\n## Logs\n[[2026-03-15]]\n- Created\n", note)
}
