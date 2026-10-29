package dns

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

func (m *Manager) TakeControl() error {
	m.log.Info("Taking control of system DNS...")

	m.stopDNSService()

	resolvPath := m.cfg.ResolvConfPath
	backupPath := filepath.Join(m.state.BackupDir(), "resolv.conf.bak")

	if info, err := os.Lstat(resolvPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(resolvPath)
			if err != nil {
				return fmt.Errorf("cannot read symlink target of '%s': %w", resolvPath, err)
			}

			m.state.Current.DNSWasSymlink = true
			m.state.Current.SymlinkTarget = target
			m.log.Info("Detected symlink, target: '%s'", target)
		}
	}

	if err := m.backupResolvConf(resolvPath, backupPath); err != nil {
		m.startDNSService()
		return fmt.Errorf("failed to backup DNS config: %w", err)
	}

	m.state.Current.BackupDNS = backupPath

	if err := m.writeNewResolvConf(resolvPath); err != nil {
		_ = m.ReleaseControl()
		return fmt.Errorf("failed to write new resolv.conf: %w", err)
	}

	if err := m.state.Save(); err != nil {
		m.log.Warn("Could not save state after DNS takeover: %v", err)
	}

	m.log.Info("DNS now routes through Tor (nameserver %s).", m.cfg.TorDNSIP)
	return nil
}

func (m *Manager) ReleaseControl() error {
	m.log.Info("Releasing control of system DNS...")

	resolvPath := m.cfg.ResolvConfPath

	if os.Remove(resolvPath) != nil {
		m.log.Warn("Could not remove current resolv.conf")
	}

	if m.state.Current.DNSWasSymlink && m.state.Current.SymlinkTarget != "" {
		if err := os.Symlink(m.state.Current.SymlinkTarget, resolvPath); err != nil {
			m.log.Error("Failed to restore symlink '%s' -> '%s': %v",
				resolvPath, m.state.Current.SymlinkTarget, err)
		} else {
			m.log.Info("Restored symlink -> '%s'", m.state.Current.SymlinkTarget)
		}
	} else if m.state.Current.BackupDNS != "" {
		if _, err := os.Stat(m.state.Current.BackupDNS); err == nil {
			data, err := os.ReadFile(m.state.Current.BackupDNS)
			if err == nil {
				if err := os.WriteFile(resolvPath, data, 0o644); err != nil {
					m.log.Error("Failed to restore resolv.conf from backup: %v", err)
				} else {
					m.log.Info("Restored resolv.conf from backup.")
				}
			}
			os.Remove(m.state.Current.BackupDNS)
		}
	} else {
		m.log.Warn("No DNS backup info found. DNS may need manual fix.")
	}

	m.startDNSService()
	m.reloadNetworkManager()

	m.log.Info("DNS control released.")
	return nil
}

func (m *Manager) IsTorDNS() bool {
	data, err := os.ReadFile(m.cfg.ResolvConfPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "Managed by Torsen")
}

func (m *Manager) backupResolvConf(src, dst string) error {
	m.log.Info("Backing up '%s' to '%s'...", src, dst)

	if _, err := os.Stat(src); os.IsNotExist(err) {
		m.log.Warn("resolv.conf does not exist. Creating empty backup.")
		return os.WriteFile(dst, []byte{}, 0o644)
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("cannot read '%s': %w", src, err)
	}

	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("cannot write backup to '%s': %w", dst, err)
	}

	m.log.Info("DNS backup successful.")
	return nil
}

func (m *Manager) writeNewResolvConf(path string) error {
	m.log.Info("Writing Tor DNS config to '%s'...", path)

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		m.log.Warn("Could not remove existing resolv.conf: %v", err)
	}

	content := fmt.Sprintf("# Managed by Torsen\nnameserver %s\n", m.cfg.TorDNSIP)

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("cannot write to '%s': %w", path, err)
	}

	return nil
}

func (m *Manager) stopDNSService() {
	if m.cfg.DNSServiceName != "" {
		_ = m.run.RunSilent("systemctl", "stop", m.cfg.DNSServiceName)
		m.log.Info("Stopped %s service.", m.cfg.DNSServiceName)
		return
	}

	services := []string{"systemd-resolved", "dnsmasq", "unbound"}
	for _, svc := range services {
		if m.run.RunSilent("systemctl", "is-active", "--quiet", svc) == nil {
			_ = m.run.RunSilent("systemctl", "stop", svc)
			m.log.Info("Stopped %s service.", svc)
		}
	}
}

func (m *Manager) startDNSService() {
	if m.cfg.DNSServiceName != "" {
		_ = m.run.RunSilent("systemctl", "restart", m.cfg.DNSServiceName)
		m.log.Info("Restarted %s service.", m.cfg.DNSServiceName)
		return
	}

	services := []string{"systemd-resolved", "dnsmasq", "unbound"}
	for _, svc := range services {
		if m.run.RunSilent("systemctl", "is-enabled", "--quiet", svc) == nil {
			_ = m.run.RunSilent("systemctl", "restart", svc)
			m.log.Info("Restarted %s service.", svc)
			return
		}
	}
}

func (m *Manager) reloadNetworkManager() {
	_ = m.run.RunSilent("nmcli", "general", "reload")
}
