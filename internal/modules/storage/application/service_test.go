package application

import "testing"

func TestCompatibleTypesAllowsZipBasedOfficeDocuments(t *testing.T) {
	cases := []struct {
		name     string
		declared string
		detected string
	}{
		{"docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "application/zip"},
		{"xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "application/zip"},
		{"pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation", "application/zip"},
		{"odt", "application/vnd.oasis.opendocument.text", "application/zip"},
		{"unknown-binary", "application/x-custom-binary", "application/octet-stream"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if !compatibleTypes(testCase.declared, testCase.detected) {
				t.Fatalf("expected %q to be compatible with %q", testCase.declared, testCase.detected)
			}
		})
	}
}

func TestCompatibleTypesRejectsMismatchedActiveContent(t *testing.T) {
	if compatibleTypes("application/pdf", "text/html") {
		t.Fatal("PDF must not be accepted as HTML")
	}
}
