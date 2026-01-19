package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
)

// GetQueryCmd returns the cli query commands for the orderbook module
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "orderbook",
		Short:                      "Querying commands for the orderbook module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdQueryOrderbook(),
		CmdQueryOrder(),
		CmdQueryOrders(),
	)

	return cmd
}

// CmdQueryOrderbook returns the command to query orderbook
func CmdQueryOrderbook() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "book [market-id]",
		Short: "Query orderbook for a market",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			marketID := args[0]

			// For MVP demo, return sample orderbook
			orderbook := map[string]interface{}{
				"market_id": marketID,
				"bids": []map[string]string{
					{"price": "49500.00", "quantity": "0.5"},
					{"price": "49400.00", "quantity": "1.2"},
					{"price": "49300.00", "quantity": "0.8"},
				},
				"asks": []map[string]string{
					{"price": "50500.00", "quantity": "0.6"},
					{"price": "50600.00", "quantity": "1.0"},
					{"price": "50700.00", "quantity": "0.4"},
				},
				"timestamp": "2026-01-18T09:00:00Z",
			}

			output, _ := json.MarshalIndent(orderbook, "", "  ")
			fmt.Println(string(output))
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryOrder returns the command to query a specific order
func CmdQueryOrder() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "order [order-id]",
		Short: "Query a specific order by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			orderID := args[0]
			fmt.Printf("Order query for ID: %s requires running node connection\n", orderID)
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryOrders returns the command to query orders by trader
func CmdQueryOrders() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "orders [trader]",
		Short: "Query all orders for a trader",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			trader := args[0]
			fmt.Printf("Orders query for trader: %s requires running node connection\n", trader)
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
