export const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
export const SHOWTIME_ID = __ENV.SHOWTIME_ID || "1";
export const SEAT_ID = __ENV.SEAT_ID || "1";

export function intEnv(name, defaultValue) {
  const raw = __ENV[name];
  if (!raw) return defaultValue;
  const parsed = Number.parseInt(raw, 10);
  return Number.isNaN(parsed) ? defaultValue : parsed;
}

