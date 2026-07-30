import { Counter } from "k6/metrics";
import { check } from "k6";
import { SEAT_ID, SHOWTIME_ID } from "./lib/config.js";
import { lockSeat, newIdempotencyKey, newSessionId, pay, syntheticEmail } from "./lib/flows.js";

const paymentProcessed = new Counter("payment_processed"); // 200 or 402
const paymentConflict = new Counter("payment_conflict");
const paymentError = new Counter("payment_error");

export const options = {
    vus: 50,
    iterations: 50,
    thresholds: {
        payment_processed: ["count==1"],  // Exactly 1 user MUST get past the idempotency lock
        payment_conflict: ["count==49"],  // Exactly 49 users MUST be safely rejected
        payment_error: ["count==0"],      // Zero system crashes allowed
    },
};

// Runs once
export function setup() {
    const sessionId = newSessionId();
    const idempotencyKey = newIdempotencyKey();

    const lockRes = lockSeat(sessionId, SEAT_ID, SHOWTIME_ID);
    if (lockRes.status !== 200) {
        throw new Error(`setup lock failed for seat ${SEAT_ID}, status=${lockRes.status}`);
    }

    return { sessionId, idempotencyKey };
}

export default function (data) {
    const res = pay(data.sessionId, data.idempotencyKey, syntheticEmail(), SHOWTIME_ID);

    check(res, {
        "pay status is expected": (r) => r.status === 200 || r.status === 409 || r.status === 402,
    });

    if (res.status === 200 || res.status === 402) {
        paymentProcessed.add(1);
    } else if (res.status === 409) {
        paymentConflict.add(1);
    } else {
        paymentError.add(1);
    }
}