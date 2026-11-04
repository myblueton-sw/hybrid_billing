import { Fragment } from "react";
export default function PercentText({ children }: { children: string }) {
  return (
    <>
      {children.split(/(100%|\(%\))/).map((part, i) => (
        <Fragment key={i}>
          {part === "100%" || part === "(%)" ? (
            <bdi dir="ltr">{part}</bdi>
          ) : (
            part
          )}
        </Fragment>
      ))}
    </>
  );
}
