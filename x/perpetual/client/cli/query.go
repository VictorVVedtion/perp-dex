package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
)

// MarketInfo is a CLI-friendly market info struct
type MarketInfo struct {
	MarketID              string `json:"market_id"`
	BaseAsset             string `json:"base_asset"`
	QuoteAsset            string `json:"quote_asset"`
	MaxLeverage           string `json:"max_leverage"`
	InitialMarginRate     string `json:"initial_margin_rate"`
	MaintenanceMarginRate string `json:"maintenance_margin_rate"`
	TakerFeeRate          string `json:"taker_fee_rate"`
	MakerFeeRate          string `json:"maker_fee_rate"`
}

// FundingInfo is a CLI-friendly funding info struct
type FundingInfo struct {
	MarketID         string `json:"market_id"`
	CurrentRate      string `json:"current_rate"`
	PredictedPayment string `json:"predicted_payment"`
	NextSettlement   string `json:"next_settlement"`
	LastSettlement   string `json:"last_settlement"`
}

// GetQueryCmd returns the cli query commands for the perpetual module
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "perpetual",
		Short:                      "Querying commands for the perpetual module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdQueryMarket(),
		CmdQueryMarkets(),
		CmdQueryPrice(),
		CmdQueryFunding(),
		CmdQueryAccount(),
		CmdQueryPosition(),
		CmdQueryAllPositions(),
	)

	return cmd
}

// CmdQueryMarket returns the command to query market info
func CmdQueryMarket() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "market [market-id]",
		Short: "Query market information",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			marketID := args[0]

			market, ok := findMarket(sampleMarkets(), marketID)
			if !ok {
				return fmt.Errorf("market not found: %s", marketID)
			}

			output, _ := json.MarshalIndent(market, "", "  ")
			fmt.Println(string(output))
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryMarkets returns the command to query all markets
func CmdQueryMarkets() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "markets",
		Short: "Query all markets",
		RunE: func(cmd *cobra.Command, args []string) error {
			markets := sampleMarkets()
			output, _ := json.MarshalIndent(markets, "", "  ")
			fmt.Println(string(output))
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryPrice returns the command to query current price
func CmdQueryPrice() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "price [market-id]",
		Short: "Query current price for a market",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// For MVP testing, we'll query via RPC
			fmt.Println("Price query requires running node connection")
			fmt.Println("Use REST API: GET /perpdex/perpetual/v1/price/{market_id}")
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryFunding returns the command to query funding info
func CmdQueryFunding() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "funding [market-id]",
		Short: "Query funding rate information for a market",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			marketID := args[0]

			info := FundingInfo{
				MarketID:         marketID,
				CurrentRate:      "0.0008",
				PredictedPayment: "40.0",
				NextSettlement:   "2026-01-18T08:00:00Z",
				LastSettlement:   "2026-01-18T00:00:00Z",
			}

			output, _ := json.MarshalIndent(info, "", "  ")
			fmt.Println(string(output))
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryAccount returns the command to query account info
func CmdQueryAccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account [address]",
		Short: "Query account balance and margin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Account query requires running node connection")
			fmt.Println("Use REST API: GET /perpdex/perpetual/v1/account/{address}")
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryPosition returns the command to query a position
func CmdQueryPosition() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "position [trader] [market-id]",
		Short: "Query a specific position",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Position query requires running node connection")
			fmt.Println("Use REST API: GET /perpdex/perpetual/v1/position/{trader}/{market_id}")
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func sampleMarkets() []MarketInfo {
	return []MarketInfo{
		{
			MarketID:              "BTC-USDC",
			BaseAsset:             "BTC",
			QuoteAsset:            "USDC",
			MaxLeverage:           "50",
			InitialMarginRate:     "0.05",
			MaintenanceMarginRate: "0.025",
			TakerFeeRate:          "0.0005",
			MakerFeeRate:          "0.0002",
		},
		{
			MarketID:              "ETH-USDC",
			BaseAsset:             "ETH",
			QuoteAsset:            "USDC",
			MaxLeverage:           "50",
			InitialMarginRate:     "0.05",
			MaintenanceMarginRate: "0.025",
			TakerFeeRate:          "0.0005",
			MakerFeeRate:          "0.0002",
		},
		{
			MarketID:              "SOL-USDC",
			BaseAsset:             "SOL",
			QuoteAsset:            "USDC",
			MaxLeverage:           "25",
			InitialMarginRate:     "0.05",
			MaintenanceMarginRate: "0.025",
			TakerFeeRate:          "0.0005",
			MakerFeeRate:          "0.0002",
		},
	}
}

func findMarket(markets []MarketInfo, marketID string) (MarketInfo, bool) {
	for _, market := range markets {
		if market.MarketID == marketID {
			return market, true
		}
	}
	return MarketInfo{}, false
}

// CmdQueryAllPositions returns the command to query all positions
func CmdQueryAllPositions() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "positions",
		Short: "Query all positions",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("All positions query requires running node connection")
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
