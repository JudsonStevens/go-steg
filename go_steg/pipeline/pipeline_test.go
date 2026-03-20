package pipeline

import (
	"reflect"
	"testing"
	"go-steg/go_steg/reed_solomon"
)

func TestPipelineEncodeDecodeRoundtrip(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		data   []byte
	}{
		{
			name: "no transforms",
			config: Config{Password: "test"},
			data: []byte("hello world"),
		},
		{
			name: "huffman only",
			config: Config{HuffmanEnabled: true, Password: "test"},
			data: []byte("hello world"),
		},
		{
			name: "RS only",
			config: Config{RSEnabled: true, RSLevel: reed_solomon.Standard, Password: "test"},
			data: []byte("hello world"),
		},
		{
			name: "huffman + RS",
			config: Config{
				HuffmanEnabled: true, RSEnabled: true,
				RSLevel: reed_solomon.Standard, Password: "test",
			},
			data: []byte("hello world"),
		},
		{
			name: "huffman + RS high",
			config: Config{
				HuffmanEnabled: true, RSEnabled: true,
				RSLevel: reed_solomon.High, Password: "test",
			},
			data: func() []byte {
				b := make([]byte, 256)
				for i := range b { b[i] = byte(i) }
				return b
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := Encode(tt.data, tt.config)
			if err != nil { t.Fatalf("Encode error: %v", err) }
			decoded, err := Decode(encoded, tt.config)
			if err != nil { t.Fatalf("Decode error: %v", err) }
			if !reflect.DeepEqual(decoded, tt.data) {
				t.Errorf("roundtrip failed")
			}
		})
	}
}
