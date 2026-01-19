package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
)

// GetQueryCmd returns the cli query commands for the clearinghouse module
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "clearinghouse",
		Short:                      "Querying commands for the clearinghouse module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdQueryLiquidations(),
		CmdQueryPositionHealth(),
		CmdQueryAtRiskPositions(),
	)

	return cmd
}

// CmdQueryLiquidations returns the command to query recent liquidations
func CmdQueryLiquidations() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "liquidations",
		Short: "Query recent liquidations",
		RunE: func(cmd *cobra.Command, args []string) error {
			// For MVP demo, return sample liquidations
			liquidations := []map[string]interface{}{
				{
					"liquidation_id": "liq-1",
					"trader":         "cosmos1abc...",
					"market_id":      "BTC-USDC",
					"position_side":  "long",
					"size":           "0.5",
					"entry_price":    "50000",
					"mark_price":     "47000",
					"status":         "executed",
					"timestamp":      "2026-01-18T08:30:00Z",
				},
			}

			output, _ := json.MarshalIndent(liquidations, "", "  ")
			fmt.Println(string(output))
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryPositionHealth returns the command to query position health
func CmdQueryPositionHealth() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health [trader] [market-id]",
		Short: "Query health status of a position",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			trader := args[0]
			marketID := args[1]

			// For MVP demo
			health := map[string]interface{}{
				"trader":             trader,
				"market_id":          marketID,
				"margin_ratio":       "0.12",
				"maintenance_margin": "500",
				"account_equity":     "1200",
				"is_healthy":         true,
				"at_risk":            false,
			}

			output, _ := json.MarshalIndent(health, "", "  ")
			fmt.Println(string(output))
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryAtRiskPositions returns the command to query at-risk positions
func CmdQueryAtRiskPositions() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "at-risk",
		Short: "Query all at-risk positions",
		RunE: func(cmd *cobra.Command, args []string) error {
			// For MVP demo, return empty list (no at-risk positions)
			positions := []map[string]interface{}{}

			output, _ := json.MarshalIndent(map[string]interface{}{
				"at_risk_positions": positions,
				"count":             0,
			}, "", "  ")
			fmt.Println(string(output))
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
