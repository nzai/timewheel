# Hierarchical Time Wheel Implementation in Go

A concurrent-safe, hierarchical time wheel implementation with asynchronous callback support, designed for efficient timed task management.

## Features

- **Hierarchical Design**: 3-layer wheel (seconds/minutes/hours) with auto-cascading
- **Asynchronous Callbacks**: Non-blocking expiration handling
- **Concurrent-Safe**: Built with RWMutex for thread safety
- **Precision Control**: Configurable base interval (1s/1m/1h)
- **Task Management**:
  - Set/Update/Delete tasks
  - Dynamic expiration adjustment
  - Bulk flush support
- **Auto-Demotion**: Handles long-duration tasks through round-based cascading

## Installation

```bash
go get github.com/nzai/timewheel
```

## Quick Start
```go
package main

import (
    "fmt"
    "time"
    "github.com/nzai/timewheel"
)

func main() {
    // Initialize with 1-second base interval, 60 slots, async callback
    tw := timewheel.NewTimeWheel(time.Second, 60, func(key string, value interface{}) {
        fmt.Printf("Expired: %s -> %v\n", key, value)
    })

    // Add task with 30s expiration
    tw.Set("task1", "data", 30*time.Second)

    // Keep main thread alive
    select {}
}
```

## Core API
### Initialization

```go
func NewTimeWheel(
    baseInterval time.Duration,          // Base time unit (e.g., 1s/1m)
    slotsPerLayer int,                   // Slots per layer (recommend 60)
    callback func(key string, value any) // Async expiration handler
) *TimeWheel
```

### Task Operations

```go
// Set/Update task
tw.Set("key", value, 2*time.Hour)

// Delete task
tw.Delete("key")

// Reschedule existing task
tw.Move("key", 15*time.Minute)

// Clear all tasks
tw.FlushAll()

// Stop time wheel
tw.Stop()
```

## Configuration Guide

### Layer Structure

| Layer | Time Coverage | Example Configuration |
|-------|---------------|-----------------------|
| L1    | 0 - (N×base)  | 60 slots × 1s = 1 minute |
| L2    | N×base - N²×base | 60 slots × 1m = 1 hour |
| L3    | N²×base - N³×base | 60 slots × 1h = 60 hours |

