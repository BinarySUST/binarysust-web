// Sponsors' API queries for the dashboard: one query key per resource, one hook per call. Public pages never
// call the API; they read the snapshot through `content.sponsors()` (src/content/).
import { type Schemas, api, unwrap } from "@binarysust/api-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { invalidate } from "~/dashboard/ui";

export type SponsorIn = Schemas["sponsors.In"];
export type Sponsor = Schemas["sponsors.Sponsor"];

export const sponsorKeys = {
  all: ["sponsors"] as const,
  one: (id: string | null) => ["sponsors", id] as const,
};

export function useSponsors() {
  return useQuery({ queryKey: sponsorKeys.all, queryFn: async () => unwrap(await api.GET("/api/v1/sponsors")) });
}

export function useSponsor(id: string | null) {
  return useQuery({
    queryKey: sponsorKeys.one(id),
    queryFn: async () => unwrap(await api.GET("/api/v1/sponsors/{id}", { params: { path: { id: id ?? "" } } })),
    enabled: id !== null,
  });
}

/** Create (id null) or update. The list updates optimistically; every write republishes the site. */
export function useSaveSponsor(id: string | null, onCreated: (s: Sponsor) => void) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: SponsorIn) =>
      unwrap(
        id
          ? await api.PUT("/api/v1/sponsors/{id}", { params: { path: { id } }, body })
          : await api.POST("/api/v1/sponsors", { body }),
      ),
    onMutate: (body) => {
      if (id) qc.setQueryData<Sponsor[]>(sponsorKeys.all, (rows) => rows?.map((r) => (r.id === id ? { ...r, ...body } : r)));
    },
    onSuccess: (s) => {
      qc.setQueryData(sponsorKeys.one(s.id), s);
      if (!id) onCreated(s);
    },
    onSettled: () => invalidate(qc, sponsorKeys.all),
  });
}

export async function deleteSponsor(id: string) {
  return unwrap(await api.DELETE("/api/v1/sponsors/{id}", { params: { path: { id } } }));
}
