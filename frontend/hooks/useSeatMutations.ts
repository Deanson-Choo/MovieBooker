import { useMutation, useQueryClient } from "@tanstack/react-query";
import { lockSeat } from "@/services/booking/lockSeat";
import { unlockSeat } from "@/services/booking/unlockSeat";
import type { SeatResponse } from "@/services/booking/getSeats";
import type { Seat } from "@/types/Seat";

type UseSeatMutationsProps = {
    showtimeId: number;
    sessionId: string;
};

type MutationContext = {
    previousSeats?: SeatResponse;
};

export function useSeatMutations({ showtimeId, sessionId }: UseSeatMutationsProps) {
    const queryClient = useQueryClient();
    const queryKey = ['seats', showtimeId, sessionId];
    const mutationKey = ['seat-mutation', showtimeId, sessionId];

    const optimisticallyUpdateSeat = async (seatId: number, newStatus: "available" | "selected") => {
        await queryClient.cancelQueries({ queryKey }); 
        const previousSeats = queryClient.getQueryData<SeatResponse>(queryKey);

        queryClient.setQueryData<SeatResponse>(queryKey, (oldData) => {
            if (!oldData) return undefined;

            return {
                ...oldData,
                seats: oldData.seats.map(seat =>
                    seat.id === seatId ? { ...seat, status: newStatus } : seat
                )
            };
        });

        return { previousSeats };
    };

    const rollbackCache = (_err: unknown, _variables: number, context: MutationContext | undefined) => {
        if (context?.previousSeats) {
            queryClient.setQueryData(queryKey, context.previousSeats);
        }
    };

    const lockMutation = useMutation({
        mutationKey,
        mutationFn: (seatId: number) => lockSeat(showtimeId, seatId, sessionId),
        onMutate: (seatId) => optimisticallyUpdateSeat(seatId, "selected"),
        onError: rollbackCache,
        onSettled: () => queryClient.invalidateQueries({ queryKey })
    });

    const unlockMutation = useMutation({
        mutationKey,
        mutationFn: (seatId: number) => unlockSeat(showtimeId, seatId, sessionId),
        onMutate: (seatId) => optimisticallyUpdateSeat(seatId, "available"),
        onError: rollbackCache,
        onSettled: () => queryClient.invalidateQueries({ queryKey })
    });

    const toggleSeat = (seat: Seat) => {
        if (seat.status === "booked" || seat.status === "locked") return;
        
        if (seat.status === "available") {
            lockMutation.mutate(seat.id);
        } else if (seat.status === "selected") {
            unlockMutation.mutate(seat.id);
        }
    };

    return {
        toggleSeat
    };
}