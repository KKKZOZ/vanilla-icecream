package main

import (
	mongoDB "benchmark/db/mongo"
	"benchmark/db/oreo"
	"benchmark/db/redis"
	"benchmark/pkg/client"
	"benchmark/pkg/measurement"
	"benchmark/ycsb"
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/kkkzoz/oreo/pkg/config"
	mongoCo "github.com/kkkzoz/oreo/pkg/datastore/mongo"
	redisCo "github.com/kkkzoz/oreo/pkg/datastore/redis"
	"github.com/kkkzoz/oreo/pkg/txn"
	goredis "github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	RedisDBAddr   = "43.139.62.221:6371"
	MongoDBAddr   = "mongodb://43.139.62.221:27017"
	OreoRedisAddr = "43.139.62.221:6380"
)

func main() {

	args := os.Args
	argsLen := len(args)
	if argsLen < 4 {
		fmt.Println("Usage: main [DBType] [load|run] [ThreadNum] [TestTypeFlag]")
		return
	}

	// TODO: Read it from file
	wp := &ycsb.WorkloadParameter{
		RecordCount:               10000,
		OperationCount:            10000,
		TxnOperationGroup:         10,
		ReadProportion:            0.95,
		UpdateProportion:          0.05,
		InsertProportion:          0,
		ScanProportion:            0,
		ReadModifyWriteProportion: 0,

		DataConsistencyTest:   false,
		InitialAmountPerKey:   1000,
		TransferAmountPerTxn:  1,
		PostCheckWorkerThread: 10,

		RedisProportion: 0.5,
		MongoProportion: 0.5,
	}
	wp.TotalAmount = wp.InitialAmountPerKey * wp.RecordCount

	config.Config.ConcurrentOptimizationLevel = 1
	config.Config.AsyncLevel = 2

	// f, err := os.Create("trace.out")
	// if err != nil {
	// 	panic(err)
	// }
	// err = trace.Start(f)
	// if err != nil {
	// 	panic(err)
	// }
	// defer trace.Stop()

	client := genClient(wp, args[1])

	measurement.InitMeasure()
	measurement.EnableWarmUp(true)

	mode := args[2]
	threadNum, err := strconv.Atoi(args[3])
	if err != nil || threadNum <= 0 {
		fmt.Println("ThreadNum should be a positive integer")
		return
	}
	wp.ThreadCount = threadNum

	if argsLen == 5 {
		switch args[4] {
		case "-dc":
			wp.DataConsistencyTest = true
			fmt.Println("This is a data consistency test")
		case "-tp":
			wp.TxnPerformanceTest = true
			fmt.Println("This is a transaction performance test")
		case "-ad":
			wp.AcrossDatastoreTest = true
			fmt.Println("This is a across datastore test")
		default:
			fmt.Println("Invalid TestTypeFlag")
		}
	}

	switch mode {
	case "load":
		// TODO:
		config.Config.ConcurrentOptimizationLevel = config.DEFAULT
		wp.DoBenchmark = false
		fmt.Println("Start to load data")
		client.RunLoad()
		fmt.Println("Load finished")
	case "run":
		wp.DoBenchmark = true
		fmt.Println("Start to run benchmark")
		measurement.EnableWarmUp(false)
		client.RunBenchmark()
	default:
		panic("Invalid mode")
	}
}

func genClient(wp *ycsb.WorkloadParameter, dbName string) *client.Client {
	var c *client.Client
	switch dbName {
	case "redis":
		wp.DBName = "redis"
		creator, err := RedisCreator()
		if err != nil {
			fmt.Printf("Error when creating redis client: %v\n", err)
			return nil
		}
		creatorMap := map[string]ycsb.DBCreator{
			dbName: creator,
		}
		c = client.NewClient(wp, creatorMap)
	case "mongo":
		wp.DBName = "mongo"
		creator, err := MongoCreator()
		if err != nil {
			fmt.Printf("Error when creating mongo client: %v\n", err)
			return nil
		}
		creatorMap := map[string]ycsb.DBCreator{
			dbName: creator,
		}
		c = client.NewClient(wp, creatorMap)
	case "redis-mongo":
		wp.DBName = "redis-mongo"
		redisCreator, err1 := RedisCreator()
		mongoCreator, err2 := MongoCreator()
		if err1 != nil || err2 != nil {
			fmt.Printf("Error when creating client: %v %v\n", err1, err2)
			return nil
		}
		creatorMap := map[string]ycsb.DBCreator{
			"redis": redisCreator,
			"mongo": mongoCreator,
		}
		c = client.NewClient(wp, creatorMap)
	case "oreo-redis":
		wp.DBName = "oreo-redis"
		creator, err := OreoRedisCreator()
		if err != nil {
			fmt.Printf("Error when creating oreo-redis client: %v\n", err)
			return nil
		}
		creatorMap := map[string]ycsb.DBCreator{
			dbName: creator,
		}
		c = client.NewClient(wp, creatorMap)
	case "oreo":
		wp.DBName = "oreo"
		creator, err := OreoCreator()
		if err != nil {
			fmt.Printf("Error when creating oreo-redis client: %v\n", err)
			return nil
		}
		creatorMap := map[string]ycsb.DBCreator{
			dbName: creator,
		}
		c = client.NewClient(wp, creatorMap)
	default:
		panic("Unsupport db type")
	}
	return c
}
func RedisCreator() (ycsb.DBCreator, error) {
	rdb := goredis.NewClient(&goredis.Options{
		Addr: RedisDBAddr,
	})

	// try to warm up the connection
	var wg sync.WaitGroup
	for i := 1; i <= 15; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rdb.Get(context.Background(), "1")
		}()
	}
	wg.Wait()
	return &redis.RedisCreator{Rdb: rdb}, nil
}

func MongoCreator() (ycsb.DBCreator, error) {
	clientOptions := options.Client().ApplyURI(MongoDBAddr)
	clientOptions.SetAuth(options.Credential{
		Username: "admin",
		Password: "admin",
	})
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &mongoDB.MongoCreator{Client: client}, nil
}

func OreoRedisCreator() (ycsb.DBCreator, error) {
	redisConn := redisCo.NewRedisConnection(&redisCo.ConnectionOptions{
		Address: OreoRedisAddr,
	})

	// try to warm up the connection
	var wg sync.WaitGroup
	for i := 1; i <= 15; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			redisConn.Get("1")
		}()
	}
	wg.Wait()
	return &oreo.OreoRedisCreator{Conn: redisConn}, nil
}

func OreoCreator() (ycsb.DBCreator, error) {
	redisConn := redisCo.NewRedisConnection(&redisCo.ConnectionOptions{
		Address: OreoRedisAddr,
	})
	mongoConn := mongoCo.NewMongoConnection(&mongoCo.ConnectionOptions{
		Address:        MongoDBAddr,
		DBName:         "oreo",
		CollectionName: "benchmark",
		Username:       "admin",
		Password:       "admin",
	})
	mongoConn.Connect()

	// try to warm up the connection
	var wg sync.WaitGroup
	for i := 1; i <= 15; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			redisConn.Get("1")
			mongoConn.Get("1")
		}()
	}
	wg.Wait()

	connMap := map[string]txn.Connector{
		"redis": redisConn,
		"mongo": mongoConn,
	}
	return &oreo.OreoCreator{
		ConnMap:             connMap,
		GlobalDatastoreName: "redis",
	}, nil
}
