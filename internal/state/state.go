package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/awmiriiw/torsen/internal/logger"
)

type AppState struct {
	Connected      bool   `json:"connected"`
	DNSWasSymlink  bool   `json:"dns_was_symlink"`
	SymlinkTarget  string `json:"symlink_target,omitempty"`
	BackupIPv4Path string `json:"backup_ipv4_path,omitempty"`
	BackupIPv6Path string `json:"backup_ipv6_path,omitempty"`
	BackupTorrc    string `json:"backup_torrc_path,omitempty"`
	BackupDNS      string `json:"backup_dns_path,omitempty"`
}

type Manager struct {
	stateDir  string
	statePath string
	log       *logger.Logger
	Current   AppState
}

func NewManager(stateDir string, log *logger.Logger) (*Manager, error) {
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return nil, fmt.Errorf("cannot create state directory '%s': %w", stateDir, err)
	}

	m := &Manager{
		stateDir:  stateDir,
		statePath: filepath.Join(stateDir, "state.json"),
		log:       log,
	}

	if err := m.load(); err != nil {
		log.Warn("Could not load existing state, starting fresh: %v", err)
		m.Current = AppState{}
	}

	return m, nil
}

func (m *Manager) load() error {
	data, err := os.ReadFile(m.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("cannot read state file: %w", err)
	}

	if err := json.Unmarshal(data, &m.Current); err != nil {
		return fmt.Errorf("cannot parse state file: %w", err)
	}

	m.log.Debug("Loaded state from '%s'", m.statePath)

	return nil
}

func (m *Manager) Save() error {
	data, err := json.MarshalIndent(m.Current, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot serialize state: %w", err)
	}

	if err := os.WriteFile(m.statePath, data, 0o600); err != nil {
		return fmt.Errorf("cannot write state file: %w", err)
	}

	m.log.Debug("State saved to '%s'", m.statePath)

	return nil
}

func (m *Manager) Clear() error {
	m.Current = AppState{}

	if err := os.Remove(m.statePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot remove state file: %w", err)
	}

	m.log.Debug("State cleared.")

	return nil
}

func (m *Manager) IsConnected() bool {
	return m.Current.Connected
}

func (m *Manager) SetConnected(connected bool) {
	m.Current.Connected = connected
}

func (m *Manager) BackupDir() string {
	return m.stateDir
}
