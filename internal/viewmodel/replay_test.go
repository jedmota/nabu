package viewmodel

import "testing"

func TestProxyHost(t *testing.T) {
	tests := []struct {
		addr string
		want string
	}{
		{"0.0.0.0", "127.0.0.1"},
		{"", "127.0.0.1"},
		{"::", "127.0.0.1"},
		{"127.0.0.1", "127.0.0.1"},
		{"192.168.1.100", "192.168.1.100"},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			vm := &ViewModel{source: &mockFlowSourceWithAddr{bindAddress: tt.addr}}
			got := vm.proxyHost()
			if got != tt.want {
				t.Errorf("proxyHost() = %q, want %q", got, tt.want)
			}
		})
	}
}

type mockFlowSourceWithAddr struct {
	mockFlowSource
	bindAddress string
}

func (m *mockFlowSourceWithAddr) BindAddress() string { return m.bindAddress }
