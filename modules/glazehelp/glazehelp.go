package glazehelp

import (
	"github.com/dop251/goja"
	"github.com/go-go-golems/glazed/pkg/help"
	"github.com/go-go-golems/go-go-goja/modules"
)

type m struct{}

// Compile-time interface check
var _ modules.NativeModule = (*m)(nil)

func (m) Name() string { return "glazehelp" }

func (m) Doc() string {
	return "Native module providing access to Glazed HelpSystem instances from JavaScript"
}

func (m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	exports := moduleObj.Get("exports").(*goja.Object)

	// JS: glazehelp.query(key, dsl) -> []Section (as JS objects)
	_ = exports.Set("query", func(key, dsl string) (interface{}, error) {
		hs, err := Get(key)
		if err != nil {
			return nil, err
		}

		sections, err := hs.QuerySections(dsl)
		if err != nil {
			return nil, err
		}

		// Convert to JS-friendly format
		result := make([]map[string]interface{}, len(sections))
		for i, section := range sections {
			result[i] = sectionToMap(section)
		}
		return result, nil
	})

	// JS: glazehelp.section(key, slug) -> Section or null
	_ = exports.Set("section", func(key, slug string) (interface{}, error) {
		hs, err := Get(key)
		if err != nil {
			return nil, err
		}

		section, err := hs.GetSectionWithSlug(slug)
		if err != nil {
			// Return null for section not found, rather than error
			return nil, nil
		}

		if section == nil {
			return nil, nil
		}

		return sectionToMap(section), nil
	})

	// JS: glazehelp.render(key) -> markdown string (top-level page)
	_ = exports.Set("render", func(key string) (interface{}, error) {
		hs, err := Get(key)
		if err != nil {
			return nil, err
		}

		page := hs.GetTopLevelHelpPage()
		if page == nil {
			return nil, nil
		}

		// Return the HelpPage structure as a JS object
		return map[string]interface{}{
			"defaultGeneralTopics": convertSections(page.DefaultGeneralTopics),
			"otherGeneralTopics":   convertSections(page.OtherGeneralTopics),
			"allGeneralTopics":     convertSections(page.AllGeneralTopics),
			"defaultExamples":      convertSections(page.DefaultExamples),
			"otherExamples":        convertSections(page.OtherExamples),
			"allExamples":          convertSections(page.AllExamples),
			"defaultApplications":  convertSections(page.DefaultApplications),
			"otherApplications":    convertSections(page.OtherApplications),
			"allApplications":      convertSections(page.AllApplications),
			"defaultTutorials":     convertSections(page.DefaultTutorials),
			"otherTutorials":       convertSections(page.OtherTutorials),
			"allTutorials":         convertSections(page.AllTutorials),
		}, nil
	})

	// JS: glazehelp.topics(key) -> []string (distinct topics across all sections)
	_ = exports.Set("topics", func(key string) (interface{}, error) {
		hs, err := Get(key)
		if err != nil {
			return nil, err
		}

		// Get all sections to extract topics
		sections, err := hs.QuerySections("")
		if err != nil {
			return nil, err
		}

		topicsSet := make(map[string]bool)
		for _, section := range sections {
			for _, topic := range section.Topics {
				topicsSet[topic] = true
			}
		}

		topics := make([]string, 0, len(topicsSet))
		for topic := range topicsSet {
			topics = append(topics, topic)
		}

		return topics, nil
	})

	// JS: glazehelp.keys() -> []string (registered help system keys)
	_ = exports.Set("keys", func() interface{} {
		return Keys()
	})
}

func init() {
	modules.Register(&m{})
}

// convertSections converts a slice of sections to JavaScript-friendly format
func convertSections(sections []*help.Section) []map[string]interface{} {
	result := make([]map[string]interface{}, len(sections))
	for i, section := range sections {
		result[i] = sectionToMap(section)
	}
	return result
}

// sectionToMap converts a help.Section to a map for JavaScript consumption
func sectionToMap(section *help.Section) map[string]interface{} {
	return map[string]interface{}{
		"title":          section.Title,
		"slug":           section.Slug,
		"short":          section.Short,
		"content":        section.Content,
		"topics":         section.Topics,
		"commands":       section.Commands,
		"flags":          section.Flags,
		"isTopLevel":     section.IsTopLevel,
		"isTemplate":     section.IsTemplate,
		"showPerDefault": section.ShowPerDefault,
		"sectionType":    section.SectionType.String(),
		"order":          section.Order,
	}
}
