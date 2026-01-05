# Design Document

## Overview

The Ranged HTTP Downloader is designed as a concurrent, memory-efficient file download system that leverages HTTP range requests to achieve high-performance downloads. The system follows a producer-consumer pattern where a central RequestManager orchestrates multiple HttpFragment workers that download file chunks in parallel.

The architecture prioritizes memory efficiency by streaming data directly from HTTP responses to disk buffers, avoiding loading entire fragments into memory simultaneously. Thread safety is achieved through careful synchronization of file I/O operations while maintaining high concurrency for network operations.

## Architecture

The system follows a layered architecture with clear separation of concerns:

```
┌─────────────────┐
│   Main Entry    │ ← Configuration & Initialization
└─────────────────┘
         │
┌─────────────────┐
│ Request Manager │ ← Orchestration & Coordination  
└─────────────────┘
         │
┌─────────────────┐
│  HTTP Fragment  │ ← Individual Download Workers
└─────────────────┘
         │
┌─────────────────┐
│   File System   │ ← Disk I/O Operations
└─────────────────┘
```

**Key Architectural Principles:**
- **Separation of Concerns**: Each component has a single, well-defined responsibility
- **Concurrent Execution**: Network I/O and disk I/O operations run in parallel where safe
- **Resource Management**: Explicit control over memory allocation and cleanup
- **Error Isolation**: Failures in individual fragments don't corrupt the overall download
- **Request Optimization**: Initial request serves dual purpose of size detection and first fragment download

## Components and Interfaces

### RequestManager
**Responsibility**: Orchestrates the entire download process, manages worker goroutines, and coordinates file I/O.

**Key Methods:**
- `NewRequestManager(src string, dest *os.File) (*RequestManager, error)`: Creates a new manager instance
- `Start() error`: Initiates the download process with optimized initial request
- `initRequest() (*http.Response, error)`: Performs initial request to determine file size and potentially download first fragment
- `processFragment(fragment *HttpFragment)`: Handles individual fragment processing

**Optimization Strategy:**
- Initial request serves dual purpose: Content-Length detection and first fragment download
- If Content-Length is missing or file size ≤ MaxFragmentSize, uses single fragment approach
- For larger files, reuses initial response for first fragment while creating additional fragments in parallel

**State Management:**
- Maintains HTTP client for connection reuse
- Manages semaphore for concurrency control
- Coordinates file writing through mutex synchronization
- Tracks download progress and timing for debug logging
- Handles proper resource cleanup on completion or failure

**Progress Monitoring:**
- Logs fragment start and completion events when debug mode is enabled
- Records byte counts for each fragment
- Maintains relative timestamps from download start
- Reports total file size and fragment count at initialization

### HttpFragment
**Responsibility**: Downloads a specific byte range of the target file using pipeline processing for memory efficiency.

**Key Methods:**
- `Start(httpClient *http.Client, destFile *os.File, writeMutex *sync.Mutex) error`: Downloads the fragment data using pipeline processing
- `GetSize() int`: Calculates the actual fragment size
- `GetRange() string`: Formats the HTTP Range header

**Pipeline Processing Data Flow:**
- Receives byte range specification (startPos, endPos)
- Creates HTTP request with appropriate Range header
- Streams response data in small chunks (64 KiB) directly to disk
- Seeks to correct file position and writes each chunk immediately
- No large memory buffers - constant memory usage per fragment
- Releases resources immediately after completion

**Memory Management:**
- Uses pipeline processing with 64 KiB streaming chunks
- Maintains constant memory usage regardless of fragment size
- Writes data to disk as it arrives from HTTP response
- Eliminates large memory buffers (no 20 MiB allocations)
- Memory usage per fragment: ~64 KiB + HTTP overhead

### Configuration Management
**Responsibility**: Provides system-wide configuration parameters with validation and safe defaults.

**Parameters:**
- `MaxFragmentSize`: Maximum size per fragment (20 MiB default)
- `Parallelization`: Maximum concurrent downloads (4 default)  
- `Debug`: Controls logging verbosity (false default)

**Configuration Validation:**
- Invalid fragment sizes default to 20 MiB
- Invalid parallelization factors default to 4 concurrent connections
- Invalid debug flags default to disabled
- All configuration parameters are validated at startup with fallback to safe defaults

## Data Models

### Fragment Specification
```go
type HttpFragment struct {
    srcUrl   *url.URL     // Source URL for the download
    startPos int          // Starting byte position (inclusive)
    endPos   int          // Ending byte position (inclusive)  
    resp     *http.Response // Cached HTTP response (optional)
}
```

### Request Coordination
```go
type RequestManager struct {
    httpClient *http.Client    // Reusable HTTP client
    srcUrl     *url.URL        // Parsed source URL
    destFile   *os.File        // Destination file handle
    wg         sync.WaitGroup  // Goroutine synchronization
    writeMutex sync.Mutex      // File write synchronization
    semaphore  chan struct{}   // Concurrency limiting
    startTime  time.Time       // Download start time for logging
    totalSize  int64           // Total file size for progress tracking
}
```

### Logging and Progress Tracking
```go
type ProgressLogger struct {
    startTime     time.Time     // Download start timestamp
    debugEnabled  bool          // Debug logging flag
    totalSize     int64         // Total file size
    fragmentCount int           // Number of fragments
}
```

### Range Calculation
- **Fragment Count**: `ceil(fileSize / MaxFragmentSize)`
- **Fragment Range**: `[i * MaxFragmentSize, min((i+1) * MaxFragmentSize - 1, fileSize - 1)]`
- **Range Header**: `"bytes=startPos-endPos"`

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Fragment size calculation correctness
*For any* file size and fragment size configuration, if Content-Length is present and file size > MaxFragmentSize, the calculated number of fragments should be `ceil(fileSize / MaxFragmentSize)`, otherwise exactly one fragment should be used
**Validates: Requirements 1.1, 1.4, 1.5**

### Property 2: Concurrency limit enforcement  
*For any* download operation, the number of simultaneously active fragment downloads should never exceed the configured Parallelization_Factor
**Validates: Requirements 1.2**

### Property 3: Download completeness and integrity
*For any* successfully downloaded file, the resulting file should be byte-for-byte identical to the source file (verified through checksum comparison)
**Validates: Requirements 1.3**

### Property 4: Fragment range continuity
*For any* file fragmentation, the byte ranges should be contiguous with no gaps or overlaps - the end position of fragment i should be exactly one less than the start position of fragment i+1
**Validates: Requirements 6.4**

### Property 5: Error propagation consistency
*For any* HTTP error response (non-2xx status codes), the system should terminate with an appropriate error message and not produce a partial file
**Validates: Requirements 2.1, 2.4**

### Property 6: File I/O error handling
*For any* file write operation failure, the system should report the error and terminate gracefully without leaving corrupted partial files
**Validates: Requirements 2.3, 2.4**

### Property 7: Seek position accuracy
*For any* fragment being written to disk, the file seek position should exactly match the fragment's start position before writing begins
**Validates: Requirements 3.2**

### Thread-safe file operations
*For any* concurrent fragment writes to non-overlapping byte ranges, the final file should contain exactly the expected data with no corruption from race conditions. Note: While the OS allows concurrent writes to different file positions, explicit synchronization ensures atomic seek-and-write operations.
**Validates: Requirements 3.3**

### Property 9: Resource cleanup consistency
*For any* download operation (successful or failed), all file handles should be properly closed and resources cleaned up
**Validates: Requirements 3.4**

### Property 10: Range header formatting
*For any* fragment with start and end positions, the generated Range header should follow the exact format "bytes=start-end"
**Validates: Requirements 6.2**

### Property 11: URL validation consistency
*For any* input string, URL validation should correctly identify valid HTTP/HTTPS URLs and reject invalid ones
**Validates: Requirements 6.1**

### Property 12: Configuration parameter validation
*For any* configuration values (fragment size, parallelization factor, debug flag), invalid inputs should result in safe default values being used
**Validates: Requirements 5.1, 5.2, 5.3, 5.4**

### Property 13: Debug logging completeness
*For any* download operation with debug mode enabled, all fragment lifecycle events (start, completion, byte counts) should be logged with relative timestamps
**Validates: Requirements 4.1, 4.2, 4.3, 4.4**

### Property 14: Memory efficiency consistency
*For any* download operation, memory usage should remain constant regardless of file size, with memory buffers released promptly after fragment completion
**Validates: Requirements 7.1, 7.2, 7.4, 7.5**

### Property 15: Range request server compatibility  
*For any* server that supports range requests, the system should utilize parallel downloading, otherwise fall back to single fragment download
**Validates: Requirements 6.3**

### Property 16: Component separation consistency
*For any* modification to individual components (RequestManager, HttpFragment, Configuration), other components should remain unaffected and maintain their interfaces
**Validates: Requirements 7.3**

### Property 17: File creation and overwriting
*For any* specified destination file path, the system should create a new file or overwrite an existing file, ensuring the final file contains only the downloaded content
**Validates: Requirements 3.1**

### Property 18: Pipeline processing memory efficiency
*For any* fragment download, memory usage should remain constant at approximately 64 KiB per fragment regardless of fragment size, with data streamed directly to disk as it arrives
**Validates: Requirements 8.1, 8.2, 8.3, 8.4**

## Error Handling

The system implements a fail-fast approach with comprehensive error handling:

### HTTP Errors
- **Non-2xx Status Codes**: Immediate termination with descriptive error message
- **Network Timeouts**: Graceful failure with connection error reporting
- **Invalid URLs**: Early validation prevents malformed requests

### File System Errors  
- **Permission Denied**: Clear error message about file access rights
- **Disk Full**: Detection and reporting of insufficient storage space
- **Invalid Paths**: Validation of destination file path before download begins
- **File Creation**: Automatic creation or overwriting of destination files as specified by user

### File Management
- **File Creation/Overwriting**: When a destination file is specified, the system creates a new file or overwrites an existing file
- **Atomic Operations**: File operations are designed to prevent partial corruption during concurrent access
- **Proper Closure**: All file handles are properly closed upon completion or failure

### Concurrency Errors
- **Race Conditions**: Mutex synchronization prevents file corruption during seek-and-write operations
- **Deadlock Prevention**: Careful lock ordering and timeout mechanisms
- **Resource Exhaustion**: Semaphore limits prevent excessive resource usage

**File I/O Concurrency Details:**
While most operating systems support concurrent writes to different positions within the same file, the current implementation uses a write mutex to ensure atomic seek-and-write operations. This prevents race conditions where one goroutine's seek operation could be interrupted by another goroutine's seek, leading to data being written to the wrong position. Alternative approaches could include:
- Using `WriteAt()` method which combines seek and write atomically
- Per-fragment file descriptors (though this may hit OS limits)
- Memory-mapped files for very large downloads

### Recovery Strategies
- **Partial Download Cleanup**: Failed downloads remove incomplete files
- **Resource Release**: Guaranteed cleanup of HTTP connections and file handles
- **Error Context**: Detailed error messages include fragment information and timestamps

## Testing Strategy

The testing approach combines unit testing for specific behaviors with property-based testing for universal correctness guarantees.

### Unit Testing Approach
Unit tests will focus on:
- **Specific Examples**: Testing known good inputs and expected outputs
- **Edge Cases**: Small files, single fragments, boundary conditions  
- **Error Conditions**: Invalid URLs, file permission errors, HTTP failures
- **Integration Points**: Component interactions and data flow

### Property-Based Testing Approach
Property-based tests will use **Go's testing/quick package** for generating random test inputs and verifying universal properties hold across all valid executions.

**Configuration Requirements:**
- Each property-based test MUST run a minimum of 100 iterations
- Each property-based test MUST be tagged with a comment referencing the specific correctness property from this design document
- Tag format: `**Feature: ranged-http-downloader, Property {number}: {property_text}**`
- Each correctness property MUST be implemented by exactly one property-based test

**Test Categories:**
- **Mathematical Properties**: Fragment calculations, range computations, size validations
- **Invariant Properties**: File integrity, concurrency limits, resource cleanup
- **Round-trip Properties**: Download-verify cycles, configuration-behavior consistency
- **Error Handling Properties**: Consistent failure modes, proper error propagation

### Test Data Generation
- **File Sizes**: Random sizes from 1 byte to 1GB
- **URLs**: Valid and invalid HTTP/HTTPS URLs with various formats
- **Configuration Values**: Valid and boundary-case parameter combinations
- **Error Scenarios**: Simulated network failures, file system errors, permission issues

The dual testing approach ensures both concrete correctness (unit tests) and universal correctness (property tests), providing comprehensive validation of the system's behavior across all possible inputs and conditions.
