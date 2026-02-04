package main

import (
"context"
"fmt"
"log"

"github.com/dianlight/srat/service"
)

// Example demonstrating how to use the filesystem adapter pattern
func main() {
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Create filesystem service (passing nil for eventBus in this example)
fsService := service.NewFilesystemService(ctx, cancel, nil)

// List all supported filesystem types
fmt.Println("=== Supported Filesystems ===")
types := fsService.ListSupportedTypes()
for _, fsType := range types {
fmt.Printf("- %s\n", fsType)
}
fmt.Println()

// Get detailed support information
fmt.Println("=== Filesystem Support Details ===")
supportInfo, err := fsService.GetSupportedFilesystems(ctx)
if err != nil {
log.Fatalf("Failed to get support info: %v", err)
}

for fsType, support := range supportInfo {
fmt.Printf("\n%s (Alpine package: %s):\n", fsType, support.AlpinePackage)
fmt.Printf("  Can mount: %v\n", support.CanMount)
fmt.Printf("  Can format: %v\n", support.CanFormat)
fmt.Printf("  Can check: %v\n", support.CanCheck)
fmt.Printf("  Can set label: %v\n", support.CanSetLabel)

if len(support.MissingTools) > 0 {
fmt.Printf("  Missing tools: %v\n", support.MissingTools)
}
}
fmt.Println()

// Example: Working with ext4 adapter
fmt.Println("=== Working with ext4 ===")
ext4Adapter, err := fsService.GetAdapter("ext4")
if err != nil {
log.Fatalf("Failed to get ext4 adapter: %v", err)
}

fmt.Printf("Name: %s\n", ext4Adapter.GetName())
fmt.Printf("Description: %s\n", ext4Adapter.GetDescription())

// Get mount flags
fmt.Println("\nMount flags:")
flags := ext4Adapter.GetMountFlags()
for _, flag := range flags {
if flag.NeedsValue {
fmt.Printf("  %s=%s - %s\n", flag.Name, flag.ValueDescription, flag.Description)
} else {
fmt.Printf("  %s - %s\n", flag.Name, flag.Description)
}
}

fmt.Println("\n=== Example Complete ===")
}
