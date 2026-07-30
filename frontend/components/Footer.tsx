import Link from "next/link"
import { FaGithub, FaLinkedin } from "react-icons/fa"

export default function Footer() {
    return (
        <footer className="bg-white/5 py-4">
            <div className="flex flex-col items-center justify-center space-y-1">
                <p className="text-md text-primary font-semibold">Deanson Choo</p>
                <p className="text-sm text-secondary">
                    &copy; {new Date().getFullYear()} Deanson Choo. All rights reserved.
                </p>
                <div className="flex gap-4 mt-3 items-center">
                    <Link href="https://github.com/Deanson-Choo" target="_blank">
                        <FaGithub className="text-secondary text-xl hover:opacity-70" />
                    </Link>
                    <Link href="https://www.linkedin.com/in/Deanson-Choo/" target="_blank">
                        <FaLinkedin className="text-secondary text-xl hover:opacity-70" />
                    </Link>
                </div>
            </div>
        </footer>
    )
}