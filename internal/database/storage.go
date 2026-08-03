package database

import (
	"log"
	"os"

	"github.com/dgraph-io/badger/v3"
)

type Storage struct {
	DB *badger.DB
}

// InitDB: Opens or creates a new Database at the given directory
func InitDB(dir string) *Storage {
	// If the directory does not exist, create it
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
	}

	opts := badger.DefaultOptions(dir)
	opts.Logger = nil // Disable verbose logging

	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal("❌ Error initializing DB:", err)
	}

	return &Storage{DB: db}
}

// Close: Closes the connection
func (s *Storage) Close() {
	s.DB.Close()
}

// SetData: Saves a Key-Value pair
func (s *Storage) SetData(key []byte, value []byte) error {
	return s.DB.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
}

// GetData: Retrieves a Value by Key
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

// HasKey: Checks if a key exists
func (s *Storage) HasKey(key []byte) bool {
	err := s.DB.View(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		return err
	})
	return err == nil
}
