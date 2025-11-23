package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// Định nghĩa hàm xử lý: Nhận vào danh sách tham số, trả về lỗi nếu có
type CommandHandler func(args []string) error

type Command struct {
	Names       []string
	Description string
	Flags       *flag.FlagSet
	Handler     CommandHandler
}

type CLI struct {
	Commands map[string]*Command
}

func NewCLI() *CLI {
	return &CLI{
		Commands: make(map[string]*Command),
	}
}

// Đăng ký lệnh mới
func (c *CLI) AddCommand(nameStr string, desc string, handler CommandHandler) {
	names := strings.Fields(nameStr)

	primaryName := names[0]

	fs := flag.NewFlagSet(primaryName, flag.ExitOnError)

	cmd := &Command{
		Names:       names,
		Description: desc,
		Flags:       fs,
		Handler:     handler,
	}

	for _, n := range names {
		c.Commands[n] = cmd
	}
}

// Hàm in hướng dẫn sử dụng chung
func (c *CLI) PrintHelp() {
	fmt.Println("🥔 KHOAI CHAIN BUILDER CLI")
	fmt.Println("Usage: khoai [command] [flags]")
	fmt.Println("\nAvailable Commands:")

	var uniqueCmds []*Command
	printed := make(map[*Command]bool)

	maxNameLen := 0
	maxAliasLen := 0

	for _, cmd := range c.Commands {
		if printed[cmd] {
			continue
		}
		printed[cmd] = true
		uniqueCmds = append(uniqueCmds, cmd)

		if len(cmd.Names) > 0 {
			if len(cmd.Names[0]) > maxNameLen {
				maxNameLen = len(cmd.Names[0])
			}
		}

		if len(cmd.Names) > 1 {
			if len(cmd.Names[1]) > maxAliasLen {
				maxAliasLen = len(cmd.Names[1])
			}
		}
	}

	maxNameLen += 4
	maxAliasLen += 4

	for _, cmd := range uniqueCmds {
		name := ""
		alias := ""

		if len(cmd.Names) > 0 {
			name = cmd.Names[0]
		}
		if len(cmd.Names) > 1 {
			alias = cmd.Names[1]
		}

		fmt.Printf("  %-*s%-*s%s\n", maxNameLen, name, maxAliasLen, alias, cmd.Description)
	}

	fmt.Println("\nDefault:")
	fmt.Println("  (no command)    Run builder directly")
}

// Phân loại và chạy lệnh
func (c *CLI) Run() {
	if len(os.Args) < 2 {
		c.PrintHelp()
		return
	}

	arg := os.Args[1]

	// 1. Nếu tham số bắt đầu bằng "-" (VD: -input), coi như là chạy lệnh Build mặc định
	if arg[0] == '-' {
		if buildCmd, ok := c.Commands["build"]; ok {
			if err := buildCmd.Handler(os.Args[1:]); err != nil {
				fmt.Println("❌ Error:", err)
				os.Exit(1)
			}
			return
		}
	}

	// 2. Nếu là lệnh cụ thể (VD: khoai help, khoai version)
	if cmd, ok := c.Commands[arg]; ok {
		if err := cmd.Handler(os.Args[2:]); err != nil {
			fmt.Println("❌ Error:", err)
			os.Exit(1)
		}
		return
	}

	// 3. Không hiểu lệnh gì
	fmt.Printf("❌ Unknown command: %s\n", arg)
	c.PrintHelp()
	os.Exit(1)
}
