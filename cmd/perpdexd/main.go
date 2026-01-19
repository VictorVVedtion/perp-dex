package main

import (
	"os"

	"cosmossdk.io/log"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/openalpha/perp-dex/app"
	"github.com/openalpha/perp-dex/cmd/perpdexd/cmd"
)

func main() {
	rootCmd := cmd.NewRootCmd()
	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		log.NewLogger(os.Stderr).Error("failure when running app", "err", err)
		os.Exit(1)
	}
}
