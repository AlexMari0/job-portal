import Image from "next/image";

export default function Footer() {
  return (
    <div className="container px-4 2xl:px-20 mx-auto flex items-center justify-between gap-4 py-3 mt-20">
      <Image src="/assets/logo.svg" alt="Job Portal" width={160} height={40} />
      <p className="flex-1 border-l border-gray-400 pl-4 text-sm text-gray-500 max-sm:hidden">
        Copyright @AlexMario | All right reserved
      </p>
      <div className="flex gap-2.5">
        {["facebook", "twitter", "instagram"].map((name) => (
          <Image key={name} src={`/assets/${name}_icon.svg`} alt={name} width={38} height={38} />
        ))}
      </div>
    </div>
  );
}
