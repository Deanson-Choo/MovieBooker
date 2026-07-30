export type Seat = {
    id: number;
    showtime_id: number;
    label: string;  // e.g. "A1", "B3"
    status: "available" | "booked" | "locked" | "selected"
};