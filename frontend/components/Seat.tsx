"use client";

import type { Seat as SeatData } from "@/services/booking/getSeats";


type SeatProps = {
    seat: SeatData;
    index: number;
    onClick: (seat: SeatData) => void;
};

const styles = {
    available: "bg-green-500/80 hover:bg-green-400 cursor-pointer",
    booked: "bg-red-500/60 cursor-not-allowed",
    locked: "bg-yellow-400/70 cursor-not-allowed",
    processing: "bg-orange-400/70 cursor-not-allowed",
    selected: "bg-blue-500/80 hover:bg-blue-400 cursor-pointer",
};

export default function Seat({ seat, index, onClick }: SeatProps) {
    return (
        <button
            type="button"
            className={`
                w-10 h-10 flex items-center justify-center rounded-t-lg rounded-b-sm text-white font-semibold text-xs transition-all
                ${styles[seat.status as keyof typeof styles]}
                ${index % 20 === 10 ? "ml-10" : ""}
            `}
            onClick={() => onClick(seat)}
        >
            {seat.label}
        </button>
    );
}