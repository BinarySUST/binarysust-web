import { Link } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { Icon } from "~/components/Icon";
import { Logo } from "@binarysust/ui/components/logo";
import { ThemeToggle } from "~/components/ThemeToggle";
import { Button, buttonVariants } from "@binarysust/ui/components/button";
import { Sheet, SheetContent, SheetTitle, SheetTrigger } from "@binarysust/ui/components/sheet";

export const NAV = [
  { to: "/events", label: "Events" },
  { to: "/people", label: "People" },
  { to: "/committee", label: "Committee" },
  { to: "/blog", label: "Blog" },
  { to: "/talks", label: "Talks" },
  { to: "/sponsors", label: "Sponsors" },
  { to: "/about", label: "About" },
] as const;

export function SiteHeader() {
  const [open, setOpen] = useState(false);
  const [scrolled, setScrolled] = useState(false);

  // Compact once the page has scrolled; a sentinel-free check that runs only on scroll.
  useEffect(() => {
    const on = () => setScrolled(window.scrollY > 8);
    on();
    window.addEventListener("scroll", on, { passive: true });
    return () => window.removeEventListener("scroll", on);
  }, []);

  return (
    <header
      id="site-header"
      data-scrolled={scrolled || undefined}
      className="group/header sticky top-0 z-30 border-b border-line bg-ground transition-[border-color] duration-200 data-[scrolled]:border-line-strong"
    >
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:absolute focus:top-3 focus:left-4 focus:z-50 focus:rounded-[4px] focus:bg-ink focus:px-3 focus:py-2 focus:text-ground"
      >
        Skip to content
      </a>
      <div className="mx-auto flex h-16 max-w-7xl items-center gap-4 px-4 transition-[height] duration-200 ease-out group-data-[scrolled]/header:h-13 sm:px-6">
        <Link to="/" className="flex shrink-0 items-center self-stretch text-ink no-underline" aria-label="Binary SUST, home">
          <Logo />
        </Link>
        <nav aria-label="Main" className="ml-auto hidden lg:block">
          <ul className="flex items-center">
            {NAV.map((n) => (
              <li key={n.to}>
                <Link
                  to={n.to}
                  className="relative block px-3 py-2 text-[0.9375rem] text-ink-2 no-underline hover:text-ink"
                  activeProps={{
                    className:
                      "!text-ink after:absolute after:inset-x-3 after:-bottom-[13px] after:h-px after:bg-ink group-data-[scrolled]/header:after:-bottom-[7px]",
                    "aria-current": "page",
                  }}
                >
                  {n.label}
                </Link>
              </li>
            ))}
          </ul>
        </nav>
        <div className="ml-auto flex items-center gap-1 lg:ml-3">
          <ThemeToggle />
          <a href="/login" className={buttonVariants({ variant: "outline", size: "sm", className: "hidden sm:inline-flex" })}>
            Sign in
          </a>
          <Sheet open={open} onOpenChange={setOpen}>
            <SheetTrigger
              render={<Button variant="ghost" size="icon" className="lg:hidden" aria-label="Open menu" />}
            >
              <Icon name="menu" size={22} />
            </SheetTrigger>
            <SheetContent
              side="top"
              showCloseButton={false}
              className="max-h-dvh gap-0 overflow-y-auto overscroll-contain border-line-strong bg-ground lg:hidden"
            >
              <div className="mx-auto flex h-16 w-full max-w-7xl items-center px-4 sm:px-6">
                <SheetTitle className="sr-only">Menu</SheetTitle>
                <Link to="/" onClick={() => setOpen(false)} className="flex items-center self-stretch text-ink no-underline" aria-label="Binary SUST, home">
                  <Logo />
                </Link>
                <Button variant="ghost" size="icon" className="ml-auto" aria-label="Close menu" onClick={() => setOpen(false)}>
                  <Icon name="close" size={22} />
                </Button>
              </div>
              <nav id="mobile-nav" aria-label="Main">
                <ul className="mx-auto max-w-7xl px-4 pb-4 sm:px-6">
                  {NAV.map((n) => (
                    <li key={n.to} className="border-b border-line">
                      <Link
                        to={n.to}
                        onClick={() => setOpen(false)}
                        className="flex min-h-12 items-center justify-between text-lg text-ink no-underline"
                        activeProps={{ "aria-current": "page", className: "font-semibold" }}
                      >
                        {n.label}
                        <Icon name="arrow" size={18} className="text-ink-3" />
                      </Link>
                    </li>
                  ))}
                  <li className="pt-4 sm:hidden">
                    <a href="/login" className={buttonVariants({ variant: "outline", className: "w-full" })}>
                      Sign in
                    </a>
                  </li>
                </ul>
              </nav>
            </SheetContent>
          </Sheet>
        </div>
      </div>
    </header>
  );
}
