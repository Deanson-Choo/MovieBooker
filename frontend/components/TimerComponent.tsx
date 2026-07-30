"use client"

import { useState, useEffect, useRef } from "react";

type TimerComponentProps = {
    expiryInMs: number | null;
    onExpire: () => void;
};

export default function TimerComponent({ expiryInMs, onExpire }: TimerComponentProps) {
    const [remainingSeconds, setRemainingSeconds] = useState<number>(0);

    const onExpireRef = useRef(onExpire);

    useEffect(() => {
        onExpireRef.current = onExpire;
    }, [onExpire]);

    useEffect(() => {
        if (!expiryInMs) {
            return;
        }

        const updateTimer = (activeIntervalId?: number) => {
            const now = Date.now();
            const distance = expiryInMs - now;
            const secondsLeft = Math.max(0, Math.ceil(distance / 1000));

            setRemainingSeconds(secondsLeft);

            if (secondsLeft <= 0) {
                onExpireRef.current(); 
                if (activeIntervalId !== undefined) {
                    window.clearInterval(activeIntervalId);
                }
            }
        };

        updateTimer(); // Initial tick
        const intervalId = window.setInterval(() => updateTimer(intervalId), 1000);

        return () => window.clearInterval(intervalId);
    }, [expiryInMs]);

    if (!expiryInMs || remainingSeconds <= 0) {
        return null;
    }

    const isUrgent = remainingSeconds <= 30;
    const minutes = Math.floor(remainingSeconds / 60);
    const seconds = remainingSeconds % 60;
    const formattedTime = `${minutes}:${seconds.toString().padStart(2, "0")}`;

    return (
        <div className="rounded-xl border border-white/20 bg-white/10 px-4 py-3 text-white shadow-md backdrop-blur-sm">
            <p className="text-[11px] uppercase tracking-[0.18em] text-white/65">Seat hold</p>
            <p className="mt-1 text-lg font-semibold text-primary">Time Remaining</p>
            <p className={`mt-1 text-3xl font-extrabold tabular-nums ${isUrgent ? "text-red-300" : "text-white"}`}>
                {formattedTime}
            </p>
            <p className={`text-xs ${isUrgent ? "text-red-200/90" : "text-white/60"}`}>
                {isUrgent ? "Almost out of time" : "Seats release automatically when timer ends"}
            </p>
        </div>
    );
}