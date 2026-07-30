"use client"

import { useQuery, useIsMutating, useQueryClient } from "@tanstack/react-query";
import { useState, useCallback } from "react";
import TimerComponent from "./TimerComponent";
import PaymentModal from "./PaymentModal";
import { useSeatMutations } from "@/hooks/useSeatMutations";
import type { ErrorResponse } from "@/types/ErrorResponse";
import type { SeatResponse as SeatType } from "@/services/booking/getSeats";
import getSeats from "@/services/booking/getSeats";
import Seat from "./Seat";

import getSessionId from "@/lib/getSessionId";

type SeatGridProps = {
    showtimeId: number;
};

export default function SeatGrid({ showtimeId }: SeatGridProps) {
    const [sessionId] = useState<string>(() => (typeof window === "undefined" ? "" : getSessionId()));

    const seatMutationCount = useIsMutating({
        mutationKey: ['seat-mutation', showtimeId, sessionId],
    });
    const isMutating = seatMutationCount > 0;
    const queryClient = useQueryClient();
    const queryKey = ['seats', showtimeId, sessionId];
    
    const [isPaymentMode, setIsPaymentMode] = useState(false);

    const handleTimerExpire = useCallback(() => {
        setIsPaymentMode(false);
        queryClient.invalidateQueries({ queryKey: ['seats', showtimeId, sessionId] });
    }, [queryClient, showtimeId, sessionId]); 

    const handlePaymentSuccess = useCallback(() => {
        setIsPaymentMode(false);
        queryClient.invalidateQueries({ queryKey: ['seats', showtimeId, sessionId] });
    }, [queryClient, showtimeId, sessionId]);

    const { data: seats, isLoading, isError, error } = useQuery<SeatType, ErrorResponse>({
        queryKey: queryKey,
        queryFn: () => getSeats(showtimeId, sessionId),
        enabled: !!sessionId,
        refetchInterval: isMutating ? false : 3000, // Refetch every 3 seconds unless a mutation is in progress
    });

    const { toggleSeat } = useSeatMutations({ 
        showtimeId, 
        sessionId,
    });

    const selectedCount = seats?.seats.filter(seat => seat.status === "selected").length ?? 0;
    const expiryInMs = seats?.expires_at ? seats.expires_at * 1000 : null;

    if (isLoading) {
        return <div className="w-full text-center text-red-500">Loading...</div>;
    }
    if (isError) {
        return <div className="w-full text-center text-red-500">{error.error}: {error.details}</div>;
    }

    return (
        <>
            <div className="grid gap-2 mx-auto" style={{ gridTemplateColumns: "repeat(20, max-content)", width: "max-content" }}>
                {seats?.seats.map((seat, index) => (
                    <Seat key={seat.id} seat={seat} index={index} onClick={() => toggleSeat(seat)}/>
                ))}
            </div>
            <div className="flex items-end justify-between w-full py-2 mt-4">
                <TimerComponent expiryInMs={expiryInMs} onExpire={handleTimerExpire} />
                {selectedCount > 0 && (
                    <button
                        className="px-4 py-2 text-sm font-semibold text-white bg-blue-600 rounded-lg hover:bg-blue-700 cursor-pointer"
                        onClick={() => setIsPaymentMode(true)}
                    >
                    Proceed to Payment
                    </button>
                )}
            </div>
            {isPaymentMode && (
                <PaymentModal showtimeId={showtimeId} sessionId={sessionId} onSuccess={handlePaymentSuccess} onClose={() => setIsPaymentMode(false)} />
            )}
        </>
    );
}