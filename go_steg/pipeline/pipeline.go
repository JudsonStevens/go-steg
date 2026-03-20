package pipeline

import (
	"go-steg/go_steg/huffman"
	"go-steg/go_steg/reed_solomon"
)

type Config struct {
	BitDepth       int
	HuffmanEnabled bool
	RSEnabled      bool
	RSLevel        reed_solomon.RedundancyLevel
	FileExtension  string
	Password       string
}

func Encode(data []byte, cfg Config) ([]byte, error) {
	result := data
	if cfg.HuffmanEnabled {
		result = huffman.HuffmanEncode(result, cfg.Password)
	}
	if cfg.RSEnabled {
		var err error
		result, err = reed_solomon.RSEncode(result, cfg.RSLevel)
		if err != nil { return nil, err }
	}
	return result, nil
}

func Decode(data []byte, cfg Config) ([]byte, error) {
	result := data
	if cfg.RSEnabled {
		var err error
		result, err = reed_solomon.RSDecode(result, cfg.RSLevel)
		if err != nil { return nil, err }
	}
	if cfg.HuffmanEnabled {
		var err error
		result, err = huffman.HuffmanDecode(result, cfg.Password)
		if err != nil { return nil, err }
	}
	return result, nil
}
