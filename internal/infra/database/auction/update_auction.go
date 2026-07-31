package auction

import (
	"context"
	"fmt"
	"time"

	"goexptauction/configuration/logger"
	"goexptauction/internal/entity/auction_entity"
	"goexptauction/internal/internal_error"

	"go.mongodb.org/mongo-driver/bson"
)

func (ar *AuctionRepository) CloseAuction(ctx context.Context, auctionId string) *internal_error.InternalError {
	filter := bson.M{
		"_id":    auctionId,
		"status": auction_entity.Active,
	}
	update := bson.M{
		"$set": bson.M{
			"status": auction_entity.Completed,
		},
	}

	result, err := ar.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		logger.Error(
			fmt.Sprintf("Error trying to close auction with id = %s", auctionId), err)
		return internal_error.NewInternalServerError("Error trying to close auction")
	}

	if result.ModifiedCount == 0 {
		logger.Info(fmt.Sprintf("Auction %s was already closed", auctionId))
		return nil
	}

	logger.Info(fmt.Sprintf("Auction %s closed automatically", auctionId))
	return nil
}

func (ar *AuctionRepository) ScheduleActiveAuctions(ctx context.Context) *internal_error.InternalError {
	cursor, err := ar.Collection.Find(ctx, bson.M{
		"status": auction_entity.Active,
	})
	if err != nil {
		logger.Error("Error trying to find active auctions", err)
		return internal_error.NewInternalServerError("Error trying to find active auctions")
	}
	defer cursor.Close(ctx)

	var auctionsMongo []AuctionEntityMongo
	if err := cursor.All(ctx, &auctionsMongo); err != nil {
		logger.Error("Error decoding active auctions", err)
		return internal_error.NewInternalServerError("Error decoding active auctions")
	}

	for _, auctionMongo := range auctionsMongo {
		endsAt := time.Unix(auctionMongo.Timestamp, 0).Add(ar.auctionInterval)
		ar.scheduleAuctionClose(auctionMongo.Id, endsAt)
	}

	if len(auctionsMongo) > 0 {
		logger.Info(fmt.Sprintf("Rescheduled %d active auction(s)", len(auctionsMongo)))
	}

	return nil
}
