#!/bin/bash
# filepath: /home/sannis/Development/golang/dmr/digiLogRT/build.sh

echo "Building DigiLogRT..."
mkdir -p bin

# Build the application and capture the exit code
go build -ldflags="-s -w" -o bin/digilogrt cmd/digilogrt/main.go
BUILD_EXIT_CODE=$?

# Check if build was successful
if [ $BUILD_EXIT_CODE -eq 0 ]; then
    echo "✓ Build completed successfully!"
    echo "Run with: ./bin/digilogrt"
else
    echo "✗ Build failed with exit code $BUILD_EXIT_CODE"
    echo "Please fix the errors above and try again."
    exit $BUILD_EXIT_CODE
fi