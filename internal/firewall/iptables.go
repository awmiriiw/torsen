package firewall

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/awmiriiw/torsen/internal/config"
	"github.com/awmiriiw/torsen/internal/logger"
	"github.com/awmiriiw/torsen/internal/runner"
	"github.com/awmiriiw/torsen/internal/state"
)

const (
	chainOutput = "TORSEN_OUTPUT"
	chainNat    = "TORSEN_NAT"
	chainPrert  = "TORSEN_PREROUTING"
)

type Manager struct {
	cfg    *config.Config
	log    *logger.Logger
	run    *runner.Runner
	state  *state.Manager
	torUID string
}

func NewManager(cfg *config.Config, log *logger.Logger, run *runner.Runner, st *state.Manager) (*Manager, error) {
	m := &Manager{
		cfg:   cfg,
		log:   log,
		run:   run,
		state: st,
	}

	uid, err := m.getTorUID()
	if err != nil {
		return nil, fmt.Errorf("cannot get UID for user '%s': %w", cfg.TorUser, err)
	}

	m.torUID = uid
	m.log.Info("Tor user '%s' has UID: %s", cfg.TorUser, uid)

	return m, nil
}

func (m *Manager) getTorUID() (string, error) {
	output, err := m.run.RunWithOutput("id", "-u", m.cfg.TorUser)
	if err != nil {
		return "", fmt.Errorf("user '%s' not found: %w", m.cfg.TorUser, err)
	}
	return strings.TrimSpace(output), nil
}

func (m *Manager) BackupRules() error {
	m.log.Info("Backing up current firewall rules...")

	v4Path := filepath.Join(m.state.BackupDir(), "iptables.v4.bak")
	v6Path := filepath.Join(m.state.BackupDir(), "ip6tables.v6.bak")

	if err := m.saveRules("iptables-save", v4Path); err != nil {
		return fmt.Errorf("failed to backup IPv4 rules: %w", err)
	}

	if err := m.saveRules("ip6tables-save", v6Path); err != nil {
		return fmt.Errorf("failed to backup IPv6 rules: %w", err)
	}

	m.state.Current.BackupIPv4Path = v4Path
	m.state.Current.BackupIPv6Path = v6Path

	if err := m.state.Save(); err != nil {
		m.log.Warn("Could not save state after firewall backup: %v", err)
	}

	m.log.Info("Firewall rules backed up successfully.")
	return nil
}

func (m *Manager) saveRules(command, path string) error {
	output, err := m.run.RunWithOutput(command)
	if err != nil {
		return fmt.Errorf("'%s' failed: %w", command, err)
	}

	if err := os.WriteFile(path, []byte(output), 0o600); err != nil {
		return fmt.Errorf("cannot write to '%s': %w", path, err)
	}

	return nil
}

func (m *Manager) RestoreRules() error {
	m.log.Warn("Restoring firewall rules from backup...")

	m.removeTorsenChains()

	m.setDefaultAcceptV6()

	v4Restored := m.restoreFromFile("iptables-restore", m.state.Current.BackupIPv4Path, "IPv4")
	v6Restored := m.restoreFromFile("ip6tables-restore", m.state.Current.BackupIPv6Path, "IPv6")

	if !v4Restored {
		m.log.Info("No IPv4 backup found. Setting default ACCEPT policies.")
		m.setDefaultAcceptV4()
	}

	if !v6Restored {
		m.log.Info("No IPv6 backup found. Setting default ACCEPT policies.")
		m.setDefaultAcceptV6()
	}

	m.log.Info("Firewall rules restored.")
	return nil
}

func (m *Manager) restoreFromFile(command, backupPath, label string) bool {
	if backupPath == "" {
		return false
	}

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return false
	}

	data, err := os.ReadFile(backupPath)
	if err != nil {
		m.log.Error("Cannot read %s backup '%s': %v", label, backupPath, err)
		return false
	}

	tmpFile := backupPath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0o600); err != nil {
		m.log.Error("Cannot write temp file for %s restore: %v", label, err)
		return false
	}
	defer os.Remove(tmpFile)

	if err := m.run.Run("bash", "-c", fmt.Sprintf("%s < %s", command, tmpFile)); err != nil {
		m.log.Error("Failed to restore %s rules: %v", label, err)
		return false
	}

	os.Remove(backupPath)
	m.log.Info("Successfully restored %s rules from backup.", label)
	return true
}

func (m *Manager) setDefaultAcceptV4() {
	_ = m.run.RunSilent("iptables", "-P", "INPUT", "ACCEPT")
	_ = m.run.RunSilent("iptables", "-P", "FORWARD", "ACCEPT")
	_ = m.run.RunSilent("iptables", "-P", "OUTPUT", "ACCEPT")
}

func (m *Manager) setDefaultAcceptV6() {
	_ = m.run.RunSilent("ip6tables", "-P", "INPUT", "ACCEPT")
	_ = m.run.RunSilent("ip6tables", "-P", "FORWARD", "ACCEPT")
	_ = m.run.RunSilent("ip6tables", "-P", "OUTPUT", "ACCEPT")
}

func (m *Manager) chainExists(table, chain string) bool {
	err := m.run.RunSilent("iptables", "-t", table, "-L", chain, "-n")
	return err == nil
}

func (m *Manager) createChain(table, chain string) error {
	if m.chainExists(table, chain) {
		_ = m.run.RunSilent("iptables", "-t", table, "-F", chain)
		return nil
	}
	return m.run.Run("iptables", "-t", table, "-N", chain)
}

func (m *Manager) removeTorsenChains() {
	m.log.Info("Removing Torsen firewall chains...")

	_ = m.run.RunSilent("iptables", "-D", "OUTPUT", "-j", chainOutput)
	_ = m.run.RunSilent("iptables", "-t", "nat", "-D", "OUTPUT", "-j", chainNat)
	_ = m.run.RunSilent("iptables", "-t", "nat", "-D", "PREROUTING", "-j", chainPrert)

	_ = m.run.RunSilent("iptables", "-F", chainOutput)
	_ = m.run.RunSilent("iptables", "-X", chainOutput)
	_ = m.run.RunSilent("iptables", "-t", "nat", "-F", chainNat)
	_ = m.run.RunSilent("iptables", "-t", "nat", "-X", chainNat)
	_ = m.run.RunSilent("iptables", "-t", "nat", "-F", chainPrert)
	_ = m.run.RunSilent("iptables", "-t", "nat", "-X", chainPrert)

	_ = m.run.RunSilent("ip6tables", "-P", "INPUT", "ACCEPT")
	_ = m.run.RunSilent("ip6tables", "-P", "FORWARD", "ACCEPT")
	_ = m.run.RunSilent("ip6tables", "-P", "OUTPUT", "ACCEPT")
	_ = m.run.RunSilent("ip6tables", "-F")
}

func (m *Manager) ApplyTorRules() error {
	m.log.Info("Applying Tor firewall rules using custom chains...")

	if m.torUID == "" {
		return fmt.Errorf("tor user UID not available")
	}

	var errors []string

	if err := m.createChain("filter", chainOutput); err != nil {
		return fmt.Errorf("cannot create chain %s: %w", chainOutput, err)
	}
	if err := m.createChain("nat", chainNat); err != nil {
		return fmt.Errorf("cannot create chain %s: %w", chainNat, err)
	}
	if err := m.createChain("nat", chainPrert); err != nil {
		return fmt.Errorf("cannot create chain %s: %w", chainPrert, err)
	}

	natRules := [][]string{
		{"-m", "owner", "--uid-owner", m.torUID, "-j", "RETURN"},
		{"-p", "udp", "--dport", "53", "-j", "REDIRECT", "--to-ports", fmt.Sprintf("%d", m.cfg.TorDNSPort)},
	}

	allNetworks := append(m.cfg.NonTorNetworks, "127.0.0.0/8")
	for _, net := range allNetworks {
		natRules = append(natRules, []string{"-d", net, "-j", "RETURN"})
	}

	natRules = append(natRules, []string{
		"-p", "tcp", "--syn", "-j", "REDIRECT", "--to-ports", fmt.Sprintf("%d", m.cfg.TorTransPort),
	})

	for _, rule := range natRules {
		args := append([]string{"iptables", "-t", "nat", "-A", chainNat}, rule...)
		if err := m.run.Run(args...); err != nil {
			errors = append(errors, fmt.Sprintf("nat rule %v: %v", rule, err))
		}
	}

	preroutingRules := [][]string{
		{"-i", "lo", "-p", "udp", "--dport", "53", "-j", "REDIRECT", "--to-ports", fmt.Sprintf("%d", m.cfg.TorDNSPort)},
	}

	for _, rule := range preroutingRules {
		args := append([]string{"iptables", "-t", "nat", "-A", chainPrert}, rule...)
		if err := m.run.Run(args...); err != nil {
			errors = append(errors, fmt.Sprintf("prerouting rule %v: %v", rule, err))
		}
	}

	outputRules := [][]string{
		{"-o", "lo", "-j", "ACCEPT"},
		{"-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT"},
	}

	for _, net := range allNetworks {
		outputRules = append(outputRules, []string{"-d", net, "-j", "ACCEPT"})
	}

	outputRules = append(outputRules,
		[]string{"-m", "owner", "--uid-owner", m.torUID, "-j", "ACCEPT"},
		[]string{"-j", "REJECT"},
	)

	for _, rule := range outputRules {
		args := append([]string{"iptables", "-A", chainOutput}, rule...)
		if err := m.run.Run(args...); err != nil {
			errors = append(errors, fmt.Sprintf("output rule %v: %v", rule, err))
		}
	}

	if err := m.run.Run("iptables", "-t", "nat", "-I", "OUTPUT", "1", "-j", chainNat); err != nil {
		errors = append(errors, fmt.Sprintf("insert nat jump: %v", err))
	}
	if err := m.run.Run("iptables", "-t", "nat", "-I", "PREROUTING", "1", "-j", chainPrert); err != nil {
		errors = append(errors, fmt.Sprintf("insert prerouting jump: %v", err))
	}
	if err := m.run.Run("iptables", "-I", "OUTPUT", "1", "-j", chainOutput); err != nil {
		errors = append(errors, fmt.Sprintf("insert output jump: %v", err))
	}

	_ = m.run.RunSilent("iptables", "-A", "INPUT", "-i", "lo", "-j", "ACCEPT")

	_ = m.run.Run("ip6tables", "-F")
	_ = m.run.Run("ip6tables", "-A", "INPUT", "-i", "lo", "-j", "ACCEPT")
	_ = m.run.Run("ip6tables", "-A", "OUTPUT", "-o", "lo", "-j", "ACCEPT")
	_ = m.run.Run("ip6tables", "-P", "INPUT", "DROP")
	_ = m.run.Run("ip6tables", "-P", "FORWARD", "DROP")
	_ = m.run.Run("ip6tables", "-P", "OUTPUT", "DROP")

	if len(errors) > 0 {
		m.log.Error("Some firewall rules failed to apply:")
		for _, e := range errors {
			m.log.Error("  - %s", e)
		}
		return fmt.Errorf("%d firewall rules failed", len(errors))
	}

	m.log.Info("✅ Tor firewall rules applied via custom chains.")
	return nil
}

func (m *Manager) HasTorsenChains() bool {
	return m.chainExists("filter", chainOutput) &&
		m.chainExists("nat", chainNat) &&
		m.chainExists("nat", chainPrert)
}
