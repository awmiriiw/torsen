package tor

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/awmiriiw/torsen/internal/config"
	"github.com/awmiriiw/torsen/internal/logger"
)

type BootstrapMonitor struct {
	cfg *config.Config
	log *logger.Logger
}

func NewBootstrapMonitor(cfg *config.Config, log *logger.Logger) *BootstrapMonitor {
	return &BootstrapMonitor{
		cfg: cfg,
		log: log,
	}
}

func (b *BootstrapMonitor) WaitForBootstrap(ctx context.Context) error {
	b.log.Info("Waiting for Tor to bootstrap (timeout: %ds)...", b.cfg.TorBootstrapTimeout)

	timeout := time.Duration(b.cfg.TorBootstrapTimeout) * time.Second
	deadline := time.Now().Add(timeout)

	lastProgress := -1

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return fmt.Errorf("bootstrap cancelled")
		default:
		}

		progress, summary, err := b.getBootstrapProgress()
		if err != nil {
			b.log.Debug("Could not read Tor bootstrap status yet: %v", err)
			if err := b.sleepWithContext(ctx, 2*time.Second); err != nil {
				return err
			}
			continue
		}

		if progress != lastProgress {
			b.log.Info("Tor bootstrap: %d%% - %s", progress, summary)
			lastProgress = progress
		}

		if progress >= 100 {
			b.log.Info("✅ Tor has successfully bootstrapped!")
			return nil
		}

		if err := b.sleepWithContext(ctx, 2*time.Second); err != nil {
			return err
		}
	}

	return fmt.Errorf("tor bootstrap timed out after %d seconds", b.cfg.TorBootstrapTimeout)
}

func (b *BootstrapMonitor) sleepWithContext(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("bootstrap cancelled")
	case <-time.After(d):
		return nil
	}
}

func (b *BootstrapMonitor) getBootstrapProgress() (int, string, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", b.cfg.TorControlPort)

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return 0, "", fmt.Errorf("cannot connect to control port: %w", err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return 0, "", fmt.Errorf("cannot set deadline: %w", err)
	}

	reader := bufio.NewReader(conn)

	authCmd := fmt.Sprintf("AUTHENTICATE \"%s\"\r\n", b.cfg.TorControlPassword)
	if _, err := fmt.Fprint(conn, authCmd); err != nil {
		return 0, "", fmt.Errorf("cannot send AUTHENTICATE command: %w", err)
	}

	authLine, err := reader.ReadString('\n')
	if err != nil {
		return 0, "", fmt.Errorf("cannot read auth response: %w", err)
	}

	authLine = strings.TrimSpace(authLine)
	b.log.Debug("AUTH response: %s", authLine)

	if !strings.HasPrefix(authLine, "250") {
		return 0, "", fmt.Errorf("authentication failed: %s", authLine)
	}

	if _, err := fmt.Fprint(conn, "GETINFO status/bootstrap-phase\r\n"); err != nil {
		return 0, "", fmt.Errorf("cannot send GETINFO command: %w", err)
	}

	infoLine, err := reader.ReadString('\n')
	if err != nil {
		return 0, "", fmt.Errorf("cannot read bootstrap info: %w", err)
	}

	infoLine = strings.TrimSpace(infoLine)
	b.log.Debug("Bootstrap info raw: %s", infoLine)

	okLine, err := reader.ReadString('\n')
	if err == nil {
		b.log.Debug("Bootstrap trailing line: %s", strings.TrimSpace(okLine))
	}

	_, _ = fmt.Fprint(conn, "QUIT\r\n")

	progress := extractProgress(infoLine)
	summary := extractSummary(infoLine)

	if summary == "" {
		summary = "working..."
	}

	return progress, summary, nil
}

func extractProgress(line string) int {
	idx := strings.Index(line, "PROGRESS=")
	if idx == -1 {
		return 0
	}

	remaining := line[idx+len("PROGRESS="):]
	num := ""

	for _, ch := range remaining {
		if ch >= '0' && ch <= '9' {
			num += string(ch)
		} else {
			break
		}
	}

	var progress int
	_, _ = fmt.Sscanf(num, "%d", &progress)

	return progress
}

func extractSummary(line string) string {
	idx := strings.Index(line, `SUMMARY="`)
	if idx == -1 {
		return ""
	}

	remaining := line[idx+len(`SUMMARY="`):]
	end := strings.Index(remaining, `"`)
	if end == -1 {
		return ""
	}

	return remaining[:end]
}
