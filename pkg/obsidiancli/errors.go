package obsidiancli

import "fmt"

// BinaryNotFoundError means the configured Obsidian binary could not be found.
type BinaryNotFoundError struct {
	Path string
}

func (e *BinaryNotFoundError) Error() string {
	return fmt.Sprintf("obsidiancli: binary not found: %s", e.Path)
}

// NotFoundError means the requested note, vault, or resource does not exist.
type NotFoundError struct {
	Ref string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("obsidiancli: not found: %s", e.Ref)
}

// AmbiguousReferenceError means a friendly note reference resolved to multiple files.
type AmbiguousReferenceError struct {
	Ref     string
	Matches []string
}

func (e *AmbiguousReferenceError) Error() string {
	return fmt.Sprintf("obsidiancli: ambiguous reference %q", e.Ref)
}

// UnsupportedVersionError means the installed CLI version cannot satisfy the caller.
type UnsupportedVersionError struct {
	Expected string
	Actual   string
}

func (e *UnsupportedVersionError) Error() string {
	return fmt.Sprintf("obsidiancli: unsupported version %q (expected %s)", e.Actual, e.Expected)
}

// CommandError wraps a failed CLI invocation.
type CommandError struct {
	Spec     CommandSpec
	Args     []string
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

func (e *CommandError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Err != nil {
		return fmt.Sprintf("obsidiancli: command %q failed: %v", e.Spec.Name, e.Err)
	}
	return fmt.Sprintf("obsidiancli: command %q failed", e.Spec.Name)
}

func (e *CommandError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// ParseError wraps a stdout parsing failure.
type ParseError struct {
	Spec   CommandSpec
	Stdout string
	Err    error
}

func (e *ParseError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("obsidiancli: parse output for %q: %v", e.Spec.Name, e.Err)
}

func (e *ParseError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
