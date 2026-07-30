import { MovieCatalogItem } from "@/services/catalog/getMovies";
import Link from "next/link";


type ShowtimeCardProps = {
    item: MovieCatalogItem
}

export default function ShowtimeCard({ item } : ShowtimeCardProps) {
    const { title, poster_url, showtimes } = item;
    
    return (
        <div className="flex gap-10 bg-white/5 p-5 rounded-lg">
            <div className="flex flex-col gap-2 items-center">
                <img src={poster_url} alt={title} className="w-64 h-96 object-cover rounded-lg" />
                <p className="text-lg text-primary font-semibold">{title}</p>
            </div>
            <div className="flex flex-wrap gap-3 w-full content-start">
                {showtimes.map((showtime) => (
                    <Link 
                        key={showtime.id} 
                        href={`/showtimes/${showtime.id}`}
                        className="p-2 flex flex-col items-center justify-center gap-1 bg-white/10 rounded-lg hover:bg-white/20 cursor-pointer transition-colors"
                    >
                        <p className="text-sm text-primary font-semibold">
                            {new Date(showtime.starts_at).toLocaleString([], { 
                                month: 'short', 
                                day: 'numeric', 
                                hour: '2-digit', 
                                minute: '2-digit' 
                            })}
                        </p>
                        <p className="text-xs text-primary font-light">
                            {showtime.hall}
                        </p>
                    </Link>
                ))}
            </div>
        </div>
    )
}