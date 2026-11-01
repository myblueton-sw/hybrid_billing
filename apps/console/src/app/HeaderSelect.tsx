import { useRef, useState } from "react";
import chevron from "../../../../packages/design-system/assets/icons/chevron-down.svg";
export default function HeaderSelect({
  value,
  options,
  onChange,
  label,
}: {
  value: string;
  options: { value: string; label: string }[];
  onChange: (value: string) => void;
  label: string;
}) {
  const selected = options.find((o) => o.value === value);
  const selectedText =
    value === "ko" ? "선택됨" : value === "ar" ? "محدد" : "Selected";
  const [open, setOpen] = useState(false),
    [active, setActive] = useState(0);
  const trigger = useRef<HTMLButtonElement>(null),
    list = useRef<HTMLDivElement>(null);
  function close() {
    setOpen(false);
    trigger.current?.focus();
  }
  function show() {
    setActive(
      Math.max(
        0,
        options.findIndex((o) => o.value === value),
      ),
    );
    setOpen(true);
  }
  return (
    <div
      className="header-select"
      onBlur={(e) => {
        if (!e.currentTarget.contains(e.relatedTarget as Node | null))
          setOpen(false);
      }}
    >
      <button
        id="language"
        ref={trigger}
        className="header-control"
        type="button"
        aria-label={`${label}: ${selected?.label ?? ""}`}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls="language-options"
        onClick={() => (open ? close() : show())}
        onKeyDown={(e) => {
          if (e.key === "ArrowDown" || e.key === "ArrowUp") {
            e.preventDefault();
            show();
          }
        }}
      >
        <span className="language-code" aria-hidden="true">
          {value.toUpperCase()}
        </span>
        <span className="language-value">{selected?.label}</span>
        <img className="icon" src={chevron} alt="" />
      </button>
      {open && (
        <div
          id="language-options"
          className="header-options"
          role="listbox"
          aria-label={label}
          ref={list}
          onKeyDown={(e) => {
            if (e.key === "Escape") {
              e.preventDefault();
              close();
            } else if (
              ["ArrowDown", "ArrowUp", "Home", "End"].includes(e.key)
            ) {
              e.preventDefault();
              const next =
                e.key === "Home"
                  ? 0
                  : e.key === "End"
                    ? options.length - 1
                    : (active +
                        (e.key === "ArrowDown" ? 1 : -1) +
                        options.length) %
                      options.length;
              setActive(next);
            }
          }}
        >
          {options.map((o, i) => (
            <button
              type="button"
              key={o.value}
              role="option"
              aria-label={o.label}
              aria-selected={o.value === value}
              tabIndex={active === i ? 0 : -1}
              ref={(node) => {
                if (node && active === i) node.focus();
              }}
              onClick={() => {
                onChange(o.value);
                close();
              }}
            >
              <span className="language-option-label">
                <bdi>{o.label}</bdi>
                <small aria-hidden="true">{o.value.toUpperCase()}</small>
              </span>
              {o.value === value && (
                <span className="language-selected" aria-hidden="true">
                  {selectedText}
                </span>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
