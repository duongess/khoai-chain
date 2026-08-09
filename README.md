# Khoai Chain

## 🥔 Project Overview

Khoai Chain is a foundational blockchain implementation designed for specific network applications, such as the "Vietnam Construction Net" as configured. It features a peer-to-peer (P2P) network for decentralized communication and synchronization, and supports pluggable smart contracts written in Go. The project includes a command-line interface (CLI) tool to manage and interact with the blockchain network.

## ✨ Features

-   **Decentralized P2P Network:** Nodes communicate and synchronize blocks and transactions across a distributed network.
-   **Pluggable Smart Contracts:** Develop and deploy smart contracts written in Go, allowing for custom business logic on the chain.
-   **Configurable Network Topology:** Easily define network nodes, their ports, database paths, and peer connections using a YAML configuration file.
-   **CLI for Management:** A powerful command-line interface (`khoai`) to build, start, and interact with the blockchain network and its smart contracts.
-   **Blockchain Core:** Basic functionalities including block creation, chain synchronization, and persistent storage.

## 🚀 Getting Started

Follow these steps to set up and run your Khoai Chain project.

### Prerequisites

Before you begin, ensure you have the following installed:

-   **Go:** Version 1.22 or higher. You can download it from golang.org.
-   **Docker (Optional but Recommended):** For building and managing containerized blockchain nodes and images.

### Installation

1.  **Clone the Repository:**
    ```bash
    git clone https://github.com/your-username/khoai-chain.git
    cd khoai-chain
    ```

2.  **Build the Khoai CLI Tool:**
    The `khoai` CLI tool is essential for interacting with your blockchain. Build it for your operating system:

    **For Linux:**
    ```bash
    GOOS=linux GOARCH=amd64 go build -o dist/khoai-builder-linux ./cmd/khoai
    ```

    **For Windows:**
    ```bash
    GOOS=windows GOARCH=amd64 go build -o dist/khoai-builder-windows.exe ./cmd/khoai
    ```

    **For macOS:**
    ```bash
    GOOS=darwin GOARCH=amd64 go build -o dist/khoai-builder-darwin ./cmd/khoai
    ```

    It's recommended to add the `dist` directory to your system's `PATH` or move the `khoai` executable to a directory already in your `PATH` (e.g., `/usr/local/bin` on Linux/macOS).

## ⚙️ Configuration

The core configuration for your blockchain network is defined in `khoai-config.yaml`.
This file is **optional**. If it doesn't exist, a default single-node network will be generated.

```yaml
network:
  name: "Vietnam_Construction_Net"
  domain: "khoai.local"

docker:
  image_base: "golang:1.22-alpine"
  image_tag: "v1.0.0"
  registry: "registry.duongess.com/khoai-chain"

organizations:
  - display_name: "Vingroup"
    metadata:
      description: "Vingroup Organization"
      website: "https://vingroup.com.vn"
    chaincodes:
      - name: "sample-contract"
        package: "khoai-chain/examples/contracts/sample" # Example path
    nodes:
      - id: "hn"
        display_name: "Hanoi Node"
        endpoint: "vingroup-hn.khoai.local:9000"
      - id: "hcm"
        display_name: "HCMC Node"
        endpoint: "vingroup-hcm.khoai.local:9001"

  - display_name: "Coteccons"
    nodes:
      - id: "main"
        display_name: "Main Node"
        endpoint: "coteccons-main.khoai.local:9002"
```

-   **`network_name`**: The name of your blockchain network.
-   **`domain`**: The domain used for internal network resolution (e.g., for Docker Compose).
-   **`image_base`, `image_tag`, `registry`**: Docker image configuration for building and distributing node images.
-   **`nodes`**: A list of blockchain nodes in your network.
    -   **`name`**: Unique identifier for the node.
    -   **`port`**: The port on which the node will listen for P2P connections.
    -   **`db_path`**: The file system path where the node's blockchain data will be stored.
    -   **`peers`**: A list of `host:port` strings of other nodes this node should connect to upon startup.
    -   **`chaincodes`**: (Currently empty in the example, but would list deployed smart contracts).

## 🏃 Usage

Once the `khoai` CLI tool is built and accessible, you can use it to manage your blockchain.

### General Help
```bash
khoai help
```

### Starting a Node
To start a specific node defined in your `khoai-config.yaml`:
```bash
khoai start <node_name>
```
Example:
```bash
khoai start node_vingroup
khoai start node_coteccons
khoai start node_thachthat
```

### Executing a Smart Contract Function
To interact with a deployed smart contract:
```bash
khoai execute <contract_name> <function_name> <sender_id> [arg1] [arg2] ...
```
Example (referencing `examples/use.go`):
```bash
# Assuming 'examplesgolang' is deployed as a contract
khoai execute examplesgolang TestAdd Alice "key1" "valueB" "valueC" "valueD"
khoai execute examplesgolang TestGet Alice "key1"
```

### Building Docker Images (if applicable)
If your `builder.go` script includes logic to build Docker images based on `khoai-config.yaml`, you might have a command like:
```bash
khoai build-images
```

## 📚 Examples

The `examples/use.go` file provides a sample smart contract (`UsageExamples`) demonstrating how to define contract functions (`TestAdd`, `TestGet`) and interact with the chain's state (`ue.Save`, `ue.Get`). This serves as a good starting point for developing your own chaincodes.

## 🤝 Contributing

Contributions are welcome! Please feel free to open issues or submit pull requests.

## 📄 License

This project is licensed under the MIT License - see the `LICENSE` file for details.
