import type { Seat } from "@/types/Seat";
import type { ErrorResponse } from "@/types/ErrorResponse";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api";

export type SeatResponse = {
    seats: Seat[];
    expires_at?: number;
};

export default async function getSeats(showtimeId: number, sessionId: string): Promise<SeatResponse> {

    if (typeof showtimeId !== "number" || isNaN(showtimeId) || !sessionId.trim()) {
        throw {
            error: "Invalid Parameters",
            details: "Both a valid showtimeId and sessionId are required."
        } as ErrorResponse;
    }

    var res: Response;

    try {
        res = await fetch(`${API_URL}/booking/showtimes/${showtimeId}/seats`, {
            method: 'GET',
            headers: {
                'X-Session-ID': sessionId
            },
        });
    } catch (error) {
        throw {
            error: "Network Error",
            details: "Failed to connect to the backend server."
        } as ErrorResponse;
    }

    const data = await res.json();

    if (!res.ok) {
        throw data as ErrorResponse;
    }

    return data as SeatResponse;
}



