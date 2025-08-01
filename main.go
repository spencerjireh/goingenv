package main

import (
	"archive/tar"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/term"
)

// Version information
const Version = "1.0.0"

// EnvFile represents a detected environment file
type EnvFile struct {
	Path         string    `json:"path"`
	RelativePath string    `json:"relative_path"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"mod_time"`
	Checksum     string    `json:"checksum"`
}

// Archive represents the structure of an encrypted archive
type Archive struct {
	CreatedAt   time.Time `json:"created_at"`
	Files       []EnvFile `json:"files"`
	TotalSize   int64     `json:"total_size"`
	Description string    `json:"description"`
}

// Config holds application configuration
type Config struct {
	DefaultDepth   int      `json:"default_depth"`
	EnvPatterns    []string `json:"env_patterns"`
	ExcludePatterns []string `json:"exclude_patterns"`
}

var config = Config{
	DefaultDepth: 3,
	EnvPatterns: []string{
		`\.env$`,
		`\.env\.local$`,
		`\.env\.development$`,
		`\.env\.production$`,
		`\.env\.staging$`,
		`\.env\.test$`,
		`\.env\.example$`,
	},
	ExcludePatterns: []string{
		`node_modules/`,
		`\.git/`,
		`vendor/`,
		`dist/`,
		`build/`,
	},
}

// Styles for the TUI
var (
	titleStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("#7D56F4")).
		Foreground(lipgloss.Color("#FAFAFA")).
		Padding(0, 1).
		MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true).
		MarginBottom(1)

	listStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1).
		MarginBottom(1)

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF5555")).
		Bold(true)

	successStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#50FA7B")).
		Bold(true)
)

// Main menu items
type menuItem struct {
	title       string
	description string
	icon        string
}

func (m menuItem) Title() string       { return m.icon + " " + m.title }
func (m menuItem) Description() string { return m.description }
func (m menuItem) FilterValue() string { return m.title }

// TUI Model
type model struct {
	state           string
	menu            list.Model
	textInput       textinput.Model
	filepicker      filepicker.Model
	progress        progress.Model
	message         string
	error           string
	selectedArchive string
	width           int
	height          int
}

// Initialize TUI model
func initialModel() model {
	items := []list.Item{
		menuItem{
			title:       "Pack Environment Files",
			description: "Scan and encrypt environment files",
			icon:        "📦",
		},
		menuItem{
			title:       "Unpack Archive",
			description: "Decrypt and restore archived files",
			icon:        "📂",
		},
		menuItem{
			title:       "List Archive Contents",
			description: "Browse archive contents without extracting",
			icon:        "📋",
		},
		menuItem{
			title:       "Status",
			description: "View current directory and available archives",
			icon:        "📊",
		},
		menuItem{
			title:       "Settings",
			description: "Configure default options",
			icon:        "⚙️",
		},
		menuItem{
			title:       "Help",
			description: "Command documentation and examples",
			icon:        "❓",
		},
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "GoingEnv - Environment File Manager"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	ti := textinput.New()
	ti.Placeholder = "Enter password..."
	ti.EchoMode = textinput.EchoPassword
	ti.CharLimit = 256

	fp := filepicker.New()
	fp.AllowedTypes = []string{".enc"}

	prog := progress.New(progress.WithDefaultGradient())

	return model{
		state:      "menu",
		menu:       l,
		textInput:  ti,
		filepicker: fp,
		progress:   prog,
	}
}

// Init implements tea.Model interface
func (m model) Init() tea.Cmd {
	return nil
}

// TUI Update function
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.menu.SetWidth(msg.Width)
		m.menu.SetHeight(msg.Height - 4)
		return m, nil

	case packCompleteMsg:
		m.message = string(msg)
		m.error = ""
		m.state = "menu"
		return m, nil

	case unpackCompleteMsg:
		m.message = string(msg)
		m.error = ""
		m.state = "menu"
		return m, nil

	case listCompleteMsg:
		m.message = string(msg)
		m.error = ""
		m.state = "listing"
		return m, nil

	case errorMsg:
		m.error = string(msg)
		m.message = ""
		m.state = "menu"
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case "menu":
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "enter":
				selectedItem := m.menu.SelectedItem().(menuItem)
				switch selectedItem.title {
				case "Pack Environment Files":
					m.state = "pack_password"
					m.textInput.Focus()
					return m, nil
				case "Unpack Archive":
					m.state = "unpack_select"
					return m, m.filepicker.Init()
				case "List Archive Contents":
					m.state = "list_select"
					return m, m.filepicker.Init()
				case "Status":
					m.state = "status"
					return m, nil
				case "Settings":
					m.state = "settings"
					return m, nil
				case "Help":
					m.state = "help"
					return m, nil
				}
			}

		case "pack_password":
			switch msg.String() {
			case "esc":
				m.state = "menu"
				m.textInput.Blur()
				m.textInput.SetValue("")
				return m, nil
			case "enter":
				password := m.textInput.Value()
				if password == "" {
					m.error = "Password cannot be empty"
					return m, nil
				}
				m.state = "packing"
				return m, packFilesCmd(password)
			}

		case "unpack_password", "list_password":
			switch msg.String() {
			case "esc":
				m.state = "menu"
				m.textInput.Blur()
				m.textInput.SetValue("")
				return m, nil
			case "enter":
				password := m.textInput.Value()
				if password == "" {
					m.error = "Password cannot be empty"
					return m, nil
				}
				if m.state == "unpack_password" {
					m.state = "unpacking"
					return m, unpackFilesCmd(password, m.selectedArchive)
				} else {
					m.state = "listing"
					return m, listFilesCmd(password, m.selectedArchive)
				}
			}

		default:
			switch msg.String() {
			case "esc", "q":
				m.state = "menu"
				m.error = ""
				m.message = ""
				return m, nil
			}
		}
	}

	// Handle component updates
	switch m.state {
	case "menu":
		m.menu, cmd = m.menu.Update(msg)
	case "pack_password", "unpack_password", "list_password":
		m.textInput, cmd = m.textInput.Update(msg)
	case "unpack_select", "list_select":
		m.filepicker, cmd = m.filepicker.Update(msg)
	}

	return m, cmd
}

// TUI View function
func (m model) View() string {
	switch m.state {
	case "menu":
		return titleStyle.Render("GoingEnv v"+Version) + "\n" + m.menu.View()

	case "pack_password":
		return titleStyle.Render("Pack Environment Files") + "\n" +
			headerStyle.Render("Enter encryption password:") + "\n" +
			m.textInput.View() + "\n\n" +
			"Press Enter to continue, Esc to go back\n" +
			m.renderError()

	case "unpack_password":
		return titleStyle.Render("Unpack Archive") + "\n" +
			headerStyle.Render("Enter decryption password:") + "\n" +
			m.textInput.View() + "\n\n" +
			"Press Enter to continue, Esc to go back\n" +
			m.renderError()

	case "list_password":
		return titleStyle.Render("List Archive Contents") + "\n" +
			headerStyle.Render("Enter decryption password:") + "\n" +
			m.textInput.View() + "\n\n" +
			"Press Enter to continue, Esc to go back\n" +
			m.renderError()

	case "unpack_select", "list_select":
		title := "Select Archive"
		if m.state == "list_select" {
			title = "Select Archive to List"
		}
		return titleStyle.Render(title) + "\n" +
			m.filepicker.View() + "\n" +
			"Select a .enc file, Esc to go back"

	case "packing":
		return titleStyle.Render("Packing Files...") + "\n" +
			m.progress.View() + "\n" +
			m.renderMessage()

	case "unpacking":
		return titleStyle.Render("Unpacking Files...") + "\n" +
			m.progress.View() + "\n" +
			m.renderMessage()

	case "listing":
		return titleStyle.Render("Listing Archive Contents") + "\n" +
			m.renderMessage() + "\n" +
			"Press Esc to go back"

	case "status":
		return m.renderStatus()

	case "settings":
		return m.renderSettings()

	case "help":
		return m.renderHelp()

	default:
		return "Unknown state"
	}
}

func (m model) renderError() string {
	if m.error != "" {
		return errorStyle.Render("Error: " + m.error)
	}
	return ""
}

func (m model) renderMessage() string {
	if m.message != "" {
		return successStyle.Render(m.message)
	}
	return ""
}

func (m model) renderStatus() string {
	cwd, _ := os.Getwd()
	archives := getAvailableArchives()
	
	status := titleStyle.Render("Status") + "\n\n" +
		headerStyle.Render("Current Directory:") + "\n" +
		cwd + "\n\n" +
		headerStyle.Render("Available Archives:") + "\n"

	if len(archives) == 0 {
		status += "No archives found in .goingenv folder\n"
	} else {
		for _, archive := range archives {
			info, _ := os.Stat(archive)
			status += fmt.Sprintf("• %s (%s)\n", filepath.Base(archive), formatSize(info.Size()))
		}
	}

	status += "\nPress Esc to go back"
	return status
}

func (m model) renderSettings() string {
	return titleStyle.Render("Settings") + "\n\n" +
		headerStyle.Render("Default Scan Depth:") + " " + fmt.Sprintf("%d", config.DefaultDepth) + "\n\n" +
		headerStyle.Render("Environment File Patterns:") + "\n" +
		strings.Join(config.EnvPatterns, "\n") + "\n\n" +
		headerStyle.Render("Exclude Patterns:") + "\n" +
		strings.Join(config.ExcludePatterns, "\n") + "\n\n" +
		"Press Esc to go back"
}

func (m model) renderHelp() string {
	return titleStyle.Render("Help") + "\n\n" +
		headerStyle.Render("Interactive Mode:") + "\n" +
		"• Navigate with arrow keys or j/k\n" +
		"• Select with Enter\n" +
		"• Go back with Esc\n" +
		"• Quit with q or Ctrl+C\n\n" +
		headerStyle.Render("Command Line Usage:") + "\n" +
		"• goingenv pack -k \"password\" [-d /path] [-o name.enc]\n" +
		"• goingenv unpack -k \"password\" [-f archive.enc] [--overwrite]\n" +
		"• goingenv list -f archive.enc -k \"password\"\n" +
		"• goingenv status\n\n" +
		"Press Esc to go back"
}

// Message types for TUI
type packCompleteMsg string
type unpackCompleteMsg string
type listCompleteMsg string
type errorMsg string

// Command execution functions for TUI
func packFilesCmd(password string) tea.Cmd {
	return func() tea.Msg {
		files, err := scanEnvFiles(".", config.DefaultDepth)
		if err != nil {
			return errorMsg(fmt.Sprintf("Error scanning files: %v", err))
		}

		if len(files) == 0 {
			return errorMsg("No environment files found")
		}

		archivePath := filepath.Join(".goingenv", fmt.Sprintf("archive-%s.enc", time.Now().Format("20060102-150405")))
		err = packFiles(files, archivePath, password)
		if err != nil {
			return errorMsg(fmt.Sprintf("Error packing files: %v", err))
		}

		return packCompleteMsg(fmt.Sprintf("Successfully packed %d files to %s", len(files), archivePath))
	}
}

func unpackFilesCmd(password, archivePath string) tea.Cmd {
	return func() tea.Msg {
		err := unpackFiles(archivePath, password, ".", false, false)
		if err != nil {
			return errorMsg(fmt.Sprintf("Error unpacking files: %v", err))
		}
		return unpackCompleteMsg("Files successfully unpacked")
	}
}

func listFilesCmd(password, archivePath string) tea.Cmd {
	return func() tea.Msg {
		archive, err := listArchiveContents(archivePath, password)
		if err != nil {
			return errorMsg(fmt.Sprintf("Error listing archive: %v", err))
		}

		var result strings.Builder
		result.WriteString(fmt.Sprintf("Archive created: %s\n", archive.CreatedAt.Format("2006-01-02 15:04:05")))
		result.WriteString(fmt.Sprintf("Total files: %d\n", len(archive.Files)))
		result.WriteString(fmt.Sprintf("Total size: %s\n\n", formatSize(archive.TotalSize)))
		
		for _, file := range archive.Files {
			result.WriteString(fmt.Sprintf("• %s (%s)\n", file.RelativePath, formatSize(file.Size)))
		}

		return listCompleteMsg(result.String())
	}
}

// Core functionality

// scanEnvFiles scans for environment files in the given directory
func scanEnvFiles(rootPath string, maxDepth int) ([]EnvFile, error) {
	var files []EnvFile
	
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check depth
		relPath, _ := filepath.Rel(rootPath, path)
		depth := strings.Count(relPath, string(filepath.Separator))
		if depth > maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories
		if info.IsDir() {
			// Check if this directory should be excluded
			for _, pattern := range config.ExcludePatterns {
				matched, _ := regexp.MatchString(pattern, path)
				if matched {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Check if file matches env patterns
		for _, pattern := range config.EnvPatterns {
			matched, _ := regexp.MatchString(pattern, info.Name())
			if matched {
				checksum, err := calculateChecksum(path)
				if err != nil {
					return err
				}

				files = append(files, EnvFile{
					Path:         path,
					RelativePath: relPath,
					Size:         info.Size(),
					ModTime:      info.ModTime(),
					Checksum:     checksum,
				})
				break
			}
		}

		return nil
	})

	return files, err
}

// packFiles creates an encrypted archive of the given files
func packFiles(files []EnvFile, outputPath, password string) error {
	// Create .goingenv directory if it doesn't exist
	goingenvDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(goingenvDir, 0755); err != nil {
		return fmt.Errorf("failed to create .goingenv directory: %v", err)
	}

	// Create .gitignore in .goingenv
	gitignorePath := filepath.Join(goingenvDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		gitignoreContent := "# Ignore all encrypted archives\n*.enc\n"
		if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
			return fmt.Errorf("failed to create .gitignore: %v", err)
		}
	}

	// Create archive metadata
	var totalSize int64
	for _, file := range files {
		totalSize += file.Size
	}

	archive := Archive{
		CreatedAt:   time.Now(),
		Files:       files,
		TotalSize:   totalSize,
		Description: fmt.Sprintf("Environment files archive created on %s", time.Now().Format("2006-01-02 15:04:05")),
	}

	// Create temporary file for the tar archive
	tmpFile, err := os.CreateTemp("", "envcase-*.tar")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(tmpFile)

	// Write metadata
	metadataJSON, err := json.Marshal(archive)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %v", err)
	}

	metadataHeader := &tar.Header{
		Name: "metadata.json",
		Mode: 0644,
		Size: int64(len(metadataJSON)),
	}
	if err := tarWriter.WriteHeader(metadataHeader); err != nil {
		return fmt.Errorf("failed to write metadata header: %v", err)
	}
	if _, err := tarWriter.Write(metadataJSON); err != nil {
		return fmt.Errorf("failed to write metadata: %v", err)
	}

	// Write files
	for _, file := range files {
		fileInfo, err := os.Stat(file.Path)
		if err != nil {
			return fmt.Errorf("failed to stat file %s: %v", file.Path, err)
		}

		header := &tar.Header{
			Name:    file.RelativePath,
			Mode:    int64(fileInfo.Mode()),
			Size:    fileInfo.Size(),
			ModTime: fileInfo.ModTime(),
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write header for %s: %v", file.Path, err)
		}

		fileContent, err := os.Open(file.Path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %v", file.Path, err)
		}

		if _, err := io.Copy(tarWriter, fileContent); err != nil {
			fileContent.Close()
			return fmt.Errorf("failed to write file %s: %v", file.Path, err)
		}
		fileContent.Close()
	}

	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %v", err)
	}

	// Read the tar data
	if _, err := tmpFile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek to beginning: %v", err)
	}

	tarData, err := io.ReadAll(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to read tar data: %v", err)
	}

	// Encrypt the data
	encryptedData, err := encrypt(tarData, password)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %v", err)
	}

	// Write encrypted data to output file
	if err := os.WriteFile(outputPath, encryptedData, 0644); err != nil {
		return fmt.Errorf("failed to write encrypted file: %v", err)
	}

	return nil
}

// unpackFiles decrypts and extracts files from an archive
func unpackFiles(archivePath, password, targetDir string, overwrite, backup bool) error {
	// Read encrypted file
	encryptedData, err := os.ReadFile(archivePath)
	if err != nil {
		return fmt.Errorf("failed to read archive: %v", err)
	}

	// Decrypt the data
	tarData, err := decrypt(encryptedData, password)
	if err != nil {
		return fmt.Errorf("failed to decrypt archive: %v", err)
	}

	// Create tar reader
	tarReader := tar.NewReader(strings.NewReader(string(tarData)))

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %v", err)
		}

		// Skip metadata file
		if header.Name == "metadata.json" {
			continue
		}

		targetPath := filepath.Join(targetDir, header.Name)

		// Create directory if needed
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}

		// Check if file exists
		if _, err := os.Stat(targetPath); err == nil {
			if !overwrite {
				fmt.Printf("Skipping existing file: %s\n", targetPath)
				continue
			}
			if backup {
				backupPath := targetPath + ".backup"
				if err := os.Rename(targetPath, backupPath); err != nil {
					return fmt.Errorf("failed to create backup: %v", err)
				}
			}
		}

		// Extract file
		file, err := os.Create(targetPath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %v", targetPath, err)
		}

		if _, err := io.Copy(file, tarReader); err != nil {
			file.Close()
			return fmt.Errorf("failed to extract file %s: %v", targetPath, err)
		}

		file.Close()

		// Set file permissions and modification time
		if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
			return fmt.Errorf("failed to set permissions: %v", err)
		}
		if err := os.Chtimes(targetPath, time.Now(), header.ModTime); err != nil {
			return fmt.Errorf("failed to set modification time: %v", err)
		}
	}

	return nil
}

// listArchiveContents lists the contents of an archive without extracting
func listArchiveContents(archivePath, password string) (*Archive, error) {
	// Read encrypted file
	encryptedData, err := os.ReadFile(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read archive: %v", err)
	}

	// Decrypt the data
	tarData, err := decrypt(encryptedData, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt archive: %v", err)
	}

	// Create tar reader
	tarReader := tar.NewReader(strings.NewReader(string(tarData)))

	// Read metadata
	header, err := tarReader.Next()
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %v", err)
	}

	if header.Name != "metadata.json" {
		return nil, fmt.Errorf("invalid archive format: missing metadata")
	}

	metadataBytes, err := io.ReadAll(tarReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %v", err)
	}

	var archive Archive
	if err := json.Unmarshal(metadataBytes, &archive); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %v", err)
	}

	return &archive, nil
}

// Encryption functions
func encrypt(data []byte, password string) ([]byte, error) {
	// Generate salt
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	// Derive key using PBKDF2
	key := pbkdf2.Key([]byte(password), salt, 100000, 32, sha256.New)

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// Encrypt data
	ciphertext := gcm.Seal(nil, nonce, data, nil)

	// Combine salt + nonce + ciphertext
	result := make([]byte, 32+len(nonce)+len(ciphertext))
	copy(result[:32], salt)
	copy(result[32:32+len(nonce)], nonce)
	copy(result[32+len(nonce):], ciphertext)

	return result, nil
}

func decrypt(data []byte, password string) ([]byte, error) {
	if len(data) < 32+12 {
		return nil, fmt.Errorf("invalid encrypted data")
	}

	// Extract salt, nonce, and ciphertext
	salt := data[:32]
	nonce := data[32 : 32+12]
	ciphertext := data[32+12:]

	// Derive key using PBKDF2
	key := pbkdf2.Key([]byte(password), salt, 100000, 32, sha256.New)

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Decrypt data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: invalid password or corrupted data")
	}

	return plaintext, nil
}

// Utility functions
func calculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func getAvailableArchives() []string {
	var archives []string
	
	goingenvDir := ".goingenv"
	if _, err := os.Stat(goingenvDir); os.IsNotExist(err) {
		return archives
	}

	files, err := os.ReadDir(goingenvDir)
	if err != nil {
		return archives
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".enc") {
			archives = append(archives, filepath.Join(goingenvDir, file.Name()))
		}
	}

	return archives
}

func readPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return string(password), nil
}

// CLI Commands
var rootCmd = &cobra.Command{
	Use:   "goingenv",
	Short: "Environment File Manager with Encryption",
	Long:  `GoingEnv is a CLI tool for managing environment files with encryption capabilities.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Launch interactive TUI
		p := tea.NewProgram(initialModel(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running program: %v\n", err)
			os.Exit(1)
		}
	},
}

var packCmd = &cobra.Command{
	Use:   "pack",
	Short: "Pack and encrypt environment files",
	Run: func(cmd *cobra.Command, args []string) {
		directory, _ := cmd.Flags().GetString("directory")
		if directory == "" {
			directory = "."
		}

		output, _ := cmd.Flags().GetString("output")
		if output == "" {
			output = filepath.Join(".goingenv", fmt.Sprintf("archive-%s.enc", time.Now().Format("20060102-150405")))
		}

		key, _ := cmd.Flags().GetString("key")
		if key == "" {
			var err error
			key, err = readPassword("Enter encryption password: ")
			if err != nil {
				fmt.Printf("Error reading password: %v\n", err)
				os.Exit(1)
			}
		}

		files, err := scanEnvFiles(directory, config.DefaultDepth)
		if err != nil {
			fmt.Printf("Error scanning files: %v\n", err)
			os.Exit(1)
		}

		if len(files) == 0 {
			fmt.Println("No environment files found")
			return
		}

		fmt.Printf("Found %d environment files:\n", len(files))
		for _, file := range files {
			fmt.Printf("  • %s (%s)\n", file.RelativePath, formatSize(file.Size))
		}

		fmt.Printf("\nPacking files to %s...\n", output)
		err = packFiles(files, output, key)
		if err != nil {
			fmt.Printf("Error packing files: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Successfully packed %d files to %s\n", len(files), output)
	},
}

var unpackCmd = &cobra.Command{
	Use:   "unpack",
	Short: "Unpack and decrypt archived files",
	Run: func(cmd *cobra.Command, args []string) {
		file, _ := cmd.Flags().GetString("file")
		if file == "" {
			// Try to find the most recent archive
			archives := getAvailableArchives()
			if len(archives) == 0 {
				fmt.Println("No archives found. Please specify a file with -f flag.")
				os.Exit(1)
			}
			file = archives[len(archives)-1] // Use the last one (most recent)
		}

		key, _ := cmd.Flags().GetString("key")
		if key == "" {
			var err error
			key, err = readPassword("Enter decryption password: ")
			if err != nil {
				fmt.Printf("Error reading password: %v\n", err)
				os.Exit(1)
			}
		}

		overwrite, _ := cmd.Flags().GetBool("overwrite")
		backup, _ := cmd.Flags().GetBool("backup")

		fmt.Printf("Unpacking %s...\n", file)
		err := unpackFiles(file, key, ".", overwrite, backup)
		if err != nil {
			fmt.Printf("Error unpacking files: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Files successfully unpacked")
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List archive contents",
	Run: func(cmd *cobra.Command, args []string) {
		file, _ := cmd.Flags().GetString("file")
		if file == "" {
			fmt.Println("Please specify an archive file with -f flag")
			os.Exit(1)
		}

		key, _ := cmd.Flags().GetString("key")
		if key == "" {
			var err error
			key, err = readPassword("Enter decryption password: ")
			if err != nil {
				fmt.Printf("Error reading password: %v\n", err)
				os.Exit(1)
			}
		}

		archive, err := listArchiveContents(file, key)
		if err != nil {
			fmt.Printf("Error listing archive: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Archive: %s\n", file)
		fmt.Printf("Created: %s\n", archive.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Files: %d\n", len(archive.Files))
		fmt.Printf("Total size: %s\n\n", formatSize(archive.TotalSize))

		for _, envFile := range archive.Files {
			fmt.Printf("  • %s (%s) - %s\n", 
				envFile.RelativePath, 
				formatSize(envFile.Size),
				envFile.ModTime.Format("2006-01-02 15:04:05"))
		}
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current status and available archives",
	Run: func(cmd *cobra.Command, args []string) {
		cwd, _ := os.Getwd()
		fmt.Printf("Current directory: %s\n\n", cwd)

		archives := getAvailableArchives()
		if len(archives) == 0 {
			fmt.Println("No archives found in .goingenv folder")
		} else {
			fmt.Printf("Available archives (%d):\n", len(archives))
			for _, archive := range archives {
				info, _ := os.Stat(archive)
				fmt.Printf("  • %s (%s) - %s\n", 
					filepath.Base(archive),
					formatSize(info.Size()),
					info.ModTime().Format("2006-01-02 15:04:05"))
			}
		}

		// Show detected env files
		files, err := scanEnvFiles(".", config.DefaultDepth)
		if err == nil && len(files) > 0 {
			fmt.Printf("\nDetected environment files (%d):\n", len(files))
			for _, file := range files {
				fmt.Printf("  • %s (%s)\n", file.RelativePath, formatSize(file.Size))
			}
		}
	},
}

func init() {
	// Pack command flags
	packCmd.Flags().StringP("key", "k", "", "Encryption key/password")
	packCmd.Flags().StringP("directory", "d", "", "Directory to scan (default: current directory)")
	packCmd.Flags().StringP("output", "o", "", "Output archive name (default: auto-generated)")

	// Unpack command flags
	unpackCmd.Flags().StringP("key", "k", "", "Decryption key/password")
	unpackCmd.Flags().StringP("file", "f", "", "Archive file to unpack")
	unpackCmd.Flags().Bool("overwrite", false, "Overwrite existing files")
	unpackCmd.Flags().Bool("backup", false, "Create backups of existing files")

	// List command flags
	listCmd.Flags().StringP("key", "k", "", "Decryption key/password")
	listCmd.Flags().StringP("file", "f", "", "Archive file to list")

	// Add commands to root
	rootCmd.AddCommand(packCmd)
	rootCmd.AddCommand(unpackCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(statusCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}