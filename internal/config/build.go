package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Cấu trúc khớp với file YAML bạn đã thiết kế
type Config struct {
	NodeName string   `yaml:"node_name"` // Tên hiển thị
	Host     string   `yaml:"host"`      // IP (thường là 127.0.0.1)
	Port     string   `yaml:"port"`      // Cổng lắng nghe (VD: 9000)
	DBPath   string   `yaml:"db_path"`   // Đường dẫn lưu data
	Peers    []string `yaml:"peers"`     // Danh sách cần kết nối
}

// Hàm đọc file
func LoadConfig(filePath string) (*Config, error) {
	// 1. Đọc nội dung file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("không tìm thấy file config: %v", err)
	}

	// 2. Giải mã YAML vào struct
	var conf Config
	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		return nil, fmt.Errorf("lỗi định dạng YAML: %v", err)
	}

	return &conf, nil
}
