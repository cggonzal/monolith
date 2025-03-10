#!/bin/bash

# Configuration
REMOTE_USER="your_username"            # Your remote server username
REMOTE_HOST="your_server_ip"           # Remote server IP or hostname
REMOTE_DIR="/path/to/remote/beego/app" # Path to your monolith project on the remote server
LOCAL_DIR="/path/to/local/beego/app"   # Path to your local monolith project
SSH_KEY="/path/to/your/private/key"    # Path to your SSH private key (optional, if using SSH keys)
SERVICE_NAME="monolith"               # The name of the systemd service (optional)
APP_NAME="monolith"                    # The app binary name

# Optional: Build the Beego app locally
echo "Building the Beego app locally..."
GOOS=linux GOARCH=amd64 go build -o $LOCAL_DIR/$APP_NAME $LOCAL_DIR

if [ $? -ne 0 ]; then
    echo "Build failed. Exiting..."
    exit 1
fi

# Step 1: Copy the built app and static files to the remote server
echo "Copying files to the remote server..."
scp -i $SSH_KEY $LOCAL_DIR/$APP_NAME $REMOTE_USER@$REMOTE_HOST:$REMOTE_DIR/
scp -i $SSH_KEY -r $LOCAL_DIR/conf $REMOTE_USER@$REMOTE_HOST:$REMOTE_DIR/
scp -i $SSH_KEY -r $LOCAL_DIR/views $REMOTE_USER@$REMOTE_HOST:$REMOTE_DIR/
scp -i $SSH_KEY -r $LOCAL_DIR/static $REMOTE_USER@$REMOTE_HOST:$REMOTE_DIR/

if [ $? -ne 0 ]; then
    echo "File transfer failed. Exiting..."
    exit 1
fi

# Step 2: Restart the Beego app on the remote server
echo "Restarting the Beego application on the remote server..."
ssh -i $SSH_KEY $REMOTE_USER@$REMOTE_HOST << EOF
    # Navigate to the application directory
    cd $REMOTE_DIR

    # this assumes that the service file is already set up inside of /etc/systemd/system/ , and that 
    # the application handles gracefully shutting down, and
    # systemd will buffer the connections that come between the time the app shuts down and restarts so that no connections are dropped (my understanding is that systemd will do this buffering automatically).
    # with the above conditions in this comment, we should have zero downtime deploys
    sudo systemctl reload $SERVICE_NAME

    echo "Deployment completed successfully."
EOF
