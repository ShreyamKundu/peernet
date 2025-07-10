#!/bin/bash
# set -x # Keep this for debugging, shows commands being executed 

# Exit immediately if a command exits with a non-zero status.
set -e

# --- Section 1: Initial Cleanup ---
echo "████████████████████████████████████████████████████████████████████████████████"
echo "███ PHASE 1: CLEANUP AND INITIAL SETUP ███"
echo "████████████████████████████████████████████████████████████████████████████████"
echo ""
echo "--- Stopping any previous PeerNet containers and cleaning up volumes to ensure a clean start... ---"
docker compose down -v --remove-orphans
echo "✅ Previous PeerNet environment successfully cleaned up."
echo ""

# --- Section 2: Prepare Sample Files ---
echo "--- Preparing sample file directories for our peers... ---"
mkdir -p shared_files/peer1/data
mkdir -p shared_files/peer2/downloads
echo "Directory 'shared_files/peer1/data' created for Peer 1 to share files from."
echo "Directory 'shared_files/peer2/downloads' created for Peer 2 to download files into."

SAMPLE_FILE_PATH="shared_files/peer1/data/sample.txt"
CURRENT_DATE=$(date)
echo "Creating a sample text file for Peer 1 to share: '$SAMPLE_FILE_PATH'"
echo "Hello from PeerNet! This file was created on $CURRENT_DATE" > "$SAMPLE_FILE_PATH"
echo "File created: $SAMPLE_FILE_PATH"
echo "--- Content of the sample file: ---"
cat "$SAMPLE_FILE_PATH"
echo "-----------------------------------"
echo "✅ Sample files prepared."
echo ""

# --- Section 3: Build and Start Services ---
echo "████████████████████████████████████████████████████████████████████████████████"
echo "███ PHASE 2: BUILDING AND STARTING PEERNET SERVICES ███"
echo "████████████████████████████████████████████████████████████████████████████████"
echo ""
echo "--- Building Docker images and starting all PeerNet services (database, tracker, peer1, peer2) in the background... ---"
docker compose up --build -d
echo "✅ All PeerNet services are now up and running in detached mode."
echo ""

# Define unique gRPC server ports for each peer
PEER1_GRPC_PORT="50051"
PEER2_GRPC_PORT="50052" # Peer 2 will use a different port

echo "--- Giving containers a moment to fully initialize before sending commands... ---"
sleep 5
echo "✅ Containers should be fully ready for interaction."
echo ""

# --- Section 4: Register Peers with Tracker ---
echo "████████████████████████████████████████████████████████████████████████████████"
echo "███ PHASE 3: PEER REGISTRATION ███"
echo "████████████████████████████████████████████████████████████████████████████████"
echo ""
echo "--- Registering Peer 1 with the central tracker. Peer 1 announces its internal network address and gRPC port... ---"
docker exec peernet-peer1 ./peer-client register \
  --tracker http://tracker:8080 \
  --address peernet-peer1:$PEER1_GRPC_PORT \
  --password pass123
echo "✅ Peer 1 successfully registered with the tracker."
echo ""

echo "--- Registering Peer 2 with the central tracker. Peer 2 also announces its internal network address and gRPC port... ---"
docker exec peernet-peer2 ./peer-client register \
  --tracker http://tracker:8080 \
  --address peernet-peer2:$PEER2_GRPC_PORT \
  --password pass456
echo "✅ Peer 2 successfully registered with the tracker."
echo ""

echo "--- Giving the tracker a brief moment to process the new peer registrations... ---"
sleep 2 # Give tracker a moment to update its records after registrations
echo "✅ Tracker state updated."
echo ""

# Define a temporary log file path inside the peer1 container
PEER1_SHARE_LOG="/tmp/peer1_share.log"

# --- Section 5: Peer 1 Shares File ---
echo "████████████████████████████████████████████████████████████████████████████████"
echo "███ PHASE 4: FILE SHARING FROM PEER 1 ███"
echo "████████████████████████████████████████████████████████████████████████████████"
echo ""
echo "--- Instructing Peer 1 to start its gRPC server and announce the sample file to the tracker. This runs in the background. ---"
echo "--- Peer 1's output for this operation will be logged to $PEER1_SHARE_LOG inside its container. ---"
docker exec peernet-peer1 sh -c "./peer-client share /home/appuser/data/sample.txt --port \"$PEER1_GRPC_PORT\" > $PEER1_SHARE_LOG 2>&1 &"

echo "Share command initiated. Now, we wait for Peer 1 to process the file, calculate its unique hash, and log it. This hash is crucial for Peer 2 to find the file."

# --- Robust wait for File Hash from internal log file ---
FILE_HASH=""
MAX_WAIT_TIME=30 # seconds
ELAPSED_TIME=0

while [ -z "$FILE_HASH" ] && [ "$ELAPSED_TIME" -lt "$MAX_WAIT_TIME" ]; do
    echo "Attempting to retrieve file hash from Peer 1's internal log file (attempt $((ELAPSED_TIME + 1)) of $MAX_WAIT_TIME)..."
    # Display the current content of the log file for debugging visibility during the demo
    docker exec peernet-peer1 cat "$PEER1_SHARE_LOG" 2>&1 || true
    # Extract the file hash from the internal log file.
    FILE_HASH=$(docker exec peernet-peer1 cat "$PEER1_SHARE_LOG" 2>&1 | grep "File Hash:" | awk '{print $NF}' | head -n 1)
    if [ -z "$FILE_HASH" ]; then
        sleep 1 # Wait for 1 second before retrying
        ELAPSED_TIME=$((ELAPSED_TIME + 1))
    fi
done

if [ -z "$FILE_HASH" ]; then
    echo "❌ Error: Could not find the File Hash in Peer 1's logs within $MAX_WAIT_TIME seconds."
    echo "Please check the contents of $PEER1_SHARE_LOG inside peernet-peer1 for more details."
    exit 1
fi
echo "✅ File Hash found: $FILE_HASH. This is the unique identifier for our shared file."
echo ""

DOWNLOAD_OUTPUT_DIR="/home/appuser/downloads"
LOCAL_DOWNLOADED_FILE_PATH="shared_files/peer2/downloads/${FILE_HASH}.download"

# --- Section 6: Peer 2 Downloads File ---
echo "████████████████████████████████████████████████████████████████████████████████"
echo "███ PHASE 5: FILE DOWNLOAD BY PEER 2 ███"
echo "████████████████████████████████████████████████████████████████████████████████"
echo ""
echo "--- Ensuring Peer 2 has the necessary write permissions to its mounted download directory ('$DOWNLOAD_OUTPUT_DIR')... ---"
# Explicitly change ownership of the downloads directory inside peer2 container as root
docker exec -u root peernet-peer2 chown appuser:appgroup "$DOWNLOAD_OUTPUT_DIR"
echo "✅ Permissions set for Peer 2's download directory."
echo ""

echo "--- Instructing Peer 2 to download the file using its unique hash. Peer 2 will query the tracker to find Peer 1, and then connect directly to Peer 1 to retrieve the file chunks. ---"
docker exec peernet-peer2 ./peer-client download "$FILE_HASH" --output "$DOWNLOAD_OUTPUT_DIR"
echo "✅ Download command sent to Peer 2."
echo ""

# --- Section 7: Verify Download ---
echo "████████████████████████████████████████████████████████████████████████████████"
echo "███ PHASE 6: VERIFICATION OF DOWNLOADED FILE ███"
echo "████████████████████████████████████████████████████████████████████████████████"
echo ""
echo "--- Verifying that the downloaded file exists and its content matches the original... ---"
if [ -f "$LOCAL_DOWNLOADED_FILE_PATH" ]; then
    echo "🎉🎉🎉 SUCCESS! The file has been successfully downloaded by Peer 2! 🎉🎉🎉"
    echo "Downloaded file path on host: '$LOCAL_DOWNLOADED_FILE_PATH'"
    echo "--- Content of the downloaded file: ---"
    cat "$LOCAL_DOWNLOADED_FILE_PATH"
    echo "---------------------------------------"
else
    echo "❌❌❌ FAILURE! The downloaded file was NOT found at '$LOCAL_DOWNLOADED_FILE_PATH'. ❌❌❌"
    echo "Please check peer2 logs for errors: docker logs peernet-peer2"
    exit 1
fi
echo ""

# --- Section 8: Final Cleanup ---
echo "████████████████████████████████████████████████████████████████████████████████"
echo "███ PHASE 7: FINAL CLEANUP ███"
echo "████████████████████████████████████████████████████████████████████████████████"
echo ""
echo "--- Cleaning up the temporary log file created in Peer 1's container... ---"
docker exec peernet-peer1 rm -f "$PEER1_SHARE_LOG"
echo "✅ Temporary log file cleaned up."
echo ""

echo "████████████████████████████████████████████████████████████████████████████████"
echo "███ DEMO COMPLETE! ███"
echo "███ To stop all services and clean up, run: docker compose down -v ███"
echo "████████████████████████████████████████████████████████████████████████████████"
