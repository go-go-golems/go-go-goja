# Changelog

## 2026-04-22

- Initial workspace created


## 2026-04-22

Created deep-dive analysis document identifying three gaps in jsverbs-example: (1) list is plain text not glazed, (2) global flags hidden from short help for glazed verbs due to ShortHelpSections filtering, (3) output mode documentation needed.

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-go-goja/cmd/jsverbs-example/main.go — Contains list command and parser config


## 2026-04-22

Implemented list as a glazed GlazeCommand with structured output (path, source, output_mode). Removed raw cobra.Command list implementation. Added schema.GlobalDefaultSlug to ShortHelpSections so global flags appear in short help for all commands.

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-go-goja/cmd/jsverbs-example/main.go — Converted list to glazed command and fixed ShortHelpSections

