package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/awmiriiw/torsen/internal/config"
	"github.com/awmiriiw/torsen/internal/dns"
	"github.com/awmiriiw/torsen/internal/firewall"
	"github.com/awmiriiw/torsen/internal/logger"
	"github.com/awmiriiw/torsen/internal/runner"
	"github.com/awmiriiw/torsen/internal/state"
	"github.com/awmiriiw/torsen/internal/tor"
)

var validCommands = []string{"connect", "disconnect", "status", "verify", "recover", "edit", "reset-template"}

func main() {
	if len(os.Args) != 2 || !isValidCommand(os.Args[1]) {
		fmt.Printf("Usage: sudo %s [%s]\n", os.Args[0], strings.Join(validCommands, "|"))
		os.Exit(1)
	}

	if os.Geteuid() != 0 {
		fmt.Println("Error: This program must be run as root.")
		os.Exit(1)
	}

	command := os.Args[1]

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New("torsen", cfg.LogDir)
	log.SetLevel(cfg.LogLevel)
	defer log.Close()

	run := runner.New(log)

	st, err := state.NewManager(cfg.StateDir, log)
	if err != nil {
		log.Fatal("Failed to initialize state manager: %v", err)
		os.Exit(1)
	}

	switch command {
	case "connect":
		runConnect(cfg, log, run, st)
	case "disconnect":
		runDisconnect(cfg, log, run, st)
	case "status":
		runStatus(cfg, log, run, st)
	case "verify":
		runVerify(cfg, log, run)
	case "recover":
		runRecover(cfg, log, run, st)
	case "edit":
		runEdit(cfg, log, run, st)
	case "reset-template":
		runResetTemplate(cfg, log, run, st)
	}
}

func isValidCommand(cmd string) bool {
	for _, c := range validCommands {
		if c == cmd {
			return true
		}
	}
	return false
}

func runConnect(cfg *config.Config, log *logger.Logger, run *runner.Runner, st *state.Manager) {
	if st.IsConnected() {
		log.Warn("Already connected. Run 'disconnect' first.")
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Info("--- Starting Secure Connection Process ---")

	dnsManager := dns.NewManager(cfg, log, run, st)
	torManager := tor.NewManager(cfg, log, run, st)
	bootstrapMonitor := tor.NewBootstrapMonitor(cfg, log)

	fwManager, err := firewall.NewManager(cfg, log, run, st)
	if err != nil {
		log.Fatal("Cannot initialize firewall manager: %v", err)
		os.Exit(1)
	}

	log.Info("Step 1: Backing up firewall rules...")
	if err := fwManager.BackupRules(); err != nil {
		log.Fatal("Failed to backup firewall rules: %v", err)
		os.Exit(1)
	}

	if ctx.Err() != nil {
		doCleanup(cfg, log, run, st)
		return
	}

	log.Info("Step 2: Taking control of DNS...")
	if err := dnsManager.TakeControl(); err != nil {
		log.Fatal("Failed to take control of DNS: %v", err)
		doCleanup(cfg, log, run, st)
		return
	}

	if ctx.Err() != nil {
		doCleanup(cfg, log, run, st)
		return
	}

	log.Info("Step 3: Configuring Tor...")
	if err := torManager.BackupTorrc(); err != nil {
		log.Fatal("Failed to backup torrc: %v", err)
		doCleanup(cfg, log, run, st)
		return
	}

	if err := torManager.ApplyTemplate(); err != nil {
		log.Fatal("Failed to apply torrc template: %v", err)
		doCleanup(cfg, log, run, st)
		return
	}

	if ctx.Err() != nil {
		doCleanup(cfg, log, run, st)
		return
	}

	log.Info("Step 4: Starting Tor service...")
	if err := torManager.StartService(); err != nil {
		log.Fatal("Failed to start Tor service: %v", err)
		doCleanup(cfg, log, run, st)
		return
	}

	log.Info("Step 5: Waiting for Tor bootstrap...")
	if err := bootstrapMonitor.WaitForBootstrap(ctx); err != nil {
		if ctx.Err() != nil {
			log.Info("Bootstrap cancelled by user signal.")
		} else {
			log.Fatal("Tor bootstrap failed: %v", err)
		}
		doCleanup(cfg, log, run, st)
		return
	}

	if ctx.Err() != nil {
		doCleanup(cfg, log, run, st)
		return
	}

	log.Info("Step 6: Applying firewall rules...")
	if err := fwManager.ApplyTorRules(); err != nil {
		log.Fatal("Failed to apply firewall rules: %v", err)
		doCleanup(cfg, log, run, st)
		return
	}

	st.SetConnected(true)
	if err := st.Save(); err != nil {
		log.Warn("Could not save connected state: %v", err)
	}

	log.Info("✅ System traffic is now securely routed through Tor.")
	fmt.Println("\n🔒 Connection is active. Press Ctrl+C to disconnect and restore settings.\n")

	<-ctx.Done()

	fmt.Println()
	log.Info("Signal received. Disconnecting...")
	doCleanup(cfg, log, run, st)
}

func runDisconnect(cfg *config.Config, log *logger.Logger, run *runner.Runner, st *state.Manager) {
	if !st.IsConnected() {
		log.Warn("Not currently connected. Nothing to do.")
		return
	}

	log.Info("Disconnect command received.")
	doCleanup(cfg, log, run, st)
}

func runStatus(cfg *config.Config, log *logger.Logger, run *runner.Runner, st *state.Manager) {
	fmt.Println("=== Torsen Status ===")
	fmt.Println()

	torManager := tor.NewManager(cfg, log, run, st)
	dnsManager := dns.NewManager(cfg, log, run, st)

	stateConnected := st.IsConnected()
	torActive := torManager.IsServiceActive()
	dnsHijacked := dnsManager.IsTorDNS()

	fwActive := false
	fwManager, err := firewall.NewManager(cfg, log, run, st)
	if err == nil {
		fwActive = fwManager.HasTorsenChains()
	}

	printStatus("State file", stateConnected)
	printStatus("Tor service", torActive)
	printStatus("DNS hijack", dnsHijacked)
	printStatus("Firewall chains", fwActive)

	fmt.Println()

	allActive := stateConnected && torActive && dnsHijacked && fwActive
	if allActive {
		fmt.Println("🔒 Overall: CONNECTED through Tor")
	} else if stateConnected || torActive || dnsHijacked || fwActive {
		fmt.Println("⚠️  Overall: PARTIALLY CONNECTED (run 'recover' to fix)")
	} else {
		fmt.Println("🔓 Overall: NOT CONNECTED")
	}
}

func printStatus(label string, active bool) {
	if active {
		fmt.Printf("  ✅ %-20s active\n", label)
	} else {
		fmt.Printf("  ❌ %-20s inactive\n", label)
	}
}

func runVerify(cfg *config.Config, log *logger.Logger, run *runner.Runner) {
	fmt.Println("=== Torsen System Verification ===")
	fmt.Println()

	allOk := true

	allOk = checkBinary("tor", run) && allOk
	allOk = checkBinary("iptables", run) && allOk
	allOk = checkBinary("iptables-save", run) && allOk
	allOk = checkBinary("iptables-restore", run) && allOk
	allOk = checkBinary("ip6tables", run) && allOk
	allOk = checkBinary("systemctl", run) && allOk

	checkBinaryOptional("obfs4proxy", run)
	checkBinaryOptional("nmcli", run)

	if _, err := run.RunWithOutput("id", "-u", cfg.TorUser); err != nil {
		fmt.Printf("  ❌ Tor user '%s' not found\n", cfg.TorUser)
		allOk = false
	} else {
		fmt.Printf("  ✅ Tor user '%s' exists\n", cfg.TorUser)
	}

	if _, err := os.Stat(cfg.ResolvConfPath); err == nil {
		fmt.Printf("  ✅ resolv.conf exists at '%s'\n", cfg.ResolvConfPath)
	} else {
		fmt.Printf("  ❌ resolv.conf not found at '%s'\n", cfg.ResolvConfPath)
		allOk = false
	}

	torrcDir := cfg.SystemTorrcPath[:strings.LastIndex(cfg.SystemTorrcPath, "/")]
	if _, err := os.Stat(torrcDir); err == nil {
		fmt.Printf("  ✅ Tor config directory exists: '%s'\n", torrcDir)
	} else {
		fmt.Printf("  ❌ Tor config directory missing: '%s'\n", torrcDir)
		allOk = false
	}

	if _, err := os.Stat(cfg.StateDir); err == nil {
		fmt.Printf("  ✅ State directory exists: '%s'\n", cfg.StateDir)
	} else {
		fmt.Printf("  ⚠️  State directory missing (will be created): '%s'\n", cfg.StateDir)
	}

	fmt.Println()

	if allOk {
		fmt.Println("✅ System is ready for Torsen.")
	} else {
		fmt.Println("❌ Some requirements are missing. Fix them before running 'connect'.")
	}
}

func checkBinary(name string, run *runner.Runner) bool {
	if _, err := run.RunWithOutput("which", name); err != nil {
		fmt.Printf("  ❌ '%s' not found\n", name)
		return false
	}
	fmt.Printf("  ✅ '%s' found\n", name)
	return true
}

func checkBinaryOptional(name string, run *runner.Runner) {
	if _, err := run.RunWithOutput("which", name); err != nil {
		fmt.Printf("  ⚠️  '%s' not found (optional)\n", name)
	} else {
		fmt.Printf("  ✅ '%s' found\n", name)
	}
}

func runRecover(cfg *config.Config, log *logger.Logger, run *runner.Runner, st *state.Manager) {
	log.Warn("--- Recovery Process Initiated ---")
	fmt.Println("Recovering system to clean state...")

	doCleanup(cfg, log, run, st)

	fmt.Println("\n✅ Recovery complete. System should be in clean state.")
	fmt.Println("Run 'sudo torsen status' to verify.")
}

func runEdit(cfg *config.Config, log *logger.Logger, run *runner.Runner, st *state.Manager) {
	torManager := tor.NewManager(cfg, log, run, st)

	if err := torManager.EditTemplate(); err != nil {
		log.Error("Failed to edit template: %v", err)
		os.Exit(1)
	}
}

func runResetTemplate(cfg *config.Config, log *logger.Logger, run *runner.Runner, st *state.Manager) {
	torManager := tor.NewManager(cfg, log, run, st)

	if err := torManager.ResetTemplate(); err != nil {
		log.Error("Failed to reset template: %v", err)
		os.Exit(1)
	}
}

func doCleanup(cfg *config.Config, log *logger.Logger, run *runner.Runner, st *state.Manager) {
	log.Warn("--- Cleanup Process Initiated ---")

	torManager := tor.NewManager(cfg, log, run, st)
	dnsManager := dns.NewManager(cfg, log, run, st)

	log.Info("Cleanup Step 1: Stopping Tor service...")
	if err := torManager.StopService(); err != nil {
		log.Error("Failed to stop Tor: %v", err)
	}

	log.Info("Cleanup Step 2: Restoring firewall rules...")
	fwManager, err := firewall.NewManager(cfg, log, run, st)
	if err != nil {
		log.Error("Cannot create firewall manager for restore: %v", err)
	} else {
		if err := fwManager.RestoreRules(); err != nil {
			log.Error("Failed to restore firewall: %v", err)
		}
	}

	log.Info("Cleanup Step 3: Restoring Tor configuration...")
	if err := torManager.RestoreTorrc(); err != nil {
		log.Error("Failed to restore torrc: %v", err)
	}

	log.Info("Cleanup Step 4: Releasing DNS control...")
	if err := dnsManager.ReleaseControl(); err != nil {
		log.Error("Failed to release DNS: %v", err)
	}

	st.SetConnected(false)
	if err := st.Clear(); err != nil {
		log.Error("Failed to clear state: %v", err)
	}

	log.Info("✅ System connectivity restored. Tor service is stopped.")
}
