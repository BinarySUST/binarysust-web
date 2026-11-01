import { QueryClient, QueryClientProvider, useQuery } from "@tanstack/react-query";
import { Link, Outlet, useRouterState } from "@tanstack/react-router";
import { AnimatePresence, motion, useReducedMotion } from "motion/react";
import { useEffect, useState } from "react";
import { Icon } from "~/components/Icon";
import { Logo } from "@binarysust/ui/components/logo";
import { ThemeToggle } from "~/components/ThemeToggle";
import { type Account, ApiError, type Me, api, loginUrl, unwrap, warmApi } from "@binarysust/api-client";
import { Avatar, AvatarFallback, AvatarImage } from "@binarysust/ui/components/avatar";
import { Button } from "@binarysust/ui/components/button";
import { Sheet, SheetContent, SheetTitle, SheetTrigger } from "@binarysust/ui/components/sheet";
import { MeContext } from "~/dashboard/me";
import { visibleSections } from "~/dashboard/sections";
import { ErrorBox, WakingBanner } from "~/dashboard/ui";

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

  const current = path.replace(/^\/dashboard\/?/, "").split("/")[0] ?? "";

  async function signOut() {
    await api.POST("/api/v1/auth/logout").catch(() => undefined);
    window.location.assign("/");
  }

  const navigation = <DashNav me={me.data} current={current} onNavigate={() => setNavOpen(false)} onSignOut={() => void signOut()} />;

  return (
    <MeContext.Provider value={me.data}>
      <div className="min-h-dvh lg:grid lg:grid-cols-[17rem_1fr]">
        <aside className="border-b border-line bg-ground-2 lg:sticky lg:top-0 lg:h-dvh lg:border-r lg:border-b-0">
          <div className="flex h-16 items-center gap-2 px-4">
            <Link to="/" className="flex items-center gap-2 text-ink no-underline">
              <Logo className="h-9" />
              <span className="sign border-l border-line pl-2">Dashboard</span>
            </Link>
            <div className="ml-auto flex items-center lg:hidden">
              <ThemeToggle />
              <Sheet open={navOpen} onOpenChange={setNavOpen}>
                <SheetTrigger render={<Button variant="ghost" size="icon" aria-label="Open menu" />}>
                  <Icon name="menu" size={24} />
                </SheetTrigger>
                <SheetContent side="left" className="w-[min(20rem,85vw)] overflow-y-auto bg-ground-2 px-3 pt-16 pb-4">
                  <SheetTitle className="sr-only">Dashboard menu</SheetTitle>
                  {navigation}
                </SheetContent>
              </Sheet>
            </div>
          </div>
          <div className="hidden px-3 pb-4 lg:block">{navigation}</div>
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

/** The section list (from sections.tsx), shown in the sidebar on wide screens and in a sheet on narrow ones. */
function DashNav({ me, current, onNavigate, onSignOut }: { me: Me; current: string; onNavigate: () => void; onSignOut: () => void }) {
  const item = "flex min-h-11 items-center gap-3 rounded-lg px-3 no-underline";
  return (
    <nav aria-label="Dashboard">
      <Identity account={me.account} roles={me.roles} />
      <ul className="space-y-1">
        {visibleSections(me).map((s) => {
          const active = s.path === current;
          return (
            <li key={s.path}>
              <Link
                to="/dashboard/$"
                params={{ _splat: s.path }}
                onClick={onNavigate}
                aria-current={active ? "page" : undefined}
                className={`${item} font-medium ${active ? "bg-ink text-ground" : "text-ink-2 hover:bg-panel hover:text-ink"}`}
              >
                <Icon name={s.icon} size={19} /> {s.title}
              </Link>
            </li>
          );
        })}
      </ul>
      <div className="mt-6 space-y-1 border-t border-line pt-4">
        <a href="/" className={`${item} text-ink-2 hover:bg-panel hover:text-ink`}>
          <Icon name="external" size={19} /> Public site
        </a>
        <Button
          variant="ghost"
          onClick={onSignOut}
          className="w-full justify-start gap-3 rounded-lg px-3 text-base font-normal text-ink-2 hover:text-ink hover:not-disabled:bg-panel"
        >
          <Icon name="logout" size={19} /> Sign out
        </Button>
        <div className="hidden px-1 pt-2 lg:block">
          <ThemeToggle />
        </div>
      </div>
    </nav>
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
      <Avatar className="size-10">
        {account.avatarUrl && <AvatarImage src={account.avatarUrl} alt="" referrerPolicy="no-referrer" />}
        <AvatarFallback className="bg-ink font-semibold text-ground">{initials}</AvatarFallback>
      </Avatar>
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
