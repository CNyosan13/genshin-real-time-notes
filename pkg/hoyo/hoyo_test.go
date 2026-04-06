package hoyo

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

// MockClient implements HTTPClient for testing.
type MockClient struct {
	Response *http.Response
	Err      error
}

func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	return m.Response, m.Err
}

func TestGenerateDS(t *testing.T) {
	ds := GenerateDS()
	parts := strings.Split(ds, ",")
	if len(parts) != 3 {
		t.Errorf("expected 3 parts in DS, got %d", len(parts))
	}
	if len(parts[1]) != 6 {
		t.Errorf("expected 6 chars in random part of DS, got %d", len(parts[1]))
	}
}

func TestGetData_Genshin(t *testing.T) {
	// Sample JSON based on the fix we made for the 'transformer' object
	jsonResponse := `{
		"retcode": 0,
		"message": "OK",
		"data": {
			"current_resin": 150,
			"max_resin": 160,
			"transformer": {
				"obtained": true,
				"recovery_time": {
					"Day": 0,
					"Hour": 5,
					"Minute": 30,
					"Second": 0,
					"reached": false
				}
			}
		}
	}`

	originalClient := Client
	defer func() { Client = originalClient }()

	Client = &MockClient{
		Response: &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(jsonResponse)),
			Header:     make(http.Header),
		},
	}

	// We need to type-assert or use any to match the expected generic T
	// For this test, we just want to ensure it doesn't fail unmarshaling
	type GenshinDataShort struct {
		Retcode int `json:"retcode"`
		Data    struct {
			CurrentResin int `json:"current_resin"`
			Transformer  struct {
				Obtained bool `json:"obtained"`
			} `json:"transformer"`
		} `json:"data"`
	}

	data, err := GetData[GenshinDataShort]("http://test", "os_asia", "800123456", "token", "uid")
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}

	if data.Retcode != 0 {
		t.Errorf("expected retcode 0, got %d", data.Retcode)
	}
	if data.Data.CurrentResin != 150 {
		t.Errorf("expected 150 resin, got %d", data.Data.CurrentResin)
	}
	if !data.Data.Transformer.Obtained {
		t.Error("expected transformer obtained to be true")
	}
}
