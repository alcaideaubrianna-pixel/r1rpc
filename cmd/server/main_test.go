package main

import "testing"

func TestHTTPAccessURL(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		wantURL  string
		wantPort string
	}{
		{name: "仅端口", address: ":9876", wantURL: "http://localhost:9876", wantPort: "9876"},
		{name: "IPv4 通配地址", address: "0.0.0.0:8080", wantURL: "http://localhost:8080", wantPort: "8080"},
		{name: "IPv6 通配地址", address: "[::]:8080", wantURL: "http://localhost:8080", wantPort: "8080"},
		{name: "指定主机", address: "127.0.0.1:9090", wantURL: "http://127.0.0.1:9090", wantPort: "9090"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotURL, gotPort := httpAccessURL(test.address)
			if gotURL != test.wantURL || gotPort != test.wantPort {
				t.Fatalf("httpAccessURL(%q) = (%q, %q), want (%q, %q)", test.address, gotURL, gotPort, test.wantURL, test.wantPort)
			}
		})
	}
}
