package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/openalpha/perp-dex/x/orderbook/types"
)

// GetTxCmd returns the transaction commands for the orderbook module
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "orderbook",
		Short:                      "Orderbook module transaction commands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdPlaceOrder(),
		CmdCancelOrder(),
	)

	return cmd
}

// CmdPlaceOrder returns the command to place an order
func CmdPlaceOrder() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "place-order [market-id] [side] [type] [price] [quantity]",
		Short: "Place a new order",
		Long: `Place a new order in the orderbook.

Examples:
  perpdexd tx orderbook place-order BTC-USDC buy limit 50000 0.1 --from alice
  perpdexd tx orderbook place-order BTC-USDC sell market 0 0.5 --from bob`,
		Args: cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			marketID := args[0]
			sideStr := strings.ToLower(args[1])
			orderTypeStr := strings.ToLower(args[2])
			priceStr := args[3]
			quantityStr := args[4]

			// Parse side
			var side types.Side
			switch sideStr {
			case "buy":
				side = types.SideBuy
			case "sell":
				side = types.SideSell
			default:
				return fmt.Errorf("invalid side: %s (use 'buy' or 'sell')", sideStr)
			}

			// Parse order type
			var orderType types.OrderType
			switch orderTypeStr {
			case "limit":
				orderType = types.OrderTypeLimit
			case "market":
				orderType = types.OrderTypeMarket
			default:
				return fmt.Errorf("invalid order type: %s (use 'limit' or 'market')", orderTypeStr)
			}

			price, err := strconv.ParseFloat(priceStr, 64)
			if err != nil {
				return fmt.Errorf("invalid price: %v", err)
			}

			quantity, err := strconv.ParseFloat(quantityStr, 64)
			if err != nil {
				return fmt.Errorf("invalid quantity: %v", err)
			}

			msg := &types.MsgPlaceOrder{
				Trader:    clientCtx.GetFromAddress().String(),
				MarketId:  marketID,
				Side:      side,
				OrderType: orderType,
				Price:     fmt.Sprintf("%f", price),
				Quantity:  fmt.Sprintf("%f", quantity),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdCancelOrder returns the command to cancel an order
func CmdCancelOrder() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel-order [order-id]",
		Short: "Cancel an existing order",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgCancelOrder{
				Trader:  clientCtx.GetFromAddress().String(),
				OrderId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
