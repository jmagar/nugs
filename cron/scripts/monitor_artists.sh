#!/bin/bash

# Daily artist monitor script for Nugs
# Add this to crontab for daily execution:
# 0 2 * * * /home/jmagar/code/nugs/cron/monitor_artists.sh >> /home/jmagar/code/nugs/cron/monitor.log 2>&1

cd /home/jmagar/code/nugs/cron

echo "========================================="
echo "Starting artist monitor check: $(date)"
echo "========================================="

# Build the Go program if needed
if [ ! -f "monitor_artists" ] || [ "monitor_artists.go" -nt "monitor_artists" ]; then
    echo "Building monitor_artists..."
    go build monitor_artists.go
fi

# Run the monitor
./monitor_artists

# Optional: Sync to remote server after downloads
# Uncomment if you want automatic sync to tootie
# if [ $? -eq 0 ]; then
#     echo "Syncing to remote server..."
#     rsync -avP -e "ssh -p 29229" /home/jmagar/code/nugs/downloads/ root@tootie:/mnt/user/data/media/music/
# fi

echo "Monitor check complete: $(date)"
echo ""