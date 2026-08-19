package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	// Import core
	"khoai-chain/internal/config"
	"khoai-chain/internal/contract"
	"khoai-chain/internal/core"
	"khoai-chain/internal/database"
	"khoai-chain/internal/server"

	"khoai-chain/chaincodes"
)

func main() {
	defaultConfigPath := filepath.Join("/app", "config.yaml")

	// Parse flags to get the config path
	configPathFlag := flag.String("config", defaultConfigPath, "Path to the configuration file")
	flag.Parse()

	// 1. Load Config
	conf, err := config.LoadConfig(*configPathFlag)
	if err != nil {
		fmt.Printf("Could not read config file at: %s\n", *configPathFlag)
		absPath, _ := filepath.Abs(*configPathFlag)
		fmt.Printf("   (Absolute path: %s)\n", absPath)
		os.Exit(1)
	}

	fmt.Println("========================================")
	fmt.Printf("KHOAI CHAIN NODE: %s\n", conf.NodeName)
	fmt.Printf("Config File: %s\n", *configPathFlag)
	fmt.Printf("Database Path: %s\n", conf.DBPath)
	fmt.Println("========================================")

	// 2. Initialize DB
	db := database.InitDB(conf.DBPath)
	defer db.Close()

	// 3. Initialize Blockchain
	chain := core.InitBlockchain(db)

	// 4. Initialize Smart Contract Manager
	contractManager := contract.NewManager(chain)

	// Register contracts
	chaincodes.Init()

	fmt.Printf("- Blockchain Height: %d\n", chain.GetBestHeight())

	// 5. Initialize P2P Server
	// The P2P server handles the core blockchain protocol (block and transaction gossip).
	srv := server.NewServer(conf.P2PEndpoint, contractManager)
	srv.ConfigurePersistence(conf, *configPathFlag)
	go srv.Start()

	// 6. Initialize HTTP Control Plane
	// The HTTP server exposes API endpoints for node management (/join, /peers, etc.).
	fmt.Printf("HTTP API server listening on %s\n", conf.HTTPListenEndpoint)
	go func() {
		_ = http.ListenAndServe(conf.HTTPListenEndpoint, srv)
	}()

	// 7. Block main thread to keep the server running forever
	select {}
}
