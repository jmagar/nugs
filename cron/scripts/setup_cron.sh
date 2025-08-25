#!/bin/bash

# Setup script to add the monitor to crontab

SCRIPT_PATH="/home/jmagar/code/nugs/cron/monitor_artists.sh"
LOG_PATH="/home/jmagar/code/nugs/cron/monitor.log"

# Check if already in crontab
if crontab -l 2>/dev/null | grep -q "monitor_artists.sh"; then
    echo "Monitor script already in crontab"
else
    # Add to crontab (runs daily at 2 AM)
    (crontab -l 2>/dev/null; echo "0 2 * * * $SCRIPT_PATH >> $LOG_PATH 2>&1") | crontab -
    echo "Added monitor script to crontab - will run daily at 2 AM"
fi

echo ""
echo "Current crontab entries:"
crontab -l | grep monitor_artists.sh

echo ""
echo "To manually run the monitor now:"
echo "  ./monitor_artists.sh"
echo ""
echo "To view the cron logs:"
echo "  tail -f monitor.log"
echo ""
echo "To edit the artists to monitor:"
echo "  nano monitor_config.json"