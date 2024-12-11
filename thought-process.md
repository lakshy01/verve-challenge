### Thought Process for Implementation

#### Design Goals
1. **High Performance**: Ensure the application can handle at least 10,000 requests per second.
2. **Simplicity**: Keep the architecture straightforward for readability and maintainability.
3. **Scalability**: Allow for distributed handling of unique ID deduplication when deployed behind a load balancer.
4. **Extensibility**: Provide hooks for additional functionalities such as POST requests and distributed streaming services.

---

### Implementation Overview

#### Request Handling
- **RESTful Endpoint**: Implemented a `/api/verve/accept` GET endpoint to accept `id` (mandatory) and `endpoint` (optional).
- **Concurrency**: Used a buffered Go channel (`requestChan`) to queue incoming requests for processing.
- **Response**: Returns "ok" for successful processing or "failed" for errors.

#### Deduplication and Logging
- **Unique ID Tracking**: A `map[int]struct{}` with a mutex lock ensures deduplication of IDs within a single instance.
- **Log File**: Writes the count of unique IDs received every minute to a file using `os` and standard logging.
- **Reset Mechanism**: Resets the unique ID map every minute to start fresh.

#### HTTP Endpoint
- **Optional Parameter**: If `endpoint` is provided, the application sends an HTTP POST request with the unique count as a JSON payload.
- **Error Handling**: Logs failures in sending the request and captures the response status code for debugging.

---

### Extensions

#### Extension 1: HTTP POST Requests
- The service sends a POST request to the specified endpoint with a JSON payload containing the unique count.
- Example Payload:
  ```json
  {
    "unique_count": 100
  }
  ```

#### Extension 2: Load Balancer Compatibility
- Distributed deduplication is achieved by:
  1. Using a shared in-memory database like Redis for global unique ID tracking.
  2. Ensuring atomic operations via Redis commands (e.g., `SETNX`).

#### Extension 3: Distributed Streaming Service
- Replace file-based logging with a streaming system such as Apache Kafka or AWS Kinesis.
- Example Flow:
  1. Serialize the unique ID count as a JSON message.
  2. Publish the message to a Kafka topic for downstream consumers.

---

### Design Considerations

#### Error Handling
- Validates query parameters (`id` and `endpoint`) and responds with appropriate HTTP status codes for missing or invalid data.
- Logs critical issues, such as file IO or network errors, for post-mortem analysis.

#### Scalability
- The use of a Go channel and mutex ensures thread-safe operations for a single instance.
- Adapts easily to distributed environments with minor modifications (e.g., integration with Redis).

#### Performance
- Buffered channels and minimal locking ensure high throughput.
- The JSON payload for POST requests is lightweight and efficient.

#### Logging
- Logs are rotated every minute to avoid accumulation and ensure real-time data tracking.

---

### Appliation Running - 

```bash
docker build -t verve . --no-cache
docker run -p 8080:8080 verve
curl "http://localhost:8080/api/verve/accept?id=123"
```