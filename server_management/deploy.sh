#!/bin/bash

# This should be run from the root of the monolith project: ./server_management/deploy.sh
# This script will cross compile the app, copy the necessary files to the remote server, and restart the application on the remote server.

# Configuration
REMOTE_USER="your_username"            # Your remote server username
REMOTE_HOST="your_server_ip"           # Remote server IP or hostname
REMOTE_DIR="/path/to/remote/monolith/app" # Path to your monolith project on the remote server
LOCAL_DIR=$(pwd)   # Path to your local monolith project
SSH_KEY="/path/to/your/private/key"    # Path to your SSH private key
SERVICE_NAME="monolith"               # The name of the systemd service
APP_NAME="monolith"                    # The app binary name


# Step 1: Ensure the relevant directories exist on the remote server
echo "Creating directories on the remote server..."
ssh -i $SSH_KEY $REMOTE_USER@$REMOTE_HOST << EOF
    mkdir -p $REMOTE_DIR
EOF

# Step 2: Copy the monolith.service systemd file to the server
echo "Copying the systemd service file to the remote server..."
scp -i $SSH_KEY $LOCAL_DIR/server_management/monolith.service $REMOTE_USER@$REMOTE_HOST:/etc/systemd/system/

# Step 3: Cross compile the app
echo "cross compiling the app..."
GOOS=linux GOARCH=amd64 go build -o $LOCAL_DIR/$APP_NAME $LOCAL_DIR

if [ $? -ne 0 ]; then
    echo "Build failed. Exiting..."
    exit 1
fi


# Step 4: Copy the built app to the remote server. This assumes the app binary has static files embedded in the binary.
echo "Copying app to the remote server..."
scp -i $SSH_KEY $LOCAL_DIR/$APP_NAME $REMOTE_USER@$REMOTE_HOST:$REMOTE_DIR/

if [ $? -ne 0 ]; then
    echo "File transfer failed. Exiting..."
    exit 1
fi


# Step 5: Restart the monolith app on the remote server
echo "Restarting the monolith application on the remote server..."
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

# Step 6: Clean up
echo "Cleaning up..."
rm $LOCAL_DIR/$APP_NAME