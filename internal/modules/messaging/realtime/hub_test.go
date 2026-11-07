package realtime

import (
	"reflect"
	"testing"
)

func TestWebSocketOriginPatternsUseConfiguredCORSOrigins(t *testing.T) {
	got := websocketOriginPatterns([]string{
		"https://admin.example.com",
		"http://localhost:3000",
		"https://admin.example.com/ignored-path",
		"not-an-origin",
	})
	want := []string{"admin.example.com", "localhost:3000"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("origin patterns = %#v, want %#v", got, want)
	}
}
