import { Counter } from "k6/metrics";
import { lockSeat, newSessionId } from "./lib/flows.js";
import { SEAT_ID, SHOWTIME_ID } from "./lib/config.js";

const lockSuccess = new Counter("lock_success");
const lockConflict = new Counter("lock_conflict");
const lockError = new Counter("lock_error"); 

export const options = {
    vus: 100,
    iterations: 100, 
    thresholds: {
        lock_success: ["count==1"],  // Exactly 1 user MUST get the seat
        lock_conflict: ["count==99"],// Exactly 99 users MUST be rejected gracefully
        lock_error: ["count==0"],    // Zero internal server errors allowed
    },
};

export default function () {
    const sessionId = newSessionId();
    
    const res = lockSeat(sessionId, SEAT_ID, SHOWTIME_ID);

    if (res.status === 200) {
        lockSuccess.add(1);
    } else if (res.status === 409) {
        lockConflict.add(1);
    } else if (res.status >= 500) {
        lockError.add(1);
    } else {
        console.error(`Unexpected status code: ${res.status}`);
    }
}

// This test simulate 100 concurrent users trying to lock the same seat 
// Run with command:
// k6 run -e SEAT_ID=30 SHOWTIME_ID=1 tests/load/double-booking.js