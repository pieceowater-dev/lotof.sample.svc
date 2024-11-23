package main

import (
	"app/internal/core/cfg"
	"ariga.io/atlas-provider-gorm/gormschema"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	sb := &strings.Builder{}
	loadModels(sb)

	_, err := io.WriteString(os.Stdout, sb.String())
	if err != nil {
		return
	}
}

func loadModels(sb *strings.Builder) {
	stmts, err := gormschema.New("postgres").Load(cfg.Inst().PostgresModels...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load gorm schema: %v\n", err)
		os.Exit(1)
	}
	sb.WriteString(stmts)
	sb.WriteString(";\n")
}
