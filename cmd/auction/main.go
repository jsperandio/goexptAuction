package main

import (
	"context"
	"log"

	"goexptauction/configuration/database/mongodb"
	"goexptauction/configuration/logger"
	"goexptauction/internal/infra/api/web/controller/auction_controller"
	"goexptauction/internal/infra/api/web/controller/bid_controller"
	"goexptauction/internal/infra/api/web/controller/user_controller"
	"goexptauction/internal/infra/database/auction"
	"goexptauction/internal/infra/database/bid"
	"goexptauction/internal/infra/database/user"
	"goexptauction/internal/usecase/auction_usecase"
	"goexptauction/internal/usecase/bid_usecase"
	"goexptauction/internal/usecase/user_usecase"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
)

func main() {
	ctx := context.Background()

	if err := godotenv.Load("cmd/auction/.env"); err != nil {
		log.Fatal("Error trying to load env variables")
		return
	}

	databaseConnection, err := mongodb.NewMongoDBConnection(ctx)
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	router := gin.Default()

	userController, bidController, auctionsController, auctionRepository := initDependencies(databaseConnection)

	// reagenda os leilões que continuam abertos no banco
	if err := auctionRepository.ScheduleActiveAuctions(ctx); err != nil {
		logger.Error("Error trying to reschedule active auctions", err)
	}

	router.GET("/auction", auctionsController.FindAuctions)
	router.GET("/auction/:auctionId", auctionsController.FindAuctionById)
	router.POST("/auction", auctionsController.CreateAuction)
	router.GET("/auction/winner/:auctionId", auctionsController.FindWinningBidByAuctionId)
	router.POST("/bid", bidController.CreateBid)
	router.GET("/bid/:auctionId", bidController.FindBidByAuctionId)
	router.GET("/user/:userId", userController.FindUserById)

	router.Run(":8080")
}

func initDependencies(database *mongo.Database) (
	userController *user_controller.UserController,
	bidController *bid_controller.BidController,
	auctionController *auction_controller.AuctionController,
	auctionRepository *auction.AuctionRepository,
) {
	auctionRepository = auction.NewAuctionRepository(database)
	bidRepository := bid.NewBidRepository(database, auctionRepository)
	userRepository := user.NewUserRepository(database)

	userController = user_controller.NewUserController(
		user_usecase.NewUserUseCase(userRepository))
	auctionController = auction_controller.NewAuctionController(
		auction_usecase.NewAuctionUseCase(auctionRepository, bidRepository))
	bidController = bid_controller.NewBidController(bid_usecase.NewBidUseCase(bidRepository))

	return
}
