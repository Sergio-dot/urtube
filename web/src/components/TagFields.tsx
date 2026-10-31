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
  { key: "artist", label: "Artist", placeholder: "Unknown Artist" },
  { key: "title", label: "Title", placeholder: "Untitled" },
  { key: "album", label: "Album", placeholder: "None (single)" },
];

const inputClass =
  "mt-1 block w-full rounded-lg border-0 px-3 py-2 text-sm text-gray-900 ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 dark:bg-white/5 dark:text-white dark:ring-white/10 dark:focus:ring-indigo-500";

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
      <div className="space-y-3">
        {fields.map(({ key, label, placeholder }) => (
          <div key={key}>
            <label
              htmlFor={`tag-${key}`}
              className="text-sm font-medium dark:text-gray-300"
            >
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
              className={inputClass}
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
        <p className="mt-2 text-xs text-amber-600 dark:text-amber-400">
          Couldn't read the video's details ({error}). Fill in what you know;
          empty fields are guessed when the download starts.
        </p>
      )}
    </div>
  );
}
