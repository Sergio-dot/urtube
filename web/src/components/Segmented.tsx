interface SegmentedProps<T extends string> {
  /** Names the group for assistive technology and groups the radios. */
  legend: string;
  name: string;
  options: readonly { value: T; label: string }[];
  value: T;
  onChange: (value: T) => void;
  /** Stretch to the container's width, with equal segments. */
  fill?: boolean;
}

/**
 * A one-of-few choice shown as adjoining buttons. Built on real radio inputs
 * so arrow keys, form semantics and screen readers work without extra code.
 */
export default function Segmented<T extends string>({
  legend,
  name,
  options,
  value,
  onChange,
  fill = false,
}: SegmentedProps<T>) {
  return (
    <fieldset
      className={`${fill ? "flex" : "inline-flex"} rounded-lg bg-raised p-1`}
    >
      <legend className="sr-only">{legend}</legend>
      {options.map((option) => (
        <label key={option.value} className={fill ? "flex-1" : undefined}>
          <input
            type="radio"
            name={name}
            value={option.value}
            checked={value === option.value}
            onChange={() => onChange(option.value)}
            className="peer sr-only"
          />
          <span className="flex h-9 cursor-pointer items-center justify-center rounded-md px-3.5 text-sm font-medium text-muted transition-colors duration-150 select-none peer-checked:bg-surface peer-checked:text-fg peer-checked:shadow-sm peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-accent-text hover:text-fg">
            {option.label}
          </span>
        </label>
      ))}
    </fieldset>
  );
}
