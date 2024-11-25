package benconfig

import "time"

var (
	ExecutorAddressList = []string{"localhost:8001"}
	TimeOracleUrl       = "http://localhost:8010"
	ZipfianConstant     = 0.5
	Latency             = 10 * time.Millisecond
	MaxLoadBatchSize    = 100
)

type BenchmarkConfig struct {
	ExecutorAddressList []string      `yaml:"executor_address_list"`
	TimeOracleUrl       string        `yaml:"time_oracle_url"`
	ZipfianConstant     float64       `yaml:"zipfian_constant"`
	Latency             time.Duration `yaml:"latency"`
	LatencyValue        int           `yaml:"latency_value"`
	MaxLoadBatchSize    int           `yaml:"max_load_batch_size"`

	RedisAddr     string `yaml:"redis_addr"`
	RedisPassword string `yaml:"redis_password"`

	MongoDBAddr1    string `yaml:"mongodb_addr1"`
	MongoDBAddr2    string `yaml:"mongodb_addr2"`
	MongoDBUsername string `yaml:"mongodb_username"`
	MongoDBPassword string `yaml:"mongodb_password"`

	KVRocksAddr     string `yaml:"kvrocks_addr"`
	KVRocksPassword string `yaml:"kvrocks_password"`

	CouchDBAddr     string `yaml:"couchdb_addr"`
	CouchDBUsername string `yaml:"couchdb_username"`
	CouchDBPassword string `yaml:"couchdb_password"`

	CassandraAddr []string `yaml:"cassandra_addr"`
	DynamoDBAddr  string   `yaml:"dynamodb_addr"`
	TiKVAddr      []string `yaml:"tikv_addr"`

	// DBCombination []string `yaml:"db_combination"`
}
