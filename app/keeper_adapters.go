package app

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	orderbookkeeper "github.com/openalpha/perp-dex/x/orderbook/keeper"
	orderbooktypes "github.com/openalpha/perp-dex/x/orderbook/types"
	perpetualkeeper "github.com/openalpha/perp-dex/x/perpetual/keeper"
	perpetualtypes "github.com/openalpha/perp-dex/x/perpetual/types"
)

type orderbookPerpetualAdapter struct {
	keeper *perpetualkeeper.Keeper
}

func newOrderbookPerpetualAdapter(keeper *perpetualkeeper.Keeper) orderbookkeeper.PerpetualKeeper {
	return orderbookPerpetualAdapter{keeper: keeper}
}

func (a orderbookPerpetualAdapter) GetMarket(ctx sdk.Context, marketID string) *orderbookkeeper.Market {
	if a.keeper == nil {
		return nil
	}

	market := a.keeper.GetMarket(ctx, marketID)
	if market == nil {
		return nil
	}

	return &orderbookkeeper.Market{
		MarketID:      market.MarketID,
		TakerFeeRate:  market.TakerFeeRate,
		MakerFeeRate:  market.MakerFeeRate,
		InitialMargin: market.InitialMarginRate,
	}
}

func (a orderbookPerpetualAdapter) GetMarkPrice(ctx sdk.Context, marketID string) (math.LegacyDec, bool) {
	if a.keeper == nil {
		return math.LegacyZeroDec(), false
	}

	priceInfo := a.keeper.GetPrice(ctx, marketID)
	if priceInfo == nil {
		return math.LegacyZeroDec(), false
	}

	return priceInfo.MarkPrice, true
}

func (a orderbookPerpetualAdapter) UpdatePosition(ctx sdk.Context, trader, marketID string, side orderbooktypes.Side, qty, price, fee interface{}) error {
	if a.keeper == nil {
		return fmt.Errorf("perpetual keeper not set")
	}

	qtyDec, err := parseLegacyDec(qty)
	if err != nil {
		return err
	}
	priceDec, err := parseLegacyDec(price)
	if err != nil {
		return err
	}
	feeDec, err := parseLegacyDecOptional(fee)
	if err != nil {
		return err
	}

	isBuy, err := orderSideIsBuy(side)
	if err != nil {
		return err
	}

	pm := perpetualkeeper.NewPositionManager(a.keeper)
	return pm.UpdatePositionFromTrade(ctx, trader, marketID, isBuy, qtyDec, priceDec, feeDec)
}

func (a orderbookPerpetualAdapter) CheckMarginRequirement(ctx sdk.Context, trader, marketID string, side orderbooktypes.Side, qty, price interface{}) error {
	if a.keeper == nil {
		return fmt.Errorf("perpetual keeper not set")
	}

	qtyDec, err := parseLegacyDec(qty)
	if err != nil {
		return err
	}
	priceDec, err := parseLegacyDec(price)
	if err != nil {
		return err
	}

	positionSide, err := mapOrderSide(side)
	if err != nil {
		return err
	}

	return a.keeper.CheckMarginRequirement(ctx, trader, marketID, positionSide, qtyDec, priceDec)
}

func parseLegacyDec(value interface{}) (math.LegacyDec, error) {
	switch v := value.(type) {
	case math.LegacyDec:
		return v, nil
	case *math.LegacyDec:
		if v == nil {
			return math.LegacyZeroDec(), fmt.Errorf("nil decimal")
		}
		return *v, nil
	default:
		return math.LegacyZeroDec(), fmt.Errorf("unsupported decimal type: %T", value)
	}
}

func parseLegacyDecOptional(value interface{}) (math.LegacyDec, error) {
	if value == nil {
		return math.LegacyZeroDec(), nil
	}
	return parseLegacyDec(value)
}

func orderSideIsBuy(side orderbooktypes.Side) (bool, error) {
	switch side {
	case orderbooktypes.SideBuy:
		return true, nil
	case orderbooktypes.SideSell:
		return false, nil
	default:
		return false, fmt.Errorf("unsupported side: %s", side.String())
	}
}

func mapOrderSide(side orderbooktypes.Side) (perpetualtypes.PositionSide, error) {
	switch side {
	case orderbooktypes.SideBuy:
		return perpetualtypes.PositionSideLong, nil
	case orderbooktypes.SideSell:
		return perpetualtypes.PositionSideShort, nil
	default:
		return perpetualtypes.PositionSideUnspecified, fmt.Errorf("unsupported side: %s", side.String())
	}
}
