package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
)

func printJSON(data interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(data)
}

func printTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	// Header
	for i, h := range headers {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, h)
	}
	fmt.Fprintln(w)

	// Separator
	for i, h := range headers {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		for range h {
			fmt.Fprint(w, "-")
		}
	}
	fmt.Fprintln(w)

	// Rows
	for _, row := range rows {
		for i, col := range row {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, col)
		}
		fmt.Fprintln(w)
	}

	w.Flush()
}

func outputResult(format string, data json.RawMessage, headers []string, rowFn func(json.RawMessage) [][]string) {
	if format == "json" {
		var parsed interface{}
		json.Unmarshal(data, &parsed)
		printJSON(parsed)
		return
	}

	rows := rowFn(data)
	printTable(headers, rows)
}
