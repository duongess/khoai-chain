package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// Định nghĩa hàm xử lý: Nhận vào danh sách tham số, trả về lỗi nếu có
type CommandHandler func(args []string) error

type Group struct {
	Name        string
	Description string
	Commands    []*Command
}

type Command struct {
	Names       []string
	Description string
	Flags       *flag.FlagSet
	Handler     CommandHandler
}

type CLI struct {
	Groups map[string]*Group
}

func NewCLI() *CLI {
	return &CLI{
		Groups: make(map[string]*Group),
	}
}

// Đăng ký lệnh mới
func (c *CLI) AddCommand(nameStr string, desc string, handler CommandHandler, groupName string, groupDesc ...string) {
	names := strings.Fields(nameStr)

	primaryName := names[0]

	fs := flag.NewFlagSet(primaryName, flag.ExitOnError)

	cmd := &Command{
		Names:       names,
		Description: desc,
		Flags:       fs,
		Handler:     handler,
	}

	if _, exists := c.Groups[groupName]; !exists {
		c.Groups[groupName] = &Group{
			Name:        groupName,
			Description: strings.Join(groupDesc, " "),
			Commands:    []*Command{},
		}
	}

	c.Groups[groupName].Commands = append(c.Groups[groupName].Commands, cmd)
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

	for _, group := range c.Groups {
		fmt.Println(group.Name)
		fmt.Println(group.Description)
		for _, cmd := range group.Commands {
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
		if buildCmd, ok := c.Groups["build"]; ok {
			for _, cmd := range buildCmd.Commands {
				if err := cmd.Handler(os.Args[1:]); err != nil {
					fmt.Println("❌ Error:", err)
					os.Exit(1)
				}
			}
			// Nếu không có lệnh build cụ thể, chạy lệnh build mặc định
			if len(buildCmd.Commands) > 0 {
				if err := buildCmd.Commands[0].Handler(os.Args[1:]); err != nil {
					fmt.Println("❌ Error:", err)
					os.Exit(1)
				}
				return
			}
		}
	}

	// 2. Nếu là lệnh cụ thể (VD: khoai help, khoai version)
	for _, group := range c.Groups {
		for _, cmd := range group.Commands {
			if cmd.Names[0] == arg {
				if err := cmd.Handler(os.Args[2:]); err != nil {
					fmt.Println("❌ Error:", err)
					os.Exit(1)
				}
				return
			}
		}
	}

	// 3. Không hiểu lệnh gì
	fmt.Printf("❌ Unknown command: %s\n", arg)
	c.PrintHelp()
	os.Exit(1)
}
