package auction

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"goexptauction/internal/entity/auction_entity"
)

const defaultTestMongoURL = "mongodb://admin:admin@localhost:27017/?authSource=admin"

var (
	testMongoOnce   sync.Once
	testMongoClient *mongo.Client
	testMongoURL    string
	testMongoErr    error
)

func testMongoClientOrSkip(t *testing.T) *mongo.Client {
	t.Helper()

	testMongoOnce.Do(func() {
		testMongoURL = os.Getenv("MONGODB_URL")
		if testMongoURL == "" {
			testMongoURL = defaultTestMongoURL
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		client, err := mongo.Connect(ctx, options.Client().ApplyURI(testMongoURL))
		if err != nil {
			testMongoErr = err

			return
		}

		if err := client.Ping(ctx, nil); err != nil {
			testMongoErr = err

			return
		}

		testMongoClient = client
	})

	if testMongoErr != nil {
		t.Skipf("MongoDB indisponível em %s (%v). Suba com: docker compose up -d mongodb",
			testMongoURL, testMongoErr)
	}

	return testMongoClient
}

func newTestDatabase(t *testing.T) *mongo.Database {
	t.Helper()

	dbName := fmt.Sprintf("auctions_test_%d_%d", os.Getpid(), time.Now().UnixNano())
	database := testMongoClientOrSkip(t).Database(dbName)

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		assert.NoError(t, database.Drop(ctx))
	})

	return database
}

func newTestAuctionRepository(t *testing.T, auctionInterval time.Duration) *AuctionRepository {
	t.Helper()

	t.Setenv("AUCTION_INTERVAL", auctionInterval.String())

	return NewAuctionRepository(newTestDatabase(t))
}

func newTestAuction(t *testing.T) *auction_entity.Auction {
	t.Helper()

	auction, err := auction_entity.CreateAuction(
		"Notebook",
		"Eletronicos",
		"Notebook usado em bom estado de conservacao",
		auction_entity.New)
	require.Nil(t, err)

	return auction
}

func findStatus(t *testing.T, repo *AuctionRepository, auctionId string) auction_entity.AuctionStatus {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var auctionMongo AuctionEntityMongo
	require.NoError(t, repo.Collection.FindOne(ctx, bson.M{"_id": auctionId}).Decode(&auctionMongo))

	return auctionMongo.Status
}

func Test_AuctionRepository_CreateAuction(t *testing.T) {
	ctx := context.Background()

	t.Run("when just created, should be active", func(t *testing.T) {
		repo := newTestAuctionRepository(t, time.Hour)
		auction := newTestAuction(t)

		require.Nil(t, repo.CreateAuction(ctx, auction))

		assert.Equal(t, auction_entity.Active, findStatus(t, repo, auction.Id))
	})

	t.Run("when the interval elapses, should close the auction", func(t *testing.T) {
		const auctionInterval = 2 * time.Second

		repo := newTestAuctionRepository(t, auctionInterval)
		auction := newTestAuction(t)

		require.Nil(t, repo.CreateAuction(ctx, auction))
		require.Equal(t, auction_entity.Active, findStatus(t, repo, auction.Id))

		assert.Eventually(
			t,
			func() bool {
				return findStatus(t, repo, auction.Id) == auction_entity.Completed
			},
			4*auctionInterval,
			100*time.Millisecond,
			"o leilão deveria fechar sozinho depois de AUCTION_INTERVAL")
	})
}

func Test_AuctionRepository_CloseAuction(t *testing.T) {
	ctx := context.Background()

	t.Run("when called, should mark the auction as completed", func(t *testing.T) {
		repo := newTestAuctionRepository(t, time.Hour)
		auction := newTestAuction(t)

		require.Nil(t, repo.CreateAuction(ctx, auction))

		assert.Nil(t, repo.CloseAuction(ctx, auction.Id))
		assert.Equal(t, auction_entity.Completed, findStatus(t, repo, auction.Id))
	})

	t.Run("when called twice, should stay completed", func(t *testing.T) {
		repo := newTestAuctionRepository(t, time.Hour)
		auction := newTestAuction(t)

		require.Nil(t, repo.CreateAuction(ctx, auction))
		require.Nil(t, repo.CloseAuction(ctx, auction.Id))

		assert.Nil(t, repo.CloseAuction(ctx, auction.Id))
		assert.Equal(t, auction_entity.Completed, findStatus(t, repo, auction.Id))
	})
}

func Test_AuctionRepository_ScheduleActiveAuctions(t *testing.T) {
	ctx := context.Background()

	t.Run("when an active auction has expired, should close it", func(t *testing.T) {
		repo := newTestAuctionRepository(t, time.Second)

		expired := &AuctionEntityMongo{
			Id:          "auction-expirado",
			ProductName: "Monitor",
			Category:    "Eletronicos",
			Description: "Monitor 27 polegadas em otimo estado",
			Condition:   auction_entity.Used,
			Status:      auction_entity.Active,
			Timestamp:   time.Now().Add(-time.Hour).Unix(),
		}
		_, err := repo.Collection.InsertOne(ctx, expired)
		require.NoError(t, err)

		require.Nil(t, repo.ScheduleActiveAuctions(ctx))

		assert.Eventually(
			t,
			func() bool {
				return findStatus(t, repo, expired.Id) == auction_entity.Completed
			},
			5*time.Second,
			50*time.Millisecond,
			"a varredura deveria fechar o leilão ja vencido")
	})

	t.Run("when the auction is still within the interval, should keep it active", func(t *testing.T) {
		repo := newTestAuctionRepository(t, time.Hour)

		running := &AuctionEntityMongo{
			Id:          "auction-em-andamento",
			ProductName: "Teclado",
			Category:    "Eletronicos",
			Description: "Teclado mecanico em otimo estado de uso",
			Condition:   auction_entity.New,
			Status:      auction_entity.Active,
			Timestamp:   time.Now().Unix(),
		}
		_, err := repo.Collection.InsertOne(ctx, running)
		require.NoError(t, err)

		require.Nil(t, repo.ScheduleActiveAuctions(ctx))

		assert.Equal(t, auction_entity.Active, findStatus(t, repo, running.Id))
	})
}
