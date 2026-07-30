"use client";
import { useState } from "react";
import { v4 as uuidv4 } from 'uuid';
import type { ErrorResponse } from "@/types/ErrorResponse";

import Pay from "@/services/payment/pay";

type PaymentModalProps = {
    showtimeId: number;
    sessionId: string;
    onSuccess: () => void;
    onClose: () => void;
};

export default function PaymentModal({ showtimeId, sessionId, onSuccess, onClose }: PaymentModalProps) {
    const [idempotencyKey] = useState(uuidv4()); // Generate a unique idempotency key for this payment session
    const [isProcessing, setIsProcessing] = useState(false);
    const [email, setEmail] = useState("");

    const handlePayment = async () => {
        setIsProcessing(true);
        try {
            const paymentResponse = await Pay(idempotencyKey, showtimeId, sessionId, email);
            console.log(paymentResponse);
            alert(`Payment Successful! Email sent!`)
            onSuccess();
            onClose();
        } catch (error) {
            const err = error as ErrorResponse;
            console.log("Payment failed:", err);
            alert(`Payment failed: ${err.error} - ${err.details}`);
        } finally {
            setIsProcessing(false);
        }
    }

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/75 px-4 backdrop-blur-sm">
            <div className="w-full max-w-lg rounded-2xl border border-white/15 bg-slate-900/95 p-6 text-white shadow-2xl shadow-black/40">
                <div className="mb-5 flex items-start justify-between gap-4">
                    <div>
                        <p className="text-[11px] uppercase tracking-[0.2em] text-white/55">Checkout</p>
                        <h2 className="mt-1 text-2xl font-semibold text-primary">Complete your payment</h2>
                        <p className="mt-2 text-sm text-white/60">We will use this email to send your booking confirmation.</p>
                    </div>
                </div>

                <label className="mb-2 block text-sm font-medium text-white/80" htmlFor="payment-email">
                    Email address
                </label>
                <input
                    id="payment-email"
                    type="email"
                    placeholder="name@example.com"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    className="mb-5 w-full rounded-xl border border-white/15 bg-white/8 px-4 py-3 text-white outline-none transition placeholder:text-white/35 focus:border-sky-400/60 focus:bg-white/10 focus:ring-2 focus:ring-sky-400/20"
                />

                <div className="flex items-center justify-end gap-3">
                    <button
                        className="rounded-xl border border-white/15 px-4 py-2.5 text-sm font-semibold text-white/75 transition hover:bg-white/8 hover:text-white cursor-pointer"
                        onClick={onClose}
                        disabled={isProcessing}
                    >
                        Cancel
                    </button>
                    <button
                        className="rounded-xl bg-sky-500 px-5 py-2.5 text-sm font-semibold text-white shadow-lg shadow-sky-500/25 transition hover:bg-sky-400 disabled:cursor-not-allowed disabled:opacity-60 cursor-pointer"
                        onClick={handlePayment}
                        disabled={isProcessing}
                    >
                        {isProcessing ? "Processing..." : "Pay Now"}
                    </button>
                </div>
            </div>
        </div>
    )
}