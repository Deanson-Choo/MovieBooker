# MovieBooker

A full-stack cinema booking application that handles the complete reservation flow, from browsing movies to processing mock payments.

Built to handle high-concurrency scenarios, it uses a Go API, Redis for temporary seat locks, and PostgreSQL for persistence to prevent **double-booking** and **double-charging** race conditions.

## Screenshots

![Home page](./screenshots/home_page.png)
<p align="center"><sub>Figure 1. Home page</sub></p>

![Showtime page](./screenshots/showtime_page.png)
<p align="center"><sub>Figure 2. Showtime page</sub></p>

![Seat hold with timer](./screenshots/seat_hold_with_timer.png)
<p align="center"><sub>Figure 3. Seat hold</sub></p>

![Payment page](./screenshots/payment_page.png)
<p align="center"><sub>Figure 4. Payment page</sub></p>

## Features

- Browse a catalog of movies and their available showtimes
- View an interactive seat map with visual seat states
- Temporarily hold seats in a user session with automatic expiry
- Use optimistic UI updates for seat selection and release actions
- Complete a mock payment flow with idempotency protection
- Prevent double-booking race conditions using Redis distributed locks and PostgreSQL transactions

## Tech Stack

### Frontend
- Next.js 16
- TanStack React Query

### Backend
- Golang
- Gin web framework
- PostgreSQL with `pgx`
- Redis with `go-redis`

### Testing and tooling
- k6 for load testing

## Setup and Run

### Prerequisites
- Go 1.26+
- Node.js 20+
- PostgreSQL running locally or remotely
- Redis running locally or remotely

### Environment Variables

Set the following environment variables before starting the backend:

```bash
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=postgres
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432
export POSTGRES_DB=moviebooker
export REDIS_ADDR=localhost:6379
```

### Backend

```bash
cd backend
go run ./api/main.go
```

The API will start on the default Gin port, typically http://localhost:8080.

### Frontend

```bash
cd frontend
npm install
npm run dev
```

Open http://localhost:4000 to use the app.

### Load Testing

```bash
k6 run -e SHOWTIME_ID=4 -e MIN_SEAT_ID=1 -e MAX_SEAT_ID=80 tests/load/end-to-end.js
```

## Architecture

The application follows a simple but practical layered architecture:

1. Frontend
   - The Next.js app renders the movie catalog and seat selection experience.
   - React Query manages async state and optimistically updates seat UI when users lock or unlock seats.
   - Manages multi-seat booking sessions with a live countdown timer to enforce temporary holds.

2. API Layer
   - The Go backend exposes REST endpoints for catalog browsing, seat retrieval, seat locking/unlocking, and payment.
   - The API uses Gin for request handling and middleware.

3. Data Layer
   - PostgreSQL stores the authoritative seat inventory and booking records.
   - Redis stores short-lived seat locks, session cart state, and idempotency keys.

4. Booking Flow
   - A user requests seats and the backend marks them as selected in the session.
   - A payment request validates the locks, extends their TTL, and commits a booking transaction in PostgreSQL.
   - Redis is then cleaned up and the seat availability cache is invalidated.

## API Endpoints

### 1. GET /api/catalog

#### Definition
- Method: GET
- Path: /api/catalog
- Description: Retrieve all movies and their showtimes
- Authentication: None

#### Request
- Headers: None
- Path Parameters: None
- Query Parameters: None
- Body: None

#### Behavior
- The backend first tries to read the catalog from Redis.
- If the cache is empty, it queries PostgreSQL and caches the result.

#### Limitation
- Thundering Herd: if many users hit the same cache miss, multiple requests can query PostgreSQL at once.
  - This was intentionally left out of scope for this project.
  - A common mitigation is single-flight cache hydration so one request populates cache while others wait.

#### Response
- Status: 200 OK

```json
[
  {
    "id": 1,
    "title": "Movie Title Here",
    "poster_url": "https://example.com/poster.jpg",
    "showtimes": [
      {
        "id": 101,
        "movie_id": 1,
        "starts_at": "2026-07-26T20:00:00Z",
        "hall": "Hall A"
      },
      {
        "id": 102,
        "movie_id": 1,
        "starts_at": "2026-07-26T23:30:00Z",
        "hall": "Hall B"
      }
    ]
  }
]
```

#### Error Handling
- 500 Internal Server Error if PostgreSQL fails or the returned data cannot be mapped correctly.

---

### 2. GET /api/booking/showtimes/:showtime_id/seats

#### Definition
- Method: GET
- Path: /api/booking/showtimes/:showtime_id/seats
- Description: Retrieve seat availability for a specific showtime and session TTL
- Authentication: None

#### Request
- Headers: X-Session-ID
- Path Parameters: showtime_id
- Query Parameters: None
- Body: None

#### Behavior
- The backend loads the seat layout from Redis if available; otherwise it queries PostgreSQL and caches the result.
- It then checks for temporary Redis seat locks and marks seats as:
  - available
  - locked
  - selected
  - booked

#### Limitation
- Thundering Herd: if many users hit the same cache miss, multiple requests can query PostgreSQL at once.
  - This was intentionally left out of scope for this project.
  - A common mitigation is single-flight cache hydration so one request populates cache while others wait.

#### Response
- Status: 200 OK

```json
{
  "seats": [
    {
      "id": 1,
      "showtime_id": 101,
      "label": "A1",
      "status": "available"
    },
    {
      "id": 2,
      "showtime_id": 101,
      "label": "A2",
      "status": "locked"
    },
    {
      "id": 3,
      "showtime_id": 101,
      "label": "A3",
      "status": "selected"
    },
    {
      "id": 4,
      "showtime_id": 101,
      "label": "A4",
      "status": "booked"
    }
  ],
  "expires_at": 1785074694
}
```

- `seats`: full seat layout with computed status per seat.
- `expires_at` (optional): UNIX timestamp (seconds) for when the current session hold expires. Omitted when there is no active hold for this session.

#### Error Handling
- 400 Bad Request if the showtime ID or session header is missing
- 500 Internal Server Error if the DB or Redis lock lookup fails

---

### 3. POST /api/booking/showtimes/:showtime_id/seats/:seat_id/lock

#### Definition
- Method: POST
- Path: /api/booking/showtimes/:showtime_id/seats/:seat_id/lock
- Description: Temporarily hold a seat for the current session
- Authentication: None

#### Request
- Headers: X-Session-ID
- Path Parameters: showtime_id, seat_id
- Query Parameters: None
- Body: None

#### Behavior
- The backend uses Redis SETNX to perform a fast soft lock.
- It also checks PostgreSQL to ensure the seat is still available before committing the lock.
- If the lock succeeds, the seat is added to the session cart and the TTL is extended.

#### Remark
- Soft lock + hard lock design: PostgreSQL is checked only after SETNX succeeds.
- This keeps Redis as the fast gate and uses DB verification for edge cases (for example, expired soft lock while DB is already processing or booked).

#### Limitation
- Non-atomic lock + session write: SETNX and SADD + EXPIREAT are separate operations.
- A crash between them can leave an orphaned soft lock until TTL expiry.
- Seat availability hard-check still queries PostgreSQL on lock attempts.

#### Response
- Status: 200 OK


#### Error Handling
- 400 Bad Request if required parameters are missing
- 409 Conflict if the seat is already locked or no longer available
- 500 Internal Server Error if Redis or PostgreSQL fails

---

### 4. POST /api/booking/showtimes/:showtime_id/seats/:seat_id/unlock

#### Definition
- Method: POST
- Path: /api/booking/showtimes/:showtime_id/seats/:seat_id/unlock
- Description: Release a previously locked seat for the current session
- Authentication: None

#### Request
- Headers: X-Session-ID
- Path Parameters: showtime_id, seat_id
- Query Parameters: None
- Body: None

#### Behavior
- The backend uses a Lua script in Redis to atomically verify ownership, delete the seat lock, and remove the seat from the session set.

#### Remarks
- All unlock operations are done inside one Lua script to avoid race conditions.
- Without atomicity, concurrent unlocks on the last seat could both observe the same pre-delete state.

#### Response
- Status: 200 OK

#### Error Handling
- 400 Bad Request if required parameters are missing
- 403 Forbidden if another session owns the lock
- 404 Not Found if the lock is missing or expired
- 500 Internal Server Error if the Lua script fails

---

### 5. POST /api/payment/pay

#### Definition
- Method: POST
- Path: /api/payment/pay
- Description: Finalize payment for seats held in the current session
- Authentication: None

#### Request
- Headers: Idempotency-Key, X-Session-ID
- Path Parameters: None
- Query Parameters: None
- Body:
  - email
  - showtime_id

#### Behavior
- The endpoint uses Redis to create an idempotency key and prevent duplicate processing.
- It validates the session and the seat locks, extends the TTL, and simulates payment processing.
- It then performs a PostgreSQL transaction to insert a booking and mark seats as booked.
- Redis locks and session state are cleaned up after a successful payment.

#### Limitation
- Synchronous blocking: the mock payment delay blocks a goroutine for around 2 seconds per payment attempt.
- Refund simulation is simplified; there is no full downstream refund workflow.
- Idempotency depends on Redis durability; if Redis is flushed, the same idempotency key could be retried.

#### Response
- Status: 200 OK

```json
{
  "booking_id": "a1b2c3d4-e5f6-7890-1234-56789abcdef0",
  "showtime_id": "st_12345",
  "seat_ids": [42, 43, 44]
}
```

#### Error Handling
- 400 Bad Request if headers or the JSON body are invalid
- 402 Payment Required if the mock payment gateway rejects the charge
- 409 Conflict if the payment is already processing, the session is expiring, or the seat locks are invalid
- 500 Internal Server Error if Redis or PostgreSQL fails during processing

## Limitations

- Cross-showtime sessions: Users can hold seats across multiple showtimes simultaneously, but checkout is restricted to a single showtime.
- Seat Hoarding: Users can technically lock and unlock a seat infinitely.
- Non-atomic Redis operations: Seat locking (SETNX) and session assignment (SADD + EXPIREAT) are executed separately. A server crash mid-execution could leave orphaned seat locks.
- Simulated payments: The checkout flow is mocked, including a 2-second artificial delay and a 10% failure rate to test error handling.
- Volatile idempotency: Payment idempotency relies entirely on Redis. If the cache is flushed, duplicate payment requests could theoretically be processed.
- Thundering herd risk: A cache miss on the movie catalog or seat map could cause a surge of direct queries to PostgreSQL under heavy load.

## Future Improvements

- Integrate a live payment gateway (e.g., Stripe) to replace the mock checkout.
- Implement WebSockets or Server-Sent Events (SSE) for real-time seat map updates.
- Resolve the thundering herd risk using cache stampede protection (e.g., singleflight or probabilistic early expiration).
- Implement NGINX for load balancing and reverse proxying.
- Containerize the application and deploy via Kubernetes for better scalability.
