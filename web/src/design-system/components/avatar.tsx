export function Avatar({
  name,
  initials,
  src,
}: {
  name: string;
  initials: string;
  src?: string;
}) {
  return (
    <span className="wk-avatar" role="img" aria-label={name}>
      <span aria-hidden="true">{initials}</span>
      {src && (
        <img
          key={src}
          src={src}
          alt=""
          onError={(event) => {
            event.currentTarget.hidden = true;
          }}
        />
      )}
    </span>
  );
}
