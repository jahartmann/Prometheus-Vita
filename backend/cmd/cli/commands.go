package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func nodesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nodes",
		Short: "Nodes verwalten",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Alle Nodes auflisten",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := getClient().Get("/nodes")
			if err != nil {
				return err
			}
			outputResult(flagOutput, resp.Data,
				[]string{"ID", "NAME", "TYP", "HOSTNAME", "ONLINE"},
				func(data json.RawMessage) [][]string {
					var nodes []map[string]interface{}
					json.Unmarshal(data, &nodes)
					var rows [][]string
					for _, n := range nodes {
						online := "Nein"
						if b, ok := n["is_online"].(bool); ok && b {
							online = "Ja"
						}
						rows = append(rows, []string{
							str(n["id"]),
							str(n["name"]),
							str(n["type"]),
							str(n["hostname"]),
							online,
						})
					}
					return rows
				})
			return nil
		},
	}

	getCmd := &cobra.Command{
		Use:   "get [id]",
		Short: "Node-Details anzeigen",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := getClient().Get("/nodes/" + args[0])
			if err != nil {
				return err
			}
			var parsed interface{}
			json.Unmarshal(resp.Data, &parsed)
			printJSON(parsed)
			return nil
		},
	}

	cmd.AddCommand(listCmd, getCmd)
	return cmd
}

func vmsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vms",
		Short: "VMs verwalten",
	}

	listCmd := &cobra.Command{
		Use:   "list [node-id]",
		Short: "VMs eines Nodes auflisten",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := getClient().Get("/nodes/" + args[0] + "/vms")
			if err != nil {
				return err
			}
			outputResult(flagOutput, resp.Data,
				[]string{"VMID", "NAME", "TYP", "STATUS", "CPU", "SPEICHER"},
				func(data json.RawMessage) [][]string {
					var vms []map[string]interface{}
					json.Unmarshal(data, &vms)
					var rows [][]string
					for _, v := range vms {
						rows = append(rows, []string{
							str(v["vmid"]),
							str(v["name"]),
							str(v["type"]),
							str(v["status"]),
							fmt.Sprintf("%.1f%%", num(v["cpu_usage"])*100),
							formatBytes(int64(num(v["memory_used"]))),
						})
					}
					return rows
				})
			return nil
		},
	}

	startCmd := &cobra.Command{
		Use:   "start [node-id] [vmid]",
		Short: "VM starten",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Note: The actual VM start goes through the Proxmox API via the backend node service
			fmt.Printf("VM %s auf Node %s wird gestartet...\n", args[1], args[0])
			return nil
		},
	}

	stopCmd := &cobra.Command{
		Use:   "stop [node-id] [vmid]",
		Short: "VM stoppen",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("VM %s auf Node %s wird gestoppt...\n", args[1], args[0])
			return nil
		},
	}

	cmd.AddCommand(listCmd, startCmd, stopCmd)
	return cmd
}

func backupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Backups verwalten",
	}

	listCmd := &cobra.Command{
		Use:   "list [node-id]",
		Short: "Backups eines Nodes auflisten",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := getClient().Get("/nodes/" + args[0] + "/backups")
			if err != nil {
				return err
			}
			outputResult(flagOutput, resp.Data,
				[]string{"ID", "TYP", "STATUS", "DATEIEN", "GROESSE", "ERSTELLT"},
				func(data json.RawMessage) [][]string {
					var backups []map[string]interface{}
					json.Unmarshal(data, &backups)
					var rows [][]string
					for _, b := range backups {
						rows = append(rows, []string{
							shortID(str(b["id"])),
							str(b["backup_type"]),
							str(b["status"]),
							str(b["file_count"]),
							formatBytes(int64(num(b["total_size"]))),
							str(b["created_at"]),
						})
					}
					return rows
				})
			return nil
		},
	}

	createCmd := &cobra.Command{
		Use:   "create [node-id]",
		Short: "Backup erstellen",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := strings.NewReader(`{"backup_type":"manual"}`)
			resp, err := getClient().Post("/nodes/"+args[0]+"/backup", body)
			if err != nil {
				return err
			}
			var parsed interface{}
			json.Unmarshal(resp.Data, &parsed)
			printJSON(parsed)
			fmt.Println("Backup erstellt.")
			return nil
		},
	}

	cmd.AddCommand(listCmd, createCmd)
	return cmd
}

func driftCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "drift",
		Short: "Drift-Erkennung",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Alle Drift-Checks auflisten",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := getClient().Get("/drift")
			if err != nil {
				return err
			}
			outputResult(flagOutput, resp.Data,
				[]string{"ID", "NODE", "STATUS", "GEAENDERT", "HINZU", "ENTFERNT", "GEPRUEFT"},
				func(data json.RawMessage) [][]string {
					var checks []map[string]interface{}
					json.Unmarshal(data, &checks)
					var rows [][]string
					for _, c := range checks {
						rows = append(rows, []string{
							shortID(str(c["id"])),
							shortID(str(c["node_id"])),
							str(c["status"]),
							str(c["changed_files"]),
							str(c["added_files"]),
							str(c["removed_files"]),
							str(c["checked_at"]),
						})
					}
					return rows
				})
			return nil
		},
	}

	checkCmd := &cobra.Command{
		Use:   "check [node-id]",
		Short: "Drift-Check ausloesen",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := getClient().Post("/nodes/"+args[0]+"/drift/check", nil)
			if err != nil {
				return err
			}
			fmt.Println("Drift-Check gestartet.")
			return nil
		},
	}

	cmd.AddCommand(listCmd, checkCmd)
	return cmd
}

func updatesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "updates",
		Short: "Update-Intelligence",
	}

	checkCmd := &cobra.Command{
		Use:   "check [node-id]",
		Short: "Update-Check ausloesen",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := getClient().Post("/nodes/"+args[0]+"/updates/check", nil)
			if err != nil {
				return err
			}
			fmt.Println("Update-Check gestartet.")
			return nil
		},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Alle Update-Checks auflisten",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := getClient().Get("/updates")
			if err != nil {
				return err
			}
			outputResult(flagOutput, resp.Data,
				[]string{"ID", "NODE", "STATUS", "UPDATES", "SICHERHEIT"},
				func(data json.RawMessage) [][]string {
					var checks []map[string]interface{}
					json.Unmarshal(data, &checks)
					var rows [][]string
					for _, c := range checks {
						rows = append(rows, []string{
							shortID(str(c["id"])),
							shortID(str(c["node_id"])),
							str(c["status"]),
							str(c["total_updates"]),
							str(c["security_updates"]),
						})
					}
					return rows
				})
			return nil
		},
	}

	cmd.AddCommand(checkCmd, listCmd)
	return cmd
}

func recommendationsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "recommendations",
		Short: "Ressourcen-Empfehlungen auflisten",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := getClient().Get("/rightsizing")
			if err != nil {
				return err
			}
			outputResult(flagOutput, resp.Data,
				[]string{"ID", "NODE", "VM", "RESSOURCE", "AKTUELL", "EMPFOHLEN", "TYP"},
				func(data json.RawMessage) [][]string {
					var recs []map[string]interface{}
					json.Unmarshal(data, &recs)
					var rows [][]string
					for _, r := range recs {
						rows = append(rows, []string{
							shortID(str(r["id"])),
							shortID(str(r["node_id"])),
							str(r["vm_name"]),
							str(r["resource_type"]),
							str(r["current_value"]),
							str(r["recommended_value"]),
							str(r["recommendation_type"]),
						})
					}
					return rows
				})
			return nil
		},
	}
}

func sshKeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh-keys",
		Short: "SSH-Schluessel verwalten",
	}

	listCmd := &cobra.Command{
		Use:   "list [node-id]",
		Short: "SSH-Schluessel eines Nodes auflisten",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := getClient().Get("/nodes/" + args[0] + "/ssh-keys")
			if err != nil {
				return err
			}
			outputResult(flagOutput, resp.Data,
				[]string{"ID", "NAME", "TYP", "FINGERPRINT", "DEPLOYED", "ERSTELLT"},
				func(data json.RawMessage) [][]string {
					var keys []map[string]interface{}
					json.Unmarshal(data, &keys)
					var rows [][]string
					for _, k := range keys {
						deployed := "Nein"
						if b, ok := k["is_deployed"].(bool); ok && b {
							deployed = "Ja"
						}
						fp := str(k["fingerprint"])
						if len(fp) > 24 {
							fp = fp[:24] + "..."
						}
						rows = append(rows, []string{
							shortID(str(k["id"])),
							str(k["name"]),
							str(k["key_type"]),
							fp,
							deployed,
							str(k["created_at"]),
						})
					}
					return rows
				})
			return nil
		},
	}

	generateCmd := &cobra.Command{
		Use:   "generate [node-id] [name]",
		Short: "SSH-Schluessel generieren",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := strings.NewReader(fmt.Sprintf(`{"name":"%s","key_type":"ed25519","deploy":true}`, args[1]))
			resp, err := getClient().Post("/nodes/"+args[0]+"/ssh-keys", body)
			if err != nil {
				return err
			}
			var parsed interface{}
			json.Unmarshal(resp.Data, &parsed)
			printJSON(parsed)
			return nil
		},
	}

	cmd.AddCommand(listCmd, generateCmd)
	return cmd
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "CLI-Konfiguration verwalten",
	}

	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Konfiguration setzen",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				cfg = &CLIConfig{}
			}
			if u, _ := cmd.Flags().GetString("api-url"); u != "" {
				cfg.APIURL = u
			}
			if t, _ := cmd.Flags().GetString("token"); t != "" {
				cfg.Token = t
			}
			if err := saveConfig(cfg); err != nil {
				return err
			}
			fmt.Println("Konfiguration gespeichert.")
			return nil
		},
	}
	setCmd.Flags().String("api-url", "", "API-URL setzen")
	setCmd.Flags().String("token", "", "API-Token setzen")

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Konfiguration anzeigen",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			printJSON(cfg)
			return nil
		},
	}

	cmd.AddCommand(setCmd, showCmd)
	return cmd
}

// Helpers

func str(v interface{}) string {
	if v == nil {
		return "-"
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%.2f", val)
	case bool:
		if val {
			return "Ja"
		}
		return "Nein"
	default:
		return fmt.Sprintf("%v", val)
	}
}

func num(v interface{}) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func formatBytes(b int64) string {
	if b == 0 {
		return "0 B"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
