#!/bin/sh
# Start MinIO in the background
minio server /data --console-address ":9001" &
MINIO_PID=$!

# Wait for MinIO to be ready (simple check, can be improved)
echo "Waiting for MinIO to start..."
sleep 1 # Adjust as needed, or implement a proper health check

echo "Configuring MinIO mc alias..."
mc alias set local http://localhost:9000 ${MINIO_ROOT_USER} ${MINIO_ROOT_PASSWORD}

echo "Creating bucket skilly if it doesn't exist..."
mc mb local/skilly || true # || true ignores error if bucket exists

echo "Adding Kafka event notification for skilly bucket..."
mc event add local/skilly arn:minio:sqs::${MINIO_NOTIFY_KAFKA_ID_kafka1}:kafka \
    --event put \
    --prefix "pfp/"

echo "MinIO setup complete. Bringing MinIO process to foreground."
# Wait for the MinIO server process to exit
wait $MINIO_PID
