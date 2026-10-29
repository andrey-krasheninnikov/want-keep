import { Progress as SharedProgress } from "@want-foundation/desyst-react";
export function Progress({
  label,
  value,
  description,
}: {
  label: string;
  value: number | null;
  description: string;
}) {
  return (
    <div className="wk-progress">
      <span className="wk-label">{label}</span>
      <SharedProgress
        label={label}
        value={value}
        aria-valuetext={description}
      />
      <p className="wk-description">{description}</p>
    </div>
  );
}
