import "./Brand.css";

// Brand is the app's logo and name. Change the name here, in index.html
// and in public/manifest.webmanifest.
export function Brand({ large = false }: { large?: boolean }) {
  return (
    <span className={large ? "brand-mark large" : "brand-mark"}>
      <img src="/icon.svg" alt="" />
      TAI<span>Go</span>
    </span>
  );
}
