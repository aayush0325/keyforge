package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// newTestClient creates a new Redis client for testing
func newTestClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
}

// TestPing tests the PING command
func TestPing(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	result, err := client.Ping(ctx).Result()
	if err != nil {
		t.Fatalf("PING failed: %v", err)
	}
	if result != "PONG" {
		t.Errorf("Expected PONG, got %s", result)
	}
}

// TestEcho tests the ECHO command
func TestEcho(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	message := "Hello, Redis!"
	result, err := client.Echo(ctx, message).Result()
	if err != nil {
		t.Fatalf("ECHO failed: %v", err)
	}
	if result != message {
		t.Errorf("Expected %s, got %s", message, result)
	}
}

// =============================================================================
// KV Store Tests
// =============================================================================

// TestSetAndGet tests basic SET and GET operations
func TestSetAndGet(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	// Test basic SET and GET
	key := "test:set:basic"
	value := "hello world"

	err := client.Set(ctx, key, value, 0).Err()
	if err != nil {
		t.Fatalf("SET failed: %v", err)
	}

	result, err := client.Get(ctx, key).Result()
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	if result != value {
		t.Errorf("Expected %s, got %s", value, result)
	}

	// Cleanup
	client.Del(ctx, key)
}

// TestSetWithExpiration tests SET with EX (seconds) expiration
func TestSetWithExpiration(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	key := "test:set:ex"
	value := "expires soon"

	// Set with 2 second expiration
	err := client.Set(ctx, key, value, 2*time.Second).Err()
	if err != nil {
		t.Fatalf("SET with EX failed: %v", err)
	}

	// Should exist immediately
	result, err := client.Get(ctx, key).Result()
	if err != nil {
		t.Fatalf("GET failed immediately after SET: %v", err)
	}
	if result != value {
		t.Errorf("Expected %s, got %s", value, result)
	}

	// Cleanup
	client.Del(ctx, key)
}

// TestSetWithMillisecondExpiration tests SET with PX (milliseconds) expiration
func TestSetWithMillisecondExpiration(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	key := "test:set:px"
	value := "expires in ms"

	// Set with 1500ms expiration
	err := client.Set(ctx, key, value, 1500*time.Millisecond).Err()
	if err != nil {
		t.Fatalf("SET with PX failed: %v", err)
	}

	// Should exist immediately
	result, err := client.Get(ctx, key).Result()
	if err != nil {
		t.Fatalf("GET failed immediately after SET: %v", err)
	}
	if result != value {
		t.Errorf("Expected %s, got %s", value, result)
	}

	// Cleanup
	client.Del(ctx, key)
}

// TestSetNX tests SET with NX (only set if not exists)
func TestSetNX(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	key := "test:set:nx"
	value1 := "first value"
	value2 := "second value"

	// First SETNX should succeed
	result, err := client.SetNX(ctx, key, value1, 0).Result()
	if err != nil {
		t.Fatalf("First SETNX failed: %v", err)
	}
	if !result {
		t.Error("First SETNX should return true")
	}

	// Second SETNX should fail (key exists)
	result, err = client.SetNX(ctx, key, value2, 0).Result()
	if err != nil {
		t.Fatalf("Second SETNX failed: %v", err)
	}
	if result {
		t.Error("Second SETNX should return false")
	}

	// Value should still be the first one
	val, err := client.Get(ctx, key).Result()
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	if val != value1 {
		t.Errorf("Expected %s, got %s", value1, val)
	}

	// Cleanup
	client.Del(ctx, key)
}

// TestGetNonExistent tests GET on a non-existent key
func TestGetNonExistent(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	_, err := client.Get(ctx, "non:existent:key").Result()
	if err != redis.Nil {
		t.Errorf("Expected redis.Nil error, got %v", err)
	}
}

// TestDel tests the DEL command
func TestDel(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	key := "test:del:single"
	client.Set(ctx, key, "value", 0)

	// Delete the key
	deleted, err := client.Del(ctx, key).Result()
	if err != nil {
		t.Fatalf("DEL failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("Expected 1 deleted, got %d", deleted)
	}

	// Verify it's gone
	_, err = client.Get(ctx, key).Result()
	if err != redis.Nil {
		t.Error("Key should not exist after DEL")
	}
}

// TestDelMultiple tests DEL with multiple keys
func TestDelMultiple(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	keys := []string{"test:del:multi:1", "test:del:multi:2", "test:del:multi:3"}
	for _, key := range keys {
		client.Set(ctx, key, "value", 0)
	}

	// Delete all keys
	deleted, err := client.Del(ctx, keys...).Result()
	if err != nil {
		t.Fatalf("DEL multiple failed: %v", err)
	}
	if deleted != 3 {
		t.Errorf("Expected 3 deleted, got %d", deleted)
	}
}

// TestDelNonExistent tests DEL on non-existent keys
func TestDelNonExistent(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	deleted, err := client.Del(ctx, "non:existent:key").Result()
	if err != nil {
		t.Fatalf("DEL failed: %v", err)
	}
	if deleted != 0 {
		t.Errorf("Expected 0 deleted for non-existent key, got %d", deleted)
	}
}

// TestExists tests the EXISTS command
func TestExists(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	key := "test:exists"
	client.Set(ctx, key, "value", 0)

	// Should exist
	count, err := client.Exists(ctx, key).Result()
	if err != nil {
		t.Fatalf("EXISTS failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected EXISTS to return 1, got %d", count)
	}

	// Cleanup
	client.Del(ctx, key)
}

// TestExistsMultiple tests EXISTS with multiple keys
func TestExistsMultiple(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	keys := []string{"test:exists:multi:1", "test:exists:multi:2"}
	for _, key := range keys {
		client.Set(ctx, key, "value", 0)
	}

	// Both should exist
	count, err := client.Exists(ctx, keys...).Result()
	if err != nil {
		t.Fatalf("EXISTS multiple failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected EXISTS to return 2, got %d", count)
	}

	// Add a non-existent key
	count, err = client.Exists(ctx, "test:exists:multi:1", "non:existent", "test:exists:multi:2").Result()
	if err != nil {
		t.Fatalf("EXISTS with mixed keys failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected EXISTS to return 2 for mixed keys, got %d", count)
	}

	// Cleanup
	client.Del(ctx, keys...)
}

// TestExistsNonExistent tests EXISTS on non-existent key
func TestExistsNonExistent(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	count, err := client.Exists(ctx, "non:existent:key").Result()
	if err != nil {
		t.Fatalf("EXISTS failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected EXISTS to return 0 for non-existent key, got %d", count)
	}
}

// TestType tests the TYPE command
func TestType(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	// Test string type
	stringKey := "test:type:string"
	client.Set(ctx, stringKey, "value", 0)

	typeResult, err := client.Type(ctx, stringKey).Result()
	if err != nil {
		t.Fatalf("TYPE failed: %v", err)
	}
	if typeResult != "string" {
		t.Errorf("Expected type 'string', got '%s'", typeResult)
	}

	// Test list type
	listKey := "test:type:list"
	client.RPush(ctx, listKey, "item")

	typeResult, err = client.Type(ctx, listKey).Result()
	if err != nil {
		t.Fatalf("TYPE failed for list: %v", err)
	}
	if typeResult != "list" {
		t.Errorf("Expected type 'list', got '%s'", typeResult)
	}

	// Test non-existent key
	typeResult, err = client.Type(ctx, "non:existent:type:key").Result()
	if err != nil {
		t.Fatalf("TYPE failed for non-existent: %v", err)
	}
	if typeResult != "none" {
		t.Errorf("Expected type 'none', got '%s'", typeResult)
	}

	// Cleanup
	client.Del(ctx, stringKey, listKey)
}

// TestSetOverwrite tests that SET overwrites existing values
func TestSetOverwrite(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	key := "test:set:overwrite"

	client.Set(ctx, key, "first", 0)
	client.Set(ctx, key, "second", 0)

	result, err := client.Get(ctx, key).Result()
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	if result != "second" {
		t.Errorf("Expected 'second', got '%s'", result)
	}

	// Cleanup
	client.Del(ctx, key)
}

// TestSetEmptyValue tests SET with empty string value
func TestSetEmptyValue(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	key := "test:set:empty"

	err := client.Set(ctx, key, "", 0).Err()
	if err != nil {
		t.Fatalf("SET with empty value failed: %v", err)
	}

	result, err := client.Get(ctx, key).Result()
	if err != nil {
		t.Fatalf("GET empty value failed: %v", err)
	}
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}

	// Cleanup
	client.Del(ctx, key)
}

// TestSetLargeValue tests SET with a large value
func TestSetLargeValue(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	key := "test:set:large"
	// Create a 1MB value
	largeValue := make([]byte, 1024*1024)
	for i := range largeValue {
		largeValue[i] = byte('A' + (i % 26))
	}

	err := client.Set(ctx, key, string(largeValue), 0).Err()
	if err != nil {
		t.Fatalf("SET with large value failed: %v", err)
	}

	result, err := client.Get(ctx, key).Result()
	if err != nil {
		t.Fatalf("GET large value failed: %v", err)
	}
	if len(result) != len(largeValue) {
		t.Errorf("Expected length %d, got %d", len(largeValue), len(result))
	}

	// Cleanup
	client.Del(ctx, key)
}

// =============================================================================
// List Tests
// =============================================================================

// TestRPush tests the RPUSH command
func TestRPush(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	key := "test:list:rpush"

	// Push single element
	length, err := client.RPush(ctx, key, "first").Result()
	if err != nil {
		t.Fatalf("RPUSH failed: %v", err)
	}
	if length != 1 {
		t.Errorf("Expected length 1, got %d", length)
	}

	// Push multiple elements
	length, err = client.RPush(ctx, key, "second", "third").Result()
	if err != nil {
		t.Fatalf("RPUSH multiple failed: %v", err)
	}
	if length != 3 {
		t.Errorf("Expected length 3, got %d", length)
	}

	// Verify order
	items, err := client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		t.Fatalf("LRANGE failed: %v", err)
	}
	expected := []string{"first", "second", "third"}
	for i, item := range items {
		if item != expected[i] {
			t.Errorf("Expected %s at index %d, got %s", expected[i], i, item)
		}
	}

	// Cleanup
	client.Del(ctx, key)
}

// TestLPush tests the LPUSH command
func TestLPush(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	key := "test:list:lpush"

	// Push single element
	length, err := client.LPush(ctx, key, "first").Result()
	if err != nil {
		t.Fatalf("LPUSH failed: %v", err)
	}
	if length != 1 {
		t.Errorf("Expected length 1, got %d", length)
	}

	// Push multiple elements
	length, err = client.LPush(ctx, key, "second", "third").Result()
	if err != nil {
		t.Fatalf("LPUSH multiple failed: %v", err)
	}
	if length != 3 {
		t.Errorf("Expected length 3, got %d", length)
	}

	// Verify order (LPUSH adds to front, so order is reversed)
	items, err := client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		t.Fatalf("LRANGE failed: %v", err)
	}
	// With LPUSH, elements are added to the left, so "third" will be first, then "second", then "first"
	expected := []string{"third", "second", "first"}
	for i, item := range items {
		if item != expected[i] {
			t.Errorf("Expected %s at index %d, got %s", expected[i], i, item)
		}
	}

	// Cleanup
	client.Del(ctx, key)
}

// TestLPop tests the LPOP command
func TestLPop(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	key := "test:list:lpop"
	client.RPush(ctx, key, "a", "b", "c")

	// Pop single element
	val, err := client.LPop(ctx, key).Result()
	if err != nil {
		t.Fatalf("LPOP failed: %v", err)
	}
	if val != "a" {
		t.Errorf("Expected 'a', got '%s'", val)
	}

	// Verify remaining elements
	items, _ := client.LRange(ctx, key, 0, -1).Result()
	if len(items) != 2 {
		t.Errorf("Expected 2 remaining elements, got %d", len(items))
	}

	// Cleanup
	client.Del(ctx, key)
}

// TestLPopCount tests LPOP with count parameter
func TestLPopCount(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	key := "test:list:lpop:count"
	client.RPush(ctx, key, "a", "b", "c", "d", "e")

	// Pop multiple elements
	vals, err := client.LPopCount(ctx, key, 3).Result()
	if err != nil {
		t.Fatalf("LPOP with count failed: %v", err)
	}
	if len(vals) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(vals))
	}

	expected := []string{"a", "b", "c"}
	for i, val := range vals {
		if val != expected[i] {
			t.Errorf("Expected %s at index %d, got %s", expected[i], i, val)
		}
	}

	// Cleanup
	client.Del(ctx, key)
}

// TestLPopEmpty tests LPOP on empty list
func TestLPopEmpty(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	_, err := client.LPop(ctx, "non:existent:list").Result()
	if err != redis.Nil {
		t.Errorf("Expected redis.Nil, got %v", err)
	}
}

// TestLLen tests the LLEN command
func TestLLen(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	key := "test:list:llen"
	client.RPush(ctx, key, "a", "b", "c", "d", "e")

	length, err := client.LLen(ctx, key).Result()
	if err != nil {
		t.Fatalf("LLEN failed: %v", err)
	}
	if length != 5 {
		t.Errorf("Expected length 5, got %d", length)
	}

	// Cleanup
	client.Del(ctx, key)
}

// TestLLenEmpty tests LLEN on non-existent list
func TestLLenEmpty(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	length, err := client.LLen(ctx, "non:existent:list").Result()
	if err != nil {
		t.Fatalf("LLEN failed: %v", err)
	}
	if length != 0 {
		t.Errorf("Expected length 0 for non-existent list, got %d", length)
	}
}

// TestLRange tests the LRANGE command
func TestLRange(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	key := "test:list:lrange"
	client.RPush(ctx, key, "a", "b", "c", "d", "e")

	// Get all elements
	items, err := client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		t.Fatalf("LRANGE all failed: %v", err)
	}
	if len(items) != 5 {
		t.Errorf("Expected 5 elements, got %d", len(items))
	}

	// Get subset
	items, err = client.LRange(ctx, key, 1, 3).Result()
	if err != nil {
		t.Fatalf("LRANGE subset failed: %v", err)
	}
	expected := []string{"b", "c", "d"}
	if len(items) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(items))
	}
	for i, item := range items {
		if item != expected[i] {
			t.Errorf("Expected %s at index %d, got %s", expected[i], i, item)
		}
	}

	// Cleanup
	client.Del(ctx, key)
}

// TestLRangeNegativeIndex tests LRANGE with negative indices
func TestLRangeNegativeIndex(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	key := "test:list:lrange:neg"
	client.RPush(ctx, key, "a", "b", "c", "d", "e")

	// Last 3 elements
	items, err := client.LRange(ctx, key, -3, -1).Result()
	if err != nil {
		t.Fatalf("LRANGE with negative indices failed: %v", err)
	}
	expected := []string{"c", "d", "e"}
	if len(items) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(items))
	}
	for i, item := range items {
		if item != expected[i] {
			t.Errorf("Expected %s at index %d, got %s", expected[i], i, item)
		}
	}

	// Mixed negative and positive
	items, err = client.LRange(ctx, key, 0, -2).Result()
	if err != nil {
		t.Fatalf("LRANGE mixed indices failed: %v", err)
	}
	expected = []string{"a", "b", "c", "d"}
	if len(items) != 4 {
		t.Errorf("Expected 4 elements, got %d", len(items))
	}

	// Cleanup
	client.Del(ctx, key)
}

// TestLRangeOutOfBounds tests LRANGE with out of bounds indices
func TestLRangeOutOfBounds(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	key := "test:list:lrange:oob"
	client.RPush(ctx, key, "a", "b", "c")

	// Start and end beyond list length
	items, err := client.LRange(ctx, key, 0, 100).Result()
	if err != nil {
		t.Fatalf("LRANGE out of bounds failed: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(items))
	}

	// Cleanup
	client.Del(ctx, key)
}

// TestBLPop tests the BLPOP command (blocking left pop)
func TestBLPop(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	key := "test:list:blpop"
	client.RPush(ctx, key, "first", "second")

	// BLPOP should return immediately since list is not empty
	result, err := client.BLPop(ctx, time.Second, key).Result()
	if err != nil {
		t.Fatalf("BLPOP failed: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("Expected 2 elements in result, got %d", len(result))
	}
	if result[0] != key {
		t.Errorf("Expected key name %s, got %s", key, result[0])
	}
	if result[1] != "first" {
		t.Errorf("Expected 'first', got %s", result[1])
	}

	// Cleanup
	client.Del(ctx, key)
}

// TestBLPopTimeout tests BLPOP timeout on empty list
func TestBLPopTimeout(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	start := time.Now()
	_, err := client.BLPop(ctx, time.Second, "non:existent:blpop:key").Result()
	elapsed := time.Since(start)

	if err != redis.Nil {
		t.Errorf("Expected redis.Nil on timeout, got %v", err)
	}

	// Should have waited approximately 1 second
	if elapsed < 900*time.Millisecond {
		t.Errorf("BLPOP returned too quickly: %v", elapsed)
	}
}

// TestBLPopBlocking tests BLPOP blocking behavior with concurrent push
func TestBLPopBlocking(t *testing.T) {
	client := newTestClient()
	pusherClient := newTestClient()
	defer client.Close()
	defer pusherClient.Close()
	ctx := context.Background()

	key := "test:list:blpop:blocking"

	// Make sure key doesn't exist
	client.Del(ctx, key)

	var wg sync.WaitGroup
	var result []string
	var blpopErr error

	// Start BLPOP in goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		result, blpopErr = client.BLPop(ctx, 5*time.Second, key).Result()
	}()

	// Wait a bit then push
	time.Sleep(500 * time.Millisecond)
	pusherClient.RPush(ctx, key, "pushed value")

	wg.Wait()

	if blpopErr != nil {
		t.Fatalf("BLPOP failed: %v", blpopErr)
	}
	if len(result) != 2 {
		t.Fatalf("Expected 2 elements in result, got %d", len(result))
	}
	if result[1] != "pushed value" {
		t.Errorf("Expected 'pushed value', got %s", result[1])
	}

	// Cleanup
	client.Del(ctx, key)
}

// TestBLPopMultipleKeys tests BLPOP with multiple keys
func TestBLPopMultipleKeys(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	key1 := "test:list:blpop:multi:1"
	key2 := "test:list:blpop:multi:2"

	// Only push to second key
	client.RPush(ctx, key2, "from key2")

	result, err := client.BLPop(ctx, time.Second, key1, key2).Result()
	if err != nil {
		t.Fatalf("BLPOP multiple keys failed: %v", err)
	}
	if result[0] != key2 {
		t.Errorf("Expected key %s, got %s", key2, result[0])
	}
	if result[1] != "from key2" {
		t.Errorf("Expected 'from key2', got %s", result[1])
	}

	// Cleanup
	client.Del(ctx, key1, key2)
}

// TestBLPopMultipleKeysBlockingOnSecond tests BLPOP blocking on multiple keys
// where data is pushed to a key other than the first one
func TestBLPopMultipleKeysBlockingOnSecond(t *testing.T) {
	client := newTestClient()
	pusherClient := newTestClient()
	defer client.Close()
	defer pusherClient.Close()
	ctx := context.Background()

	key1 := "test:list:blpop:multiblock:1"
	key2 := "test:list:blpop:multiblock:2"
	key3 := "test:list:blpop:multiblock:3"

	// Make sure keys don't exist
	client.Del(ctx, key1, key2, key3)

	var wg sync.WaitGroup
	var result []string
	var blpopErr error

	// Start BLPOP on all three keys
	wg.Add(1)
	go func() {
		defer wg.Done()
		result, blpopErr = client.BLPop(ctx, 5*time.Second, key1, key2, key3).Result()
	}()

	// Wait a bit, then push to the second key
	time.Sleep(300 * time.Millisecond)
	pusherClient.RPush(ctx, key2, "value from key2")

	wg.Wait()

	if blpopErr != nil {
		t.Fatalf("BLPOP failed: %v", blpopErr)
	}
	if len(result) != 2 {
		t.Fatalf("Expected 2 elements in result, got %d", len(result))
	}
	if result[0] != key2 {
		t.Errorf("Expected key %s, got %s", key2, result[0])
	}
	if result[1] != "value from key2" {
		t.Errorf("Expected 'value from key2', got %s", result[1])
	}

	// Cleanup
	client.Del(ctx, key1, key2, key3)
}

// TestBLPopMultipleKeysBlockingOnThird tests BLPOP blocking on multiple keys
// where data is pushed to the third key
func TestBLPopMultipleKeysBlockingOnThird(t *testing.T) {
	client := newTestClient()
	pusherClient := newTestClient()
	defer client.Close()
	defer pusherClient.Close()
	ctx := context.Background()

	key1 := "test:list:blpop:multiblock3:1"
	key2 := "test:list:blpop:multiblock3:2"
	key3 := "test:list:blpop:multiblock3:3"

	// Make sure keys don't exist
	client.Del(ctx, key1, key2, key3)

	var wg sync.WaitGroup
	var result []string
	var blpopErr error

	// Start BLPOP on all three keys
	wg.Add(1)
	go func() {
		defer wg.Done()
		result, blpopErr = client.BLPop(ctx, 5*time.Second, key1, key2, key3).Result()
	}()

	// Wait a bit, then push to the third key
	time.Sleep(300 * time.Millisecond)
	pusherClient.RPush(ctx, key3, "value from key3")

	wg.Wait()

	if blpopErr != nil {
		t.Fatalf("BLPOP failed: %v", blpopErr)
	}
	if len(result) != 2 {
		t.Fatalf("Expected 2 elements in result, got %d", len(result))
	}
	if result[0] != key3 {
		t.Errorf("Expected key %s, got %s", key3, result[0])
	}
	if result[1] != "value from key3" {
		t.Errorf("Expected 'value from key3', got %s", result[1])
	}

	// Cleanup
	client.Del(ctx, key1, key2, key3)
}

// TestBLPopMultipleKeysTimeoutAllEmpty tests BLPOP timeout when all keys are empty
func TestBLPopMultipleKeysTimeoutAllEmpty(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	key1 := "test:list:blpop:timeout:1"
	key2 := "test:list:blpop:timeout:2"
	key3 := "test:list:blpop:timeout:3"

	// Make sure keys don't exist
	client.Del(ctx, key1, key2, key3)

	start := time.Now()
	_, err := client.BLPop(ctx, time.Second, key1, key2, key3).Result()
	elapsed := time.Since(start)

	if err != redis.Nil {
		t.Errorf("Expected redis.Nil on timeout, got %v", err)
	}

	// Should have waited approximately 1 second
	if elapsed < 900*time.Millisecond {
		t.Errorf("BLPOP returned too quickly: %v", elapsed)
	}
}

// TestBLPopMultipleKeysPriority tests that BLPOP returns from first key with data
// when multiple keys have data (priority by key order)
func TestBLPopMultipleKeysPriority(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	key1 := "test:list:blpop:priority:1"
	key2 := "test:list:blpop:priority:2"
	key3 := "test:list:blpop:priority:3"

	// Make sure keys don't exist
	client.Del(ctx, key1, key2, key3)

	// Push to keys 2 and 3 (not key 1)
	client.RPush(ctx, key2, "value2")
	client.RPush(ctx, key3, "value3")

	// BLPOP should return from key2 (first one with data in order)
	result, err := client.BLPop(ctx, time.Second, key1, key2, key3).Result()
	if err != nil {
		t.Fatalf("BLPOP failed: %v", err)
	}
	if result[0] != key2 {
		t.Errorf("Expected key %s (first with data), got %s", key2, result[0])
	}
	if result[1] != "value2" {
		t.Errorf("Expected 'value2', got %s", result[1])
	}

	// Cleanup
	client.Del(ctx, key1, key2, key3)
}

// TestBLPopMultipleKeysConcurrentPushers tests BLPOP with concurrent pushers to different keys
func TestBLPopMultipleKeysConcurrentPushers(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	key1 := "test:list:blpop:concurrent:1"
	key2 := "test:list:blpop:concurrent:2"
	key3 := "test:list:blpop:concurrent:3"

	// Clean up first
	client.Del(ctx, key1, key2, key3)

	var wg sync.WaitGroup
	var result []string
	var blpopErr error

	// Start BLPOP
	wg.Add(1)
	go func() {
		defer wg.Done()
		result, blpopErr = client.BLPop(ctx, 5*time.Second, key1, key2, key3).Result()
	}()

	// Wait for BLPOP to start blocking
	time.Sleep(200 * time.Millisecond)

	// Start multiple concurrent pushers (only one should win)
	pusher1 := newTestClient()
	pusher2 := newTestClient()
	pusher3 := newTestClient()
	defer pusher1.Close()
	defer pusher2.Close()
	defer pusher3.Close()

	var pushWg sync.WaitGroup
	pushWg.Add(3)
	go func() {
		defer pushWg.Done()
		time.Sleep(50 * time.Millisecond)
		pusher1.RPush(ctx, key1, "from pusher1")
	}()
	go func() {
		defer pushWg.Done()
		time.Sleep(50 * time.Millisecond)
		pusher2.RPush(ctx, key2, "from pusher2")
	}()
	go func() {
		defer pushWg.Done()
		time.Sleep(50 * time.Millisecond)
		pusher3.RPush(ctx, key3, "from pusher3")
	}()

	wg.Wait()
	pushWg.Wait()

	if blpopErr != nil {
		t.Fatalf("BLPOP failed: %v", blpopErr)
	}
	if len(result) != 2 {
		t.Fatalf("Expected 2 elements in result, got %d", len(result))
	}

	// One of the keys should have been returned
	validKeys := map[string]bool{key1: true, key2: true, key3: true}
	if !validKeys[result[0]] {
		t.Errorf("Unexpected key: %s", result[0])
	}

	// Cleanup
	client.Del(ctx, key1, key2, key3)
}

// TestBLPopMultipleKeysCleanup tests that blocked channels are cleaned up properly
// after one key returns data
func TestBLPopMultipleKeysCleanup(t *testing.T) {
	client := newTestClient()
	pusherClient := newTestClient()
	defer client.Close()
	defer pusherClient.Close()
	ctx := context.Background()

	key1 := "test:list:blpop:cleanup:1"
	key2 := "test:list:blpop:cleanup:2"

	// Clean up first
	client.Del(ctx, key1, key2)

	// First BLPOP - push to key2
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(200 * time.Millisecond)
		pusherClient.RPush(ctx, key2, "first value")
	}()

	result, err := client.BLPop(ctx, 5*time.Second, key1, key2).Result()
	wg.Wait()

	if err != nil {
		t.Fatalf("First BLPOP failed: %v", err)
	}
	if result[0] != key2 || result[1] != "first value" {
		t.Errorf("Unexpected first result: %v", result)
	}

	// Second BLPOP - this time push to key1
	// This verifies that the blocking channel for key1 was properly cleaned up
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(200 * time.Millisecond)
		pusherClient.RPush(ctx, key1, "second value")
	}()

	result, err = client.BLPop(ctx, 5*time.Second, key1, key2).Result()
	wg.Wait()

	if err != nil {
		t.Fatalf("Second BLPOP failed: %v", err)
	}
	if result[0] != key1 || result[1] != "second value" {
		t.Errorf("Unexpected second result: %v", result)
	}

	// Cleanup
	client.Del(ctx, key1, key2)
}

// TestListAsFIFOQueue tests using list as a FIFO queue
func TestListAsFIFOQueue(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	key := "test:list:fifo"

	// Push to right, pop from left = FIFO
	for i := 1; i <= 5; i++ {
		client.RPush(ctx, key, i)
	}

	for i := 1; i <= 5; i++ {
		val, err := client.LPop(ctx, key).Result()
		if err != nil {
			t.Fatalf("LPOP failed: %v", err)
		}
		expected := string(rune('0' + i))
		if val != expected {
			t.Errorf("Expected %s, got %s", expected, val)
		}
	}

	// Cleanup
	client.Del(ctx, key)
}

// TestListAsStack tests using list as a stack (LIFO)
func TestListAsStack(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	key := "test:list:stack"

	// Push to left, pop from left = LIFO
	for i := 1; i <= 5; i++ {
		client.LPush(ctx, key, i)
	}

	for i := 5; i >= 1; i-- {
		val, err := client.LPop(ctx, key).Result()
		if err != nil {
			t.Fatalf("LPOP failed: %v", err)
		}
		expected := string(rune('0' + i))
		if val != expected {
			t.Errorf("Expected %s, got %s", expected, val)
		}
	}

	// Cleanup
	client.Del(ctx, key)
}

// =============================================================================
// Pub/Sub Tests
// =============================================================================

// TestPubSubBasic tests basic publish/subscribe functionality
func TestPubSubBasic(t *testing.T) {
	publisher := newTestClient()
	subscriber := newTestClient()
	defer publisher.Close()
	defer subscriber.Close()
	ctx := context.Background()

	channel := "test:pubsub:basic"
	message := "Hello, PubSub!"

	// Subscribe to channel
	pubsub := subscriber.Subscribe(ctx, channel)
	defer pubsub.Close()

	// Wait for subscription confirmation
	_, err := pubsub.Receive(ctx)
	if err != nil {
		t.Fatalf("Failed to receive subscription confirmation: %v", err)
	}

	// Publish message
	receiversCount, err := publisher.Publish(ctx, channel, message).Result()
	if err != nil {
		t.Fatalf("PUBLISH failed: %v", err)
	}
	if receiversCount != 1 {
		t.Errorf("Expected 1 receiver, got %d", receiversCount)
	}

	// Receive the message
	msg, err := pubsub.ReceiveMessage(ctx)
	if err != nil {
		t.Fatalf("Failed to receive message: %v", err)
	}
	if msg.Channel != channel {
		t.Errorf("Expected channel %s, got %s", channel, msg.Channel)
	}
	if msg.Payload != message {
		t.Errorf("Expected message %s, got %s", message, msg.Payload)
	}
}

// TestPubSubMultipleSubscribers tests publishing to multiple subscribers
func TestPubSubMultipleSubscribers(t *testing.T) {
	publisher := newTestClient()
	sub1 := newTestClient()
	sub2 := newTestClient()
	defer publisher.Close()
	defer sub1.Close()
	defer sub2.Close()
	ctx := context.Background()

	channel := "test:pubsub:multi"

	// Subscribe both clients
	pubsub1 := sub1.Subscribe(ctx, channel)
	pubsub2 := sub2.Subscribe(ctx, channel)
	defer pubsub1.Close()
	defer pubsub2.Close()

	// Wait for subscription confirmations
	pubsub1.Receive(ctx)
	pubsub2.Receive(ctx)

	// Publish
	receiversCount, err := publisher.Publish(ctx, channel, "broadcast").Result()
	if err != nil {
		t.Fatalf("PUBLISH failed: %v", err)
	}
	if receiversCount != 2 {
		t.Errorf("Expected 2 receivers, got %d", receiversCount)
	}

	// Both should receive
	msg1, _ := pubsub1.ReceiveMessage(ctx)
	msg2, _ := pubsub2.ReceiveMessage(ctx)

	if msg1.Payload != "broadcast" {
		t.Errorf("Subscriber 1: expected 'broadcast', got '%s'", msg1.Payload)
	}
	if msg2.Payload != "broadcast" {
		t.Errorf("Subscriber 2: expected 'broadcast', got '%s'", msg2.Payload)
	}
}

// TestPubSubMultipleChannels tests subscribing to multiple channels
func TestPubSubMultipleChannels(t *testing.T) {
	publisher := newTestClient()
	subscriber := newTestClient()
	defer publisher.Close()
	defer subscriber.Close()
	ctx := context.Background()

	ch1 := "test:pubsub:ch1"
	ch2 := "test:pubsub:ch2"

	// Subscribe to both channels
	pubsub := subscriber.Subscribe(ctx, ch1, ch2)
	defer pubsub.Close()

	// Wait for both subscription confirmations
	pubsub.Receive(ctx)
	pubsub.Receive(ctx)

	// Publish to channel 1
	publisher.Publish(ctx, ch1, "message1")
	msg, _ := pubsub.ReceiveMessage(ctx)
	if msg.Channel != ch1 || msg.Payload != "message1" {
		t.Errorf("Expected message1 on %s, got %s on %s", ch1, msg.Payload, msg.Channel)
	}

	// Publish to channel 2
	publisher.Publish(ctx, ch2, "message2")
	msg, _ = pubsub.ReceiveMessage(ctx)
	if msg.Channel != ch2 || msg.Payload != "message2" {
		t.Errorf("Expected message2 on %s, got %s on %s", ch2, msg.Payload, msg.Channel)
	}
}

// TestPubSubUnsubscribe tests unsubscribing from channels
func TestPubSubUnsubscribe(t *testing.T) {
	publisher := newTestClient()
	subscriber := newTestClient()
	defer publisher.Close()
	defer subscriber.Close()
	ctx := context.Background()

	channel := "test:pubsub:unsub"

	// Subscribe
	pubsub := subscriber.Subscribe(ctx, channel)
	defer pubsub.Close()
	pubsub.Receive(ctx)

	// Initial publish should reach subscriber
	receiversCount, _ := publisher.Publish(ctx, channel, "before unsub").Result()
	if receiversCount != 1 {
		t.Errorf("Expected 1 receiver before unsub, got %d", receiversCount)
	}

	// Consume the message
	pubsub.ReceiveMessage(ctx)

	// Unsubscribe
	err := pubsub.Unsubscribe(ctx, channel)
	if err != nil {
		t.Fatalf("UNSUBSCRIBE failed: %v", err)
	}

	// Wait a bit for unsubscribe to process
	time.Sleep(100 * time.Millisecond)

	// Publish after unsubscribe should reach 0 subscribers
	receiversCount, _ = publisher.Publish(ctx, channel, "after unsub").Result()
	if receiversCount != 0 {
		t.Errorf("Expected 0 receivers after unsub, got %d", receiversCount)
	}
}

// TestPubSubPublishNoSubscribers tests publishing with no subscribers
func TestPubSubPublishNoSubscribers(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	receiversCount, err := client.Publish(ctx, "no:subscribers:channel", "message").Result()
	if err != nil {
		t.Fatalf("PUBLISH failed: %v", err)
	}
	if receiversCount != 0 {
		t.Errorf("Expected 0 receivers, got %d", receiversCount)
	}
}

// TestPubSubChannel tests using the channel interface
func TestPubSubChannel(t *testing.T) {
	publisher := newTestClient()
	subscriber := newTestClient()
	defer publisher.Close()
	defer subscriber.Close()
	ctx := context.Background()

	channel := "test:pubsub:channel"
	numMessages := 5

	pubsub := subscriber.Subscribe(ctx, channel)
	defer pubsub.Close()

	// Get channel for messages
	ch := pubsub.Channel()

	// Wait for subscription
	time.Sleep(100 * time.Millisecond)

	// Publish messages
	for i := 0; i < numMessages; i++ {
		publisher.Publish(ctx, channel, i)
	}

	// Receive messages
	received := 0
	timeout := time.After(2 * time.Second)
	for received < numMessages {
		select {
		case msg := <-ch:
			if msg.Channel != channel {
				t.Errorf("Expected channel %s, got %s", channel, msg.Channel)
			}
			received++
		case <-timeout:
			t.Fatalf("Timeout: only received %d of %d messages", received, numMessages)
		}
	}
}

// TestPubSubPingInSubscribedMode tests PING while subscribed
func TestPubSubPingInSubscribedMode(t *testing.T) {
	subscriber := newTestClient()
	defer subscriber.Close()
	ctx := context.Background()

	pubsub := subscriber.Subscribe(ctx, "test:pubsub:ping")
	defer pubsub.Close()

	// Wait for subscription
	pubsub.Receive(ctx)

	// Ping should still work in subscribed mode
	err := pubsub.Ping(ctx)
	if err != nil {
		t.Fatalf("PING in subscribed mode failed: %v", err)
	}
}

// TestPubSubConcurrentPublish tests concurrent publishing
func TestPubSubConcurrentPublish(t *testing.T) {
	subscriber := newTestClient()
	defer subscriber.Close()
	ctx := context.Background()

	channel := "test:pubsub:concurrent"
	numPublishers := 5
	messagesPerPublisher := 10

	pubsub := subscriber.Subscribe(ctx, channel)
	defer pubsub.Close()
	pubsub.Receive(ctx)

	// Get channel
	ch := pubsub.Channel()

	// Start concurrent publishers
	var wg sync.WaitGroup
	for i := 0; i < numPublishers; i++ {
		wg.Add(1)
		go func(publisherID int) {
			defer wg.Done()
			pub := newTestClient()
			defer pub.Close()
			for j := 0; j < messagesPerPublisher; j++ {
				pub.Publish(ctx, channel, publisherID*100+j)
			}
		}(i)
	}

	// Receive all messages
	totalExpected := numPublishers * messagesPerPublisher
	received := 0
	timeout := time.After(5 * time.Second)

	for received < totalExpected {
		select {
		case <-ch:
			received++
		case <-timeout:
			t.Fatalf("Timeout: received %d of %d messages", received, totalExpected)
		}
	}

	wg.Wait()
}

// =============================================================================
// Integration Tests
// =============================================================================

// TestConcurrentSetGet tests concurrent SET and GET operations
func TestConcurrentSetGet(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	numGoroutines := 10
	numOperations := 100

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := "test:concurrent:" + string(rune('0'+id)) + ":" + string(rune('0'+j%10))
				value := "value" + string(rune('0'+j%10))
				client.Set(ctx, key, value, 0)
				client.Get(ctx, key)
			}
		}(i)
	}
	wg.Wait()

	// Cleanup
	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < 10; j++ {
			key := "test:concurrent:" + string(rune('0'+i)) + ":" + string(rune('0'+j))
			client.Del(ctx, key)
		}
	}
}

// TestMixedOperations tests various operations in sequence
func TestMixedOperations(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	// KV operations
	client.Set(ctx, "mixed:kv", "value", 0)
	val, _ := client.Get(ctx, "mixed:kv").Result()
	if val != "value" {
		t.Errorf("KV test failed: expected 'value', got '%s'", val)
	}

	// List operations
	client.RPush(ctx, "mixed:list", "a", "b", "c")
	length, _ := client.LLen(ctx, "mixed:list").Result()
	if length != 3 {
		t.Errorf("List test failed: expected length 3, got %d", length)
	}

	// Type checking
	kvType, _ := client.Type(ctx, "mixed:kv").Result()
	listType, _ := client.Type(ctx, "mixed:list").Result()
	if kvType != "string" {
		t.Errorf("Type test failed: expected 'string', got '%s'", kvType)
	}
	if listType != "list" {
		t.Errorf("Type test failed: expected 'list', got '%s'", listType)
	}

	// EXISTS check
	exists, _ := client.Exists(ctx, "mixed:kv", "mixed:list", "mixed:nonexistent").Result()
	if exists != 2 {
		t.Errorf("EXISTS test failed: expected 2, got %d", exists)
	}

	// Cleanup
	client.Del(ctx, "mixed:kv", "mixed:list")
}

// TestSpecialCharacters tests handling of special characters in keys and values
func TestSpecialCharacters(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	testCases := []struct {
		key   string
		value string
	}{
		{"test:special:space", "hello world"},
		{"test:special:newline", "hello\nworld"},
		{"test:special:tab", "hello\tworld"},
		{"test:special:unicode", "hello 世界 мир 🌍"},
		{"test:special:binary", "\x00\x01\x02\x03"},
		{"test:special:quotes", `"hello" 'world'`},
	}

	for _, tc := range testCases {
		err := client.Set(ctx, tc.key, tc.value, 0).Err()
		if err != nil {
			t.Errorf("SET failed for key %s: %v", tc.key, err)
			continue
		}

		result, err := client.Get(ctx, tc.key).Result()
		if err != nil {
			t.Errorf("GET failed for key %s: %v", tc.key, err)
			continue
		}
		if result != tc.value {
			t.Errorf("Value mismatch for key %s: expected %q, got %q", tc.key, tc.value, result)
		}

		// Cleanup
		client.Del(ctx, tc.key)
	}
}

// TestClientInfo tests the CLIENT command (basic connectivity check)
func TestClientInfo(t *testing.T) {
	client := newTestClient()
	defer client.Close()
	ctx := context.Background()

	// Set client name using raw command
	err := client.Do(ctx, "CLIENT", "SETNAME", "test-client").Err()
	if err != nil {
		t.Fatalf("CLIENT SETNAME failed: %v", err)
	}

	// Get client name using raw command
	result, err := client.Do(ctx, "CLIENT", "GETNAME").Result()
	if err != nil {
		t.Fatalf("CLIENT GETNAME failed: %v", err)
	}
	name, ok := result.(string)
	if !ok {
		t.Fatalf("Expected string result, got %T", result)
	}
	if name != "test-client" {
		t.Errorf("Expected client name 'test-client', got '%s'", name)
	}
}

// TestConnectionPooling tests multiple operations with connection pooling
func TestConnectionPooling(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		PoolSize: 10,
	})
	defer client.Close()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := "pool:test:" + string(rune('0'+id%10))
			client.Set(ctx, key, id, 0)
			client.Get(ctx, key)
		}(i)
	}
	wg.Wait()

	// Cleanup
	for i := 0; i < 10; i++ {
		client.Del(ctx, "pool:test:"+string(rune('0'+i)))
	}
}
