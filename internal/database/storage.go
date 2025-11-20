package database

import (
	"log"
	"os"

	"github.com/dgraph-io/badger/v3"
)

type Storage struct {
	DB *badger.DB
}

// InitDB: Mở hoặc tạo mới Database tại đường dẫn dir
func InitDB(dir string) *Storage {
	// Nếu thư mục chưa có thì tạo mới
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
	}

	opts := badger.DefaultOptions(dir)
	opts.Logger = nil // Tắt log rác

	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal("❌ Lỗi khởi tạo DB:", err)
	}

	return &Storage{DB: db}
}

// Close: Đóng kết nối
func (s *Storage) Close() {
	s.DB.Close()
}

// SetData: Lưu Key-Value
func (s *Storage) SetData(key []byte, value []byte) error {
	return s.DB.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
}

// GetData: Lấy Value từ Key
func (s *Storage) GetData(key []byte) ([]byte, error) {
	var valCopy []byte
	err := s.DB.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		valCopy, err = item.ValueCopy(nil)
		return err
	})
	return valCopy, err
}

// HasKey: Kiểm tra key có tồn tại không
func (s *Storage) HasKey(key []byte) bool {
	err := s.DB.View(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		return err
	})
	return err == nil
}
