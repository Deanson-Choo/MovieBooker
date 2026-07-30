import type { ErrorResponse } from "@/types/ErrorResponse";

type PaymentResponse = {
    booking_id: string;
    showtime_id: string;
    seat_ids: number[];
};

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api";

export default async function Pay(idempotencyKey: string, showtimeId: number, sessionId: string, email: string): Promise<PaymentResponse> {
    if (!idempotencyKey || !showtimeId || !sessionId || !email) {
        throw {
            error: "Missing required parameters",
            details: "idempotencyKey, showtimeId, sessionId, and email are required"
        } as ErrorResponse;
    }

    let response: Response;
    try {
        response = await fetch(`${API_URL}/payment/pay`, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                "Idempotency-Key": idempotencyKey,
                "X-Session-ID": sessionId,
            },
            body: JSON.stringify({
                email,
                showtime_id: String(showtimeId),
            })
        });
    } catch {
        throw {
            error: "Network Error",
            details: "Failed to connect to the backend server."
        } as ErrorResponse;
    }

    const data = await response.json();

    if (!response.ok) {
        throw data as ErrorResponse;
    }

    return data as PaymentResponse;
}