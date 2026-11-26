package feature

import (
	"reflect"
	"strings"
	"testing"

	"github.com/samznd/layrgo/internal/templates"
)

func TestParseFields(t *testing.T) {
	t.Parallel()

	got, err := ParseFields(" customer:string, note:text? ,total:INT,created_by:time,")
	if err != nil {
		t.Fatal(err)
	}
	want := []templates.Field{
		{Name: "customer", Kind: "string"},
		{Name: "note", Kind: "text", Optional: true},
		{Name: "total", Kind: "int"},
		{Name: "created_by", Kind: "time"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseFields = %+v\nwant %+v", got, want)
	}
}

func TestParseFields_Rejects(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ spec, wantErr string }{
		{"", "at least one field"},
		{" , ", "at least one field"},
		{"title", "name:type"},
		{"title:", "name:type"},
		{"Title:string", "snake_case"},
		{"created__by:string", "snake_case"},
		{"id:int", "added automatically"},
		{"table_name:string", "TableName"},
		{"order:int", "reserved word in SQL"},
		{"title:varchar", "invalid type"},
		{"title:string,title:text", "declared twice"},
	} {
		_, err := ParseFields(tc.spec)
		if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
			t.Errorf("ParseFields(%q) error = %v, want it to mention %q", tc.spec, err, tc.wantErr)
		}
	}
}
