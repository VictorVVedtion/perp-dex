package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/openalpha/perp-dex/x/perpetual/types"
)

// GetTxCmd returns the transaction commands for the perpetual module
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "perpetual",
		Short:                      "Perpetual module transaction commands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdDeposit(),
		CmdWithdraw(),
	)

	return cmd
}

// CmdDeposit returns the command to deposit margin
func CmdDeposit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deposit [amount]",
		Short: "Deposit margin to trading account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			amount, err := strconv.ParseFloat(args[0], 64)
			if err != nil {
				return fmt.Errorf("invalid amount: %v", err)
			}

			msg := &types.MsgDeposit{
				Trader: clientCtx.GetFromAddress().String(),
				Amount: fmt.Sprintf("%f", amount),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdWithdraw returns the command to withdraw margin
func CmdWithdraw() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "withdraw [amount]",
		Short: "Withdraw margin from trading account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			amount, err := strconv.ParseFloat(args[0], 64)
			if err != nil {
				return fmt.Errorf("invalid amount: %v", err)
			}

			msg := &types.MsgWithdraw{
				Trader: clientCtx.GetFromAddress().String(),
				Amount: fmt.Sprintf("%f", amount),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
