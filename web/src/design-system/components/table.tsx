import { Table as SharedTable } from "@want-foundation/desyst-react";
import type { ComponentProps } from "react";
export { TableHead, TableCell } from "@want-foundation/desyst-react";
export function Table({
  label,
  ...props
}: ComponentProps<typeof SharedTable> & { label: string }) {
  return (
    <section className="wk-table-region" aria-label={label}>
      <SharedTable aria-label={label} className="wk-table" {...props} />
    </section>
  );
}
