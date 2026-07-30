"use client";

import Link from "next/link";
import { LuPopcorn } from "react-icons/lu";
import { RxAvatar } from "react-icons/rx";

export default function NavBar() {
    return (
        <div className="flex justify-between items-center px-16 py-5 sticky top-0 z-50 backdrop-blur-md opacity-80">
            <div className="flex items-center gap-1">
                <LuPopcorn size={24} color="var(--color-primary)" />
                <Link
                    href="/"
                    className="font-bold text-xl text-primary hover:opacity-80 hover:cursor-pointer"
                >
                    Movie Booker
                </Link>
            </div>
            <div className="flex gap-5 items-center">
                <RxAvatar size={36} color="var(--color-primary)" />
            </div>
        </div>
    );
}