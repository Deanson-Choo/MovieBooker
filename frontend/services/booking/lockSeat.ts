import type { ErrorResponse } from "@/types/ErrorResponse";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api";

export async function lockSeat(showtimeId: number, seatId: number, sessionId: string) {

    if (typeof showtimeId != "number" || isNaN(showtimeId) || typeof seatId != "number" || isNaN(seatId) || !sessionId.trim()) {
        throw {
            error: "Invalid Parameters",
            details: "Both a valid showtimeId, seatId and sessionId are required."
        } as ErrorResponse;
    }

    let res: Response;
    try {
        res = await fetch(`${API_URL}/booking/showtimes/${showtimeId}/seats/${seatId}/lock`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-Session-ID': sessionId
            },
        });
    } catch {
        throw {
            error: "Network Error",
            details: "Failed to connect to the backend server."
        } as ErrorResponse;
    }

    const data = await res.json();

    if (!res.ok) {
        throw data as ErrorResponse;
    }
}