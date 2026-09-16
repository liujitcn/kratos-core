package middleware

import (
	"testing"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"google.golang.org/protobuf/proto"
)

func TestStandardValidationMessageKey(t *testing.T) {
	tests := []struct {
		name     string
		ruleID   string
		elements []*validate.FieldPathElement
		want     string
	}{
		{
			name:   "standard email rule",
			ruleID: "string.email_empty",
			elements: []*validate.FieldPathElement{
				{FieldName: proto.String("string")},
				{FieldName: proto.String("email")},
			},
			want: "common.validation.invalid",
		},
		{
			name:   "required rule",
			ruleID: "required",
			elements: []*validate.FieldPathElement{
				{FieldName: proto.String("required")},
			},
			want: "common.validation.required",
		},
		{
			name:   "custom rule",
			ruleID: "system.admin.base.user.entity.email.format",
			elements: []*validate.FieldPathElement{
				{FieldName: proto.String("cel")},
			},
			want: "system.admin.base.user.entity.email.format",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if got := standardValidationMessageKey(testCase.ruleID, testCase.elements); got != testCase.want {
				t.Fatalf("message key = %q, want %q", got, testCase.want)
			}
		})
	}
}
