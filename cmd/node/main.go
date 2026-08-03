package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	// Import internal packages
	"khoai-chain/examples"
	"khoai-chain/internal/config"
	"khoai-chain/internal/contract"
	"khoai-chain/internal/core"
	"khoai-chain/internal/database"
	"khoai-chain/internal/p2p"
	"khoai-chain/pkg/cli"
)

var (
	BuiltInNodeName string = "Unknown Node"
)

func main() {
	// 1. Setup config file discovery (logic to find it next to the exe)
	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exeDir := filepath.Dir(exePath)
	defaultConfigPath := filepath.Join(exeDir, "config.yaml")

	// Parse flags to get the config path
	configPathFlag := flag.String("config", defaultConfigPath, "Path to the configuration file")
	flag.Parse()

	// 2. Initialize CLI
	nodeCLI := cli.NewCLI()

	// --- COMMAND: RUN (For Docker/Production) ---
	// This mode strictly respects the path in the config file (e.g., /app/data)
	nodeCLI.AddCommand("run", "Run node in Production (Docker) mode", func(args []string) error {
		fmt.Println("🐳 Mode: DOCKER / PRODUCTION")
		startNode(*configPathFlag, false)
		return nil
	})

	// --- COMMAND: DEV (For quick developer testing) ---
	// This mode forces the DB to be next to the exe, regardless of the config
	nodeCLI.AddCommand("dev", "Run node in Dev mode (DB saved next to the exe)", func(args []string) error {
		fmt.Println("🛠️  Mode: DEVELOPMENT")
		startNode(*configPathFlag, true)
		return nil
	})

	nodeCLI.Run()
}

func startNode(configPath string, isDevMode bool) {
	// 1. Load Config
	conf, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("❌ Could not read config file at: %s\n", configPath)
		// Suggest absolute path for easier debugging
		absPath, _ := filepath.Abs(configPath)
		fmt.Printf("   (Absolute path: %s)\n", absPath)
		os.Exit(1)
	}

	fmt.Println("========================================")
	fmt.Printf("🏭 KHOAI CHAIN NODE: %s\n", BuiltInNodeName)
	fmt.Printf("📂 Config File: %s\n", configPath)

	// 2. HANDLE DATABASE PATH (Most important logic here)
	finalDBPath := conf.DBPath

	if isDevMode {
		// LOGIC FOR DEV:
		dbName := filepath.Base(conf.DBPath)

		// And force it to be next to the exe file
		exePath, _ := os.Executable()
		exeDir := filepath.Dir(exePath)
		finalDBPath = filepath.Join(exeDir, dbName)

		fmt.Println("🔧 Dev Override: Forcing DB to local directory")
	} else {
		// LOGIC FOR DOCKER / RUN:
		// Keep the config. If config is "/app/data", use it as is.
		fmt.Println("🐳 Docker Mode: Using DB path from config")
	}

	fmt.Printf("💾 Database Path: %s\n", finalDBPath)
	fmt.Println("========================================")

	// 3. Initialize DB
	db := database.InitDB(finalDBPath)
	// Note: In server's infinite loop mode (select{}), this defer only runs on app shutdown (Ctrl+C)
	defer db.Close()

	// 4. Initialize Blockchain
	chain := core.InitBlockchain(db)

	// 5. Initialize Smart Contract Manager
	contractManager := contract.NewManager(chain)

	// Register example contracts (if needed)
	// contractManager.RegisterApp(examples.NewUsageExamples())
	// (Or use .Imports .Registrations from your template)
	contractManager.RegisterApp(examples.NewUsageExamples())

	fmt.Printf("⛓️  Blockchain Height: %d\n", chain.GetBestHeight())

	// 6. Initialize P2P Server
	srv := p2p.NewServer(conf.Port, contractManager)
	go srv.Start()

	// 7. Connect to Peers (After 2s)
	go func() {
		time.Sleep(2 * time.Second)
		if len(conf.Peers) > 0 {
			fmt.Println("🌐 Peers list in config:", conf.Peers)

			for _, peerAddr := range conf.Peers {
				targetAddr := peerAddr

				if isDevMode {
					parts := strings.Split(peerAddr, ":")
					// Ensure correct host:port format
					if len(parts) == 2 {
						port := parts[1]

						// Create a new address pointing to localhost
						targetAddr = fmt.Sprintf("localhost:%s", port)
					}
				}

				// Connect to the address (processed or original)
				srv.ConnectToPeer(targetAddr)
			}
		}
	}()

	// 8. Block main thread to keep the server running forever
	select {}
}
