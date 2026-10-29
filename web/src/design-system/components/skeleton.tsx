import { Skeleton as SharedSkeleton } from "@want-foundation/desyst-react";
export function Skeleton({ label }: { label: string }) {
  return (
    <div className="wk-skeleton" role="status" aria-label={label}>
      {[1, 2, 3].map((index) => (
        <SharedSkeleton key={index} />
      ))}
    </div>
  );
}
