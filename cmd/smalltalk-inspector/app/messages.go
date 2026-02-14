package app

import (
	"github.com/go-go-golems/go-go-goja/pkg/inspector/runtime"
	"github.com/go-go-golems/go-go-goja/pkg/jsparse"
)

// MsgFileLoaded is sent after a successful :load command.
type MsgFileLoaded struct {
	Filename string
	Source   string
	Analysis *jsparse.AnalysisResult
}

// MsgFileLoadError is sent when :load fails.
type MsgFileLoadError struct {
	Filename string
	Err      error
}

// MsgGlobalSelected is sent when the user selects a global binding.
type MsgGlobalSelected struct {
	Name       string
	BindingIdx int
}

// MsgMemberSelected is sent when the user selects a member of the current target.
type MsgMemberSelected struct {
	Name      string
	MemberIdx int
}

// MsgEvalResult is sent after a successful REPL eval.
type MsgEvalResult struct {
	Result runtime.EvalResult
}

// MsgStatusNotice is a transient status message.
type MsgStatusNotice struct {
	Text string
}
