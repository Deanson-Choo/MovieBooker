import type { ErrorResponse } from "../../types/ErrorResponse";
import type { Showtime } from "../../types/Showtime";

export type MovieCatalogItem = {
    id: number
    title: string
    poster_url: string
    showtimes: Showtime[]
}

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api";

export async function getMovies(): Promise<MovieCatalogItem[]> {
    let res: Response;

    try {
        res = await fetch(`${API_URL}/catalog`);
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
    return data as MovieCatalogItem[];
}
