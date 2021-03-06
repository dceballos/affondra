package affondra

import (
	"fmt"

	"github.com/EG-easy/affondra/x/affondra/keeper"
	"github.com/EG-easy/affondra/x/affondra/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	NFT "github.com/cosmos/modules/incubator/nft"
)

func handleMsgBuyItem(ctx sdk.Context, k keeper.Keeper, msg types.MsgBuyItem) (*sdk.Result, error) {

	fmt.Print("===start handle BuyMsg===\n")

	item, err := k.GetItem(ctx, msg.ID)
	// check if item is exsit
	if err != nil {
		return nil, err
	}

	nft, err := k.NFTKeeper.GetNFT(ctx, item.Denom, item.NftId)

	//check if nft is exist
	if err != nil {
		return nil, err
	}
	// check if owner is correct
	if !item.Creator.Equals(nft.GetOwner()) {
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "Incorrect Owner")
	}
	//check if on sale
	if !item.InSale {
		return nil, sdkerrors.Wrap(types.ErrOutOfSale, "Alrady out of sale")
	}
	// check if the receiver has enough amount
	coins := k.CoinKeeper.GetCoins(ctx, msg.Receiver)

	if !coins.AmountOf(item.Price.Denom).GT(item.Price.Amount) {
		return nil, sdkerrors.Wrap(types.ErrNotEnoughCoin, "Shortage amount of coin to buy")
	}

	// update item receiver/insale info
	item.SetReceiver(msg.Receiver)
	item.ChangeInSaleStatus()

	k.SetItem(ctx, item)

	// update NFT owner
	nft.SetOwner(msg.Receiver)
	// update the NFT (owners are updated within the keeper)
	err = k.NFTKeeper.UpdateNFT(ctx, item.Denom, nft)
	if err != nil {
		return nil, err
	}

	// pay for referral fee
	referralFee := sdk.NewCoins(item.Affiliate)
	if err := k.CoinKeeper.SendCoins(ctx, msg.Receiver, msg.IntroducedBy, referralFee); err != nil {
		return nil, err
	}
	// payment to creator
	payment, _ := sdk.NewCoins(item.Price).SafeSub(referralFee)
	if err := k.CoinKeeper.SendCoins(ctx, msg.Receiver, item.Creator, payment); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeBuyItem,
			sdk.NewAttribute(types.AttributeKeyID, item.ID),
			sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver.String()),
		),
		sdk.NewEvent(
			NFT.EventTypeTransfer,
			sdk.NewAttribute(NFT.AttributeKeyRecipient, msg.Receiver.String()),
			sdk.NewAttribute(NFT.AttributeKeyDenom, item.Denom),
			sdk.NewAttribute(NFT.AttributeKeyNFTID, item.NftId),
		),
	})

	return &sdk.Result{Events: ctx.EventManager().Events()}, nil
}
