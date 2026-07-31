package auction

import (
	"context"
	"fmt"
	"os"
	"time"

	"goexptauction/configuration/logger"
	"goexptauction/internal/entity/auction_entity"
	"goexptauction/internal/internal_error"

	"go.mongodb.org/mongo-driver/mongo"
)

type AuctionEntityMongo struct {
	Id          string                          `bson:"_id"`
	ProductName string                          `bson:"product_name"`
	Category    string                          `bson:"category"`
	Description string                          `bson:"description"`
	Condition   auction_entity.ProductCondition `bson:"condition"`
	Status      auction_entity.AuctionStatus    `bson:"status"`
	Timestamp   int64                           `bson:"timestamp"`
}
type AuctionRepository struct {
	Collection      *mongo.Collection
	auctionInterval time.Duration
}

func NewAuctionRepository(database *mongo.Database) *AuctionRepository {
	return &AuctionRepository{
		Collection:      database.Collection("auctions"),
		auctionInterval: getAuctionInterval(),
	}
}

func (ar *AuctionRepository) CreateAuction(ctx context.Context, auctionEntity *auction_entity.Auction) *internal_error.InternalError {
	auctionEntityMongo := &AuctionEntityMongo{
		Id:          auctionEntity.Id,
		ProductName: auctionEntity.ProductName,
		Category:    auctionEntity.Category,
		Description: auctionEntity.Description,
		Condition:   auctionEntity.Condition,
		Status:      auctionEntity.Status,
		Timestamp:   auctionEntity.Timestamp.Unix(),
	}
	_, err := ar.Collection.InsertOne(ctx, auctionEntityMongo)
	if err != nil {
		logger.Error("Error trying to insert auction", err)
		return internal_error.NewInternalServerError("Error trying to insert auction")
	}

	ar.scheduleAuctionClose(auctionEntity.Id, auctionEntity.Timestamp.Add(ar.auctionInterval))

	return nil
}

func (ar *AuctionRepository) scheduleAuctionClose(auctionId string, closesAt time.Time) {
	go closeAuctionAfter(closesAt, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := ar.CloseAuction(ctx, auctionId); err != nil {
			logger.Error(
				fmt.Sprintf("Error trying to close auction with id = %s", auctionId), err)
		}
	})
}

func closeAuctionAfter(closesAt time.Time, closeFn func()) {
	if left := time.Until(closesAt); left > 0 {
		timer := time.NewTimer(left)
		defer timer.Stop()
		<-timer.C
	}

	closeFn()
}

func getAuctionInterval() time.Duration {
	auctionInterval := os.Getenv("AUCTION_INTERVAL")
	duration, err := time.ParseDuration(auctionInterval)
	if err != nil {
		return time.Minute * 5
	}

	return duration
}
