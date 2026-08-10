package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"khoai-chain/internal/config"
	"khoai-chain/pkg/cli"
	"os"
	"path/filepath"
	"strings"
)

// registerOrgCommands đăng ký các lệnh liên quan đến tổ chức như 'org', 'package', 'install'.
func registerOrgCommands(app *cli.CLI, configPath string) {
	// Lệnh 'org' để quản lý tổ chức.
	app.AddCommand("org", "Manage organizations", func(args []string) error {
		if len(args) < 2 || args[0] != "package" {
			fmt.Println("Invalid command. Use 'org package <name>'.")
			return nil
		}
		action, name := args[0], args[1]
		switch action {
		case "package":
			return packageOrganization(name, configPath)
		default:
			return fmt.Errorf("unknown org command: %s", action)
		}
	})

	// Lệnh 'package' (shortcut) để đóng gói một tổ chức.
	app.AddCommand("package", "Package an organization (e.g., package org <name>)", func(args []string) error {
		if len(args) < 2 || args[0] != "org" {
			return fmt.Errorf("invalid command. Use 'package org <name>'")
		}
		return packageOrganization(args[1], configPath)
	})

	// Lệnh 'install' để cài đặt một tổ chức từ file package.
	app.AddCommand("install", "Install an organization from a package file", func(args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("package file path is required")
		}
		packagePath := args[0]
		return installOrganization(packagePath)
	})
}

// packageOrganization tạo một file lưu trữ .tar.gz của một tổ chức đã được tạo.
func packageOrganization(orgName, configPath string) error {
	fmt.Printf("Packaging organization: %s\n", orgName)

	sanitizedName := sanitize(orgName)
	orgSrcDir := filepath.Join(config.BuildDir, config.OrgsDir, sanitizedName)
	if _, err := os.Stat(orgSrcDir); os.IsNotExist(err) {
		return fmt.Errorf("organization '%s' not found in build directory. Have you run 'khoai generate'?", orgName)
	}

	if err := os.MkdirAll(config.DistDir, 0755); err != nil {
		return err
	}

	versionData, err := os.ReadFile(filepath.Join(orgSrcDir, ".version"))
	if err != nil {
		return fmt.Errorf("'.version' file not found in organization build directory '%s'. Please run 'khoai generate' first: %w", orgSrcDir, err)
	}
	version := strings.TrimSpace(string(versionData))

	packageName := fmt.Sprintf("%s-khoai-%s.tar.gz", sanitizedName, version)
	packagePath := filepath.Join(config.DistDir, packageName)
	file, err := os.Create(packagePath)
	if err != nil {
		return fmt.Errorf("could not create package file: %w", err)
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	err = filepath.Walk(orgSrcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == orgSrcDir {
			return nil
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(orgSrcDir, path)
		if err != nil {
			return err
		}
		header.Name = filepath.Join(sanitizedName, relPath)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to archive organization directory: %w", err)
	}
	fmt.Printf("Successfully created package: %s\n", packagePath)
	return nil
}

// installOrganization giải nén một package vào một workspace mới.
func installOrganization(packagePath string) error {
	fmt.Printf("Installing organization from: %s\n", packagePath)

	absPath, err := filepath.Abs(packagePath)
	if err != nil {
		return fmt.Errorf("could not get absolute path for package: %w", err)
	}

	installDir := filepath.Dir(absPath)

	file, err := os.Open(absPath)
	if err != nil {
		return fmt.Errorf("could not open package: %w", err)
	}
	defer file.Close()

	gr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("could not create gzip reader: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	var targetDir string
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		targetDir = filepath.Join(installDir, header.Name)
		if !strings.HasPrefix(targetDir, installDir) {
			return fmt.Errorf("invalid path in package: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetDir), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", targetDir, err)
			}

			f, err := os.OpenFile(targetDir, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", targetDir, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("failed to write content to file %s: %w", targetDir, err)
			}
			f.Close()
		}
	}

	fmt.Printf("Successfully extracted organization to '%s'\n", targetDir)

	// Read the .version file from the newly extracted workspace
	versionFilePath := filepath.Join(targetDir, ".version")
	versionData, err := os.ReadFile(versionFilePath)
	if err != nil {
		return fmt.Errorf("installation failed: package is missing required '.version' file: %w", err)
	}
	version := strings.TrimSpace(string(versionData))
	if version == "" {
		return fmt.Errorf("installation failed: '.version' file is empty or invalid")
	}

	// Download the exact source code version required by the package
	fmt.Printf("Organization requires Khoai source version: %s. Downloading...\n", version)
	_, err = downloadViaScript(version, targetDir)
	if err != nil {
		return fmt.Errorf("failed to download required source code version '%s': %w", version, err)
	}

	fmt.Printf("Successfully downloaded source code for version %s.\n", version)
	fmt.Println("Installation complete.")
	return nil
}
