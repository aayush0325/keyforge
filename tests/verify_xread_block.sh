#!/bin/bash

# Start the server
./your_program.sh > server.log 2>&1 &
SERVER_PID=$!
sleep 1

echo "Test 1: Blocking with Data Arrival"
# Client A blocks
# We need to run this in background and capture output
(
    sleep 0.5
    redis-cli -p 6379 XADD s 0-1 foo bar > /dev/null
) &

START=$(date +%s%N)
OUTPUT=$(redis-cli -p 6379 XREAD BLOCK 2000 STREAMS s 0-0)
END=$(date +%s%N)
DURATION=$((($END - $START) / 1000000))

echo "Output: $OUTPUT"
echo "Duration: ${DURATION}ms"

if [[ "$OUTPUT" == *"foo"* ]] && [[ "$OUTPUT" == *"bar"* ]]; then
    echo "PASS: Data received"
else
    echo "FAIL: Data not received"
fi

if [ $DURATION -lt 2000 ] && [ $DURATION -gt 400 ]; then
    echo "PASS: Unblocked immediately after XADD"
else
    echo "FAIL: Duration unexpected ($DURATION ms)"
fi

echo "------------------------------------------------"

echo "Test 2: Blocking Timeout"
START=$(date +%s%N)
# Use nc to see raw output
OUTPUT=$(echo -e "XREAD BLOCK 1000 STREAMS s 0-1\r\n" | nc localhost 6379)
END=$(date +%s%N)
DURATION=$((($END - $START) / 1000000))

echo "Output: $OUTPUT"
echo "Duration: ${DURATION}ms"

if [[ "$OUTPUT" == *"*-1"* ]]; then
    echo "PASS: Timed out with nil (*-1)"
else
    echo "FAIL: Did not time out correctly. Output: '$OUTPUT'"
    echo "Server Log:"
    cat server.log
fi

if [ $DURATION -ge 1000 ]; then
    echo "PASS: Waited for timeout"
else
    echo "FAIL: Returned too early ($DURATION ms)"
fi

echo "------------------------------------------------"

echo "Test 3: Blocking with $"
# Client A blocks with $
(
    sleep 0.5
    redis-cli -p 6379 XADD s 0-2 foo baz > /dev/null
) &

START=$(date +%s%N)
OUTPUT=$(echo -e "XREAD BLOCK 2000 STREAMS s $\r\n" | nc localhost 6379)
END=$(date +%s%N)
DURATION=$((($END - $START) / 1000000))

echo "Output: $OUTPUT"
echo "Duration: ${DURATION}ms"

if [[ "$OUTPUT" == *"foo"* ]] && [[ "$OUTPUT" == *"baz"* ]]; then
    echo "PASS: Data received with $"
else
    echo "FAIL: Data not received with $. Output: '$OUTPUT'"
    cat server.log
fi

if [ $DURATION -lt 2000 ] && [ $DURATION -gt 400 ]; then
    echo "PASS: Unblocked immediately after XADD"
else
    echo "FAIL: Duration unexpected ($DURATION ms)"
fi

# Cleanup
kill $SERVER_PID
