package tor

import (
	"bufio"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/awmiriiw/torsen/internal/config"
	"github.com/awmiriiw/torsen/internal/logger"
	"github.com/awmiriiw/torsen/internal/runner"
	"github.com/awmiriiw/torsen/internal/state"
)

//go:embed assets/torrc.template
var embeddedTorrcTemplate string

type Manager struct {
	cfg   *config.Config
	log   *logger.Logger
	run   *runner.Runner
	state *state.Manager
}

func NewManager(cfg *config.Config, log *logger.Logger, run *runner.Runner, st *state.Manager) *Manager {
	return &Manager{
		cfg:   cfg,
		log:   log,
		run:   run,
		state: st,
	}
}

func (m *Manager) BackupTorrc() error {
	src := m.cfg.SystemTorrcPath
	dst := filepath.Join(m.state.BackupDir(), "torrc.bak")

	m.log.Info("Backing up '%s' to '%s'...", src, dst)

	if _, err := os.Stat(src); os.IsNotExist(err) {
		m.log.Warn("Original torrc not found. Creating blank backup.")

		if err := os.WriteFile(dst, []byte{}, 0o644); err != nil {
			return fmt.Errorf("cannot create blank backup: %w", err)
		}

		m.state.Current.BackupTorrc = dst

		if err := m.state.Save(); err != nil {
			m.log.Warn("Could not save state after blank torrc backup: %v", err)
		}

		return nil
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("cannot read '%s': %w", src, err)
	}

	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("cannot write backup to '%s': %w", dst, err)
	}

	m.state.Current.BackupTorrc = dst

	if err := m.state.Save(); err != nil {
		m.log.Warn("Could not save state after torrc backup: %v", err)
	}

	m.log.Info("Torrc backup successful.")
	return nil
}

func (m *Manager) ApplyTemplate() error {
	dst := m.cfg.SystemTorrcPath

	m.log.Info("Rendering and applying torrc template to '%s'...", dst)

	template := m.loadTemplate()
	if len(template) == 0 {
		return fmt.Errorf("torrc template is empty")
	}

	hashedPassword, err := m.hashPassword(m.cfg.TorControlPassword)
	if err != nil {
		return fmt.Errorf("cannot hash control password: %w", err)
	}

	rendered := m.renderTemplate(template, hashedPassword)

	tmpPath := filepath.Join(m.state.BackupDir(), "torrc.rendered.tmp")
	if err := os.WriteFile(tmpPath, []byte(rendered), 0o644); err != nil {
		return fmt.Errorf("cannot write temp torrc: %w", err)
	}

	if err := m.verifyConfig(tmpPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("torrc verification failed: %w", err)
	}

	if err := os.WriteFile(dst, []byte(rendered), 0o644); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("cannot write torrc to '%s': %w", dst, err)
	}

	os.Remove(tmpPath)

	m.log.Info("Torrc template rendered and applied successfully.")
	return nil
}

func (m *Manager) RestoreTorrc() error {
	backupPath := m.state.Current.BackupTorrc
	if backupPath == "" {
		m.log.Warn("No torrc backup path in state. Skipping restore.")
		return nil
	}

	dst := m.cfg.SystemTorrcPath

	m.log.Info("Restoring torrc from '%s' to '%s'...", backupPath, dst)

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file '%s' not found", backupPath)
	}

	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("cannot read backup '%s': %w", backupPath, err)
	}

	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("cannot write torrc to '%s': %w", dst, err)
	}

	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		m.log.Warn("Could not remove torrc backup '%s': %v", backupPath, err)
	}

	m.log.Info("Torrc restored successfully.")
	return nil
}

func (m *Manager) StartService() error {
	m.log.Info("Starting Tor service...")

	if err := m.run.Run("systemctl", "restart", "tor"); err != nil {
		return fmt.Errorf("failed to start tor service: %w", err)
	}

	m.log.Info("Tor service started.")
	return nil
}

func (m *Manager) StopService() error {
	m.log.Info("Stopping Tor service...")

	if err := m.run.Run("systemctl", "stop", "tor"); err != nil {
		m.log.Warn("Could not stop tor service: %v", err)
		return err
	}

	m.log.Info("Tor service stopped.")
	return nil
}

func (m *Manager) IsServiceActive() bool {
	return m.run.RunSilent("systemctl", "is-active", "--quiet", "tor") == nil
}

func (m *Manager) EditTemplate() error {
	templatePath := m.customTemplatePath()

	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		m.log.Info("Custom template not found. Creating from embedded default...")
		if err := m.exportEmbeddedTemplate(templatePath); err != nil {
			return fmt.Errorf("cannot create custom template: %w", err)
		}
	}

	fmt.Println()
	fmt.Println("┌─────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│  ⚠️  WARNING: Runtime Placeholders                                  │")
	fmt.Println("├─────────────────────────────────────────────────────────────────────┤")
	fmt.Println("│                                                                     │")
	fmt.Println("│  The following placeholders are replaced automatically at runtime:  │")
	fmt.Println("│                                                                     │")
	fmt.Println("│    {{TOR_USER}}                → TOR_USER from .env                 │")
	fmt.Println("│    {{TOR_CONTROL_PORT}}        → TOR_CONTROL_PORT from .env         │")
	fmt.Println("│    {{HASHED_CONTROL_PASSWORD}} → auto-generated hash                │")
	fmt.Println("│    {{TOR_TRANSPARENT_PORT}}    → TOR_TRANSPARENT_PORT from .env     │")
	fmt.Println("│    {{TOR_DNS_PORT}}            → TOR_DNS_PORT from .env             │")
	fmt.Println("│                                                                     │")
	fmt.Println("│  Do NOT modify these unless you know what you are doing.            │")
	fmt.Println("│  You CAN safely edit: bridges, ExitNodes, logging, etc.             │")
	fmt.Println("│                                                                     │")
	fmt.Printf("│  Template: %-56s │\n", templatePath)
	fmt.Println("│                                                                     │")
	fmt.Println("└─────────────────────────────────────────────────────────────────────┘")
	fmt.Println()

	editor := m.findEditor()
	if editor == "" {
		fmt.Println("❌ No text editor found on this system.")
		fmt.Println("   Install one (nano, vim, micro) or set the EDITOR environment variable.")
		fmt.Printf("   Example: EDITOR=nano sudo torsen edit\n")
		return fmt.Errorf("no text editor available")
	}

	fmt.Printf("Editor: %s\n", editor)
	fmt.Println()

	if !askConfirm("Do you want to open the template for editing?") {
		fmt.Println("Edit cancelled.")
		return nil
	}

	fmt.Println()

	cmd := exec.Command(editor, templatePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor exited with error: %w", err)
	}

	data, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("cannot read edited template: %w", err)
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		m.log.Warn("Template is empty after editing. Restoring default...")
		if err := m.exportEmbeddedTemplate(templatePath); err != nil {
			return fmt.Errorf("cannot restore default template: %w", err)
		}
		fmt.Println("⚠️  Empty template detected. Default template restored.")
		return nil
	}

	fmt.Println()
	fmt.Println("✅ Template saved successfully.")
	fmt.Println("   Changes will take effect on next 'torsen connect'.")
	return nil
}

func (m *Manager) ResetTemplate() error {
	templatePath := m.customTemplatePath()

	m.log.Info("Resetting template to embedded default...")

	if err := m.exportEmbeddedTemplate(templatePath); err != nil {
		return fmt.Errorf("cannot reset template: %w", err)
	}

	fmt.Println("✅ Template reset to default.")
	return nil
}

func (m *Manager) customTemplatePath() string {
	return filepath.Join(m.state.BackupDir(), "torrc.template")
}

func (m *Manager) exportEmbeddedTemplate(path string) error {
	return os.WriteFile(path, []byte(embeddedTorrcTemplate), 0o644)
}

func (m *Manager) loadTemplate() string {
	customPath := m.customTemplatePath()

	if data, err := os.ReadFile(customPath); err == nil {
		content := strings.TrimSpace(string(data))
		if len(content) > 0 {
			m.log.Info("Using custom template from '%s'", customPath)
			return string(data)
		}
	}

	m.log.Info("Using embedded default template.")
	return embeddedTorrcTemplate
}

func (m *Manager) renderTemplate(template, hashedPassword string) string {
	r := strings.NewReplacer(
		"{{TOR_USER}}", m.cfg.TorUser,
		"{{TOR_CONTROL_PORT}}", fmt.Sprintf("%d", m.cfg.TorControlPort),
		"{{HASHED_CONTROL_PASSWORD}}", hashedPassword,
		"{{TOR_TRANSPARENT_PORT}}", fmt.Sprintf("%d", m.cfg.TorTransPort),
		"{{TOR_DNS_PORT}}", fmt.Sprintf("%d", m.cfg.TorDNSPort),
	)

	return r.Replace(template)
}

func (m *Manager) hashPassword(password string) (string, error) {
	m.log.Info("Generating hashed control password...")

	output, err := m.run.RunWithOutput("tor", "--hash-password", password)
	if err != nil {
		return "", fmt.Errorf("tor --hash-password failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "16:") {
			m.log.Info("Password hashed successfully.")
			return line, nil
		}
	}

	return "", fmt.Errorf("could not find hashed password in tor output: %s", output)
}

func (m *Manager) verifyConfig(torrcPath string) error {
	m.log.Info("Verifying torrc configuration...")

	cmd := exec.Command("tor", "--verify-config", "-f", torrcPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		m.log.Error("Torrc verification failed:\n%s", string(output))
		return fmt.Errorf("torrc verification failed: %w", err)
	}

	m.log.Info("Torrc configuration verified successfully.")
	return nil
}

func (m *Manager) findEditor() string {
	if editor := os.Getenv("EDITOR"); editor != "" {
		if _, err := exec.LookPath(editor); err == nil {
			return editor
		}
	}

	editors := []string{"nano", "vim", "vi", "micro", "joe", "ne"}
	for _, e := range editors {
		if _, err := exec.LookPath(e); err == nil {
			return e
		}
	}

	return ""
}

func askConfirm(question string) bool {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s [y/N]: ", question)

	answer, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	answer = strings.TrimSpace(strings.ToLower(answer))

	return answer == "y" || answer == "yes"
}
