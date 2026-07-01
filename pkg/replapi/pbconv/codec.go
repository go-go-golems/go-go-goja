package pbconv

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	replapiv1 "github.com/go-go-golems/go-go-goja/pkg/replapi/pb/proto/goja/replapi/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const SchemaVersion uint32 = 1

var MarshalOptions = protojson.MarshalOptions{EmitUnpopulated: false, UseProtoNames: false}
var UnmarshalOptions = protojson.UnmarshalOptions{DiscardUnknown: false}

func MarshalJSON(msg proto.Message) ([]byte, error) { return MarshalOptions.Marshal(msg) }

func UnmarshalEvaluateRequestJSON(b []byte) (*replapiv1.EvaluateRequest, error) {
	var req replapiv1.EvaluateRequest
	if err := UnmarshalOptions.Unmarshal(b, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func timestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func timestampPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil || t.IsZero() {
		return nil
	}
	return timestamppb.New(*t)
}

func RawJSONToValue(raw json.RawMessage) (*structpb.Value, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return structpb.NewNullValue(), nil
	}
	var decoded any
	if err := json.Unmarshal(trimmed, &decoded); err != nil {
		return nil, fmt.Errorf("decode raw JSON value: %w", err)
	}
	return structpb.NewValue(decoded)
}

func ValueToRawJSON(value *structpb.Value) (json.RawMessage, error) {
	if value == nil {
		return json.RawMessage("null"), nil
	}
	b, err := value.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("encode protobuf value as raw JSON: %w", err)
	}
	return json.RawMessage(b), nil
}
