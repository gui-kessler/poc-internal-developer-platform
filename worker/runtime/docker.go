package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/ixcsoft/idp/shared/spec"
)

const (
	LabelManaged  = "idp.managed"
	LabelApp      = "idp.app"
	LabelDeployID = "idp.deploy_id"

	defaultMemory = "256m"
	defaultCPUs   = "0.5"
	readyTimeout  = 15 * time.Second
)

type Docker struct {
	log        *slog.Logger
	baseDomain string
	traefikNet string
}

func NewDocker(baseDomain, traefikNet string, log *slog.Logger) (*Docker, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, fmt.Errorf("docker CLI não encontrado no PATH: %w", err)
	}
	return &Docker{baseDomain: baseDomain, traefikNet: traefikNet, log: log}, nil
}

func (d *Docker) Close() error { return nil }

// DeployApp materializa o container do app: pull → remove versão anterior →
// run com labels Traefik geradas pela plataforma + limites default + restart
// policy → aguarda transição para `running`. Em caso de falha pós-criação,
// o container parcial é removido.
func (d *Docker) DeployApp(ctx context.Context, deployID string, intent spec.Intent) (string, error) {
	name := "idp-" + intent.App
	log := d.log.With("deploy_id", deployID, "app", intent.App, "container", name)

	if intent.Replicas > 1 {
		log.Warn("replicas > 1 não suportado no runtime Docker, subindo instância única", "requested", intent.Replicas)
	}

	if err := d.run(ctx, "pull", intent.Image); err != nil {
		return "", fmt.Errorf("pull: %w", err)
	}
	log.Info("image pulled", "image", intent.Image)

	if err := d.removeIfExists(ctx, log, name); err != nil {
		return "", fmt.Errorf("remove existing: %w", err)
	}

	host := intent.App + "." + d.baseDomain
	if err := d.run(ctx, d.runArgs(deployID, name, intent, host)...); err != nil {
		return "", fmt.Errorf("run: %w", err)
	}
	log.Info("container started")

	if err := d.waitRunning(ctx, log, name); err != nil {
		_ = d.run(context.Background(), "rm", "-f", name)
		return "", fmt.Errorf("readiness: %w", err)
	}

	return "https://" + host, nil
}

func (d *Docker) runArgs(deployID, name string, intent spec.Intent, host string) []string {
	port := fmt.Sprintf("%d", intent.Port)
	app := intent.App
	return []string{
		"run", "-d",
		"--name", name,
		"--network", d.traefikNet,
		"--restart", "unless-stopped",
		"--memory", defaultMemory,
		"--cpus", defaultCPUs,
		"--label", LabelManaged + "=true",
		"--label", LabelApp + "=" + app,
		"--label", LabelDeployID + "=" + deployID,
		"--label", "traefik.enable=true",
		"--label", "traefik.docker.network=" + d.traefikNet,
		"--label", "traefik.http.routers." + app + ".rule=Host(`" + host + "`)",
		"--label", "traefik.http.routers." + app + ".entrypoints=websecure",
		"--label", "traefik.http.routers." + app + ".tls.certresolver=letsencrypt",
		"--label", "traefik.http.services." + app + ".loadbalancer.server.port=" + port,
		intent.Image,
	}
}

func (d *Docker) run(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "docker", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func (d *Docker) runOutput(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

func (d *Docker) removeIfExists(ctx context.Context, log *slog.Logger, name string) error {
	out, err := d.runOutput(ctx, "ps", "-a", "--filter", "name=^"+name+"$", "--format", "{{.Names}}")
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) == "" {
		return nil
	}
	log.Info("removendo container anterior", "name", name)
	return d.run(ctx, "rm", "-f", name)
}

type containerState struct {
	Status   string `json:"Status"`
	ExitCode int    `json:"ExitCode"`
	Error    string `json:"Error"`
}

type containerInfo struct {
	State *containerState `json:"State"`
}

func (d *Docker) waitRunning(ctx context.Context, log *slog.Logger, name string) error {
	deadline := time.Now().Add(readyTimeout)
	for time.Now().Before(deadline) {
		out, err := d.runOutput(ctx, "inspect", name)
		if err != nil {
			return err
		}
		var infos []containerInfo
		if err := json.Unmarshal([]byte(out), &infos); err != nil {
			return fmt.Errorf("parse inspect: %w", err)
		}
		if len(infos) == 0 || infos[0].State == nil {
			return errors.New("inspect retornou sem State")
		}
		s := infos[0].State
		switch s.Status {
		case "running":
			log.Info("container ready")
			return nil
		case "exited", "dead":
			return fmt.Errorf("container terminou (code=%d): %s", s.ExitCode, s.Error)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	return errors.New("timeout esperando container ficar running")
}
