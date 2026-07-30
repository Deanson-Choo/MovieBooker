import ShowtimeCard from "@/components/ShowtimeCard";
import { getMovies, type MovieCatalogItem } from "@/services/catalog/getMovies";
import type { ErrorResponse } from "@/types/ErrorResponse";

export default async function Home() {

  let movies: MovieCatalogItem[] | undefined = undefined;
  try {
    movies = await getMovies();
  } catch (error) {
    const err = error as ErrorResponse;
    console.error(`${err.error}: ${err.details}`);
  }

  return (
    <div className="flex flex-col items-center justify-center p-16 w-full">
      <p className="text-xl font-bold mb-10 text-primary underline underline-offset-4 decoration-gray-400">Now Showing</p>
      {Array.isArray(movies) ? (
        <div className="flex w-full flex-col gap-4">
          {movies.map((movie) => (
            <ShowtimeCard
              key={movie.id}
              item={movie}
            />
          ))}
        </div>
      ) : (
        <p className="text-sm text-red-600">Something went wrong. Please try again later.</p>
      )}
    </div>
  );
}
