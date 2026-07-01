package pbconv_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	replapiv1 "github.com/go-go-golems/go-go-goja/pkg/replapi/pb/proto/goja/replapi/v1"
	"github.com/go-go-golems/go-go-goja/pkg/replapi/pbconv"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestEvaluateRequestFromProto(t *testing.T) {
	got := pbconv.EvaluateRequestFromProto(&replapiv1.EvaluateRequest{SchemaVersion: 1, Source: "1 + 2"})
	if got.Source != "1 + 2" {
		t.Fatalf("bad source: %q", got.Source)
	}
}

func TestEvaluateResponseToProto(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	resp := &replsession.EvaluateResponse{
		Session: &replsession.SessionSummary{
			ID:           "sess-1",
			Profile:      "interactive",
			Policy:       replsession.InteractiveSessionOptions().Policy,
			CreatedAt:    now,
			CellCount:    1,
			BindingCount: 1,
			Bindings: []replsession.BindingView{{
				Name: "answer", Kind: "const", Origin: "cell", DeclaredInCell: 1, LastUpdatedCell: 1, DeclaredLine: 1,
				Runtime: replsession.BindingRuntimeView{ValueKind: "number", Preview: "42"},
			}},
		},
		Cell: &replsession.CellReport{
			ID:        1,
			CreatedAt: now,
			Source:    "const answer = 42; answer",
			Execution: replsession.ExecutionReport{Status: "ok", Result: "42", ResultJSON: `{"result":42}`, DurationMS: 7, Console: []replsession.ConsoleEvent{{Kind: "log", Message: "hello"}}},
			Runtime:   replsession.RuntimeReport{NewBindings: []string{"answer"}, CurrentCellValue: "42"},
		},
	}
	got := pbconv.EvaluateResponseToProto(resp)
	if got.GetSchemaVersion() != pbconv.SchemaVersion || got.GetSession().GetId() != "sess-1" || got.GetCell().GetExecution().GetResult() != "42" {
		t.Fatalf("bad converted response: %#v", got)
	}
	if got.GetSession().GetPolicy().GetEval().GetMode() != replapiv1.EvalMode_EVAL_MODE_INSTRUMENTED {
		t.Fatalf("bad eval mode: %s", got.GetSession().GetPolicy().GetEval().GetMode())
	}
	b, err := pbconv.MarshalJSON(got)
	if err != nil {
		t.Fatalf("marshal protojson: %v", err)
	}
	body := string(b)
	for _, want := range []string{`"schemaVersion":1`, `"session"`, `"cell"`, `"resultJson":"{\"result\":42}"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %s in %s", want, body)
		}
	}
	golden, err := os.ReadFile(filepath.Join("testdata", "evaluate_response.golden.json"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var wantJSON, gotJSON any
	if err := json.Unmarshal(golden, &wantJSON); err != nil {
		t.Fatalf("decode golden: %v", err)
	}
	if err := json.Unmarshal(b, &gotJSON); err != nil {
		t.Fatalf("decode got: %v", err)
	}
	if jsonValue(wantJSON) != jsonValue(gotJSON) {
		t.Fatalf("golden mismatch\nwant: %s\ngot: %s", string(golden), body)
	}
}

func TestEmptySourceEvaluateResponseToProto(t *testing.T) {
	resp := &replsession.EvaluateResponse{Cell: &replsession.CellReport{Execution: replsession.ExecutionReport{Status: "empty-source", Result: "undefined"}}}
	got := pbconv.EvaluateResponseToProto(resp)
	if got.GetCell().GetExecution().GetStatus() != "empty-source" || got.GetCell().GetExecution().GetResult() != "undefined" {
		t.Fatalf("bad empty source conversion: %#v", got)
	}
}

func TestEvalModeRoundTrip(t *testing.T) {
	if got := pbconv.EvalModeToProto(replsession.EvalModeRaw); got != replapiv1.EvalMode_EVAL_MODE_RAW {
		t.Fatalf("raw -> %s", got)
	}
	if got := pbconv.EvalModeFromProto(replapiv1.EvalMode_EVAL_MODE_INSTRUMENTED); got != replsession.EvalModeInstrumented {
		t.Fatalf("instrumented -> %q", got)
	}
}

func TestRawJSONToValuePreservesShapes(t *testing.T) {
	cases := []json.RawMessage{
		json.RawMessage(`{"a":1,"b":[true,"x"]}`),
		json.RawMessage(`[1,2,3]`),
		json.RawMessage(`"hello"`),
		json.RawMessage(`42`),
		json.RawMessage(`true`),
		json.RawMessage(`null`),
	}
	for _, tc := range cases {
		value, err := pbconv.RawJSONToValue(tc)
		if err != nil {
			t.Fatalf("RawJSONToValue(%s): %v", tc, err)
		}
		roundTrip, err := pbconv.ValueToRawJSON(value)
		if err != nil {
			t.Fatalf("ValueToRawJSON(%s): %v", tc, err)
		}
		var want, got any
		if err := json.Unmarshal(tc, &want); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(roundTrip, &got); err != nil {
			t.Fatal(err)
		}
		if jsonValue(want) != jsonValue(got) {
			t.Fatalf("round trip mismatch: want %s got %s", jsonValue(want), jsonValue(got))
		}
	}
}

func TestSessionExportToProto(t *testing.T) {
	now := time.Unix(200, 0).UTC()
	exported := &repldb.SessionExport{
		Session: repldb.SessionRecord{SessionID: "sess-1", CreatedAt: now, UpdatedAt: now, EngineKind: "goja", MetadataJSON: json.RawMessage(`{"profile":"persistent"}`)},
		Evaluations: []repldb.EvaluationRecord{{
			EvaluationID:      1,
			SessionID:         "sess-1",
			CellID:            1,
			CreatedAt:         now,
			RawSource:         "answer",
			OK:                true,
			ResultJSON:        json.RawMessage(`{"result":42}`),
			AnalysisJSON:      json.RawMessage(`{"diagnostics":[]}`),
			GlobalsBeforeJSON: json.RawMessage(`[]`),
			GlobalsAfterJSON:  json.RawMessage(`[{"name":"answer"}]`),
			BindingVersions:   []repldb.BindingVersionRecord{{Name: "answer", CreatedAt: now, CellID: 1, Action: "insert", SummaryJSON: json.RawMessage(`{"name":"answer"}`), ExportKind: "json", ExportJSON: json.RawMessage(`42`)}},
			BindingDocs:       []repldb.BindingDocRecord{{SymbolName: "answer", CellID: 1, SourceKind: "jsdoc", RawDoc: "answer docs", NormalizedJSON: json.RawMessage(`{"description":"answer docs"}`)}},
		}},
	}
	got, err := pbconv.SessionExportToProto(exported)
	if err != nil {
		t.Fatalf("SessionExportToProto: %v", err)
	}
	if got.GetSession().GetSessionId() != "sess-1" || len(got.GetEvaluations()) != 1 || len(got.GetEvaluations()[0].GetBindingVersions()) != 1 {
		t.Fatalf("bad export conversion: %#v", got)
	}
	if _, err := protojson.Marshal(got); err != nil {
		t.Fatalf("protojson marshal export: %v", err)
	}
}

func jsonValue(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
