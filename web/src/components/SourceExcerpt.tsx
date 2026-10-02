import type { Excerpt } from "../types";
import "./SourceExcerpt.css";

// SourceExcerpt shows the quoted fragment highlighted inside its source text.
export function SourceExcerpt({ excerpt }: { excerpt: Excerpt }) {
  return (
    <p className="excerpt">
      {excerpt.clipped_start && "…"}
      {excerpt.before}
      <mark>{excerpt.match}</mark>
      {excerpt.after}
      {excerpt.clipped_end && "…"}
    </p>
  );
}
