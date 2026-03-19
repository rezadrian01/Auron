#!/bin/bash
# Kafka Topics Creation Script for Auron E-Commerce Platform
# This script creates all required Kafka topics

set -e

# Configuration
KAFKA_BROKER="${KAFKA_BROKER:-localhost:9092}"
PARTITIONS="${PARTITIONS:-6}"
REPLICATION_FACTOR="${REPLICATION_FACTOR:-1}"
RETENTION_HOURS="${RETENTION_HOURS:-168}"  # 7 days

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Auron Kafka Topics Creator${NC}"
echo "Broker: $KAFKA_BROKER"
echo "Partitions: $PARTITIONS"
echo "Replication Factor: $REPLICATION_FACTOR"
echo ""

# Function to create a topic
create_topic() {
    local topic=$1
    local partitions=$2

    echo -n "Creating topic '$topic'... "

    # Check if topic exists
    if kafka-topics --describe --topic "$topic" --bootstrap-server "$KAFKA_BROKER" &> /dev/null; then
        echo -e "${YELLOW}already exists${NC}"
        return 0
    fi

    # Create topic
    kafka-topics \
        --create \
        --if-not-exists \
        --topic "$topic" \
        --bootstrap-server "$KAFKA_BROKER" \
        --partitions "$partitions" \
        --replication-factor "$REPLICATION_FACTOR" \
        --config "retention.hours=$RETENTION_HOURS" \
        --config "cleanup.policy=delete" \
        --config "compression.type=snappy" \
        &> /dev/null

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}created${NC}"
    else
        echo -e "${RED}failed${NC}"
        return 1
    fi
}

# Wait for Kafka to be ready
wait_for_kafka() {
    echo -n "Waiting for Kafka to be ready... "
    local max_attempts=30
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        if kafka-broker-api-versions --bootstrap-server "$KAFKA_BROKER" &> /dev/null; then
            echo -e "${GREEN}ready${NC}"
            return 0
        fi
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done

    echo -e "${RED}timeout${NC}"
    return 1
}

# Main execution
main() {
    # Wait for Kafka
    wait_for_kafka || exit 1

    echo ""
    echo "Creating topics..."

    # User events (3 partitions - lower volume)
    create_topic "user.registered" 3

    # Order events (6 partitions - high volume)
    create_topic "order.created" 6

    # Payment events (6 partitions - high volume)
    create_topic "payment.processed" 6
    create_topic "payment.failed" 6

    # Inventory events (6 partitions - high volume)
    create_topic "inventory.updated" 6
    create_topic "inventory.failed" 3

    # Notification events (3 partitions - lower volume)
    create_topic "notification.dlq" 3

    echo ""
    echo -e "${GREEN}All topics created successfully!${NC}"
    echo ""
    echo "Topics:"
    kafka-topics --list --bootstrap-server "$KAFKA_BROKER"
}

# Run main function
main "$@"
