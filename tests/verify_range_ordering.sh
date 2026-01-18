#!/bin/bash

# Start the server
./your_program.sh > server_range.log 2>&1 &
SERVER_PID=$!
sleep 1

echo "Test: XRANGE Ordering with Mixed ID Lengths"

# Add entries with different ID lengths
redis-cli -p 6379 XADD s 2-0 f v > /dev/null
redis-cli -p 6379 XADD s 10-0 f v > /dev/null
redis-cli -p 6379 XADD s 100-0 f v > /dev/null

# Range from 0 to 1000
OUTPUT=$(redis-cli -p 6379 XRANGE s - +)

echo "Output:"
echo "$OUTPUT"

# Verify order: 2-0, 10-0, 100-0
if [[ "$OUTPUT" == *"2-0"* ]] && [[ "$OUTPUT" == *"10-0"* ]] && [[ "$OUTPUT" == *"100-0"* ]]; then
    # Check relative order
    POS2=$(echo "$OUTPUT" | grep -n "2-0" | cut -d: -f1)
    POS10=$(echo "$OUTPUT" | grep -n "10-0" | cut -d: -f1)
    POS100=$(echo "$OUTPUT" | grep -n "100-0" | cut -d: -f1)
    
    if [ $POS2 -lt $POS10 ] && [ $POS10 -lt $POS100 ]; then
        echo "PASS: Correct ordering"
    else
        echo "FAIL: Incorrect ordering"
    fi
else
    echo "FAIL: Missing entries"
fi

# Cleanup
kill $SERVER_PID
