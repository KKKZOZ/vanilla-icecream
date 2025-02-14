package redis

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/oreo-dtx-lab/oreo/internal/testutil"
	"github.com/oreo-dtx-lab/oreo/internal/util"
	"github.com/oreo-dtx-lab/oreo/pkg/config"
	"github.com/oreo-dtx-lab/oreo/pkg/errs"
	"github.com/oreo-dtx-lab/oreo/pkg/serializer"
	"github.com/oreo-dtx-lab/oreo/pkg/txn"
	"github.com/stretchr/testify/assert"
)

// func TestRedisConnection_Connect(t *testing.T) {
// 	RedisClient, _ := redismock.NewClientMock()
// 	connection := &RedisConnection{rdb: RedisClient}

// 	err := connection.Connect()
// 	assert.Nil(t, err)
// }

func TestTimestamp(t *testing.T) {
	tValid := time.Now().Add(-3 * time.Second)
	// tValidStr :=
	t1, _ := time.Parse(time.RFC3339Nano, tValid.Format(time.RFC3339Nano))
	// assert.Equal(t, tValid, t1)
	if !t1.Equal(tValid) {
		t.Error("Not Equal")
	}
	// if t1 != tValid {
	// 	t.Error("Not Equal")
	// }
}

func TestRedisConnection_GetItemNotFound(t *testing.T) {
	key := "test_key"
	connection := NewRedisConnection(nil)
	connection.Connect()

	connection.Delete("test_key")
	_, err := connection.GetItem(key)

	assert.ErrorIs(t, err, errs.NewKeyNotFoundError(key, errs.NotFoundInDB))
}

func TestRedisConnectionPutItemAndGetItem(t *testing.T) {
	conn := NewRedisConnection(nil)
	conn.Connect()
	conn.Delete("test_key")

	key := "test_key"
	expectedValue := testutil.NewDefaultPerson()
	expectedItem := &RedisItem{
		RKey:          key,
		RValue:        util.ToJSONString(expectedValue),
		RGroupKeyList: "redis:1",
		RTxnState:     config.COMMITTED,
		RTValid:       111,
		RTLease:       time.Now().Add(-2 * time.Second),
		RPrev:         "",
		RIsDeleted:    false,
		RVersion:      "2",
	}

	_, err := conn.PutItem(key, expectedItem)
	assert.NoError(t, err)

	item, err := conn.GetItem(key)

	assert.NoError(t, err)
	if !item.Equal(expectedItem) {
		t.Errorf("\nexpect: \n%v, \nactual: \n%v", expectedItem, item)
	}
}

func TestRedisConnectionReplaceAndGetItem(t *testing.T) {
	conn := NewRedisConnection(nil)

	key := "test_key"
	olderPerson := testutil.NewDefaultPerson()
	olderItem := &RedisItem{
		RKey:          key,
		RValue:        util.ToJSONString(olderPerson),
		RGroupKeyList: "redis:1",
		RTxnState:     config.COMMITTED,
		RTValid:       111,
		RTLease:       time.Now().Add(-2 * time.Second),
		RPrev:         "",
		RIsDeleted:    false,
		RVersion:      "2",
	}

	_, err := conn.PutItem(key, olderItem)
	assert.NoError(t, err)

	newerPerson := testutil.NewDefaultPerson()
	newerPerson.Name = "newer"
	newerItem := &RedisItem{
		RKey:          key,
		RValue:        util.ToJSONString(newerPerson),
		RGroupKeyList: "redis:2",
		RTxnState:     config.COMMITTED,
		RTValid:       222,
		RTLease:       time.Now().Add(1 * time.Second),
		RPrev:         util.ToJSONString(olderItem),
		RIsDeleted:    false,
		RVersion:      "3",
	}

	_, err = conn.PutItem(key, newerItem)
	assert.NoError(t, err)

	item, err := conn.GetItem(key)
	assert.NoError(t, err)
	if !item.Equal(newerItem) {
		t.Errorf("\nexpect: \n%v, \nactual: \n%v", newerItem, item)
	}
}

func TestRedisConnectionConditionalUpdateSuccess(t *testing.T) {
	conn := NewRedisConnection(nil)

	key := "test_key"
	olderPerson := testutil.NewDefaultPerson()
	olderItem := &RedisItem{
		RKey:          key,
		RValue:        util.ToJSONString(olderPerson),
		RGroupKeyList: "redis:1",
		RTxnState:     config.COMMITTED,
		RTValid:       111,
		RTLease:       time.Now().Add(-2 * time.Second),
		RPrev:         "",
		RIsDeleted:    false,
		RVersion:      "2",
	}
	_, err := conn.PutItem(key, olderItem)
	assert.NoError(t, err)

	newerPerson := testutil.NewDefaultPerson()
	newerPerson.Name = "newer"
	newerItem := &RedisItem{
		RKey:          key,
		RValue:        util.ToJSONString(newerPerson),
		RGroupKeyList: "redis:2",
		RTxnState:     config.COMMITTED,
		RTValid:       222,
		RTLease:       time.Now().Add(-1 * time.Second),
		RPrev:         "",
		RIsDeleted:    false,
		RVersion:      "2",
	}

	_, err = conn.ConditionalUpdate(key, newerItem, false)
	assert.NoError(t, err)

	item, err := conn.GetItem(key)
	assert.NoError(t, err)

	newerItem.RVersion = util.AddToString(newerItem.RVersion, 1)

	if !item.Equal(newerItem) {
		t.Errorf("\nexpect: \n%v, \nactual: \n%v", newerItem, item)
	}

}

func TestRedisConnectionConditionalUpdateFail(t *testing.T) {
	conn := NewRedisConnection(nil)
	key := "test_key"
	olderPerson := testutil.NewDefaultPerson()
	olderItem := &RedisItem{
		RKey:          key,
		RValue:        util.ToJSONString(olderPerson),
		RGroupKeyList: "redis:1",
		RTxnState:     config.COMMITTED,
		RTValid:       111,
		RTLease:       time.Now().Add(-2 * time.Second),
		RPrev:         "",
		RIsDeleted:    false,
		RVersion:      "2",
	}
	_, err := conn.PutItem(key, olderItem)
	assert.NoError(t, err)

	newerPerson := testutil.NewDefaultPerson()
	newerPerson.Name = "newer"
	newerItem := &RedisItem{
		RKey:          key,
		RValue:        util.ToJSONString(olderPerson),
		RGroupKeyList: "2",
		RTxnState:     config.COMMITTED,
		RTValid:       222,
		RTLease:       time.Now().Add(-1 * time.Second),
		RPrev:         "",
		RIsDeleted:    false,
		RVersion:      "3",
	}

	_, err = conn.ConditionalUpdate(key, newerItem, false)
	assert.EqualError(t, err, "version mismatch")

	item, err := conn.GetItem(key)
	assert.NoError(t, err)
	if !item.Equal(olderItem) {
		t.Errorf("\nexpect: \n%v, \nactual: \n%v", olderItem, item)
	}
}

func TestRedisConnectionConditionalUpdateNonExist(t *testing.T) {
	conn := NewRedisConnection(nil)

	key := "test_key"
	conn.Delete(key)
	newerPerson := testutil.NewDefaultPerson()
	newerPerson.Name = "newer"
	newerItem := &RedisItem{
		RKey:          key,
		RValue:        util.ToJSONString(newerPerson),
		RGroupKeyList: "redis:2",
		RTxnState:     config.COMMITTED,
		RTValid:       222,
		RTLease:       time.Now().Add(-1 * time.Second),
		RPrev:         "",
		RIsDeleted:    false,
		RVersion:      "1",
	}

	_, err := conn.ConditionalUpdate(key, newerItem, true)
	assert.NoError(t, err)

	item, err := conn.GetItem(key)
	assert.NoError(t, err)

	newerItem.RVersion = util.AddToString(newerItem.RVersion, 1)
	if !item.Equal(newerItem) {
		t.Errorf("\nexpect: \n%v, \nactual: \n%v", newerItem, item)
	}
}

func TestRedisConnectionConditionalUpdateConcurrently(t *testing.T) {

	t.Run("this is a update", func(t *testing.T) {
		conn := NewRedisConnection(nil)
		conn.Connect()

		key := "test_key"
		olderPerson := testutil.NewDefaultPerson()
		olderItem := &RedisItem{
			RKey:          key,
			RValue:        util.ToJSONString(olderPerson),
			RGroupKeyList: "redis:1",
			RTxnState:     config.COMMITTED,
			RTValid:       111,
			RTLease:       time.Now().Add(-2 * time.Second),
			RPrev:         "",
			RIsDeleted:    false,
			RVersion:      "2",
		}
		_, err := conn.PutItem(key, olderItem)
		assert.NoError(t, err)

		resChan := make(chan bool)
		currentNum := 100
		globalId := 0
		for i := 1; i <= currentNum; i++ {
			go func(id int) {
				newerPerson := testutil.NewDefaultPerson()
				newerPerson.Name = "newer"
				newerItem := &RedisItem{
					RKey:          key,
					RValue:        util.ToJSONString(newerPerson),
					RGroupKeyList: strconv.Itoa(id),
					RTxnState:     config.COMMITTED,
					RTValid:       222,
					RTLease:       time.Now().Add(-1 * time.Second),
					RPrev:         "",
					RIsDeleted:    false,
					RVersion:      "2",
				}

				_, err = conn.ConditionalUpdate(key, newerItem, false)
				if err == nil {
					globalId = id
					resChan <- true
				} else {
					fmt.Printf("error: %v\n", err)
					resChan <- false
				}
			}(i)
		}
		successCnt := 0
		for i := 1; i <= currentNum; i++ {
			res := <-resChan
			if res {
				successCnt++
			}
		}
		assert.Equal(t, 1, successCnt)

		item, err := conn.GetItem(key)
		assert.NoError(t, err)
		if item.GroupKeyList() != strconv.Itoa(globalId) {
			t.Errorf("\nexpect: \n%v, \nactual: \n%v", globalId, item.GroupKeyList())
		}
	})

	t.Run("this is a create", func(t *testing.T) {
		conn := NewRedisConnection(nil)
		conn.Connect()
		key := "test_key"
		conn.Delete(key)

		resChan := make(chan bool)
		currentNum := 100
		globalId := 0
		for i := 1; i <= currentNum; i++ {
			go func(id int) {
				newerPerson := testutil.NewDefaultPerson()
				newerPerson.Name = "newer"
				newerItem := &RedisItem{
					RKey:          key,
					RValue:        util.ToJSONString(newerPerson),
					RGroupKeyList: strconv.Itoa(id),
					RTxnState:     config.COMMITTED,
					RTValid:       222,
					RTLease:       time.Now().Add(-1 * time.Second),
					RPrev:         "",
					RIsDeleted:    false,
					RVersion:      "2",
				}

				_, err := conn.ConditionalUpdate(key, newerItem, true)
				if err == nil {
					globalId = id
					resChan <- true
				} else {
					resChan <- false
				}
			}(i)
		}
		successCnt := 0
		for i := 1; i <= currentNum; i++ {
			res := <-resChan
			if res {
				successCnt++
			}
		}
		assert.Equal(t, 1, successCnt)

		item, err := conn.GetItem(key)
		assert.NoError(t, err)
		if item.GroupKeyList() != strconv.Itoa(globalId) {
			t.Errorf("\nexpect: \n%v, \nactual: \n%v",
				globalId, item.GroupKeyList())
		}
	})
}

func TestRedisConnectionPutAndGet(t *testing.T) {
	conn := NewRedisConnection(nil)
	se := serializer.NewJSONSerializer()

	key := "test_key"
	person := testutil.NewDefaultPerson()
	item := &RedisItem{
		RKey:          key,
		RValue:        util.ToJSONString(person),
		RGroupKeyList: "redis:1",
		RTxnState:     config.COMMITTED,
		RTValid:       111,
		RTLease:       time.Now().Add(-2 * time.Second),
		RPrev:         "",
		RIsDeleted:    false,
		RVersion:      "2",
	}
	bs, err := se.Serialize(item)
	assert.NoError(t, err)
	err = conn.Put(key, bs)
	assert.NoError(t, err)

	str, err := conn.Get(key)
	assert.NoError(t, err)
	var actualItem RedisItem
	err = se.Deserialize([]byte(str), &actualItem)
	assert.NoError(t, err)
	if !actualItem.Equal(item) {
		t.Errorf("\nexpect: \n%v, \nactual: \n%v", item, actualItem)
	}
}

func TestRedisConnectionReplaceAndGet(t *testing.T) {
	conn := NewRedisConnection(nil)
	se := serializer.NewJSONSerializer()

	key := "test_key"
	person := testutil.NewDefaultPerson()
	item := &RedisItem{
		RKey:          key,
		RValue:        util.ToJSONString(person),
		RGroupKeyList: "redis:1",
		RTxnState:     config.COMMITTED,
		RTValid:       111,
		RTLease:       time.Now().Add(-2 * time.Second),
		RPrev:         "",
		RIsDeleted:    false,
		RVersion:      "2",
	}
	bs, err := se.Serialize(item)
	assert.NoError(t, err)
	err = conn.Put(key, bs)
	assert.NoError(t, err)

	item.RVersion = util.AddToString(item.RVersion, 1)
	bs, _ = se.Serialize(item)
	err = conn.Put(key, bs)
	assert.NoError(t, err)

	str, err := conn.Get(key)
	assert.NoError(t, err)
	var actualItem RedisItem
	err = se.Deserialize([]byte(str), &actualItem)
	assert.NoError(t, err)
	if !actualItem.Equal(item) {
		t.Errorf("\nexpect: \n%v, \nactual: \n%v", item, actualItem)
	}
}

func TestRedisConnectionGetNoExist(t *testing.T) {
	conn := NewRedisConnection(nil)

	key := "test_key"
	conn.Delete(key)

	_, err := conn.Get(key)
	assert.EqualError(t, err, "key not found")
}

func TestRedisConnectionPutDirectItem(t *testing.T) {
	conn := NewRedisConnection(nil)

	key := "test_key"
	conn.Delete(key)

	person := testutil.NewDefaultPerson()
	item := &RedisItem{
		RKey:          key,
		RValue:        util.ToJSONString(person),
		RGroupKeyList: "redis:1",
		RTxnState:     config.COMMITTED,
		RTValid:       111,
		RTLease:       time.Now().Add(-2 * time.Second),
		RPrev:         "",
		RIsDeleted:    false,
		RVersion:      "2",
	}

	err := conn.Put(key, item)
	assert.NoError(t, err)

	// post check
	str, err := conn.Get(key)
	assert.NoError(t, err)
	var actualItem RedisItem
	err = json.Unmarshal([]byte(str), &actualItem)
	assert.NoError(t, err)
	if !actualItem.Equal(item) {
		t.Errorf("\nexpect: \n%v, \nactual: \n%v", item, actualItem)
	}
}

func TestRedisConnectionDeleteTwice(t *testing.T) {

	conn := NewRedisConnection(nil)
	conn.Put("test_key", "test_value")
	err := conn.Delete("test_key")
	assert.NoError(t, err)
	err = conn.Delete("test_key")
	assert.NoError(t, err)
}

func TestRedisConnectionConditionalUpdateDoCreate(t *testing.T) {

	dbItem := &RedisItem{
		RKey:          "item1",
		RValue:        util.ToJSONString(testutil.NewTestItem("item1-db")),
		RGroupKeyList: "redis:1",
		RTxnState:     config.COMMITTED,
		RTValid:       111,
		RTLease:       time.Now().Add(-2 * time.Second),
		RPrev:         "",
		RIsDeleted:    false,
		RLinkedLen:    1,
		RVersion:      "1",
	}

	cacheItem := &RedisItem{
		RKey:          "item1",
		RValue:        util.ToJSONString(testutil.NewTestItem("item1-cache")),
		RGroupKeyList: "redis:2",
		RTxnState:     config.COMMITTED,
		RTValid:       222,
		RTLease:       time.Now().Add(-1 * time.Second),
		RPrev:         util.ToJSONString(dbItem),
		RLinkedLen:    2,
		RVersion:      "1",
	}

	t.Run("there is no item and doCreate is true ", func(t *testing.T) {
		conn := NewRedisConnection(nil)
		conn.Delete(cacheItem.Key())

		_, err := conn.ConditionalUpdate(cacheItem.Key(), cacheItem, true)
		assert.NoError(t, err)
	})

	t.Run("there is an item and doCreate is true ", func(t *testing.T) {
		conn := NewRedisConnection(nil)
		conn.PutItem(dbItem.Key(), dbItem)

		_, err := conn.ConditionalUpdate(cacheItem.Key(), cacheItem, true)
		assert.EqualError(t, err, txn.VersionMismatch.Error())
	})

	t.Run("there is no item and doCreate is false ", func(t *testing.T) {
		conn := NewRedisConnection(nil)
		conn.Delete(cacheItem.Key())

		_, err := conn.ConditionalUpdate(cacheItem.Key(), cacheItem, false)
		assert.EqualError(t, err, txn.VersionMismatch.Error())
	})

	t.Run("there is an item and doCreate is false ", func(t *testing.T) {
		conn := NewRedisConnection(nil)
		conn.PutItem(dbItem.Key(), dbItem)

		_, err := conn.ConditionalUpdate(cacheItem.Key(), cacheItem, false)
		assert.NoError(t, err)
	})

}

func TestRedisConnectionConditionalCommit(t *testing.T) {

	dbItem := &RedisItem{
		RKey:          "item1",
		RValue:        util.ToJSONString(testutil.NewTestItem("item1-db")),
		RGroupKeyList: "redis:1",
		RTxnState:     config.COMMITTED,
		RTValid:       111,
		RTLease:       time.Now().Add(-2 * time.Second),
		RPrev:         "",
		RIsDeleted:    false,
		RLinkedLen:    1,
		RVersion:      "1",
	}

	conn := NewRedisConnection(nil)
	conn.Connect()
	conn.PutItem(dbItem.Key(), dbItem)

	_, err := conn.ConditionalCommit(dbItem.Key(), dbItem.Version(), 222)
	assert.NoError(t, err)

	item, err := conn.GetItem(dbItem.Key())
	assert.NoError(t, err)

	dbItem.RVersion = util.AddToString(dbItem.RVersion, 1)
	dbItem.RTxnState = config.COMMITTED

	if !dbItem.Equal(item) {
		t.Fail()
	}

}
