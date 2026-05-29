// ABOUTME: beaconctl is a small admin CLI to migrate the DB and inspect jobs/machines.
// ABOUTME: Used for manual verification of the data layer before the agent/MCP exist.

// Command beaconctl is the Beacon admin CLI over the store package.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/notpritam/beacon/internal/config"
	"github.com/notpritam/beacon/internal/store"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: beaconctl <migrate|enqueue|get|machines> [args]")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer st.Close()

	switch args[0] {
	case "migrate":
		if err := st.Migrate(ctx); err != nil {
			return err
		}
		fmt.Println("migrated")
		return nil
	case "machines":
		machines, err := st.ListMachines(ctx)
		if err != nil {
			return err
		}
		out, err := json.MarshalIndent(machines, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal machines: %w", err)
		}
		fmt.Println(string(out))
		return nil
	case "enqueue":
		if len(args) < 3 {
			return fmt.Errorf("usage: beaconctl enqueue <machine-name> <shell-cmd>")
		}
		m, err := st.MachineByName(ctx, args[1])
		if err != nil {
			return err
		}
		payload, err := json.Marshal(map[string]string{"cmd": args[2]})
		if err != nil {
			return fmt.Errorf("marshal payload: %w", err)
		}
		j, err := st.EnqueueJob(ctx, m.ID, store.JobShell, payload, 0, nil, "beaconctl")
		if err != nil {
			return err
		}
		fmt.Println("enqueued job", j.ID)
		return nil
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: beaconctl get <job-id>")
		}
		j, err := st.GetJob(ctx, args[1])
		if err != nil {
			return err
		}
		out, err := json.MarshalIndent(j, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal job: %w", err)
		}
		fmt.Println(string(out))
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}
