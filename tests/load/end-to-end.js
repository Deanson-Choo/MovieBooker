import { Counter } from "k6/metrics";
import { check, sleep } from "k6";
import {
  browseCatalog,
  getSeats,
  lockSeat,
  pay,
  newSessionId,
  newIdempotencyKey,
  syntheticEmail,
  chooseSeatId,
} from "./lib/flows.js";
import { SHOWTIME_ID } from "./lib/config.js";

const catalogError = new Counter("catalog_error");
const seatsError   = new Counter("seats_error");
const lockError    = new Counter("lock_error");
const payError     = new Counter("payment_error");

export const options = {
  scenarios: {
    booking_spike: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "10s", target: 30 },
        { duration: "30s", target: 30 },
        { duration: "10s", target: 0  },
      ],
    },
  },
  thresholds: {
    catalog_error:    ["count==0"],
    seats_error:      ["count==0"],
    lock_error:       ["count==0"],
    payment_error:    ["count==0"],

    http_req_duration: ["p(95)<2000"],
  },
};

// ─── Main VU function ─────────────────────────────────────────────────────────
export default function () {
  const sessionId      = newSessionId();
  const idempotencyKey = newIdempotencyKey();
  const email          = syntheticEmail();
  const seatId         = chooseSeatId(); // random from MIN_SEAT_ID..MAX_SEAT_ID

  // Step 1: Browse catalog — only 200 is acceptable
  const catalogRes = browseCatalog();
  if (catalogRes.status !== 200) {
    catalogError.add(1);
    return;
  }
  check(catalogRes, { "catalog: 200": () => true });

  sleep(0.2);

  // Step 2: Fetch seat map — only 200 is acceptable
  const seatsRes = getSeats(sessionId, SHOWTIME_ID);
  if (seatsRes.status !== 200) {
    seatsError.add(1);
    return;
  }
  check(seatsRes, { "seats: 200": () => true });

  sleep(0.1);

  // Step 3: Lock seat
  // 200 = acquired, 409 = taken by another session (expected contention), else = unexpected
  const lockRes = lockSeat(sessionId, seatId, SHOWTIME_ID);
  if (lockRes.status === 200) {
    // acquired
  } else if (lockRes.status === 409) {
    return;
  } else {
    lockError.add(1);
    return;
  }

  sleep(0.1);

  // Step 4: Pay
  // 200 = booked, 402 = mock gateway rejection (~10%), 409 = session / idempotency conflict
  // Anything else (400 / 5xx) is unexpected and fails the test
  const payRes = pay(sessionId, idempotencyKey, email, SHOWTIME_ID);

  check(payRes, {
    "pay: expected status": (r) =>
      r.status === 200 || r.status === 402 || r.status === 409,
  });

  if (payRes.status !== 200 && payRes.status !== 402 && payRes.status !== 409) {
    payError.add(1);
  }
}

// ─── Run ──────────────────────────────────────────────────────────────────────
// k6 run \
//   -e SHOWTIME_ID=1 \
//   -e MIN_SEAT_ID=1 \
//   -e MAX_SEAT_ID=100 \
//   tests/load/end-to-end.js

