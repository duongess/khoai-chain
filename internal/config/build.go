package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ConfigContent struct {
	NodeName   string            `yaml:"node_name"`
	Host       string            `yaml:"host"`
	Port       string            `yaml:"port"`
	DBPath     string            `yaml:"db_path"`
	Peers      []string          `yaml:"peers"`
	Chaincodes []ChaincodeConfig `yaml:"chaincodes"`
}

func LoadConfig(filePath string) (*ConfigContent, error) {
	// 1. Đọc nội dung file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("không tìm thấy file config: %v", err)
	}

	// 2. Giải mã YAML vào struct
	var conf ConfigContent
	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		return nil, fmt.Errorf("lỗi định dạng YAML: %v", err)
	}

	return &conf, nil
}
