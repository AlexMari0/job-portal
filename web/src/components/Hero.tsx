import Image from "next/image";
import HeroSearch from "./HeroSearch";

const trustedBy = [
  ["microsoft_logo.svg", "Microsoft"],
  ["walmart_logo.svg", "Walmart"],
  ["accenture_logo.png", "Accenture"],
  ["samsung_logo.png", "Samsung"],
  ["amazon_logo.png", "Amazon"],
  ["adobe_logo.png", "Adobe"],
] as const;

export default function Hero() {
  return (
    <div className="container 2xl:px-20 mx-auto my-10">
      <div className="bg-linear-to-r from-purple-800 to-purple-950 text-white py-16 text-center mx-2 rounded-xl">
        <h2 className="text-2xl md:text-3xl lg:text-4xl font-medium mb-4">Over 10.000+ jobs to apply</h2>
        <p className="mb-8 max-w-xl mx-auto text-sm font-light px-5">
          Your Next Big Career Move Starts Right Here - Explore the Best Job Opportunities and Take the First
          Step Toward Your Future!
        </p>
        <HeroSearch />
      </div>

      <div className="border border-gray-300 shadow-md mx-2 mt-5 p-6 rounded-md flex">
        <div className="flex justify-center gap-10 lg:gap-16 flex-wrap">
          <p className="font-medium">Trusted by</p>
          {trustedBy.map(([file, name]) => (
            <Image key={file} className="h-6 w-auto" src={`/assets/${file}`} alt={name} width={96} height={24} />
          ))}
        </div>
      </div>
    </div>
  );
}
