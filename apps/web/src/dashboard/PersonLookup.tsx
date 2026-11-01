// Pickers for people referenced by handle (talk speaker, event speakers). Suggestions come from
// GET /people/lookup, which any signed-in user may call, so an event manager without
// people.manage can still find a speaker.
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useEffect, useId, useState } from "react";
import { Icon } from "~/components/Icon";
import { type Schemas, api, unwrap } from "@binarysust/api-client";
import { Field } from "./ui";
import { Input } from "@binarysust/ui/components/input";
import { Button } from "@binarysust/ui/components/button";

type PersonRef = Schemas["people.PersonRef"];

function useLookup(text: string) {
  const [q, setQ] = useState(text);
  useEffect(() => {
    const t = setTimeout(() => setQ(text.trim().replace(/^@/, "")), 200);
    return () => clearTimeout(t);
  }, [text]);
  return useQuery({
    queryKey: ["people-lookup", q],
    queryFn: async () => unwrap(await api.GET("/api/v1/people/lookup", { params: { query: { q } } })),
    staleTime: 60_000,
    placeholderData: keepPreviousData,
  });
}

function Options({ id, people }: { id: string; people: PersonRef[] | undefined }) {
  return (
    <datalist id={id}>
      {people?.map((p) => (
        <option key={p.handle} value={p.handle}>
          {p.name}
        </option>
      ))}
    </datalist>
  );
}

/** One handle, typed or picked from suggestions. */
export function HandlePicker({
  label,
  hint,
  error,
  value,
  onChange,
}: {
  label: string;
  hint?: string;
  error?: string;
  value: string;
  onChange: (handle: string) => void;
}) {
  const listId = useId();
  const found = useLookup(value);
  const match = found.data?.find((p) => p.handle === value);
  return (
    <Field label={label} hint={match ? `${match.name}${hint ? ` · ${hint}` : ""}` : hint} error={error}>
      <Input
        
        list={listId}
        autoComplete="off"
        value={value}
        onChange={(e) => onChange(e.target.value.trim().replace(/^@/, "").toLowerCase())}
 />
      <Options id={listId} people={found.data} />
    </Field>
  );
}

/** An ordered list of handles: pick to add, × to remove. */
export function HandleListPicker({
  label,
  hint,
  error,
  value,
  onChange,
}: {
  label: string;
  hint?: string;
  error?: string;
  value: string[];
  onChange: (handles: string[]) => void;
}) {
  const listId = useId();
  const [text, setText] = useState("");
  const found = useLookup(text);
  const [names, setNames] = useState<Record<string, string>>({});
  const add = (handle: string) => {
    const p = found.data?.find((x) => x.handle === handle);
    if (!p) return false;
    if (!value.includes(p.handle)) onChange([...value, p.handle]);
    setNames((n) => ({ ...n, [p.handle]: p.name }));
    setText("");
    return true;
  };
  return (
    <div>
      <Field label={label} hint={hint ?? "Type a name or handle and pick from the list."} error={error}>
        <Input
          
          list={listId}
          autoComplete="off"
          value={text}
          onChange={(e) => {
            const v = e.target.value;
            // Picking a suggestion replaces the text in one go; typing adds only on Enter.
            const ev = e.nativeEvent;
            const picked = !(ev instanceof InputEvent) || ev.inputType === "insertReplacementText";
            if (!(picked && add(v.trim().replace(/^@/, "")))) setText(v);
          }}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              add(text.trim().replace(/^@/, ""));
            }
          }}
 />
        <Options id={listId} people={found.data?.filter((p) => !value.includes(p.handle))} />
      </Field>
      {value.length > 0 && (
        <ul className="mt-3 flex flex-wrap gap-2" aria-label={`${label} chosen`}>
          {value.map((h) => (
            <li key={h} className="inline-flex items-center gap-1 rounded-full border border-line bg-panel py-0.5 pr-1 pl-3 text-sm">
              {names[h] ? `${names[h]} ` : ""}
              <span className="text-ink-3">@{h}</span>
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                className="ml-1 size-7 rounded-full"
                aria-label={`Remove @${h}`}
                onClick={() => onChange(value.filter((x) => x !== h))}
              >
                <Icon name="close" size={14} />
              </Button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
