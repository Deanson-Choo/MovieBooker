import SeatGrid from "@/components/SeatGrid";
import { getMovies } from "@/services/catalog/getMovies";
import { SEAT_STATUS_LEGEND as legend } from "@/constants/seatStatusLegend";

type ShowtimePageProps = {
    params: Promise<{
        showtimeId: string;
    }>;
};

async function getShowtimeDetails(showtimeId: number) {
    const movies = await getMovies();
    const movie = Array.isArray(movies)
        ? movies.find(m => m.showtimes.some(s => s.id === showtimeId))
        : null;

    if (!movie) return null;

    const showtime = movie.showtimes.find(s => s.id === showtimeId);
    if (!showtime) return null;

    return {
        title: movie.title,
        poster_url: movie.poster_url,
        hall: showtime.hall,
        starts_at: showtime.starts_at,
    };
}

export default async function ShowtimePage({ params }: ShowtimePageProps) {
    let { showtimeId } = await params;
    const id = parseInt(showtimeId);

    const details = await getShowtimeDetails(id);
    const title = details?.title || "Movie Details Unavailable";
    const poster_url = details?.poster_url || "";
    const hall = details?.hall || "TBD";
    const startsAt = details?.starts_at
        ? new Date(details.starts_at).toLocaleString([], { 
            month: 'short', 
            day: 'numeric', 
            hour: '2-digit', 
            minute: '2-digit' 
            })
        : "TBD";

    return (
        <div className="flex gap-10 bg-white/5 p-6 rounded-lg w-[90%] mx-auto mt-10">
           
            {/* LEFT SIDE: Poster & Title */}
            <div className="flex flex-col gap-2 items-center shrink-0">
                {poster_url ? (
                    <img src={poster_url} alt={title} className="w-64 h-96 object-cover rounded-lg" />
                ) : (
                    <div className="w-64 h-96 bg-white/10 rounded-lg" />
                )}
                <p className="text-lg text-primary font-semibold">{title}</p>
                <p className="text-sm text-secondary">{hall} ({startsAt})</p>
            </div>

            {/* RIGHT SIDE: Seating Layout Container */}
            <div className="flex-1 flex flex-col items-center p-4 min-w-0 gap-8">

                {/* Screen Indicator */}
                <div className="w-full max-w-4xl h-8 bg-white/20 rounded-t-xl flex items-center justify-center text-sm text-white/50 tracking-widest uppercase shadow-md">
                    Screen
                </div>

                {/* Seat Grid — scrollable so it never overflows the container */}
                <div className="w-full overflow-x-auto">
                    <SeatGrid showtimeId={id} />
                </div>

                {/* Legend */}
                <div className="flex flex-wrap justify-center gap-4 mt-2">
                    {legend.map(({ label, color }) => (
                        <div key={label} className="flex items-center gap-2 text-sm text-white/70">
                            <div className={`w-5 h-5 rounded ${color}`} />
                            <span>{label}</span>
                        </div>
                    ))}
                </div>
            </div>
        </div>
    );
}