import { QueryClient, QueryClientProvider, useQuery } from "@tanstack/react-query";
import { Link, Outlet, useRouterState } from "@tanstack/react-router";
import { AnimatePresence, motion, useReducedMotion } from "motion/react";
import { useEffect, useState } from "react";
import { Icon } from "~/components/Icon";
import { Mark } from "~/components/Mark";
import { ThemeToggle } from "~/components/ThemeToggle";
import { type Account, ApiError, type Me, api, loginUrl, unwrap, warmApi } from "@binarysust/api-client";
import { MeContext } from "./me";
import { visibleSections } from "./sections";
import { ErrorBox, WakingBanner } from "./ui";

export function DashboardRoot() {
  const [client] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 30_000,
            // A sleeping Render instance answers slowly, not with an error; only retry real failures once.
            retry: (n, e) => !(e instanceof ApiError && e.status < 500) && n < 1,
            refetchOnWindowFocus: false,
          },
        },
      }),
  );
  useEffect(() => warmApi(), []);
  return (
    <QueryClientProvider client={client}>
      <Shell />
    </QueryClientProvider>
  );
}

function Shell() {
  const path = useRouterState({ select: (s) => s.location.pathname });
  const [navOpen, setNavOpen] = useState(false);
  const reduced = useReducedMotion();
  const me = useQuery({
    queryKey: ["me"],
    queryFn: async (): Promise<Me> => {
      const auth = unwrap(await api.GET("/api/v1/auth/me"));
      // /auth/me answers 200 with an empty userId when nobody is signed in.
      if (!auth.userId) throw new ApiError(401, undefined);
      const account = unwrap(await api.GET("/api/v1/me"));
      return { ...auth, account };
    },
    retry: (n, e) => !(e instanceof ApiError && e.status === 401) && n < 2,
  });

  useEffect(() => {
    if (me.error instanceof ApiError && me.error.status === 401) {
      window.location.replace(loginUrl(window.location.pathname + window.location.search));
    }
  }, [me.error]);

  if (!me.data) {
    return (
      <main id="main" className="mx-auto max-w-xl px-4 py-24">
        <WakingBanner />
        {me.error && !(me.error instanceof ApiError && me.error.status === 401) ? (
          <ErrorBox error={me.error} retry={() => void me.refetch()} />
        ) : (
          <p className="text-ink-2" role="status">
            Checking who you are…
          </p>
        )}
      </main>
    );
  }

  const nav = visibleSections(me.data);
  const current = path.replace(/^\/dashboard\/?/, "").split("/")[0] ?? "";

  async function signOut() {
    await api.POST("/api/v1/auth/logout").catch(() => undefined);
    window.location.assign("/");
  }

  return (
    <MeContext.Provider value={me.data}>
      <div className="min-h-dvh lg:grid lg:grid-cols-[17rem_1fr]">
        <aside className="border-b border-line bg-ground-2 lg:sticky lg:top-0 lg:h-dvh lg:border-r lg:border-b-0">
          <div className="flex h-16 items-center gap-2 px-4">
            <Link to="/" className="flex items-center gap-2 text-ink no-underline">
              <Mark className="h-4 w-[70px] text-ink-2" />
              <span className="sign">Dashboard</span>
            </Link>
            <div className="ml-auto flex items-center lg:hidden">
              <ThemeToggle />
              <button
                type="button"
                className="inline-flex size-11 items-center justify-center rounded-lg"
                aria-expanded={navOpen}
                aria-controls="dash-nav"
                aria-label={navOpen ? "Close menu" : "Open menu"}
                onClick={() => setNavOpen((o) => !o)}
              >
                <Icon name={navOpen ? "close" : "menu"} size={24} />
              </button>
            </div>
          </div>
          <nav id="dash-nav" aria-label="Dashboard" className={`${navOpen ? "block" : "hidden"} px-3 pb-4 lg:block`}>
            <Identity account={me.data.account} roles={me.data.roles} />
            <ul className="space-y-1">
              {nav.map((s) => {
                const active = s.path === current;
                return (
                  <li key={s.path}>
                    <Link
                      to="/dashboard/$"
                      params={{ _splat: s.path }}
                      onClick={() => setNavOpen(false)}
                      aria-current={active ? "page" : undefined}
                      className={`flex min-h-11 items-center gap-3 rounded-lg px-3 font-medium no-underline ${active ? "bg-ink text-ground" : "text-ink-2 hover:bg-panel hover:text-ink"}`}
                    >
                      <Icon name={s.icon} size={19} /> {s.title}
                    </Link>
                  </li>
                );
              })}
            </ul>
            <div className="mt-6 space-y-1 border-t border-line pt-4">
              <a href="/" className="flex min-h-11 items-center gap-3 rounded-lg px-3 text-ink-2 no-underline hover:bg-panel hover:text-ink">
                <Icon name="external" size={19} /> Public site
              </a>
              <button type="button" onClick={signOut} className="flex min-h-11 w-full items-center gap-3 rounded-lg px-3 text-left text-ink-2 hover:bg-panel hover:text-ink">
                <Icon name="logout" size={19} /> Sign out
              </button>
              <div className="hidden px-1 pt-2 lg:block">
                <ThemeToggle />
              </div>
            </div>
          </nav>
        </aside>
        <main id="main" className="min-w-0 px-4 py-8 sm:px-8 lg:py-12">
          <WakingBanner />
          <AnimatePresence mode="wait" initial={false}>
            <motion.div
              key={current}
              initial={reduced ? false : { opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              exit={reduced ? undefined : { opacity: 0, y: -4 }}
              transition={{ duration: 0.22, ease: [0.16, 1, 0.3, 1] }}
            >
              <Outlet />
            </motion.div>
          </AnimatePresence>
        </main>
      </div>
    </MeContext.Provider>
  );
}

function Identity({ account, roles }: { account: Account; roles: string[] }) {
  const name = account.name || account.email;
  const initials = name
    .split(/[\s@.]+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase())
    .join("");
  return (
    <div className="mb-4 flex items-center gap-3 rounded-lg border border-line bg-panel p-3">
      {account.avatarUrl ? (
        <img src={account.avatarUrl} alt="" className="size-10 shrink-0 rounded-full object-cover" referrerPolicy="no-referrer" />
      ) : (
        <span className="grid size-10 shrink-0 place-items-center rounded-full bg-ink font-semibold text-ground" aria-hidden="true">
          {initials}
        </span>
      )}
      <div className="min-w-0">
        <p className="truncate font-semibold text-ink">{name}</p>
        <p className="truncate text-sm text-ink-3">
          {account.personHandle ? (
            <a href={`/people/${account.personHandle}`} className="link">
              @{account.personHandle}
            </a>
          ) : (
            "No member profile yet"
          )}
          {roles.length > 0 && ` · ${roles.join(", ").replace(/_/g, " ")}`}
        </p>
      </div>
    </div>
  );
}
