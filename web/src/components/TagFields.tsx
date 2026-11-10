type TagField = "artist" | "title" | "album";

interface TagFieldsProps {
  values: Record<TagField, string>;
  onChange: (field: TagField, value: string) => void;
  /** Artist folders already in the library, offered as suggestions. */
  knownArtists: string[];
  loading: boolean;
  error: string | null;
}

const fields: { key: TagField; label: string; placeholder: string }[] = [
  { key: "title", label: "Title", placeholder: "Untitled" },
  { key: "artist", label: "Artist", placeholder: "Unknown Artist" },
  { key: "album", label: "Album", placeholder: "None (single)" },
];

/** Editable artist, title and album for a download. */
export default function TagFields({
  values,
  onChange,
  knownArtists,
  loading,
  error,
}: TagFieldsProps) {
  return (
    <div>
      <div className="grid gap-3 sm:grid-cols-2">
        {fields.map(({ key, label, placeholder }) => (
          <div key={key} className={key === "title" ? "sm:col-span-2" : ""}>
            <label htmlFor={`tag-${key}`} className="label">
              {label}
            </label>
            <input
              id={`tag-${key}`}
              type="text"
              autoComplete="off"
              value={values[key]}
              onChange={(e) => onChange(key, e.target.value)}
              placeholder={loading ? "Detecting…" : placeholder}
              list={key === "artist" ? "known-artists" : undefined}
              className="input mt-1.5"
            />
          </div>
        ))}
        <datalist id="known-artists">
          {knownArtists.map((a) => (
            <option key={a} value={a} />
          ))}
        </datalist>
      </div>
      {error && (
        <p role="status" className="mt-2 text-xs leading-relaxed text-warn">
          Couldn't read the video's details ({error}). Fill in what you know;
          empty fields are guessed when the download starts.
        </p>
      )}
    </div>
  );
}
