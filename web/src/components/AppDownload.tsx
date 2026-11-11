import Image from "next/image";

export default function AppDownload() {
  return (
    <div className="container px-4 2xl:px-20 mx-auto my-20">
      <div className="relative bg-linear-to-r from-violet-50 to-purple-50 p-12 sm:p-24 lg:p-32 rounded-lg">
        <div>
          <h1 className="text-2xl sm:text-4xl font-bold mb-8 max-w-md">
            Download Mobile App For Better Experience
          </h1>
          <div className="flex gap-4">
            <Image className="h-12 w-auto" src="/assets/play_store.svg" alt="Get it on Google Play" width={160} height={48} />
            <Image className="h-12 w-auto" src="/assets/app_store.svg" alt="Download on the App Store" width={160} height={48} />
          </div>
        </div>
        <Image
          className="absolute w-80 right-0 bottom-0 mr-32 max-lg:hidden"
          src="/assets/app_main_img.png"
          alt=""
          width={320}
          height={400}
        />
      </div>
    </div>
  );
}
