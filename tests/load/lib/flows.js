import http from "k6/http";
import { check } from "k6";
import exec from "k6/execution"; 
import { randomIntBetween, uuidv4 } from "https://jslib.k6.io/k6-utils/1.4.0/index.js";
import { BASE_URL, SHOWTIME_ID, intEnv } from "./config.js";

export function newSessionId() {
  return uuidv4();
}

export function newIdempotencyKey() {
  return uuidv4();
}

export function syntheticEmail() {
  return `loadtest-${uuidv4()}@hotmail.com`;
}

export function chooseSeatId(showtimeId = SHOWTIME_ID) {
  const minSeat = intEnv("MIN_SEAT_ID", 1) + (showtimeId - 1) * 100;
  const maxSeat = intEnv("MAX_SEAT_ID", 100) + (showtimeId - 1) * 100;
  return randomIntBetween(minSeat, maxSeat);
}


const url = {
  catalog: () => `${BASE_URL}/api/catalog`,
  seats: (showtimeId) => `${BASE_URL}/api/booking/showtimes/${showtimeId}/seats`,
  lock: (showtimeId, seatId) => `${BASE_URL}/api/booking/showtimes/${showtimeId}/seats/${seatId}/lock`,
  unlock: (showtimeId, seatId) => `${BASE_URL}/api/booking/showtimes/${showtimeId}/seats/${seatId}/unlock`,
  pay: () => `${BASE_URL}/api/payment/pay`,
};


export function browseCatalog() {
  const res = http.get(url.catalog());
  return res;
}

export function getSeats(sessionId, showtimeId = SHOWTIME_ID) {
  const res = http.get(url.seats(showtimeId), {
    headers: { "X-Session-ID": sessionId },
  });
  return res;
}

export function lockSeat(sessionId, seatId, showtimeId = SHOWTIME_ID) {
  return http.post(url.lock(showtimeId, seatId), null, {
    headers: { "X-Session-ID": sessionId },
  });
}

export function unlockSeat(sessionId, seatId, showtimeId = SHOWTIME_ID) {
  const res = http.post(url.unlock(showtimeId, seatId), null, {
    headers: { "X-Session-ID": sessionId },
  });
  
  return res;
}

export function pay(sessionId, idempotencyKey, email, showtimeId = SHOWTIME_ID) {
  const payload = JSON.stringify({ email, showtime_id: String(showtimeId) });
  const res = http.post(url.pay(), payload, {
    headers: {
      "Content-Type": "application/json",
      "X-Session-ID": sessionId,
      "Idempotency-Key": idempotencyKey,
    },
  });
  return res;
}