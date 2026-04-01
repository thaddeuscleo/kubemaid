# --- Variables ---
APP_NAME := "kubemaid"
OUT_DIR := "bin"

# --- Default ---
# Running `just` without arguments will list all available commands
default:
    @just --list

# --- Grouped Builds ---
# Build for all platforms (macOS and Windows)
all: macos windows

# Build both architectures for macOS
macos: macos-arm64 macos-amd64

# Build both architectures for Windows
windows: windows-amd64 windows-arm64

# --- Specific macOS Targets ---
# Build for macOS Apple Silicon
macos-arm64:
    @echo "Building for macOS (arm64)..."
    GOOS=darwin GOARCH=arm64 go build -o {{OUT_DIR}}/{{APP_NAME}}-darwin-arm64 .

# Build for macOS Intel
macos-amd64:
    @echo "Building for macOS (amd64)..."
    GOOS=darwin GOARCH=amd64 go build -o {{OUT_DIR}}/{{APP_NAME}}-darwin-amd64 .

# --- Specific Windows Targets ---
# Build for Windows Intel
windows-amd64:
    @echo "Building for Windows (amd64)..."
    GOOS=windows GOARCH=amd64 go build -o {{OUT_DIR}}/{{APP_NAME}}-windows-amd64.exe .

# Build for Windows ARM
windows-arm64:
    @echo "Building for Windows (arm64)..."
    GOOS=windows GOARCH=arm64 go build -o {{OUT_DIR}}/{{APP_NAME}}-windows-arm64.exe .

# --- Utilities ---
# Clean the build directory
clean:
    @echo "Cleaning build directory..."
    rm -rf {{OUT_DIR}}