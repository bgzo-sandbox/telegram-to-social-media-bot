package Handler

import "testing"

func TestParseAdminCallback(t *testing.T) {
	tests := []struct {
		name string
		data string
		kind string
	}{
		{name: "home", data: "admin:home", kind: "home"},
		{name: "sources", data: "admin:sources:all:0", kind: "sources"},
		{name: "source", data: "admin:source:targets:3:5", kind: "source"},
		{name: "failed", data: "admin:failed:0", kind: "failed"},
		{name: "detail-source", data: "admin:detail:source:all:1:0:2", kind: "detail-source"},
		{name: "detail-failed", data: "admin:detail:failed:0:3", kind: "detail-failed"},
		{name: "resync-source", data: "admin:resync-entry:source:all:1:0:2", kind: "resync-entry-source"},
		{name: "resync-entry-id", data: "admin:resync-entry-id:42", kind: "resync-entry-id"},
		{name: "detail-id", data: "admin:detail-id:42", kind: "detail-id"},
		{name: "resync-run", data: "admin:resync-run:42:all", kind: "resync-run"},
	}

	for _, tc := range tests {
		page, err := parseAdminCallback(tc.data)
		if err != nil {
			t.Fatalf("%s parse failed: %v", tc.name, err)
		}
		if page.Kind != tc.kind {
			t.Fatalf("%s unexpected kind: %+v", tc.name, page)
		}
	}
}

func TestPageBounds(t *testing.T) {
	start, end := pageBounds(5, 12)
	if start != 5 || end != 10 {
		t.Fatalf("unexpected page bounds: %d %d", start, end)
	}

	start, end = pageBounds(10, 12)
	if start != 10 || end != 12 {
		t.Fatalf("unexpected last page bounds: %d %d", start, end)
	}
}
